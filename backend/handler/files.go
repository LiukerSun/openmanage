package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/docker"
	"openmanage/backend/model"
)

type FilesHandler struct {
	Docker      *docker.Client
	MountPrefix string // e.g. "/host" when containerized, "" for dev
}

// getHostPath resolves the host filesystem path for a container's .openclaw directory.
func (h *FilesHandler) getHostPath(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	info, err := h.Docker.InspectContainer(r.Context(), id)
	if err != nil {
		return "", err
	}
	for _, m := range info.Mounts {
		if strings.Contains(m.Destination, ".openclaw") {
			return h.MountPrefix + m.Source, nil
		}
	}
	return "", os.ErrNotExist
}

// safePath validates that the resolved path stays within the root directory.
func safePath(root, rel string) (string, error) {
	full := filepath.Join(root, rel)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, absRoot) {
		return "", os.ErrPermission
	}
	return abs, nil
}

func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
	root, err := h.getHostPath(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "mount not found")
		return
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]model.FileEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		result = append(result, model.FileEntry{
			Name:  e.Name(),
			Path:  e.Name(),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *FilesHandler) Read(w http.ResponseWriter, r *http.Request) {
	root, err := h.getHostPath(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "mount not found")
		return
	}

	relPath := chi.URLParam(r, "*")
	absPath, err := safePath(root, relPath)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if stat.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		result := make([]model.FileEntry, 0, len(entries))
		for _, e := range entries {
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			result = append(result, model.FileEntry{
				Name:  e.Name(),
				Path:  filepath.Join(relPath, e.Name()),
				IsDir: e.IsDir(),
				Size:  size,
			})
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, model.FileContent{Path: relPath, Content: string(data)})
}

func (h *FilesHandler) Write(w http.ResponseWriter, r *http.Request) {
	root, err := h.getHostPath(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "mount not found")
		return
	}

	relPath := chi.URLParam(r, "*")
	absPath, err := safePath(root, relPath)
	if err != nil {
		writeError(w, http.StatusForbidden, "path not allowed")
		return
	}

	var body model.FileContent
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := os.WriteFile(absPath, []byte(body.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
