package app

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (a *App) handleDouyinLivePhoto(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 不能为空"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}

	imageIndex, err := parseOptionalInt(r.URL.Query().Get("imageIndex"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "imageIndex 非法"})
		return
	}
	videoIndex, err := parseOptionalInt(r.URL.Query().Get("videoIndex"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "videoIndex 非法"})
		return
	}

	imgIdx, vidIdx, errMsg := selectDouyinLivePhotoPair(cached.Downloads, imageIndex, videoIndex)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": errMsg})
		return
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务器缺少 ffmpeg，无法生成实况文件"})
		return
	}
	if _, err := exec.LookPath("exiftool"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务器缺少 exiftool，无法生成 iOS 实况文件"})
		return
	}

	remoteImageURL := strings.TrimSpace(cached.Downloads[imgIdx])
	remoteVideoURL := strings.TrimSpace(cached.Downloads[vidIdx])
	if remoteImageURL == "" || remoteVideoURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "实况资源链接为空"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "douyin-livephoto-")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "创建临时目录失败"})
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	rawImagePath := filepath.Join(tmpDir, "image.raw")
	rawVideoPath := filepath.Join(tmpDir, "video.raw")

	imgContentType, err := downloadDouyinResourceToFile(ctx, a.douyinDownloader.api.httpClient, remoteImageURL, rawImagePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载图片失败: " + err.Error()})
		return
	}

	vidContentType, err := downloadDouyinResourceToFile(ctx, a.douyinDownloader.api.httpClient, remoteVideoURL, rawVideoPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载视频失败: " + err.Error()})
		return
	}

	stillPath := filepath.Join(tmpDir, "still.jpg")
	if err := normalizeLivePhotoStillImage(ctx, rawImagePath, imgContentType, stillPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "图片转码失败: " + err.Error()})
		return
	}

	motionPath := filepath.Join(tmpDir, "motion.mov")
	if err := normalizeLivePhotoMotionVideo(ctx, rawVideoPath, vidContentType, motionPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "视频转码失败: " + err.Error()})
		return
	}

	assetID := strings.ToUpper(uuid.NewString())
	if err := tagLivePhotoAsset(ctx, stillPath, motionPath, assetID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "写入实况元数据失败: " + err.Error()})
		return
	}

	base := sanitizeFilename(cached.Title)
	if strings.TrimSpace(base) == "" {
		base = strings.TrimSpace(cached.DetailID)
	}
	if strings.TrimSpace(base) == "" {
		base = "livephoto"
	}

	zipFilename := base + "_livephoto.zip"

	// 处理完成后再开始写入响应，避免中途失败导致已输出不可恢复
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition("livephoto.zip", zipFilename))
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer func() { _ = zw.Close() }()

	if err := zipFile(zw, stillPath, base+".jpg"); err != nil {
		return
	}
	if err := zipFile(zw, motionPath, base+".mov"); err != nil {
		return
	}
}

