package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type RepairMediaDimensionsRequest struct {
	Commit       bool   `json:"commit"`
	Force        bool   `json:"force,omitempty"`
	Source       string `json:"source,omitempty"`
	StartAfterID int64  `json:"startAfterId,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type RepairMediaDimensionsResult struct {
	Commit bool   `json:"commit"`
	Source string `json:"source"`
	Force  bool   `json:"force"`

	StartAfterID int64 `json:"startAfterId"`
	NextAfterID  int64 `json:"nextAfterId"`
	HasMore      bool  `json:"hasMore"`
	Limit        int   `json:"limit"`

	Scanned      int `json:"scanned"`
	NeedUpdate   int `json:"needUpdate"`
	Updated      int `json:"updated"`
	FileMissing  int `json:"fileMissing"`
	InvalidPath  int `json:"invalidPath"`
	DecodeFailed int `json:"decodeFailed"`
	Unsupported  int `json:"unsupported"`
	Skipped      int `json:"skipped"`

	Warnings []string `json:"warnings,omitempty"`
}

func (s *MediaUploadService) RepairMediaDimensions(ctx context.Context, req RepairMediaDimensionsRequest) (RepairMediaDimensionsResult, error) {
	var res RepairMediaDimensionsResult
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

	res.Commit = req.Commit
	res.Source = source
	res.Force = req.Force
	res.StartAfterID = req.StartAfterID
	res.NextAfterID = req.StartAfterID
	res.Limit = req.Limit

	table := "media_file"
	if source == "douyin" {
		table = "douyin_media_file"
	}

	where := `id > ?`
	if !req.Force {
		where += ` AND (media_width IS NULL OR media_width <= 0 OR media_height IS NULL OR media_height <= 0)`
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, local_path, file_type, file_extension
FROM %s
WHERE %s
ORDER BY id ASC
LIMIT ?`, table, where), req.StartAfterID, req.Limit+1)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var localPathRaw, fileType, fileExtension string
		if err := rows.Scan(&id, &localPathRaw, &fileType, &fileExtension); err != nil {
			return res, err
		}

		if res.Scanned >= req.Limit {
			res.HasMore = true
			break
		}

		res.Scanned++
		res.NextAfterID = id

		localPath := normalizeUploadLocalPathInput(localPathRaw)
		if localPath == "" {
			res.InvalidPath++
			appendRepairDimensionWarning(&res, "id=%d invalid local_path: %q", id, localPathRaw)
			continue
		}

		if inferTypeFromMediaMeta(fileType, fileExtension, localPath) != "image" {
			res.Unsupported++
			continue
		}

		width, height, err := s.fileStore.ReadImageDimensions(localPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				res.FileMissing++
				appendRepairDimensionWarning(&res, "id=%d file missing: %s", id, localPath)
				continue
			}
			if strings.Contains(err.Error(), "路径越界") || strings.Contains(err.Error(), "localPath") || strings.Contains(err.Error(), "路径解析") {
				res.InvalidPath++
				appendRepairDimensionWarning(&res, "id=%d invalid path: %v (local_path=%q)", id, err, localPath)
				continue
			}
			res.DecodeFailed++
			appendRepairDimensionWarning(&res, "id=%d decode dimensions failed: %v (local_path=%q)", id, err, localPath)
			continue
		}

		res.NeedUpdate++
		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET media_width = ?, media_height = ? WHERE id = ?", table), width, height, id)
		if err != nil {
			res.Skipped++
			appendRepairDimensionWarning(&res, "id=%d update dimensions failed: %v", id, err)
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.Updated++
		}
	}
	if err := rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func appendRepairDimensionWarning(res *RepairMediaDimensionsResult, format string, args ...any) {
	if res == nil || len(res.Warnings) >= 200 {
		return
	}
	res.Warnings = append(res.Warnings, fmt.Sprintf(format, args...))
}
