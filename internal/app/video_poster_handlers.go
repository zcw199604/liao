package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func (a *App) handleRepairVideoPosters(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mediaUpload == nil || a.fileStorage == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req RepairVideoPostersRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	res, err := a.mediaUpload.RepairVideoPosters(r.Context(), a.cfg.FFmpegPath, a.cfg.FFprobePath, req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}
