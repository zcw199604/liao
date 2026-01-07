package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func (a *App) handleRepairMediaHistory(w http.ResponseWriter, r *http.Request) {
	var req RepairMediaHistoryRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid json body",
		})
		return
	}

	result, err := a.mediaUpload.RepairMediaHistory(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

