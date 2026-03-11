package handler

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"openmanage/backend/docker"
)

type LogsHandler struct {
	Docker *docker.Client
}

func (h *LogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}
	follow := r.URL.Query().Get("follow") == "true"

	reader, err := h.Docker.ContainerLogs(r.Context(), id, tail, follow)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	scanner := bufio.NewScanner(reader)
	// Docker multiplexed stream has 8-byte header per frame.
	// For simplicity, we strip the header bytes and send raw lines.
	for scanner.Scan() {
		line := scanner.Bytes()
		// Docker log stream: first 8 bytes are header (stream type + size)
		// If line is longer than 8 bytes and starts with 0x01 or 0x02, strip header
		if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
			line = line[8:]
		}
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}
}
