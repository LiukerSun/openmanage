package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/docker"
	"openmanage/backend/model"
	"openmanage/backend/openclaw"
)

type ChatHandler struct {
	OpenClaw *openclaw.Client
	Docker   *docker.Client
}

// Chat handles POST /api/containers/{id}/chat
// Sends a message to a single agent and streams the response via SSE.
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Get container config
	cfg, err := h.OpenClaw.GetConfig(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read agent config: %v", err))
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	sendSSE := func(data interface{}) {
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	sendSSE(map[string]string{"status": "sending", "message": "正在发送消息..."})

	// If config was patched, restart container first
	if cfg.NeedsRestart {
		sendSSE(map[string]string{"status": "sending", "message": "已更新配置，正在重启 Agent..."})
		if err := h.Docker.RestartContainer(r.Context(), id); err != nil {
			sendSSE(map[string]string{"status": "error", "message": fmt.Sprintf("重启失败: %v", err)})
			return
		}
		// Wait for gateway to come back up
		time.Sleep(5 * time.Second)
		sendSSE(map[string]string{"status": "sending", "message": "Agent 已重启，正在发送消息..."})
	}

	reply, err := h.OpenClaw.SendMessage(r.Context(), id, cfg, req.Message)
	if err != nil {
		sendSSE(map[string]string{"status": "error", "message": fmt.Sprintf("发送失败: %v", err)})
		return
	}

	sendSSE(map[string]string{"status": "done", "reply": reply})
}

// BatchChat handles POST /api/batch/chat
// Sends the same message to multiple agents concurrently, streaming progress via SSE.
func (h *ChatHandler) BatchChat(w http.ResponseWriter, r *http.Request) {
	var req model.BatchChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ContainerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "containerIds is required")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	var mu sync.Mutex
	sendSSE := func(data interface{}) {
		mu.Lock()
		defer mu.Unlock()
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	// Concurrency limiter
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	successCount := 0
	failCount := 0
	var countMu sync.Mutex

	for i, cid := range req.ContainerIDs {
		wg.Add(1)
		go func(index int, containerID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			agentName := containerID[:12]

			sendSSE(map[string]interface{}{
				"agent":   agentName,
				"index":   index,
				"step":    "sending",
				"status":  "running",
				"message": "正在发送消息...",
			})

			cfg, err := h.OpenClaw.GetConfig(r.Context(), containerID)
			if err != nil {
				log.Printf("batch chat: get config for %s: %v", containerID, err)
				sendSSE(map[string]interface{}{
					"agent":   agentName,
					"index":   index,
					"step":    "error",
					"status":  "error",
					"message": fmt.Sprintf("读取配置失败: %v", err),
				})
				countMu.Lock()
				failCount++
				countMu.Unlock()
				return
			}

			if cfg.NeedsRestart {
				sendSSE(map[string]interface{}{
					"agent":   agentName,
					"index":   index,
					"step":    "restarting",
					"status":  "running",
					"message": "已更新配置，正在重启...",
				})
				if err := h.Docker.RestartContainer(r.Context(), containerID); err != nil {
					sendSSE(map[string]interface{}{
						"agent":   agentName,
						"index":   index,
						"step":    "error",
						"status":  "error",
						"message": fmt.Sprintf("重启失败: %v", err),
					})
					countMu.Lock()
					failCount++
					countMu.Unlock()
					return
				}
				time.Sleep(5 * time.Second)
			}

			reply, err := h.OpenClaw.SendMessage(r.Context(), containerID, cfg, req.Message)
			if err != nil {
				log.Printf("batch chat: send to %s: %v", containerID, err)
				sendSSE(map[string]interface{}{
					"agent":   agentName,
					"index":   index,
					"step":    "error",
					"status":  "error",
					"message": fmt.Sprintf("发送失败: %v", err),
				})
				countMu.Lock()
				failCount++
				countMu.Unlock()
				return
			}

			sendSSE(map[string]interface{}{
				"agent":   agentName,
				"index":   index,
				"step":    "done",
				"status":  "done",
				"message": truncate(reply, 100),
			})
			countMu.Lock()
			successCount++
			countMu.Unlock()
		}(i, cid)
	}

	wg.Wait()

	sendSSE(map[string]interface{}{
		"step":    "batch-done",
		"status":  "done",
		"message": fmt.Sprintf("完成 %d/%d，失败 %d", successCount, len(req.ContainerIDs), failCount),
	})
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
