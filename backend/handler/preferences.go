package handler

import (
	"encoding/json"
	"io"
	"net/http"

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
