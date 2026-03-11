package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/ai"
	"openmanage/backend/docker"
	"openmanage/backend/model"
	"openmanage/backend/preferences"
)

type ContainerHandler struct {
	Docker      *docker.Client
	TemplateDir  string // path to templates/ directory
	MountPrefix  string // "/host" in container, "" in dev
	AI          *ai.Client // nil if GLM_API_KEY not set
	Prefs       *preferences.Store
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
		"conversations",
		"credentials",
		"canvas",
		"workspace",
	} {
		os.MkdirAll(filepath.Join(hostDataPath, sub), 0700)
	}
	sendSSE(w, flusher, "prepare", "done", "数据目录准备完成")

	// Step 2: Copy templates
	sendSSE(w, flusher, "template", "running", "正在复制模板文件...")
	if h.TemplateDir != "" {
		log.Printf("Copying templates from %s to %s", h.TemplateDir, hostDataPath)
		now := time.Now().UTC().Format(time.RFC3339Nano)
		replacements := map[string]string{
			"{{NAME}}":          req.Name,
			"{{TIMESTAMP}}":     now,
			"{{WORKSPACE}}":     req.DataPath + "/workspace",
			"{{PORT}}":          fmt.Sprintf("%d", req.Port),
			"{{GATEWAY_TOKEN}}": generateToken(),
		}
		if err := copyTemplates(h.TemplateDir, hostDataPath, replacements); err != nil {
			log.Printf("Template copy error: %v", err)
			sendSSE(w, flusher, "template", "error", "复制模板失败: "+err.Error())
			return
		}
	}
	sendSSE(w, flusher, "template", "done", "模板文件复制完成")

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
			Name:        req.Name,
			Description: req.Description,
			Username:    userPrefs.Username,
			Style:       userPrefs.Style,
			Tools:       userPrefs.Tools,
			ExtraContext: userPrefs.ExtraContext,
		}, func(chars int) {
			sendSSE(w, flusher, "ai", "running", fmt.Sprintf("AI 正在生成配置... 已生成 %d 字符", chars))
		})
		if err != nil {
			log.Printf("AI generate error (falling back to defaults): %v", err)
			sendSSE(w, flusher, "ai", "warn", "AI 生成失败，将使用默认模板: "+err.Error())
		} else {
			for filename, content := range files {
				dst := filepath.Join(hostDataPath, filename)
				if err := os.WriteFile(dst, []byte(content), 0600); err != nil {
					log.Printf("Failed to write AI-generated %s: %v", filename, err)
				}
			}
			sendSSE(w, flusher, "ai", "done", fmt.Sprintf("AI 配置文件生成完成，共 %d 个文件", len(files)))
		}
	}

	// Step 4: Create container
	sendSSE(w, flusher, "container", "running", "正在创建 Docker 容器...")
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
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(templateDir, e.Name())
		dst := filepath.Join(dataPath, e.Name())

		// Skip if file already exists
		if _, err := os.Stat(dst); err == nil {
			continue
		}

		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		content := string(data)
		for placeholder, value := range replacements {
			content = strings.ReplaceAll(content, placeholder, value)
		}
		if err := os.WriteFile(dst, []byte(content), 0600); err != nil {
			return err
		}
	}
	return nil
}

func generateToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// fallback: not ideal but won't happen in practice
		return "0000000000000000000000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
