package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RepairVideoPostersRequest struct {
	// Commit=true 才会真实生成 poster；默认 false 为 dry-run。
	Commit bool `json:"commit"`

	// Source 表示处理范围：
	// - local: media_file
	// - douyin: douyin_media_file
	Source string `json:"source,omitempty"`

	// StartAfterID 用于分页游标：只处理 id > StartAfterID 的记录。
	StartAfterID int64 `json:"startAfterId,omitempty"`

	// Limit 表示本次调用最多扫描多少条视频记录；默认 200，最大 2000。
	Limit int `json:"limit,omitempty"`
}

type RepairVideoPostersResult struct {
	Commit bool   `json:"commit"`
	Source string `json:"source"`

	StartAfterID int64 `json:"startAfterId"`
	NextAfterID  int64 `json:"nextAfterId"`
	HasMore      bool  `json:"hasMore"`
	Limit        int   `json:"limit"`

	Scanned int `json:"scanned"`

	VideoMissing int `json:"videoMissing"`

	PosterExisting  int `json:"posterExisting"`
	PosterMissing   int `json:"posterMissing"`
	PosterGenerated int `json:"posterGenerated"`
	PosterFailed    int `json:"posterFailed"`
	Skipped         int `json:"skipped"`

	Warnings []string `json:"warnings,omitempty"`
}

func (s *MediaUploadService) RepairVideoPosters(ctx context.Context, ffmpegPath string, req RepairVideoPostersRequest) (RepairVideoPostersResult, error) {
	var res RepairVideoPostersResult
	if s == nil || s.db == nil || s.fileStore == nil {
		return res, errors.New("服务未初始化")
	}

	source := strings.ToLower(strings.TrimSpace(req.Source))
	if source == "" {
		source = "local"
	}
	if source != "local" && source != "douyin" {
		return res, fmt.Errorf("source 非法: %s（仅支持 local/douyin）", source)
	}

	if req.Limit < 0 {
		return res, errors.New("invalid limits: negative value")
	}
	if req.Limit == 0 {
		req.Limit = 200
	}
	if req.Limit > 2000 {
		req.Limit = 2000
	}

	ffmpegPath = strings.TrimSpace(ffmpegPath)
	if req.Commit {
		if ffmpegPath == "" {
			return res, errors.New("ffmpeg 未配置：请设置环境变量 FFMPEG_PATH 或安装 ffmpeg")
		}
		if _, err := exec.LookPath(ffmpegPath); err != nil {
			return res, fmt.Errorf("ffmpeg 不可用: %v", err)
		}
	}

	res.Commit = req.Commit
	res.Source = source
	res.StartAfterID = req.StartAfterID
	res.NextAfterID = req.StartAfterID
	res.Limit = req.Limit

	table := "media_file"
	if source == "douyin" {
		table = "douyin_media_file"
	}

	// 使用 LIMIT+1 探测 hasMore，避免额外 COUNT 查询。
	queryLimit := req.Limit + 1
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, local_path, file_type, file_extension
FROM %s
WHERE (LOWER(file_type) LIKE 'video/%%' OR LOWER(file_extension) = 'mp4')
  AND local_path IS NOT NULL AND local_path <> ''
  AND id > ?
ORDER BY id ASC
LIMIT ?`, table), req.StartAfterID, queryLimit)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id            int64
			localPathRaw  string
			fileType      string
			fileExtension string
		)
		if err := rows.Scan(&id, &localPathRaw, &fileType, &fileExtension); err != nil {
			return res, err
		}

		// extra row => there are more results, but do not process it in this call.
		if res.Scanned >= req.Limit {
			res.HasMore = true
			break
		}

		res.Scanned++
		res.NextAfterID = id

		localPath := normalizeUploadLocalPathInput(localPathRaw)
		if localPath == "" {
			res.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d local_path invalid: %q", id, localPathRaw))
			}
			continue
		}

		videoAbs, err := s.fileStore.resolveUploadAbsPath(localPath)
		if err != nil {
			res.VideoMissing++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d resolve video abs failed: %v (local_path=%q)", id, err, localPath))
			}
			continue
		}
		if fi, err := os.Stat(videoAbs); err != nil || fi.IsDir() {
			res.VideoMissing++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d video missing: %s (file_type=%q ext=%q)", id, localPath, fileType, fileExtension))
			}
			continue
		}

		posterLocalPath := buildVideoPosterLocalPath(localPath)
		if posterLocalPath == "" {
			res.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d build poster path failed: %q", id, localPath))
			}
			continue
		}

		posterAbs, err := s.fileStore.resolveUploadAbsPath(posterLocalPath)
		if err != nil {
			res.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d resolve poster abs failed: %v (poster=%q)", id, err, posterLocalPath))
			}
			continue
		}

		// Poster already exists.
		if fi, err := os.Stat(posterAbs); err == nil && !fi.IsDir() && fi.Size() > 0 {
			res.PosterExisting++
			continue
		}

		res.PosterMissing++
		if !req.Commit {
			continue
		}

		created, err := s.fileStore.EnsureVideoPoster(ctx, ffmpegPath, localPath)
		if err != nil {
			res.PosterFailed++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d generate poster failed: %v (local_path=%q)", id, err, localPath))
			}
			continue
		}
		if strings.TrimSpace(created) == "" {
			res.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d poster skipped: returned empty poster path (local_path=%q)", id, localPath))
			}
			continue
		}
		res.PosterGenerated++
	}
	if err := rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}
