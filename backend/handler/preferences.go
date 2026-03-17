package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"openmanage/backend/preferences"
)

type PreferencesHandler struct {
	Store *preferences.Store
}

func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	prefs, err := h.Store.Get()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prefs.Masked())
}

func (h *PreferencesHandler) Save(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var prefs preferences.UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.Store.Save(&prefs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// ProbeModels proxies a GET /models request to an AI provider and returns the model list.
func (h *PreferencesHandler) ProbeModels(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": "invalid request"})
		return
	}
	if req.BaseURL == "" || req.APIKey == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": "baseUrl and apiKey required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", req.BaseURL+"/models", nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": fmt.Sprintf("bad url: %v", err)})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": fmt.Sprintf("request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": fmt.Sprintf("provider returned %d", resp.StatusCode)})
		return
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"models": []string{}, "error": "failed to parse response"})
		return
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"models": models})
}
