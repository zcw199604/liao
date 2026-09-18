package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type mtPhotoFolderAlbumInput struct {
	Name    string `json:"name"`
	Folders []struct {
		ID   int64  `json:"id"`
		Path string `json:"path"`
	} `json:"folders"`
}

// A non-nil album with an error means creation succeeded but configuration did not
// finish. Preserve its identity so callers do not blindly create another album.
func (s *MtPhotoService) createFolderAlbum(ctx context.Context, input mtPhotoFolderAlbumInput) (*MtPhotoAlbum, error) {
	var album MtPhotoAlbum
	if err := s.postAlbumJSON(ctx, "/api-album", map[string]any{"name": input.Name, "files": []string{}}, &album); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.albumsCache = nil
	s.albumsCacheExpire = time.Time{}
	s.mu.Unlock()
	if album.ID <= 0 {
		return nil, fmt.Errorf("mtPhoto 创建响应缺少相册 ID，请刷新相册列表确认创建结果后再操作")
	}
	seen := make(map[int64]bool)
	for _, folder := range input.Folders {
		if seen[folder.ID] {
			continue
		}
		seen[folder.ID] = true
		value, _ := json.Marshal(map[string]any{"id": folder.ID, "label": folder.Path})
		var result struct {
			N int `json:"n"`
		}
		// Official web client uses false for inclusion; the OpenAPI field description is reversed.
		err := s.postAlbumJSON(ctx, "/api-album/link/"+strconv.Itoa(album.ID), map[string]any{
			"type": "folder", "value": string(value), "exclude": false,
		}, &result)
		if err == nil && result.N != 1 {
			err = fmt.Errorf("mtPhoto 未确认文件夹关联成功")
		}
		if err != nil {
			return &album, fmt.Errorf("相册已创建（ID %d），关联文件夹 %s 失败；请在 mtPhoto 中补充关联，避免重复创建：%w", album.ID, folder.Path, err)
		}
	}
	var result struct {
		N int `json:"n"`
	}
	err := s.postAlbumJSON(ctx, "/api-album/linkSyncFiles/"+strconv.Itoa(album.ID), map[string]any{}, &result)
	if err == nil && result.N != 1 {
		err = fmt.Errorf("mtPhoto 未确认同步成功")
	}
	if err != nil {
		return &album, fmt.Errorf("相册已创建（ID %d）且文件夹已关联，但同步失败；请在 mtPhoto 中重新同步，避免重复创建：%w", album.ID, err)
	}
	return &album, nil
}

func (s *MtPhotoService) postAlbumJSON(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	resp, err := s.doRequest(ctx, http.MethodPost, s.baseURL+path, map[string]string{
		"Content-Type": "application/json", "Accept": "application/json",
	}, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &mtPhotoStatusError{StatusCode: resp.StatusCode, Status: resp.Status, Action: "mtPhoto 相册操作失败"}
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func (a *App) handleCreateMtPhotoFolderAlbum(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}
	var input mtPhotoFolderAlbumInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求 JSON 非法"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Folders) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请填写相册名称并选择至少一个文件夹"})
		return
	}
	for i := range input.Folders {
		folder := &input.Folders[i]
		folder.Path = strings.TrimSpace(folder.Path)
		if folder.ID <= 0 || folder.Path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "文件夹 ID 或路径非法"})
			return
		}
	}
	album, err := a.mtPhoto.createFolderAlbum(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"success": false, "album": album, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "album": album})
}
