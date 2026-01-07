package app

import (
	"net/http"
	"strconv"
	"strings"
)

func (a *App) handleFavoriteAdd(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	identityID := strings.TrimSpace(r.FormValue("identityId"))
	targetUserID := strings.TrimSpace(r.FormValue("targetUserId"))
	targetUserName := strings.TrimSpace(r.FormValue("targetUserName"))

	fav, err := a.favoriteService.Add(r.Context(), identityID, targetUserID, targetUserName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": fav,
	})
}

func (a *App) handleFavoriteRemove(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	identityID := strings.TrimSpace(r.FormValue("identityId"))
	targetUserID := strings.TrimSpace(r.FormValue("targetUserId"))

	_ = a.favoriteService.Remove(r.Context(), identityID, targetUserID)

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
	})
}

func (a *App) handleFavoriteRemoveByID(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	raw := strings.TrimSpace(r.FormValue("id"))
	id, _ := strconv.ParseInt(raw, 10, 64)

	_ = a.favoriteService.RemoveByID(r.Context(), id)

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
	})
}

func (a *App) handleFavoriteListAll(w http.ResponseWriter, r *http.Request) {
	list, err := a.favoriteService.ListAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "查询失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": list,
	})
}

func (a *App) handleFavoriteCheck(w http.ResponseWriter, r *http.Request) {
	identityID := strings.TrimSpace(r.URL.Query().Get("identityId"))
	targetUserID := strings.TrimSpace(r.URL.Query().Get("targetUserId"))

	isFav, err := a.favoriteService.IsFavorite(r.Context(), identityID, targetUserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "查询失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{"isFavorite": isFav},
	})
}

