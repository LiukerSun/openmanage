package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/ai"
	"openmanage/backend/discourse"
	"openmanage/backend/docker"
	"openmanage/backend/model"
	"openmanage/backend/preferences"
)

type ContainerHandler struct {
	Docker      *docker.Client
	TemplateDir  string // path to templates/ directory
	MountPrefix  string // "/host" in container, "" in dev
	AI          *ai.Client        // nil if GLM_API_KEY not set
	GLMAPIKey   string            // raw GLM API key, written into agent auth-profiles.json
	Prefs       *preferences.Store
	Discourse   *discourse.Client  // nil if Discourse not configured
}

func (h *ContainerHandler) List(w http.ResponseWriter, r *http.Request) {
	containers, err := h.Docker.ListOpenClawContainers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]model.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		mounts := make([]model.MountInfo, 0, len(c.Mounts))
		for _, m := range c.Mounts {
			mounts = append(mounts, model.MountInfo{
				Source:      m.Source,
				Destination: m.Destination,
				RW:          m.RW,
			})
		}
		result = append(result, model.ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
			Labels:  c.Labels,
			Mounts:  mounts,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ContainerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, err := h.Docker.InspectContainer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}

	mounts := make([]model.MountInfo, 0, len(info.Mounts))
	for _, m := range info.Mounts {
		mounts = append(mounts, model.MountInfo{
			Source:      m.Source,
			Destination: m.Destination,
			RW:          m.RW,
		})
	}

	created := int64(0)
	if t, err := time.Parse(time.RFC3339Nano, info.Created); err == nil {
		created = t.Unix()
	}

	result := model.ContainerInfo{
		ID:      info.ID[:12],
		Name:    strings.TrimPrefix(info.Name, "/"),
		Image:   info.Config.Image,
		State:   info.State.Status,
		Status:  info.State.Status,
		Created: created,
		Labels:  info.Config.Labels,
		Mounts:  mounts,
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ContainerHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Docker.StartContainer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (h *ContainerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Docker.StopContainer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *ContainerHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Docker.RestartContainer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (h *ContainerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stats, err := h.Docker.ContainerStats(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg})
}

// sendSSE writes a Server-Sent Event to the response writer and flushes.
func sendSSE(w http.ResponseWriter, flusher http.Flusher, step, status, message string) {
	evt := map[string]string{"step": step, "status": status, "message": message}
	data, _ := json.Marshal(evt)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (h *ContainerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateContainerRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Image == "" {
		req.Image = "fourplayers/openclaw:latest"
	}
	if req.DataPath == "" {
		req.DataPath = "/home/evan/.openclaw-" + req.Name
	}
	if req.Port == 0 {
		req.Port = 18789
	}

	// Set up SSE streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Step 1: Prepare directories
	sendSSE(w, flusher, "prepare", "running", "正在准备数据目录...")

	hostDataPath := h.MountPrefix + req.DataPath
	if err := os.MkdirAll(hostDataPath, 0700); err != nil {
		sendSSE(w, flusher, "prepare", "error", "创建数据目录失败: "+err.Error())
		return
	}
	for _, sub := range []string{
		"agents/main/sessions",
		"agents/main/agent",
		"conversations",
		"credentials",
		"canvas",
		"workspace",
	} {
		os.MkdirAll(filepath.Join(hostDataPath, sub), 0700)
	}
	// Write auth-profiles.json so the agent can authenticate with the AI provider
	if h.GLMAPIKey != "" {
		authProfile := fmt.Sprintf(`{"version":1,"profiles":{"zai:default":{"provider":"zai","type":"api_key","key":"%s"}}}`, h.GLMAPIKey)
		os.WriteFile(filepath.Join(hostDataPath, "agents/main/agent/auth-profiles.json"), []byte(authProfile), 0600)
	}
	sendSSE(w, flusher, "prepare", "done", "数据目录准备完成")

	// Step 1.5: Check port availability before templates & AI
	sendSSE(w, flusher, "port", "running", fmt.Sprintf("正在检测端口 %d 可用性...", req.Port))
	availablePort, err := findAvailablePort(req.Port, 10)
	if err != nil {
		sendSSE(w, flusher, "port", "error", "无法找到可用端口，请手动指定")
		return
	}
	if availablePort != req.Port {
		sendSSE(w, flusher, "port", "done", fmt.Sprintf("端口 %d 被占用，自动切换到 %d", req.Port, availablePort))
		req.Port = availablePort
	} else {
		sendSSE(w, flusher, "port", "done", fmt.Sprintf("端口 %d 可用", req.Port))
	}

	// Load Discourse settings from preferences (used by both templates and AI)
	discourseURL := "https://discourse.liukersun.com"
	discourseAPIKey := ""
	discourseUsername := req.Name
	if h.Prefs != nil {
		if p, err := h.Prefs.Get(); err == nil {
			if p.DiscourseURL != "" {
				discourseURL = p.DiscourseURL
			}
			if p.DiscourseAPIKey != "" {
				discourseAPIKey = p.DiscourseAPIKey
			}
		}
	}

	// Step 2: Copy templates
	sendSSE(w, flusher, "template", "running", "正在复制模板文件...")
	gatewayToken := generateToken()
	if h.TemplateDir != "" {
		log.Printf("Copying templates from %s to %s", h.TemplateDir, hostDataPath)
		now := time.Now().UTC().Format(time.RFC3339Nano)

		replacements := map[string]string{
			"{{NAME}}":               req.Name,
			"{{TIMESTAMP}}":          now,
			"{{WORKSPACE}}":          "/home/node/.openclaw/workspace",
			"{{PORT}}":               fmt.Sprintf("%d", req.Port),
			"{{GATEWAY_TOKEN}}":      gatewayToken,
			"{{DISCOURSE_URL}}":      discourseURL,
			"{{DISCOURSE_API_KEY}}":  discourseAPIKey,
			"{{DISCOURSE_USERNAME}}": discourseUsername,
		}
		if err := copyTemplates(h.TemplateDir, hostDataPath, replacements); err != nil {
			log.Printf("Template copy error: %v", err)
			sendSSE(w, flusher, "template", "error", "复制模板失败: "+err.Error())
			return
		}
	}
	sendSSE(w, flusher, "template", "done", "模板文件复制完成")

	// Step 2.5: Create Discourse account
	if h.Discourse != nil && discourseAPIKey != "" {
		sendSSE(w, flusher, "discourse", "running", "正在创建 Discourse 账号...")
		discoursePass := generateToken()[:16]
		email := fmt.Sprintf("%s@agents.openmanage.local", req.Name)
		if err := h.Discourse.CreateUser(req.Name, req.Name, email, discoursePass); err != nil {
			log.Printf("Discourse create user %s error: %v", req.Name, err)
			sendSSE(w, flusher, "discourse", "warn", "Discourse 账号创建失败（不影响容器创建）: "+err.Error())
		} else {
			sendSSE(w, flusher, "discourse", "done", fmt.Sprintf("Discourse 账号 %s 已就绪", req.Name))
		}
	}

	// Step 3: AI generate (optional, may be slow)
	if req.Description != "" && h.AI != nil {
		sendSSE(w, flusher, "ai", "running", "AI 正在连接...")

		// Load user preferences for AI context (with variables resolved)
		var userPrefs *preferences.UserPreferences
		if h.Prefs != nil {
			if p, err := h.Prefs.Get(); err == nil {
				userPrefs = p.Resolved()
			}
		}
		if userPrefs == nil {
			userPrefs = &preferences.UserPreferences{}
		}

		files, err := h.AI.GenerateStream(r.Context(), ai.GenerateRequest{
			Name:              req.Name,
			Description:       req.Description,
			Username:          userPrefs.Username,
			Style:             userPrefs.Style,
			Tools:             userPrefs.Tools,
			ExtraContext:      userPrefs.ExtraContext,
			DiscourseURL:      discourseURL,
			DiscourseUsername: req.Name,
		}, func(chars int) {
			sendSSE(w, flusher, "ai", "running", fmt.Sprintf("AI 正在生成配置... 已生成 %d 字符", chars))
		})
		if err != nil {
			log.Printf("AI generate error (falling back to defaults): %v", err)
			sendSSE(w, flusher, "ai", "warn", "AI 生成失败，将使用默认模板: "+err.Error())
		} else {
			for filename, content := range files {
				// .md files go to workspace/ where OpenClaw reads them
				dst := filepath.Join(hostDataPath, filename)
				if strings.HasSuffix(filename, ".md") {
					dst = filepath.Join(hostDataPath, "workspace", filename)
				}
				if err := os.WriteFile(dst, []byte(content), 0600); err != nil {
					log.Printf("Failed to write AI-generated %s: %v", filename, err)
				}
			}
			sendSSE(w, flusher, "ai", "done", fmt.Sprintf("AI 配置文件生成完成，共 %d 个文件", len(files)))
		}
	}

	// Step 4: Create container
	sendSSE(w, flusher, "container", "running", "正在创建 Docker 容器...")
	// Inject gateway env vars so supervisor passes them to openclaw gateway
	if req.Env == nil {
		req.Env = make(map[string]string)
	}
	req.Env["OPENCLAW_GATEWAY_TOKEN"] = gatewayToken
	req.Env["OPENCLAW_GATEWAY_PORT"] = fmt.Sprintf("%d", req.Port)
	req.Env["OPENCLAW_GATEWAY_BIND"] = "loopback"
	id, err := h.Docker.CreateContainer(r.Context(), req.Name, req.Image, req.DataPath, req.Port, req.Env)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "address already in use") {
			msg = fmt.Sprintf("端口 %d 已被占用，请更换其他端口", req.Port)
		}
		sendSSE(w, flusher, "container", "error", msg)
		return
	}

	// Step 5: Done
	sendSSE(w, flusher, "done", "done", id[:12])
}

