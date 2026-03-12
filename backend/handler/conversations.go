package handler

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/docker"
	"openmanage/backend/model"
)

type ConversationsHandler struct {
	Docker      *docker.Client
	MountPrefix string
}

// getSessionsDir returns the host path to agents/main/sessions/ for a container.
func (h *ConversationsHandler) getSessionsDir(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	info, err := h.Docker.InspectContainer(r.Context(), id)
	if err != nil {
		return "", err
	}
	for _, m := range info.Mounts {
		if strings.Contains(m.Destination, ".openclaw") {
			return filepath.Join(h.MountPrefix+m.Source, "agents", "main", "sessions"), nil
		}
	}
	return "", os.ErrNotExist
}

// sessionsIndex represents the sessions.json index file structure.
// Key: "agent:main:openresponses:{uuid}", Value: session metadata.
type sessionEntry struct {
	SessionID   string `json:"sessionId"`
	UpdatedAt   int64  `json:"updatedAt"` // milliseconds
	SessionFile string `json:"sessionFile"`
}

// jsonlMessage represents a line in the .jsonl session file with type "message".
type jsonlLine struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   *jsonlMsgDetail `json:"message,omitempty"`
}

type jsonlMsgDetail struct {
	Role    string           `json:"role"`
	Content []jsonlMsgContent `json:"content"`
}

type jsonlMsgContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (h *ConversationsHandler) List(w http.ResponseWriter, r *http.Request) {
	dir, err := h.getSessionsDir(r)
	if err != nil {
		writeJSON(w, http.StatusOK, []model.Conversation{})
		return
	}

	// Try to read sessions.json index first
	indexPath := filepath.Join(dir, "sessions.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		writeJSON(w, http.StatusOK, []model.Conversation{})
		return
	}

	var index map[string]sessionEntry
	if err := json.Unmarshal(indexData, &index); err != nil {
		writeJSON(w, http.StatusOK, []model.Conversation{})
		return
	}

	// Deduplicate by sessionId (multiple openresponses keys can map to same session)
	seen := make(map[string]bool)
	result := make([]model.Conversation, 0, len(index))

	for _, entry := range index {
		if seen[entry.SessionID] {
			continue
		}
		seen[entry.SessionID] = true

		// Skip sessions whose .jsonl file doesn't exist on disk
		jsonlPath := filepath.Join(dir, entry.SessionID+".jsonl")
		if _, err := os.Stat(jsonlPath); err != nil {
			continue
		}

		updatedAt := time.UnixMilli(entry.UpdatedAt).Format("2006-01-02 15:04:05")
		title := extractSessionTitle(jsonlPath)

		result = append(result, model.Conversation{
			SessionID: entry.SessionID,
			Title:     title,
			UpdatedAt: updatedAt,
		})
	}

	// Also scan for .jsonl files not in the index
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		sid := strings.TrimSuffix(e.Name(), ".jsonl")
		if seen[sid] {
			continue
		}
		seen[sid] = true
		info, _ := e.Info()
		updatedAt := ""
		if info != nil {
			updatedAt = info.ModTime().Format("2006-01-02 15:04:05")
		}
		title := extractSessionTitle(filepath.Join(dir, e.Name()))
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
	dir, err := h.getSessionsDir(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "sessions directory not found")
		return
	}

	sid := chi.URLParam(r, "sid")
	filePath := filepath.Join(dir, sid+".jsonl")

	// Path traversal check
	abs, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(abs, dir) {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	messages, err := parseSessionJSONL(filePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	title := ""
	for _, m := range messages {
		if m.Role == "user" && m.Content != "" {
			title = m.Content
			if len([]rune(title)) > 50 {
				title = string([]rune(title)[:50]) + "..."
			}
			break
		}
	}
	if title == "" {
		title = sid[:8]
	}

	detail := model.ConversationDetail{
		SessionID: sid,
		Title:     title,
		Messages:  messages,
	}
	writeJSON(w, http.StatusOK, detail)
}

// parseSessionJSONL reads a .jsonl session file and extracts user/assistant messages.
func parseSessionJSONL(path string) ([]model.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []model.Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024) // up to 1MB per line

	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "message" || line.Message == nil {
			continue
		}
		role := line.Message.Role
		if role != "user" && role != "assistant" {
			continue
		}
		// Extract text content, skip thinking blocks
		var textParts []string
		for _, c := range line.Message.Content {
			if c.Type == "text" || c.Type == "output_text" {
				if c.Text != "" {
					textParts = append(textParts, c.Text)
				}
			}
		}
		if len(textParts) == 0 {
			continue
		}
		messages = append(messages, model.Message{
			Role:    role,
			Content: strings.Join(textParts, "\n"),
		})
	}

	return messages, nil
}

// extractSessionTitle reads the first user message from a .jsonl session file.
func extractSessionTitle(path string) string {
	messages, err := parseSessionJSONL(path)
	if err != nil {
		return filepath.Base(path)
	}
	for _, m := range messages {
		if m.Role == "user" && m.Content != "" {
			title := m.Content
			if len([]rune(title)) > 50 {
				return string([]rune(title)[:50]) + "..."
			}
			return title
		}
	}
	return strings.TrimSuffix(filepath.Base(path), ".jsonl")
}
