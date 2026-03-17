package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/ai"
	"openmanage/backend/docker"
	"openmanage/backend/model"
	"openmanage/backend/openclaw"
)

type CronHandler struct {
	OpenClaw *openclaw.Client
	Docker   *docker.Client
	AI       *ai.Client // nil if GLM_API_KEY not set
}

// ListCronJobs returns cron jobs + heartbeat config for a container.
func (h *CronHandler) List(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	jobs, err := h.OpenClaw.ListCronJobs(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadGateway, "获取 cron 列表失败: "+err.Error())
		return
	}

	heartbeat, _ := h.OpenClaw.GetHeartbeat(r.Context(), id)

	result := map[string]interface{}{
		"jobs":      jobs,
		"heartbeat": heartbeat,
	}
	writeJSON(w, http.StatusOK, result)
}

// AddCronJob creates a new cron job.
func (h *CronHandler) Add(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req model.CronJobRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Schedule == "" || req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "schedule and prompt are required")
		return
	}

	jobID, err := h.OpenClaw.AddCronJob(r.Context(), id, req.Name, req.Schedule, req.Prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, "创建 cron 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": jobID})
}

// ToggleCronJob enables or disables a cron job.
func (h *CronHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	jobID := chi.URLParam(r, "jobId")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	json.Unmarshal(data, &req)

	if err := h.OpenClaw.ToggleCronJob(r.Context(), id, jobID, req.Enabled); err != nil {
		writeError(w, http.StatusBadGateway, "切换 cron 状态失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RunCronJob triggers immediate execution.
func (h *CronHandler) Run(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	jobID := chi.URLParam(r, "jobId")

	_, err := h.OpenClaw.RunCronJob(r.Context(), id, jobID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "执行 cron 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RemoveCronJob deletes a cron job.
func (h *CronHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	jobID := chi.URLParam(r, "jobId")

	if err := h.OpenClaw.RemoveCronJob(r.Context(), id, jobID); err != nil {
		writeError(w, http.StatusBadGateway, "删除 cron 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdateHeartbeat modifies the heartbeat interval.
func (h *CronHandler) UpdateHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req model.HeartbeatRequest
	data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.OpenClaw.SetHeartbeat(r.Context(), id, req.Every); err != nil {
		writeError(w, http.StatusBadGateway, "更新 heartbeat 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "every": req.Every})
}

// GenerateCronJob uses AI to generate schedule + prompt from natural language.
func (h *CronHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if h.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI 未配置（需要 GLM_API_KEY）")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err := json.Unmarshal(data, &req); err != nil || req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	schedule, prompt, err := h.AI.GenerateCronJob(r.Context(), req.Description)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI 生成失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"schedule": schedule, "prompt": prompt})
}
