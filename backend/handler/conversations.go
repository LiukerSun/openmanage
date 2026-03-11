package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/docker"
	"openmanage/backend/model"
)

type ConversationsHandler struct {
	Docker      *docker.Client
	MountPrefix string
}

func (h *ConversationsHandler) getConversationsDir(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	info, err := h.Docker.InspectContainer(r.Context(), id)
	if err != nil {
		return "", err
	}
	for _, m := range info.Mounts {
		if strings.Contains(m.Destination, ".openclaw") {
			return filepath.Join(h.MountPrefix+m.Source, "conversations"), nil
		}
	}
	return "", os.ErrNotExist
}

func (h *ConversationsHandler) List(w http.ResponseWriter, r *http.Request) {
	dir, err := h.getConversationsDir(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversations directory not found")
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []model.Conversation{})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]model.Conversation, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, _ := e.Info()
		updatedAt := ""
		if info != nil {
			updatedAt = info.ModTime().Format("2006-01-02 15:04:05")
		}
		sid := strings.TrimSuffix(e.Name(), ".json")
		title := extractTitle(filepath.Join(dir, e.Name()))
		result = append(result, model.Conversation{
			SessionID: sid,
			Title:     title,
			UpdatedAt: updatedAt,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt > result[j].UpdatedAt
	})

	writeJSON(w, http.StatusOK, result)
}

func (h *ConversationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	dir, err := h.getConversationsDir(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversations directory not found")
		return
	}

	sid := chi.URLParam(r, "sid")
	filePath := filepath.Join(dir, sid+".json")

	// Path traversal check
	abs, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(abs, dir) {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	// Try to parse as array of messages
	var messages []model.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		// Try as object with messages field
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse conversation")
			return
		}
		if raw, ok := obj["messages"]; ok {
			json.Unmarshal(raw, &messages)
		}
	}

	detail := model.ConversationDetail{
		SessionID: sid,
		Title:     extractTitle(filePath),
		Messages:  messages,
	}
	writeJSON(w, http.StatusOK, detail)
}

func extractTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return filepath.Base(path)
	}
	// Try object with title field
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err == nil {
		if raw, ok := obj["title"]; ok {
			var title string
			if json.Unmarshal(raw, &title) == nil && title != "" {
				return title
			}
		}
	}
	// Fallback: first user message
	var messages []model.Message
	if json.Unmarshal(data, &messages) == nil {
		for _, m := range messages {
			if m.Role == "user" && len(m.Content) > 0 {
				if len(m.Content) > 50 {
					return m.Content[:50] + "..."
				}
				return m.Content
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(path), ".json")
}
