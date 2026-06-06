package app

import "net/http"

func (a *App) handleRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]string{
			"randomVipCode": a.cfg.RandomVIPCode,
		},
	})
}
