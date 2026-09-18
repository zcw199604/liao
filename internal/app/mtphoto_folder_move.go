package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Read raw IDs: display mapping intentionally omits files without MD5 thumbnails.
func (s *MtPhotoService) readMoveFolder(ctx context.Context, id int64) (*mtPhotoFolderResponse, error) {
	resp, err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("%s/gateway/foldersV2/%d", s.baseURL, id), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取文件夹失败：%s", resp.Status)
	}
	var folder mtPhotoFolderResponse
	if err = json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return nil, err
	}
	if strings.TrimSpace(folder.Path) == "" {
		return nil, fmt.Errorf("文件夹响应缺少路径")
	}
	return &folder, nil
}

func (a *App) handleMoveMtPhotoFolderFiles(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, 500, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}
	var in struct {
		SourceID          int64 `json:"sourceId"`
		TargetID          int64 `json:"targetId"`
		IncludeSubfolders bool  `json:"includeSubfolders"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil || in.SourceID <= 0 || in.TargetID <= 0 || in.SourceID == in.TargetID {
		writeJSON(w, 400, map[string]any{"error": "请选择不同的源文件夹和目标文件夹"})
		return
	}
	s := a.mtPhoto
	source, err := s.readMoveFolder(r.Context(), in.SourceID)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	target, err := s.readMoveFolder(r.Context(), in.TargetID)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	sourcePath := strings.TrimRight(source.Path, "/")
	targetPath := strings.TrimRight(target.Path, "/")
	if sourcePath == targetPath || (in.IncludeSubfolders && strings.HasPrefix(targetPath, sourcePath+"/")) {
		writeJSON(w, 400, map[string]any{"error": "目标不能是源目录；包含子目录时，目标也不能位于源目录内"})
		return
	}
	ids := []int64{}
	seenFiles := map[int64]bool{}
	seenFolders := map[int64]bool{in.SourceID: true}
	queue := []*mtPhotoFolderResponse{source}
	for len(queue) > 0 {
		folder := queue[0]
		queue = queue[1:]
		for _, file := range folder.FileList {
			if file.ID <= 0 {
				writeJSON(w, 502, map[string]any{"error": "源目录返回无效文件 ID，未执行移动"})
				return
			}
			if !seenFiles[file.ID] {
				seenFiles[file.ID] = true
				ids = append(ids, file.ID)
			}
		}
		if !in.IncludeSubfolders {
			continue
		}
		for _, child := range folder.FolderList {
			if child.ID <= 0 {
				writeJSON(w, 502, map[string]any{"error": "源目录返回无效子目录 ID，未执行移动"})
				return
			}
			if seenFolders[child.ID] {
				continue
			}
			seenFolders[child.ID] = true
			next, e := s.readMoveFolder(r.Context(), child.ID)
			if e != nil {
				writeJSON(w, 502, map[string]any{"error": "读取子目录失败，未执行移动：" + e.Error()})
				return
			}
			queue = append(queue, next)
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]any{"success": true, "count": 0})
		return
	}
	body, _ := json.Marshal(map[string]any{"type": "move", "fileIds": ids, "distId": in.TargetID, "overwrite": 2})
	resp, err := s.doRequest(r.Context(), http.MethodPost, s.baseURL+"/gateway/filePathEdit", map[string]string{"Content-Type": "application/json"}, body)
	// Even a failed response can follow partial upstream writes.
	s.mu.Lock()
	s.resetCachesLocked()
	s.mu.Unlock()
	fail := func(message string) {
		writeJSON(w, 502, map[string]any{"error": message + "；请刷新源目录和目标目录确认实际结果后再操作"})
	}
	if err != nil {
		fail("移动结果未确认：" + err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fail("mtPhoto 移动失败：" + resp.Status)
		return
	}
	var result struct {
		N    int    `json:"n"`
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fail("移动响应解析失败")
		return
	}
	if result.Code != "" || result.Msg != "" || result.N != 1 {
		fail("mtPhoto 未确认移动成功 " + result.Code + " " + result.Msg)
		return
	}
	writeJSON(w, 200, map[string]any{"success": true, "count": len(ids)})
}
