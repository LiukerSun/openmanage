package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"openmanage/backend/ai"
	"openmanage/backend/discourse"
	"openmanage/backend/docker"
	"openmanage/backend/handler"
	"openmanage/backend/middleware"
	"openmanage/backend/openclaw"
	"openmanage/backend/preferences"
)

func main() {
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatal("failed to create docker client:", err)
	}
	defer dockerClient.Close()

	mountPrefix := os.Getenv("MOUNT_PREFIX") // "/host" in container, "" in dev

	// JWT secret from environment, with fallback
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "openmanage-default-secret-change-in-production"
		log.Println("Warning: Using default JWT secret. Set JWT_SECRET environment variable in production.")
	}

	// Resolve template directory relative to executable
	execPath, _ := os.Executable()
	templateDir := filepath.Join(filepath.Dir(execPath), "templates")
	// Fallback: check if templates/ exists in working directory
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		templateDir = "templates"
	}

	// AI client for generating agent configs
	var aiClient *ai.Client
	glmAPIKey := os.Getenv("GLM_API_KEY")

	authH := handler.NewAuthHandler(jwtSecret)
	prefsStore, err := preferences.NewStore(mountPrefix)
	if err != nil {
		log.Fatal("failed to create preferences store:", err)
	}

	// Try to init AI client from saved model provider preferences first
	if p, err := prefsStore.Get(); err == nil {
		if mp := p.ActiveProvider(); mp != nil {
			aiClient = ai.NewClientWithProvider(mp.BaseURL, mp.APIKey, mp.Model)
			log.Printf("AI config generation enabled (provider: %s, model: %s)", mp.Name, mp.Model)
		}
	}
	// Fallback to GLM_API_KEY env var
	if aiClient == nil && glmAPIKey != "" {
		aiClient = ai.NewClient(glmAPIKey)
		log.Println("AI config generation enabled (GLM_API_KEY env)")
	}

	containerH := &handler.ContainerHandler{Docker: dockerClient, TemplateDir: templateDir, MountPrefix: mountPrefix, AI: aiClient, GLMAPIKey: glmAPIKey, Prefs: prefsStore}

	// Discourse client (lazy init from preferences)
	if p, err := prefsStore.Get(); err == nil && p.DiscourseURL != "" && p.DiscourseAPIKey != "" {
		containerH.Discourse = discourse.NewClient(p.DiscourseURL, p.DiscourseAPIKey)
		log.Printf("Discourse integration enabled (%s)", p.DiscourseURL)
	}
	logsH := &handler.LogsHandler{Docker: dockerClient}
	filesH := &handler.FilesHandler{Docker: dockerClient, MountPrefix: mountPrefix}
	convH := &handler.ConversationsHandler{Docker: dockerClient, MountPrefix: mountPrefix}
	openclawClient := &openclaw.Client{Docker: dockerClient, MountPrefix: mountPrefix}
	chatH := &handler.ChatHandler{OpenClaw: openclawClient, Docker: dockerClient}
	cronH := &handler.CronHandler{OpenClaw: openclawClient, Docker: dockerClient, AI: aiClient}

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// WebSocket handler (authenticates internally via query param / cookie)
	wsH := handler.NewWSHandler(dockerClient, []byte(jwtSecret))

	// Public routes - no authentication required
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/logout", authH.Logout)
	r.Get("/api/auth/status", authH.Status)
	r.Get("/api/ws", wsH.Handle)

	// Protected routes - require authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth([]byte(jwtSecret)))

		r.Post("/api/create-container", containerH.Create)
		r.Post("/api/batch-create", containerH.BatchCreate)
		r.Post("/api/batch/chat", chatH.BatchChat)
		r.Post("/api/auth/change-password", authH.ChangePassword)
		r.Put("/api/auth/password", authH.ChangePassword)
		r.Get("/api/auth/me", authH.Me)

		prefsH := &handler.PreferencesHandler{Store: prefsStore}
		r.Get("/api/preferences", prefsH.Get)
		r.Put("/api/preferences", prefsH.Save)
		r.Post("/api/models/probe", prefsH.ProbeModels)

		templateH := &handler.TemplateHandler{TemplateDir: templateDir}
		r.Get("/api/templates", templateH.List)
		r.Post("/api/templates", templateH.Create)
		r.Get("/api/templates/*", templateH.Read)
		r.Put("/api/templates/*", templateH.Write)
		r.Delete("/api/templates/*", templateH.Delete)

		r.Route("/api/containers", func(r chi.Router) {
			r.Get("/", containerH.List)
			r.Get("/{id}", containerH.Get)
			r.Delete("/{id}", containerH.Delete)
			r.Put("/{id}", containerH.Update)
			r.Post("/{id}/start", containerH.Start)
			r.Post("/{id}/stop", containerH.Stop)
			r.Post("/{id}/restart", containerH.Restart)
			r.Get("/{id}/stats", containerH.Stats)
			r.Get("/{id}/logs", logsH.Stream)
			r.Get("/{id}/files", filesH.List)
			r.Get("/{id}/files/*", filesH.Read)
			r.Put("/{id}/files/*", filesH.Write)
			r.Get("/{id}/conversations", convH.List)
			r.Get("/{id}/conversations/{sid}", convH.Get)
			r.Post("/{id}/chat", chatH.Chat)
			r.Get("/{id}/forum-activity", containerH.ForumActivity)
			r.Get("/{id}/cron", cronH.List)
			r.Post("/{id}/cron", cronH.Add)
			r.Post("/{id}/cron/{jobId}/toggle", cronH.Toggle)
			r.Post("/{id}/cron/{jobId}/run", cronH.Run)
			r.Delete("/{id}/cron/{jobId}", cronH.Remove)
			r.Put("/{id}/heartbeat", cronH.UpdateHeartbeat)
			r.Post("/{id}/cron/generate", cronH.Generate)
		})
	})

	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("  %s %s\n", method, route)
		return nil
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("OpenManage backend listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
