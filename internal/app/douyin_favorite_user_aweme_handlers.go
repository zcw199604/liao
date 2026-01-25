package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type douyinFavoriteUserAwemeUpsertBatchRequest struct {
	SecUserID string                              `json:"secUserId"`
	Items     []douyinFavoriteUserAwemeUpsertItem `json:"items"`
}

type douyinFavoriteUserAwemeUpsertItem struct {
	AwemeID   string   `json:"awemeId"`
	Type      string   `json:"type,omitempty"`
	Desc      string   `json:"desc,omitempty"`
	CoverURL  string   `json:"coverUrl,omitempty"`
	Downloads []string `json:"downloads,omitempty"`
}

func (a *App) handleDouyinFavoriteUserAwemeUpsert(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	var req douyinFavoriteUserAwemeUpsertBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	secUserID := strings.TrimSpace(req.SecUserID)
	if secUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secUserId 不能为空"})
		return
	}

	upserts := make([]DouyinFavoriteUserAwemeUpsert, 0, len(req.Items))
	for _, it := range req.Items {
		awemeID := strings.TrimSpace(it.AwemeID)
		if awemeID == "" {
			continue
		}
		upserts = append(upserts, DouyinFavoriteUserAwemeUpsert{
			AwemeID:   awemeID,
			Type:      strings.TrimSpace(it.Type),
			Desc:      strings.TrimSpace(it.Desc),
			CoverURL:  strings.TrimSpace(it.CoverURL),
			Downloads: it.Downloads,
		})
	}

	added, err := a.douyinFavorite.UpsertUserAwemes(r.Context(), secUserID, upserts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "added": added})
}

func (a *App) handleDouyinFavoriteUserAwemeList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	secUserID := strings.TrimSpace(r.URL.Query().Get("secUserId"))
	if secUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secUserId 不能为空"})
		return
	}

	cursor := 0
	if v := strings.TrimSpace(r.URL.Query().Get("cursor")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cursor = n
		}
	}
	count := 20
	if v := strings.TrimSpace(r.URL.Query().Get("count")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			count = n
		}
	}

	rows, nextCursor, hasMore, err := a.douyinFavorite.ListUserAwemes(r.Context(), secUserID, cursor, count)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询失败"})
		return
	}

	items := make([]douyinAccountItem, 0, len(rows))
	for _, row := range rows {
		it := douyinAccountItem{
			DetailID: strings.TrimSpace(row.AwemeID),
			Type:     strings.TrimSpace(row.Type),
			Desc:     strings.TrimSpace(row.Desc),
			CoverURL: strings.TrimSpace(row.CoverURL),
			Key:      "",
			Items:    []douyinMediaItem{},
		}

		downloads := normalizeStringList(row.Downloads)
		if a.douyinDownloader != nil && len(downloads) > 0 {
			displayType := strings.TrimSpace(it.Type)
			if displayType != "image" && displayType != "video" {
				displayType = "video"
				for _, u := range downloads {
					if guessDouyinMediaTypeFromURL(u) == "image" {
						displayType = "image"
						break
					}
				}
			}
			typeLabel := map[string]string{"video": "视频", "image": "图集"}[displayType]

			cached := &douyinCachedDetail{
				DetailID:  it.DetailID,
				Title:     strings.TrimSpace(it.Desc),
				Type:      typeLabel,
				CoverURL:  strings.TrimSpace(it.CoverURL),
				Downloads: downloads,
			}
			key := a.douyinDownloader.CacheDetail(cached)
			if key != "" {
				it.Key = key
				if strings.TrimSpace(it.CoverURL) != "" {
					it.CoverDownloadURL = fmt.Sprintf("/api/douyin/cover?key=%s", url.QueryEscape(key))
				}

				previewItems := make([]douyinMediaItem, 0, len(downloads))
				for i, u := range downloads {
					u = strings.TrimSpace(u)
					if u == "" {
						continue
					}
					mediaType := guessDouyinMediaTypeFromURL(u)
					ext := guessExtFromURL(u)
					if ext == "" {
						if mediaType == "video" {
							ext = ".mp4"
						} else {
							ext = ".jpg"
						}
					}
					previewItems = append(previewItems, douyinMediaItem{
						Index:            i,
						Type:             mediaType,
						URL:              u,
						DownloadURL:      fmt.Sprintf("/api/douyin/download?key=%s&index=%d", url.QueryEscape(key), i),
						OriginalFilename: buildDouyinOriginalFilename(strings.TrimSpace(it.Desc), it.DetailID, i, len(downloads), ext),
					})
				}
				it.Items = previewItems
			}
		}

		items = append(items, it)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"cursor":    nextCursor,
		"hasMore":   hasMore,
		"secUserId": secUserID,
	})
}

type douyinFavoriteUserAwemePullLatestRequest struct {
	SecUserID string `json:"secUserId"`
	Cookie    string `json:"cookie,omitempty"`
	Count     int    `json:"count,omitempty"`
}

func (a *App) handleDouyinFavoriteUserAwemePullLatest(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinFavorite == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	var req douyinFavoriteUserAwemePullLatestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	secUserID := strings.TrimSpace(req.SecUserID)
	if secUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "secUserId 不能为空"})
		return
	}
	count := req.Count
	if count <= 0 {
		count = 50
	}
	if count > 200 {
		count = 200
	}

	data, err := a.douyinDownloader.FetchAccount(r.Context(), secUserID, "post", strings.TrimSpace(req.Cookie), "", 0, count)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	accountItems := extractDouyinAccountItems(a.douyinDownloader, secUserID, data)
	upserts := make([]DouyinFavoriteUserAwemeUpsert, 0, len(accountItems))
	for _, it := range accountItems {
		awemeID := strings.TrimSpace(it.DetailID)
		if awemeID == "" {
			continue
		}
		downloads := make([]string, 0, len(it.Items))
		for _, m := range it.Items {
			if u := strings.TrimSpace(m.URL); u != "" {
				downloads = append(downloads, u)
			}
		}
		upserts = append(upserts, DouyinFavoriteUserAwemeUpsert{
			AwemeID:   awemeID,
			Type:      strings.TrimSpace(it.Type),
			Desc:      strings.TrimSpace(it.Desc),
			CoverURL:  strings.TrimSpace(it.CoverURL),
			Downloads: downloads,
		})
	}

	added, err := a.douyinFavorite.UpsertUserAwemes(r.Context(), secUserID, upserts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"added":     added,
		"fetched":   len(upserts),
		"secUserId": secUserID,
	})
}
