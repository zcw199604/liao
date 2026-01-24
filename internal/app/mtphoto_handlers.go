package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) handleGetMtPhotoAlbums(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}

	albums, err := a.mtPhoto.GetAlbums(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": albums})
}

func (a *App) handleGetMtPhotoAlbumFiles(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}

	q := r.URL.Query()
	albumID := parseIntDefault(q.Get("albumId"), 0)
	page := parseIntDefault(q.Get("page"), 1)
	pageSize := parseIntDefault(q.Get("pageSize"), 60)

	items, total, totalPages, err := a.mtPhoto.GetAlbumFilesPage(r.Context(), albumID, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":       items,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
	})
}

func (a *App) handleGetMtPhotoThumb(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query()
	size := strings.TrimSpace(q.Get("size"))
	md5Value := strings.TrimSpace(q.Get("md5"))
	if size == "" || md5Value == "" {
		http.NotFound(w, r)
		return
	}

	// 白名单：避免被滥用为开放代理
	switch size {
	case "s260", "h220":
	default:
		http.NotFound(w, r)
		return
	}

	resp, err := a.mtPhoto.GatewayGet(r.Context(), size, md5Value)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()

	// 仅透传必要响应头，避免 Set-Cookie 等敏感头下发到前端
	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cc := strings.TrimSpace(resp.Header.Get("Cache-Control")); cc != "" {
		w.Header().Set("Cache-Control", cc)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleDownloadMtPhotoOriginal(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}

	q := r.URL.Query()
	fileID := int64(parseIntDefault(q.Get("id"), 0))
	md5Value := strings.TrimSpace(q.Get("md5"))
	if fileID <= 0 || !isValidMD5Hex(md5Value) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id/md5 参数非法"})
		return
	}

	resp, err := a.mtPhoto.GatewayFileDownload(r.Context(), fileID, md5Value)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "mtPhoto 下载失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "mtPhoto 下载失败: " + resp.Status})
		return
	}

	// 仅透传必要响应头，避免 Set-Cookie 等敏感头下发到前端
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if cc := strings.TrimSpace(resp.Header.Get("Cache-Control")); cc != "" {
		w.Header().Set("Cache-Control", cc)
	}
	if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	if ar := strings.TrimSpace(resp.Header.Get("Accept-Ranges")); ar != "" {
		w.Header().Set("Accept-Ranges", ar)
	}

	disp := strings.TrimSpace(resp.Header.Get("Content-Disposition"))
	if disp == "" {
		// mtPhoto 部分版本不会返回文件名；缺失时用 filesInMD5 解析文件名以改善下载体验。
		originalFilename := ""
		if item, err := a.mtPhoto.ResolveFilePath(r.Context(), md5Value); err == nil && item != nil {
			originalFilename = filepath.Base(strings.TrimSpace(item.FilePath))
		}
		disp = buildDownloadContentDisposition(originalFilename, md5Value, contentType)
	}
	w.Header().Set("Content-Disposition", disp)

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleResolveMtPhotoFilePath(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "mtPhoto 服务未初始化"})
		return
	}

	md5Value := strings.TrimSpace(r.URL.Query().Get("md5"))
	item, err := a.mtPhoto.ResolveFilePath(r.Context(), md5Value)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":       item.ID,
		"filePath": item.FilePath,
	})
}

