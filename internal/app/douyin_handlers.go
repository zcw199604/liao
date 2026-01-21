package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

type douyinDetailRequest struct {
	Input  string `json:"input"`
	Cookie string `json:"cookie,omitempty"`
	Proxy  string `json:"proxy,omitempty"`
}

type douyinDetailResponse struct {
	Key      string            `json:"key"`
	DetailID string            `json:"detailId"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	CoverURL string            `json:"coverUrl,omitempty"`
	Items    []douyinMediaItem `json:"items"`
}

type douyinMediaItem struct {
	Index            int    `json:"index"`
	Type             string `json:"type"` // image|video
	URL              string `json:"url"`
	DownloadURL      string `json:"downloadUrl"`
	OriginalFilename string `json:"originalFilename,omitempty"`
}

func (a *App) handleDouyinDetail(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	var req douyinDetailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	input := strings.TrimSpace(req.Input)
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "input 不能为空"})
		return
	}

	detailID, resolvedURL, err := a.douyinDownloader.ResolveDetailID(r.Context(), input, req.Proxy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	detail, err := a.douyinDownloader.FetchDetail(r.Context(), detailID, req.Cookie, req.Proxy)
	if err != nil {
		msg := err.Error()
		if resolvedURL != "" {
			msg = msg + "（resolved=" + resolvedURL + "）"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": msg})
		return
	}

	key := a.douyinDownloader.CacheDetail(detail)
	if key == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "缓存失败"})
		return
	}

	itemType := "video"
	if strings.Contains(detail.Type, "图集") {
		itemType = "image"
	}

	items := make([]douyinMediaItem, 0, len(detail.Downloads))
	for i, u := range detail.Downloads {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		ext := guessExtFromURL(u)
		if ext == "" {
			if itemType == "video" {
				ext = ".mp4"
			} else {
				ext = ".jpg"
			}
		}

		items = append(items, douyinMediaItem{
			Index:            i,
			Type:             itemType,
			URL:              u,
			DownloadURL:      fmt.Sprintf("/api/douyin/download?key=%s&index=%d", url.QueryEscape(key), i),
			OriginalFilename: buildDouyinOriginalFilename(detail.Title, detail.DetailID, i, len(detail.Downloads), ext),
		})
	}

	writeJSON(w, http.StatusOK, douyinDetailResponse{
		Key:      key,
		DetailID: detail.DetailID,
		Type:     detail.Type,
		Title:    detail.Title,
		CoverURL: detail.CoverURL,
		Items:    items,
	})
}

func (a *App) handleDouyinDownload(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	indexValue := strings.TrimSpace(r.URL.Query().Get("index"))
	index, err := strconv.Atoi(indexValue)
	if key == "" || indexValue == "" || err != nil || index < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key/index 非法"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}
	if index >= len(cached.Downloads) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "index 越界"})
		return
	}

	remoteURL := strings.TrimSpace(cached.Downloads[index])
	if remoteURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接为空"})
		return
	}

	if r.Method == http.MethodHead {
		a.handleDouyinDownloadHead(w, r, cached, index, remoteURL)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, remoteURL, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接非法"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.douyin.com/")

	resp, err := a.douyinDownloader.api.httpClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", resp.Status, strings.TrimSpace(string(body)))})
		return
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	ext := guessExtFromContentType(contentType)
	if ext == "" {
		ext = guessExtFromURL(remoteURL)
	}
	if ext == "" {
		ext = ".bin"
	}

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	fallback := buildDouyinFallbackFilename(cached.DetailID, index, len(cached.Downloads), ext)

	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, originalFilename))

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleDouyinDownloadHead(w http.ResponseWriter, r *http.Request, cached *douyinCachedDetail, index int, remoteURL string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodHead, remoteURL, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接非法"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.douyin.com/")

	resp, err := a.douyinDownloader.api.httpClient.Do(req)
	if err == nil && resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// 部分 CDN 不支持 HEAD：fallback 到 Range=0-0 的 GET，最佳努力拿到 Content-Length。
	if err != nil || resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rangeReq, reqErr := http.NewRequestWithContext(r.Context(), http.MethodGet, remoteURL, nil)
		if reqErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接非法"})
			return
		}
		rangeReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		rangeReq.Header.Set("Referer", "https://www.douyin.com/")
		rangeReq.Header.Set("Range", "bytes=0-0")

		rangeResp, rangeErr := a.douyinDownloader.api.httpClient.Do(rangeReq)
		if rangeErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + rangeErr.Error()})
			return
		}
		defer rangeResp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(rangeResp.Body, 1))

		if rangeResp.StatusCode < 200 || rangeResp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(rangeResp.Body, 1024))
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", rangeResp.Status, strings.TrimSpace(string(body)))})
			return
		}

		a.writeDouyinDownloadHeaders(w, cached, index, remoteURL, rangeResp.Header, true)
		w.WriteHeader(http.StatusOK)
		return
	}

	a.writeDouyinDownloadHeaders(w, cached, index, remoteURL, resp.Header, false)
	w.WriteHeader(http.StatusOK)
}

func (a *App) writeDouyinDownloadHeaders(w http.ResponseWriter, cached *douyinCachedDetail, index int, remoteURL string, hdr http.Header, ranged bool) {
	contentType := strings.TrimSpace(hdr.Get("Content-Type"))
	ext := guessExtFromContentType(contentType)
	if ext == "" {
		ext = guessExtFromURL(remoteURL)
	}
	if ext == "" {
		ext = ".bin"
	}

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	fallback := buildDouyinFallbackFilename(cached.DetailID, index, len(cached.Downloads), ext)

	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, originalFilename))

	acceptRanges := strings.TrimSpace(hdr.Get("Accept-Ranges"))
	if acceptRanges != "" {
		w.Header().Set("Accept-Ranges", acceptRanges)
	}

	// ranged=true 时优先从 Content-Range 获取总大小，避免 Content-Length=1 的情况。
	var lengthValue string
	if ranged {
		if total := parseContentRangeTotal(hdr.Get("Content-Range")); total > 0 {
			lengthValue = fmt.Sprintf("%d", total)
		}
	}
	if lengthValue == "" {
		lengthValue = strings.TrimSpace(hdr.Get("Content-Length"))
	}
	if lengthValue != "" {
		w.Header().Set("Content-Length", lengthValue)
	}
}

func (a *App) handleDouyinImport(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil || a.fileStorage == nil || a.mediaUpload == nil || a.imageServer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	userID := strings.TrimSpace(r.FormValue("userid"))
	key := strings.TrimSpace(r.FormValue("key"))
	indexValue := strings.TrimSpace(r.FormValue("index"))
	index, err := strconv.Atoi(indexValue)
	if userID == "" || key == "" || indexValue == "" || err != nil || index < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "userid/key/index 不能为空或非法"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}
	if index >= len(cached.Downloads) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "index 越界"})
		return
	}

	remoteURL := strings.TrimSpace(cached.Downloads[index])
	if remoteURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接为空"})
		return
	}

	downloadReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, remoteURL, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接非法"})
		return
	}
	downloadReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	downloadReq.Header.Set("Referer", "https://www.douyin.com/")

	downloadResp, err := a.douyinDownloader.api.httpClient.Do(downloadReq)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err.Error()})
		return
	}
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode < 200 || downloadResp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(downloadResp.Body, 1024))
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", downloadResp.Status, strings.TrimSpace(string(body)))})
		return
	}

	contentType := strings.TrimSpace(downloadResp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = inferContentTypeFromFilename("x" + guessExtFromURL(remoteURL))
	}
	if strings.TrimSpace(contentType) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "无法识别文件类型"})
		return
	}
	if !a.fileStorage.IsValidMediaType(contentType) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "不支持的文件类型"})
		return
	}

	ext := guessExtFromContentType(contentType)
	if ext == "" {
		ext = guessExtFromURL(remoteURL)
	}
	if ext == "" {
		ext = ".bin"
	}

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	localPath, fileSize, md5Value, err := a.fileStorage.SaveFileFromReader(originalFilename, contentType, downloadResp.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "本地存储失败: " + err.Error()})
		return
	}

	// 已导入过则直接复用：避免重复写文件导致孤儿文件
	if md5Value != "" {
		if existing, err := a.mediaUpload.findMediaFileByUserAndMD5(r.Context(), userID, md5Value); err == nil && existing != nil {
			_ = a.fileStorage.DeleteFile(localPath)
			a.imageCache.AddImageToCache(userID, existing.LocalPath)

			port := ""
			if strings.HasPrefix(strings.ToLower(existing.FileType), "video/") || strings.EqualFold(existing.FileExtension, "mp4") {
				port = "8006"
			} else {
				port = a.resolveImagePortByConfig(r.Context(), existing.RemoteFilename)
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"state":         "OK",
				"msg":           existing.RemoteFilename,
				"port":          port,
				"localFilename": existing.LocalFilename,
				"dedup":         true,
			})
			return
		}
	}

	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	imgServerHost := a.imageServer.GetImgServerHost()
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userID)

	uploadAbs := filepath.Join(a.fileStorage.baseUploadAbs, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	respBody, err := a.uploadAbsPathToUpstream(r.Context(), uploadURL, imgServerHost, uploadAbs, originalFilename, cookieData, referer, userAgent)
	if err != nil {
		slog.Error("抖音导入上传失败", "error", err, "userId", userID, "localPath", localPath)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "导入上传失败: " + err.Error(),
			"localPath": localPath,
		})
		return
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(respBody), &parsed); err == nil {
		if state, _ := parsed["state"].(string); state == "OK" {
			if msg, ok := parsed["msg"].(string); ok && strings.TrimSpace(msg) != "" {
				imgHostClean := strings.Split(imgServerHost, ":")[0]
				availablePort := ""
				if strings.HasPrefix(strings.ToLower(contentType), "video/") {
					availablePort = "8006"
				} else {
					availablePort = a.resolveImagePortByConfig(r.Context(), msg)
				}
				imageURL := fmt.Sprintf("http://%s:%s/img/Upload/%s", imgHostClean, availablePort, msg)

				localFilename := filepath.Base(strings.TrimPrefix(localPath, "/"))

				_, _ = a.mediaUpload.SaveUploadRecord(r.Context(), UploadRecord{
					UserID:           userID,
					OriginalFilename: originalFilename,
					LocalFilename:    localFilename,
					RemoteFilename:   msg,
					RemoteURL:        imageURL,
					LocalPath:        localPath,
					FileSize:         fileSize,
					FileType:         contentType,
					FileExtension:    a.fileStorage.FileExtension(originalFilename),
					FileMD5:          md5Value,
				})

				a.imageCache.AddImageToCache(userID, localPath)

				writeJSON(w, http.StatusOK, map[string]any{
					"state":         "OK",
					"msg":           msg,
					"port":          availablePort,
					"localFilename": localFilename,
					"dedup":         false,
				})
				return
			}
		}
	}

	// 未增强：保持兼容
	writeText(w, http.StatusOK, respBody)
}

func guessExtFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err == nil && u != nil {
		ext := strings.ToLower(filepath.Ext(u.Path))
		if ext != "" && len(ext) <= 10 {
			return ext
		}
		return ""
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if ext != "" && len(ext) <= 10 {
		return ext
	}
	return ""
}

func buildDouyinOriginalFilename(title, detailID string, index, total int, ext string) string {
	base := sanitizeFilename(title)
	if base == "" {
		base = strings.TrimSpace(detailID)
	}
	if strings.TrimSpace(ext) == "" {
		ext = ".bin"
	}
	if total > 1 {
		return fmt.Sprintf("%s_%02d%s", base, index+1, ext)
	}
	return base + ext
}

func buildDouyinFallbackFilename(detailID string, index, total int, ext string) string {
	id := strings.TrimSpace(detailID)
	if id == "" {
		id = "unknown"
	}
	if strings.TrimSpace(ext) == "" {
		ext = ".bin"
	}
	if total > 1 {
		return fmt.Sprintf("douyin_%s_%02d%s", id, index+1, ext)
	}
	return fmt.Sprintf("douyin_%s%s", id, ext)
}

func buildAttachmentContentDisposition(fallback, originalFilename string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "download"
	}
	originalFilename = strings.TrimSpace(originalFilename)
	if originalFilename == "" {
		originalFilename = fallback
	}
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, url.PathEscape(originalFilename))
}

func sanitizeFilename(input string) string {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return ""
	}

	// 替换常见非法字符（Windows/URL/控制字符）
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	raw = replacer.Replace(raw)
	raw = strings.TrimSpace(strings.Join(strings.Fields(raw), " "))

	// 控制长度（按 rune 避免截断中文）
	rs := []rune(raw)
	if len(rs) > 100 {
		raw = string(rs[:100])
	}
	return strings.TrimSpace(raw)
}

func parseContentRangeTotal(contentRange string) int64 {
	raw := strings.TrimSpace(contentRange)
	if raw == "" {
		return 0
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return 0
	}
	totalStr := strings.TrimSpace(parts[1])
	if totalStr == "" || totalStr == "*" {
		return 0
	}
	n, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
