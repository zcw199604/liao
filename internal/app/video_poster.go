package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// buildVideoPosterLocalPath converts a video localPath (e.g. /videos/.../a.mp4)
// into a deterministic poster localPath (e.g. /videos/.../a.poster.jpg).
//
// The poster file is served by the existing /upload static handler, so callers
// typically build a URL via "/upload"+posterLocalPath or convertToLocalURL().
func buildVideoPosterLocalPath(videoLocalPath string) string {
	lp := normalizeUploadLocalPathInput(videoLocalPath)
	if lp == "" {
		return ""
	}

	// Always treat as URL-path to avoid OS-specific separators.
	dir := path.Dir(lp)
	base := path.Base(lp)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if strings.TrimSpace(name) == "" {
		name = base
	}
	return path.Join(dir, name+".poster.jpg")
}

func (s *FileStorageService) resolveUploadAbsPath(localPath string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("文件服务未初始化")
	}

	localPath = normalizeUploadLocalPathInput(localPath)
	if localPath == "" {
		return "", fmt.Errorf("localPath 为空")
	}

	clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if clean == "." || clean == string(filepath.Separator) {
		return "", fmt.Errorf("localPath 非法")
	}

	full := filepath.Join(s.baseUploadAbs, clean)
	rel, err := filepathRelFn(s.baseUploadAbs, full)
	if err != nil {
		return "", fmt.Errorf("路径解析失败: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("检测到路径越界")
	}
	return full, nil
}

func (s *FileStorageService) posterURLFromLocalPath(posterLocalPath string) string {
	posterLocalPath = normalizeUploadLocalPathInput(posterLocalPath)
	if posterLocalPath == "" {
		return ""
	}
	return "/upload" + posterLocalPath
}

// EnsureVideoPoster generates a JPEG poster image for a locally stored video.
// The poster generation is best-effort: callers may ignore the error and keep
// the upload workflow unblocked.
func (s *FileStorageService) EnsureVideoPoster(ctx context.Context, ffmpegPath string, videoLocalPath string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("文件服务未初始化")
	}

	ffmpegPath = strings.TrimSpace(ffmpegPath)
	if ffmpegPath == "" {
		// Not configured: allow callers to keep working without a poster.
		return "", nil
	}

	videoLocalPath = normalizeUploadLocalPathInput(videoLocalPath)
	if videoLocalPath == "" {
		return "", fmt.Errorf("videoLocalPath 为空")
	}

	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	if posterLocalPath == "" {
		return "", fmt.Errorf("posterLocalPath 为空")
	}

	inputAbs, err := s.resolveUploadAbsPath(videoLocalPath)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(inputAbs); err != nil || fi.IsDir() {
		return "", fmt.Errorf("视频文件不存在")
	}

	outputAbs, err := s.resolveUploadAbsPath(posterLocalPath)
	if err != nil {
		return "", err
	}

	// Fast path: poster already exists.
	if fi, err := os.Stat(outputAbs); err == nil && !fi.IsDir() && fi.Size() > 0 {
		return posterLocalPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(outputAbs), 0o755); err != nil {
		return "", fmt.Errorf("无法创建封面目录: %w", err)
	}

	// Try to avoid black frames: seek to ~1s first, and fall back to 0s if needed.
	tryArgs := [][]string{
		{"-y", "-ss", "00:00:01.000", "-i", inputAbs, "-frames:v", "1", "-vf", "scale='min(480,iw)':-2", "-q:v", "4", outputAbs},
		{"-y", "-ss", "00:00:00.000", "-i", inputAbs, "-frames:v", "1", "-vf", "scale='min(480,iw)':-2", "-q:v", "4", outputAbs},
	}

	var lastErr error
	for _, args := range tryArgs {
		if err := runCommand(ctx, ffmpegPath, args); err != nil {
			lastErr = err
			continue
		}
		// Best-effort verification.
		if fi, err := os.Stat(outputAbs); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return posterLocalPath, nil
		}
		lastErr = fmt.Errorf("封面文件未生成")
	}

	return "", lastErr
}

func (s *FileStorageService) DeleteVideoPoster(videoLocalPath string) bool {
	if s == nil {
		return false
	}
	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	if posterLocalPath == "" {
		return false
	}
	return s.DeleteFile(posterLocalPath)
}

func (s *FileStorageService) EnsureVideoPosterLogged(ctx context.Context, ffmpegPath, videoLocalPath string) (posterLocalPath string, posterURL string) {
	posterLocalPath, err := s.EnsureVideoPoster(ctx, ffmpegPath, videoLocalPath)
	if err != nil {
		slog.Warn("生成视频封面失败(将跳过)", "error", err, "localPath", videoLocalPath)
		return "", ""
	}
	if posterLocalPath == "" {
		return "", ""
	}
	return posterLocalPath, s.posterURLFromLocalPath(posterLocalPath)
}