func (a *App) handleImportMtPhotoMedia(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.mtPhoto == nil || a.fileStorage == nil || a.mediaUpload == nil || a.imageServer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	userID := strings.TrimSpace(r.FormValue("userid"))
	md5Value := strings.TrimSpace(r.FormValue("md5"))
	if userID == "" || md5Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "userid/md5 不能为空"})
		return
	}

	// 已导入过则直接复用：避免重复写文件导致孤儿文件
	if existing, err := a.mediaUpload.findMediaFileByUserAndMD5(r.Context(), userID, md5Value); err == nil && existing != nil {
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
		})
		return
	}

	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// 1) 通过 mtPhoto 查询本地文件路径
	item, err := a.mtPhoto.ResolveFilePath(r.Context(), md5Value)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "获取文件路径失败: " + err.Error()})
		return
	}

	// 2) 将 /lsp/... 映射到实际文件系统路径（支持 LSP_ROOT 重定向）
	absPath, err := resolveLspLocalPath(a.cfg.LspRoot, item.FilePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "文件路径非法: " + err.Error()})
		return
	}

	originalFilename := filepath.Base(absPath)
	contentType := inferContentTypeFromFilename(originalFilename)
	if contentType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "不支持的文件类型"})
		return
	}

	// 3) 先保存到本地 upload/，即便上游失败也可在“全站图片库”中重试
	src, err := openLocalFileForRead(absPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "读取本地文件失败: " + err.Error()})
		return
	}
	defer src.Close()

	localPath, fileSize, computedMD5, err := a.fileStorage.SaveFileFromReader(originalFilename, contentType, src)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "本地存储失败: " + err.Error()})
		return
	}

	// 轻量一致性提示：不阻断，只打日志
	if computedMD5 != "" && strings.EqualFold(computedMD5, md5Value) == false {
		slog.Warn("mtPhoto md5 与本地文件 md5 不一致(继续导入)", "md5Input", md5Value, "md5Local", computedMD5)
	}

	// 4) 上传到上游
	imgServerHost := a.imageServer.GetImgServerHost()
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userID)

	uploadAbs := filepath.Join(a.fileStorage.baseUploadAbs, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	respBody, err := a.uploadAbsPathToUpstream(r.Context(), uploadURL, imgServerHost, uploadAbs, originalFilename, cookieData, referer, userAgent)
	if err != nil {
		slog.Error("导入上传失败", "error", err, "userId", userID, "localPath", localPath)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "导入上传失败: " + err.Error(),
			"localPath": localPath,
		})
		return
	}

	// 5) 解析并增强返回（对齐 /api/uploadMedia）
	var parsed map[string]any
	if err := json.Unmarshal([]byte(respBody), &parsed); err == nil {
		if state, _ := parsed["state"].(string); state == "OK" {
			if msg, ok := parsed["msg"].(string); ok && msg != "" {
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

				enhanced := map[string]any{
					"state":         "OK",
					"msg":           msg,
					"port":          availablePort,
					"localFilename": localFilename,
				}
				writeJSON(w, http.StatusOK, enhanced)
				return
			}
		}
	}

	// 未增强：保持兼容
	writeText(w, http.StatusOK, respBody)
}

func inferContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	default:
		return ""
	}
}

func isValidMD5Hex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		case ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
}

func guessExtFromContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" {
		return ""
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	default:
		return ""
	}
}

func buildDownloadContentDisposition(originalFilename, md5Value, contentType string) string {
	originalFilename = strings.TrimSpace(originalFilename)
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		ext = guessExtFromContentType(contentType)
	}

	fallback := "mtphoto_" + md5Value
	if ext != "" {
		fallback += ext
	}
	if originalFilename == "" {
		originalFilename = fallback
	}

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, url.PathEscape(originalFilename))
}

func openLocalFileForRead(absPath string) (*os.File, error) {
	return os.Open(absPath)
}

func (a *App) uploadAbsPathToUpstream(ctx context.Context, uploadURL, imgServerHost, absPath, filename, cookieData, referer, userAgent string) (string, error) {
	// 说明：复用现有 uploadToUpstream 的协议与 headers，只是数据源改为本地路径。
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func() { _ = pw.Close() }()
		defer func() { _ = writer.Close() }()

		part, err := writer.CreateFormFile("upload_file", filename)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		src, err := openLocalFileForRead(absPath)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		defer src.Close()

		if _, err := io.Copy(part, src); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Host = strings.Split(imgServerHost, ":")[0]
	req.Header.Set("Origin", "http://v1.chat2019.cn")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)
	if cookieData != "" {
		req.Header.Set("Cookie", cookieData)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("上游响应异常: %s", resp.Status)
	}
	return string(b), nil
}
