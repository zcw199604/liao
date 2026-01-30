package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

func normalizeRotationDegrees(raw int) int {
	deg := raw % 360
	if deg < 0 {
		deg += 360
	}
	switch deg {
	case 0, 90, 180, 270:
		return deg
	default:
		// Best-effort: snap to nearest 90 degrees.
		return ((deg + 45) / 90 * 90) % 360
	}
}

func probeVideoRotationDegrees(ctx context.Context, ffprobePath, inputAbsPath string) int {
	ffprobePath = strings.TrimSpace(ffprobePath)
	inputAbsPath = strings.TrimSpace(inputAbsPath)
	if ffprobePath == "" || inputAbsPath == "" {
		return 0
	}

	// Fast path: try common rotate tag first (e.g. iPhone videos).
	{
		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		out, err := execCommandContext(ctx2,
			ffprobePath,
			"-v", "error",
			"-select_streams", "v:0",
			"-show_entries", "stream_tags=rotate",
			"-of", "default=nw=1:nk=1",
			inputAbsPath,
		).Output()
		if err == nil {
			raw := strings.TrimSpace(string(out))
			if raw != "" {
				if v, err := strconv.Atoi(raw); err == nil {
					return normalizeRotationDegrees(v)
				}
			}
		}
	}

	// Fallback: inspect side_data_list.rotation from JSON output.
	{
		ctx2, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()

		out, err := execCommandContext(ctx2,
			ffprobePath,
			"-v", "error",
			"-select_streams", "v:0",
			"-show_streams",
			"-of", "json",
			inputAbsPath,
		).Output()
		if err != nil {
			return 0
		}

		var parsed struct {
			Streams []struct {
				Tags         map[string]string `json:"tags"`
				SideDataList []struct {
					Rotation *float64 `json:"rotation"`
				} `json:"side_data_list"`
			} `json:"streams"`
		}
		if err := json.Unmarshal(out, &parsed); err != nil {
			return 0
		}
		if len(parsed.Streams) == 0 {
			return 0
		}

		if v := strings.TrimSpace(parsed.Streams[0].Tags["rotate"]); v != "" {
			if deg, err := strconv.Atoi(v); err == nil {
				return normalizeRotationDegrees(deg)
			}
		}

		for _, sd := range parsed.Streams[0].SideDataList {
			if sd.Rotation != nil {
				return normalizeRotationDegrees(int(math.Round(*sd.Rotation)))
			}
		}
		return 0
	}
}

// EnsureVideoPoster generates a JPEG poster image for a locally stored video.
// The poster generation is best-effort: callers may ignore the error and keep
// the upload workflow unblocked.
func (s *FileStorageService) EnsureVideoPoster(ctx context.Context, ffmpegPath string, ffprobePath string, videoLocalPath string, force bool) (string, error) {
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
	if !force {
		if fi, err := os.Stat(outputAbs); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return posterLocalPath, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputAbs), 0o755); err != nil {
		return "", fmt.Errorf("无法创建封面目录: %w", err)
	}

	rotation := probeVideoRotationDegrees(ctx, ffprobePath, inputAbs)
	rotateFilter := ""
	switch rotation {
	case 90:
		rotateFilter = "transpose=1"
	case 180:
		rotateFilter = "transpose=2,transpose=2"
	case 270:
		rotateFilter = "transpose=2"
	}

	// Keep poster deterministic across ffmpeg versions:
	// - if rotation metadata is detected, disable ffmpeg auto-rotate and apply
	//   transpose based on metadata ourselves
	// - if rotation metadata is unknown, keep ffmpeg default behavior (auto-rotate)
	vf := "scale='min(480,iw)':-2,setsar=1"
	if rotateFilter != "" {
		vf = rotateFilter + "," + vf
	}

	// Try to avoid black frames: seek to ~1s first, and fall back to 0s if needed.
	tryArgs := [][]string{
		{"-y", "-ss", "00:00:01.000", "-i", inputAbs, "-frames:v", "1", "-vf", vf, "-q:v", "4", outputAbs},
		{"-y", "-ss", "00:00:00.000", "-i", inputAbs, "-frames:v", "1", "-vf", vf, "-q:v", "4", outputAbs},
	}
	if rotateFilter != "" {
		for i := range tryArgs {
			// Ensure ffmpeg won't auto-rotate, since we already apply transpose.
			tryArgs[i] = append([]string{"-y", "-noautorotate"}, tryArgs[i][1:]...)
		}
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

func (s *FileStorageService) EnsureVideoPosterLogged(ctx context.Context, ffmpegPath, ffprobePath, videoLocalPath string, force bool) (posterLocalPath string, posterURL string) {
	posterLocalPath, err := s.EnsureVideoPoster(ctx, ffmpegPath, ffprobePath, videoLocalPath, force)
	if err != nil {
		slog.Warn("生成视频封面失败(将跳过)", "error", err, "localPath", videoLocalPath)
		return "", ""
	}
	if posterLocalPath == "" {
		return "", ""
	}
	return posterLocalPath, s.posterURLFromLocalPath(posterLocalPath)
}
