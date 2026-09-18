package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

func (s *MtPhotoService) albumFolderIDs(ctx context.Context, albumID int) ([]int64, error) {
	resp, err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api-album/link/%d", s.baseURL, albumID), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取相册 %d 关联失败：%s", albumID, resp.Status)
	}
	var links []struct {
		Type    string `json:"type"`
		Value   string `json:"value"`
		Exclude bool   `json:"exclude"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return nil, err
	}
	ids := []int64{}
	seen := map[int64]bool{}
	for _, link := range links {
		if link.Type != "folder" || link.Exclude {
			continue
		}
		var value struct {
			ID int64 `json:"id"`
		}
		if err = json.Unmarshal([]byte(link.Value), &value); err != nil || value.ID <= 0 {
			return nil, fmt.Errorf("相册 %d 的文件夹关联格式无效", albumID)
		}
		if !seen[value.ID] {
			seen[value.ID] = true
			ids = append(ids, value.ID)
		}
	}
	return ids, nil
}

func (a *App) handleGetMtPhotoFolderAlbumLinks(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, 500, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}
	albums, err := a.mtPhoto.GetAlbums(r.Context())
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	// Limit upstream concurrency while retaining the album list order in the response.
	ids := make([][]int64, len(albums))
	errs := make([]error, len(albums))
	for start := 0; start < len(albums); start += 4 {
		var wg sync.WaitGroup
		for i := start; i < len(albums) && i < start+4; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				ids[index], errs[index] = a.mtPhoto.albumFolderIDs(r.Context(), albums[index].ID)
			}(i)
		}
		wg.Wait()
		for i := start; i < len(albums) && i < start+4; i++ {
			if errs[i] != nil {
				writeJSON(w, 502, map[string]any{"error": "文件夹相册状态查询失败：" + errs[i].Error()})
				return
			}
		}
	}
	items := map[string][]MtPhotoAlbum{}
	for i, album := range albums {
		for _, id := range ids[i] {
			key := strconv.FormatInt(id, 10)
			items[key] = append(items[key], album)
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
