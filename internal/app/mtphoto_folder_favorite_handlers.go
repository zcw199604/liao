package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"
)

func (a *App) handleGetMtPhotoFolderFavorites(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhotoFolderFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 文件夹收藏服务未初始化"})
		return
	}

	options := parseMtPhotoFolderFavoriteListOptions(r)
	items, err := a.mtPhotoFolderFavorite.ListWithOptions(r.Context(), options)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询收藏失败: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func parseMtPhotoFolderFavoriteListOptions(r *http.Request) MtPhotoFolderFavoriteListOptions {
	if r == nil {
		return MtPhotoFolderFavoriteListOptions{}
	}

	query := r.URL.Query()
	options := MtPhotoFolderFavoriteListOptions{
		TagKeyword: strings.TrimSpace(query.Get("tagKeyword")),
		TagMode:    strings.ToLower(strings.TrimSpace(query.Get("tagMode"))),
		SortBy:     strings.TrimSpace(query.Get("sortBy")),
		SortOrder:  strings.ToLower(strings.TrimSpace(query.Get("sortOrder"))),
		GroupBy:    strings.TrimSpace(query.Get("groupBy")),
	}

	switch options.TagMode {
	case "all", "any":
	default:
		options.TagMode = "any"
	}

	switch options.SortBy {
	case "updatedAt", "name", "tagCount":
	default:
		options.SortBy = "updatedAt"
	}

	switch options.SortOrder {
	case "asc", "desc":
	default:
		options.SortOrder = "desc"
	}

	switch options.GroupBy {
	case "none", "tag":
	default:
		options.GroupBy = "none"
	}
	return options
}

func sanitizeMtPhotoFolderFavoriteUpsertInput(in *MtPhotoFolderFavoriteUpsertInput) error {
	if in == nil {
		return http.ErrBodyNotAllowed
	}

	in.FolderName = strings.TrimSpace(in.FolderName)
	in.FolderPath = strings.TrimSpace(in.FolderPath)
	in.CoverMD5 = strings.TrimSpace(in.CoverMD5)
	in.Note = strings.TrimSpace(in.Note)

	if in.FolderID <= 0 {
		return errBadRequest("folderId 参数非法")
	}
	if in.FolderName == "" {
		return errBadRequest("folderName 不能为空")
	}
	if in.FolderPath == "" {
		return errBadRequest("folderPath 不能为空")
	}
	if in.CoverMD5 != "" && !isValidMD5Hex(in.CoverMD5) {
		return errBadRequest("coverMd5 参数非法")
	}

	tags, err := normalizeMtPhotoFolderFavoriteTags(in.Tags)
	if err != nil {
		return errBadRequest(err.Error())
	}
	if utf8.RuneCountInString(in.Note) > mtPhotoFolderFavoriteMaxNoteRunes {
		return errBadRequest("note 长度不能超过 500")
	}
	in.Tags = tags
	return nil
}

type badRequestError string

func (e badRequestError) Error() string {
	return string(e)
}

func errBadRequest(message string) error {
	return badRequestError(strings.TrimSpace(message))
}

func isBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(badRequestError)
	return ok
}

func (a *App) handleUpsertMtPhotoFolderFavorite(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhotoFolderFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 文件夹收藏服务未初始化"})
		return
	}

	var req MtPhotoFolderFavoriteUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求体格式错误"})
		return
	}
	if err := sanitizeMtPhotoFolderFavoriteUpsertInput(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	item, err := a.mtPhotoFolderFavorite.Upsert(r.Context(), req)
	if err != nil {
		if isBadRequestError(err) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存收藏失败: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"item":    item,
	})
}

func (a *App) handleRemoveMtPhotoFolderFavorite(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhotoFolderFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 文件夹收藏服务未初始化"})
		return
	}

	var req struct {
		FolderID int64 `json:"folderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求体格式错误"})
		return
	}
	if req.FolderID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "folderId 参数非法"})
		return
	}

	if err := a.mtPhotoFolderFavorite.Remove(r.Context(), req.FolderID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "移除收藏失败: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
