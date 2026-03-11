package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"openmanage/backend/model"
)

type TemplateHandler struct {
	TemplateDir string
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.TemplateDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]model.FileEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		result = append(result, model.FileEntry{
			Name:  e.Name(),
			Path:  e.Name(),
			IsDir: false,
			Size:  size,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *TemplateHandler) Read(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "file name required")
		return
	}

	absPath, err := safePath(h.TemplateDir, name)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	writeJSON(w, http.StatusOK, model.FileContent{Path: name, Content: string(data)})
}

func (h *TemplateHandler) Write(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "file name required")
		return
	}

	absPath, err := safePath(h.TemplateDir, name)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	var body model.FileContent
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := os.WriteFile(absPath, []byte(body.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "file name required")
		return
	}

	absPath, err := safePath(h.TemplateDir, name)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	if err := os.Remove(absPath); err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body model.FileContent
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Path == "" {
		writeError(w, http.StatusBadRequest, "path (filename) required")
		return
	}

	absPath, err := safePath(h.TemplateDir, body.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	if _, err := os.Stat(absPath); err == nil {
		writeError(w, http.StatusConflict, "template already exists")
		return
	}

	if err := os.WriteFile(absPath, []byte(body.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}
