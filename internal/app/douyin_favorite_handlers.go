package app

import (
	"encoding/json"
	"net/http"
	"strings"
)

type douyinFavoriteUserAddRequest struct {
	SecUserID       string          `json:"secUserId"`
	SourceInput     string          `json:"sourceInput,omitempty"`
	DisplayName     string          `json:"displayName,omitempty"`
	AvatarURL       string          `json:"avatarUrl,omitempty"`
	ProfileURL      string          `json:"profileUrl,omitempty"`
	LastParsedCount *int            `json:"lastParsedCount,omitempty"`
	LastParsedRaw   json.RawMessage `json:"lastParsedRaw,omitempty"`
}

type douyinFavoriteUserRemoveRequest struct {
	SecUserID string `json:"secUserId"`
}

func (a *App) handleDouyinFavoriteUserList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	list, err := a.douyinFavorite.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询失败"})
		return
	}
	if list == nil {
		list = []DouyinFavoriteUser{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *App) handleDouyinFavoriteUserAdd(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteUserAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	secUserID := strings.TrimSpace(req.SecUserID)
	if secUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secUserId 不能为空"})
		return
	}

	raw := strings.TrimSpace(string(req.LastParsedRaw))
	out, err := a.douyinFavorite.UpsertUser(r.Context(), DouyinFavoriteUserUpsert{
		SecUserID:       secUserID,
		SourceInput:     strings.TrimSpace(req.SourceInput),
		DisplayName:     strings.TrimSpace(req.DisplayName),
		AvatarURL:       strings.TrimSpace(req.AvatarURL),
		ProfileURL:      strings.TrimSpace(req.ProfileURL),
		LastParsedCount: req.LastParsedCount,
		LastParsedRaw:   raw,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteUserRemove(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteUserRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}
	secUserID := strings.TrimSpace(req.SecUserID)
	if secUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secUserId 不能为空"})
		return
	}

	_ = a.douyinFavorite.RemoveUser(r.Context(), secUserID)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

type douyinFavoriteAwemeAddRequest struct {
	AwemeID   string          `json:"awemeId"`
	SecUserID string          `json:"secUserId,omitempty"`
	Type      string          `json:"type,omitempty"`
	Desc      string          `json:"desc,omitempty"`
	CoverURL  string          `json:"coverUrl,omitempty"`
	RawDetail json.RawMessage `json:"rawDetail,omitempty"`
}

type douyinFavoriteAwemeRemoveRequest struct {
	AwemeID string `json:"awemeId"`
}

func (a *App) handleDouyinFavoriteAwemeList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	list, err := a.douyinFavorite.ListAwemes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询失败"})
		return
	}
	if list == nil {
		list = []DouyinFavoriteAweme{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *App) handleDouyinFavoriteAwemeAdd(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteAwemeAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	awemeID := strings.TrimSpace(req.AwemeID)
	if awemeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "awemeId 不能为空"})
		return
	}

	raw := strings.TrimSpace(string(req.RawDetail))
	out, err := a.douyinFavorite.UpsertAweme(r.Context(), DouyinFavoriteAwemeUpsert{
		AwemeID:   awemeID,
		SecUserID: strings.TrimSpace(req.SecUserID),
		Type:      strings.TrimSpace(req.Type),
		Desc:      strings.TrimSpace(req.Desc),
		CoverURL:  strings.TrimSpace(req.CoverURL),
		RawDetail: raw,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteAwemeRemove(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteAwemeRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}
	awemeID := strings.TrimSpace(req.AwemeID)
	if awemeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "awemeId 不能为空"})
		return
	}

	_ = a.douyinFavorite.RemoveAweme(r.Context(), awemeID)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