func (h *ContainerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Docker.RemoveContainer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ContainerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req model.UpdateContainerRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name != "" {
		if err := h.Docker.RenameContainer(r.Context(), id, req.Name); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func copyTemplates(templateDir, dataPath string, replacements map[string]string) error {
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return err
	}

	// Ensure workspace directory exists for .md files
	workspaceDir := filepath.Join(dataPath, "workspace")
	os.MkdirAll(workspaceDir, 0700)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(templateDir, e.Name())

		// .md files go to workspace/ (where OpenClaw reads them from);
		// everything else (openclaw.json etc.) stays in the root.
		dst := filepath.Join(dataPath, e.Name())
		if strings.HasSuffix(e.Name(), ".md") {
			dst = filepath.Join(workspaceDir, e.Name())
		}

		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		content := string(data)
		for placeholder, value := range replacements {
			content = strings.ReplaceAll(content, placeholder, value)
		}
		// Always overwrite — our templates should take precedence over
		// OpenClaw's auto-generated defaults in workspace/.
		if err := os.WriteFile(dst, []byte(content), 0600); err != nil {
			return err
		}
	}
	return nil
}

func (h *ContainerHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req model.BatchCreateRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Prefix == "" {
		writeError(w, http.StatusBadRequest, "prefix is required")
		return
	}
	if req.Count < 1 || req.Count > 20 {
		writeError(w, http.StatusBadRequest, "count must be 1-20")
		return
	}
	if req.StartPort == 0 {
		req.StartPort = 18790
	}
	if req.Image == "" {
		req.Image = "fourplayers/openclaw:latest"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Load Discourse settings once
	discourseURL := "https://discourse.liukersun.com"
	discourseAPIKey := ""
	if h.Prefs != nil {
		if p, err := h.Prefs.Get(); err == nil {
			if p.DiscourseURL != "" {
				discourseURL = p.DiscourseURL
			}
			if p.DiscourseAPIKey != "" {
				discourseAPIKey = p.DiscourseAPIKey
			}
		}
	}

	var userPrefs *preferences.UserPreferences
	if h.Prefs != nil {
		if p, err := h.Prefs.Get(); err == nil {
			userPrefs = p.Resolved()
		}
	}
	if userPrefs == nil {
		userPrefs = &preferences.UserPreferences{}
	}

	success, failed := 0, 0
	nextPort := req.StartPort

	for i := 0; i < req.Count; i++ {
		agentName := fmt.Sprintf("%s-%d", req.Prefix, i+1)
		agentPort := nextPort
		dataPath := "/home/evan/.openclaw-" + agentName

		sendBatchSSE(w, flusher, agentName, i, "prepare", "running", "正在准备数据目录...")

		hostDataPath := h.MountPrefix + dataPath
		if err := os.MkdirAll(hostDataPath, 0700); err != nil {
			sendBatchSSE(w, flusher, agentName, i, "prepare", "error", "创建目录失败: "+err.Error())
			failed++
			continue
		}
		for _, sub := range []string{"agents/main/sessions", "agents/main/agent", "conversations", "credentials", "canvas", "workspace"} {
			os.MkdirAll(filepath.Join(hostDataPath, sub), 0700)
		}
		if h.GLMAPIKey != "" {
			authProfile := fmt.Sprintf(`{"version":1,"profiles":{"zai:default":{"provider":"zai","type":"api_key","key":"%s"}}}`, h.GLMAPIKey)
			os.WriteFile(filepath.Join(hostDataPath, "agents/main/agent/auth-profiles.json"), []byte(authProfile), 0600)
		}

		// Check port availability before templates & AI
		sendBatchSSE(w, flusher, agentName, i, "port", "running", fmt.Sprintf("检测端口 %d...", agentPort))
		availPort, err := findAvailablePort(agentPort, 10)
		if err != nil {
			sendBatchSSE(w, flusher, agentName, i, "port", "error", "无法找到可用端口")
			failed++
			continue
		}
		if availPort != agentPort {
			sendBatchSSE(w, flusher, agentName, i, "port", "done", fmt.Sprintf("端口 %d 被占用，切换到 %d", agentPort, availPort))
			agentPort = availPort
		}

		// Copy templates
		sendBatchSSE(w, flusher, agentName, i, "template", "running", "正在复制模板...")
		gatewayToken := generateToken()
		if h.TemplateDir != "" {
			now := time.Now().UTC().Format(time.RFC3339Nano)
			replacements := map[string]string{
				"{{NAME}}":               agentName,
				"{{TIMESTAMP}}":          now,
				"{{WORKSPACE}}":          dataPath + "/workspace",
				"{{PORT}}":               fmt.Sprintf("%d", agentPort),
				"{{GATEWAY_TOKEN}}":      gatewayToken,
				"{{DISCOURSE_URL}}":      discourseURL,
				"{{DISCOURSE_API_KEY}}":  discourseAPIKey,
				"{{DISCOURSE_USERNAME}}": agentName,
			}
			if err := copyTemplates(h.TemplateDir, hostDataPath, replacements); err != nil {
				sendBatchSSE(w, flusher, agentName, i, "template", "error", "模板复制失败: "+err.Error())
				failed++
				continue
			}
		}

		// Create Discourse account
		if h.Discourse != nil && discourseAPIKey != "" {
			sendBatchSSE(w, flusher, agentName, i, "discourse", "running", "正在创建 Discourse 账号...")
			discoursePass := generateToken()[:16]
			email := fmt.Sprintf("%s@agents.openmanage.local", agentName)
			if err := h.Discourse.CreateUser(agentName, agentName, email, discoursePass); err != nil {
				log.Printf("Discourse create user %s error: %v", agentName, err)
				sendBatchSSE(w, flusher, agentName, i, "discourse", "warn", "Discourse 账号创建失败: "+err.Error())
			} else {
				sendBatchSSE(w, flusher, agentName, i, "discourse", "done", fmt.Sprintf("Discourse 账号 %s 已就绪", agentName))
			}
		}

		// AI generate (optional)
		if req.Description != "" && h.AI != nil {
			sendBatchSSE(w, flusher, agentName, i, "ai", "running", "AI 正在生成配置...")
			files, err := h.AI.GenerateStream(r.Context(), ai.GenerateRequest{
				Name:              agentName,
				Description:       req.Description,
				Username:          userPrefs.Username,
				Style:             userPrefs.Style,
				Tools:             userPrefs.Tools,
				ExtraContext:      userPrefs.ExtraContext,
				DiscourseURL:      discourseURL,
				DiscourseUsername: agentName,
			}, func(chars int) {
				sendBatchSSE(w, flusher, agentName, i, "ai", "running", fmt.Sprintf("AI 已生成 %d 字符", chars))
			})
			if err != nil {
				sendBatchSSE(w, flusher, agentName, i, "ai", "warn", "AI 生成失败，使用默认模板")
			} else {
				for filename, content := range files {
					dst := filepath.Join(hostDataPath, filename)
					if strings.HasSuffix(filename, ".md") {
						dst = filepath.Join(hostDataPath, "workspace", filename)
					}
					os.WriteFile(dst, []byte(content), 0600)
				}
			}
		}

		// Create container — retry with next port if port conflict
		sendBatchSSE(w, flusher, agentName, i, "container", "running", "正在创建容器...")
		agentEnv := map[string]string{
			"OPENCLAW_GATEWAY_TOKEN": gatewayToken,
			"OPENCLAW_GATEWAY_PORT":  fmt.Sprintf("%d", agentPort),
			"OPENCLAW_GATEWAY_BIND":  "loopback",
		}
		created := false
		for retry := 0; retry < 10; retry++ {
			agentEnv["OPENCLAW_GATEWAY_PORT"] = fmt.Sprintf("%d", agentPort)
			_, err := h.Docker.CreateContainer(r.Context(), agentName, req.Image, dataPath, agentPort, agentEnv)
			if err == nil {
				created = true
				break
			}
			if strings.Contains(err.Error(), "address already in use") {
				sendBatchSSE(w, flusher, agentName, i, "container", "running", fmt.Sprintf("端口 %d 被占用，尝试 %d...", agentPort, agentPort+1))
				agentPort++
				continue
			}
			sendBatchSSE(w, flusher, agentName, i, "container", "error", err.Error())
			break
		}
		if !created {
			failed++
			continue
		}

		nextPort = availPort + 1
		sendBatchSSE(w, flusher, agentName, i, "done", "done", fmt.Sprintf("创建成功 (端口 %d)", agentPort))
		success++
	}

	// Final summary
	evt := map[string]interface{}{"step": "batch-done", "status": "done", "total": req.Count, "success": success, "failed": failed}
	d, _ := json.Marshal(evt)
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()
}

func sendBatchSSE(w http.ResponseWriter, flusher http.Flusher, agent string, index int, step, status, message string) {
	evt := map[string]interface{}{"agent": agent, "index": index, "step": step, "status": status, "message": message}
	data, _ := json.Marshal(evt)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (h *ContainerHandler) ForumActivity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, err := h.Docker.InspectContainer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}

	username := strings.TrimPrefix(info.Name, "/")

	if h.Discourse == nil {
		writeError(w, http.StatusServiceUnavailable, "Discourse not configured")
		return
	}

	activity, err := h.Discourse.GetUserActivity(username)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Discourse API error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, activity)
}

// findAvailablePort checks if the given port is available, and if not,
// increments until it finds one (up to maxRetries attempts).
func findAvailablePort(port, maxRetries int) (int, error) {
	for i := 0; i < maxRetries; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
		port++
	}
	return 0, fmt.Errorf("无法找到可用端口（已尝试 %d 个）", maxRetries)
}

func generateToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// fallback: not ideal but won't happen in practice
		return "0000000000000000000000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