func parseOptionalInt(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func selectDouyinLivePhotoPair(downloads []string, imageIndex, videoIndex *int) (imgIdx int, vidIdx int, errMsg string) {
	if len(downloads) == 0 {
		return 0, 0, "downloads 为空"
	}
	if imageIndex != nil && (*imageIndex < 0 || *imageIndex >= len(downloads)) {
		return 0, 0, "imageIndex 越界"
	}
	if videoIndex != nil && (*videoIndex < 0 || *videoIndex >= len(downloads)) {
		return 0, 0, "videoIndex 越界"
	}

	if imageIndex != nil && videoIndex != nil {
		if guessDouyinMediaTypeFromURL(downloads[*imageIndex]) != "image" {
			return 0, 0, "imageIndex 不是图片资源"
		}
		if guessDouyinMediaTypeFromURL(downloads[*videoIndex]) != "video" {
			return 0, 0, "videoIndex 不是视频资源"
		}
		return *imageIndex, *videoIndex, ""
	}

	// 若只提供其中一个：尽量推断另一个（best-effort）
	imgIdx = -1
	vidIdx = -1

	if imageIndex != nil {
		imgIdx = *imageIndex
		for i := imgIdx + 1; i < len(downloads); i++ {
			if guessDouyinMediaTypeFromURL(downloads[i]) == "video" {
				vidIdx = i
				break
			}
		}
		if vidIdx < 0 {
			for i := 0; i < len(downloads); i++ {
				if guessDouyinMediaTypeFromURL(downloads[i]) == "video" {
					vidIdx = i
					break
				}
			}
		}
	} else if videoIndex != nil {
		vidIdx = *videoIndex
		for i := 0; i < vidIdx; i++ {
			if guessDouyinMediaTypeFromURL(downloads[i]) == "image" {
				imgIdx = i
				break
			}
		}
		if imgIdx < 0 {
			for i := 0; i < len(downloads); i++ {
				if guessDouyinMediaTypeFromURL(downloads[i]) == "image" {
					imgIdx = i
					break
				}
			}
		}
	} else {
		for i := 0; i < len(downloads); i++ {
			if guessDouyinMediaTypeFromURL(downloads[i]) == "image" {
				imgIdx = i
				break
			}
		}
		for i := 0; i < len(downloads); i++ {
			if guessDouyinMediaTypeFromURL(downloads[i]) == "video" {
				vidIdx = i
				break
			}
		}
	}

	if imgIdx < 0 || vidIdx < 0 {
		return 0, 0, "未找到图片/视频资源，无法生成实况"
	}
	return imgIdx, vidIdx, ""
}

func downloadDouyinResourceToFile(ctx context.Context, client *http.Client, remoteURL, dstPath string) (contentType string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", fmt.Errorf("remoteURL 为空")
	}

	u, err := url.Parse(remoteURL)
	if err != nil || u == nil || strings.TrimSpace(u.Scheme) == "" {
		return "", fmt.Errorf("remoteURL 非法")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return "", fmt.Errorf("remoteURL 非法")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.douyin.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("%s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))

	f, err := os.Create(dstPath)
	if err != nil {
		return contentType, err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return contentType, err
	}
	return contentType, nil
}

func normalizeLivePhotoStillImage(ctx context.Context, inputPath, contentType, outputJPGPath string) error {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "image/jpeg") || strings.HasSuffix(strings.ToLower(inputPath), ".jpg") || strings.HasSuffix(strings.ToLower(inputPath), ".jpeg") {
		return copyFile(inputPath, outputJPGPath)
	}
	// 统一转为 JPG：iOS Live Photo 静态图建议使用 JPG/HEIC，这里固定输出 JPG。
	return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-frames:v", "1", outputJPGPath})
}

func normalizeLivePhotoMotionVideo(ctx context.Context, inputPath, contentType, outputMOVPath string) error {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "video/quicktime") || strings.HasSuffix(strings.ToLower(inputPath), ".mov") {
		// 仍走一次 remux，保证 metadata atom 结构一致（best-effort）
		return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-f", "mov", outputMOVPath})
	}
	return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-f", "mov", outputMOVPath})
}

func tagLivePhotoAsset(ctx context.Context, stillPath, motionPath, assetID string) error {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return fmt.Errorf("assetID 为空")
	}

	// 图片：写入 Apple ContentIdentifier（MakerApple），并补齐 Make/Model 增加 iOS 识别概率。
	if err := runCommand(ctx, "exiftool", []string{
		"-overwrite_original",
		"-Make=Apple",
		"-Model=iPhone",
		"-ContentIdentifier=" + assetID,
		stillPath,
	}); err != nil {
		return err
	}

	// 视频：写入 QuickTime ContentIdentifier（用于与图片配对）。
	if err := runCommand(ctx, "exiftool", []string{
		"-overwrite_original",
		"-ContentIdentifier=" + assetID,
		motionPath,
	}); err != nil {
		return err
	}
	return nil
}

func runCommand(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(out.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func zipFile(zw *zip.Writer, srcPath, zipName string) error {
	srcPath = strings.TrimSpace(srcPath)
	zipName = strings.TrimSpace(zipName)
	if srcPath == "" || zipName == "" {
		return fmt.Errorf("zip 参数为空")
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = zipName
	hdr.Method = zip.Deflate

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}
