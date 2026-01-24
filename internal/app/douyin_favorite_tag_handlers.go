package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type douyinFavoriteTagAddRequest struct {
	Name string `json:"name"`
}

type douyinFavoriteTagUpdateRequest struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type douyinFavoriteTagRemoveRequest struct {
	ID int64 `json:"id"`
}

type douyinFavoriteUserTagApplyRequest struct {
	SecUserIDs []string `json:"secUserIds"`
	TagIDs     []int64  `json:"tagIds"`
	Mode       string   `json:"mode,omitempty"` // set/add/remove
}

type douyinFavoriteAwemeTagApplyRequest struct {
	AwemeIDs []string `json:"awemeIds"`
	TagIDs   []int64  `json:"tagIds"`
	Mode     string   `json:"mode,omitempty"` // set/add/remove
}

type douyinFavoriteTagReorderRequest struct {
	TagIDs []int64 `json:"tagIds"`
}

func (a *App) handleDouyinFavoriteUserTagList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	list, err := a.douyinFavorite.ListUserTags(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询失败"})
		return
	}
	if list == nil {
		list = []DouyinFavoriteTag{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *App) handleDouyinFavoriteUserTagAdd(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name 不能为空"})
		return
	}

	out, err := a.douyinFavorite.AddUserTag(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagAlreadyExists) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteUserTagUpdate(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if req.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id 不能为空"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name 不能为空"})
		return
	}

	out, err := a.douyinFavorite.UpdateUserTag(r.Context(), req.ID, name)
	if err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagAlreadyExists) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "标签不存在"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteUserTagRemove(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}
	if req.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id 不能为空"})
		return
	}

	if err := a.douyinFavorite.RemoveUserTag(r.Context(), req.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "删除失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleDouyinFavoriteUserTagApply(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteUserTagApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if err := a.douyinFavorite.ApplyUserTags(r.Context(), req.SecUserIDs, req.TagIDs, req.Mode); err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagInvalidMode) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleDouyinFavoriteUserTagReorder(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if err := a.douyinFavorite.ReorderUserTags(r.Context(), req.TagIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleDouyinFavoriteAwemeTagList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	list, err := a.douyinFavorite.ListAwemeTags(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询失败"})
		return
	}
	if list == nil {
		list = []DouyinFavoriteTag{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (a *App) handleDouyinFavoriteAwemeTagAdd(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name 不能为空"})
		return
	}

	out, err := a.douyinFavorite.AddAwemeTag(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagAlreadyExists) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteAwemeTagUpdate(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if req.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id 不能为空"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name 不能为空"})
		return
	}

	out, err := a.douyinFavorite.UpdateAwemeTag(r.Context(), req.ID, name)
	if err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagAlreadyExists) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	if out == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "标签不存在"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDouyinFavoriteAwemeTagRemove(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}
	if req.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id 不能为空"})
		return
	}

	if err := a.douyinFavorite.RemoveAwemeTag(r.Context(), req.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "删除失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleDouyinFavoriteAwemeTagReorder(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteTagReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if err := a.douyinFavorite.ReorderAwemeTags(r.Context(), req.TagIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleDouyinFavoriteAwemeTagApply(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteAwemeTagApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	if err := a.douyinFavorite.ApplyAwemeTags(r.Context(), req.AwemeIDs, req.TagIDs, req.Mode); err != nil {
		if errors.Is(err, ErrDouyinFavoriteTagInvalidMode) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
