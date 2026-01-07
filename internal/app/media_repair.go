package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type RepairMediaHistoryRequest struct {
	// Commit=true 才会真实写入/删除；默认 false 为 dry-run。
	Commit bool `json:"commit"`

	// UserID 非空时仅处理该用户的数据。
	UserID string `json:"userId,omitempty"`

	// 三个开关若全部为 false，则默认全部启用（仍受 Commit 控制）。
	FixMissingMD5         bool `json:"fixMissingMd5"`
	DeduplicateByMD5      bool `json:"deduplicateByMd5"`
	DeduplicateByLocalPath bool `json:"deduplicateByLocalPath"`

	// 每次调用的处理上限，避免一次性跑太久。
	LimitMissingMD5    int `json:"limitMissingMd5,omitempty"`
	MaxDuplicateGroups int `json:"maxDuplicateGroups,omitempty"`
	SampleLimit        int `json:"sampleLimit,omitempty"`
}

type RepairMediaHistoryResult struct {
	Commit bool `json:"commit"`
	UserID string `json:"userId,omitempty"`

	FixMissingMD5         bool `json:"fixMissingMd5"`
	DeduplicateByMD5      bool `json:"deduplicateByMd5"`
	DeduplicateByLocalPath bool `json:"deduplicateByLocalPath"`

	MissingMD5            RepairMissingMD5Result `json:"missingMd5,omitempty"`
	DuplicatesByMD5       RepairDedupResult      `json:"duplicatesByMd5,omitempty"`
	DuplicatesByLocalPath RepairDedupResult      `json:"duplicatesByLocalPath,omitempty"`

	Samples  []RepairMediaHistorySample `json:"samples,omitempty"`
	Warnings []string                   `json:"warnings,omitempty"`
}

type RepairMissingMD5Result struct {
	Scanned    int `json:"scanned"`
	NeedUpdate int `json:"needUpdate"`
	Updated    int `json:"updated"`
	Skipped    int `json:"skipped"`
}

type RepairDedupResult struct {
	Groups   int `json:"groups"`
	Rows     int `json:"rows"`
	ToDelete int `json:"toDelete"`
	Deleted  int `json:"deleted"`
}

type RepairMediaHistorySample struct {
	Kind        string `json:"kind"` // md5 | localPath
	UserID      string `json:"userId"`
	MD5         string `json:"md5,omitempty"`
	KeepID      int64  `json:"keepId"`
	DeleteCount int    `json:"deleteCount"`
}

func (s *MediaUploadService) RepairMediaHistory(ctx context.Context, req RepairMediaHistoryRequest) (RepairMediaHistoryResult, error) {
	var res RepairMediaHistoryResult

	req.UserID = strings.TrimSpace(req.UserID)
	if req.LimitMissingMD5 < 0 || req.MaxDuplicateGroups < 0 || req.SampleLimit < 0 {
		return res, errors.New("invalid limits: negative value")
	}
	if req.LimitMissingMD5 == 0 {
		req.LimitMissingMD5 = 500
	}
	if req.MaxDuplicateGroups == 0 {
		req.MaxDuplicateGroups = 500
	}
	if req.SampleLimit == 0 {
		req.SampleLimit = 20
	}
	if req.SampleLimit > 200 {
		req.SampleLimit = 200
	}

	if !req.FixMissingMD5 && !req.DeduplicateByMD5 && !req.DeduplicateByLocalPath {
		req.FixMissingMD5 = true
		req.DeduplicateByMD5 = true
		req.DeduplicateByLocalPath = true
	}

	res.Commit = req.Commit
	res.UserID = req.UserID
	res.FixMissingMD5 = req.FixMissingMD5
	res.DeduplicateByMD5 = req.DeduplicateByMD5
	res.DeduplicateByLocalPath = req.DeduplicateByLocalPath

	if req.FixMissingMD5 {
		if err := s.repairMissingMD5(ctx, &req, &res); err != nil {
			return res, err
		}
	}
	if req.DeduplicateByMD5 {
		if err := s.dedupByMD5(ctx, &req, &res); err != nil {
			return res, err
		}
	}
	if req.DeduplicateByLocalPath {
		if err := s.dedupByLocalPath(ctx, &req, &res); err != nil {
			return res, err
		}
	}

	return res, nil
}

func (s *MediaUploadService) repairMissingMD5(ctx context.Context, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) error {
	var b strings.Builder
	b.WriteString(`
SELECT id, user_id, local_path
FROM media_file
WHERE (file_md5 IS NULL OR file_md5 = '')
  AND local_path IS NOT NULL AND local_path <> ''`)
	args := make([]any, 0, 2)
	if req.UserID != "" {
		b.WriteString("\n  AND user_id = ?")
		args = append(args, req.UserID)
	}
	b.WriteString("\nORDER BY id ASC\nLIMIT ?")
	args = append(args, req.LimitMissingMD5)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("query missing md5 rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var userID, localPath string
		if err := rows.Scan(&id, &userID, &localPath); err != nil {
			return fmt.Errorf("scan missing md5 row: %w", err)
		}
		res.MissingMD5.Scanned++

		md5Value, err := s.fileStore.CalculateMD5FromLocalPath(localPath)
		if err != nil {
			res.MissingMD5.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d user_id=%s calculate md5 failed: %v", id, userID, err))
			}
			continue
		}

		if md5Value == "" {
			res.MissingMD5.Skipped++
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d user_id=%s calculated empty md5", id, userID))
			}
			continue
		}

		res.MissingMD5.NeedUpdate++
		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, `
UPDATE media_file
SET file_md5 = ?, update_time = NOW()
WHERE id = ? AND (file_md5 IS NULL OR file_md5 = '')`, md5Value, id)
		if err != nil {
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("id=%d user_id=%s update md5 failed: %v", id, userID, err))
			}
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.MissingMD5.Updated++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate missing md5 rows: %w", err)
	}
	return nil
}

func (s *MediaUploadService) dedupByMD5(ctx context.Context, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) error {
	var b strings.Builder
	b.WriteString(`
SELECT user_id, file_md5, COUNT(*) AS cnt
FROM media_file
WHERE file_md5 IS NOT NULL AND file_md5 <> ''`)
	args := make([]any, 0, 2)
	if req.UserID != "" {
		b.WriteString("\n  AND user_id = ?")
		args = append(args, req.UserID)
	}
	b.WriteString(`
GROUP BY user_id, file_md5
HAVING COUNT(*) > 1
ORDER BY cnt DESC
LIMIT ?`)
	args = append(args, req.MaxDuplicateGroups)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("query duplicate md5 groups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID, md5Value string
		var cnt int64
		if err := rows.Scan(&userID, &md5Value, &cnt); err != nil {
			return fmt.Errorf("scan duplicate md5 group: %w", err)
		}

		res.DuplicatesByMD5.Groups++
		res.DuplicatesByMD5.Rows += int(cnt)
		if cnt > 0 {
			res.DuplicatesByMD5.ToDelete += int(cnt - 1)
		}

		var keepID int64
		if err := s.db.QueryRowContext(ctx, `
SELECT id
FROM media_file
WHERE user_id = ? AND file_md5 = ?
ORDER BY
  (CASE WHEN remote_url IS NULL OR remote_url = '' THEN 0 ELSE 1 END) DESC,
  upload_time DESC,
  id DESC
LIMIT 1`, userID, md5Value).Scan(&keepID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return fmt.Errorf("select keep row for md5 dedup: %w", err)
		}

		if len(res.Samples) < req.SampleLimit {
			res.Samples = append(res.Samples, RepairMediaHistorySample{
				Kind:        "md5",
				UserID:      userID,
				MD5:         md5Value,
				KeepID:      keepID,
				DeleteCount: int(max64(cnt-1, 0)),
			})
		}

		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, `
DELETE FROM media_file
WHERE user_id = ? AND file_md5 = ? AND id <> ?`, userID, md5Value, keepID)
		if err != nil {
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("md5 dedup delete failed user_id=%s md5=%s: %v", userID, md5Value, err))
			}
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.DuplicatesByMD5.Deleted += int(n)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate duplicate md5 groups: %w", err)
	}
	return nil
}

func (s *MediaUploadService) dedupByLocalPath(ctx context.Context, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) error {
	var b strings.Builder
	b.WriteString(`
SELECT user_id, local_path, COUNT(*) AS cnt
FROM media_file
WHERE (file_md5 IS NULL OR file_md5 = '')
  AND local_path IS NOT NULL AND local_path <> ''`)
	args := make([]any, 0, 2)
	if req.UserID != "" {
		b.WriteString("\n  AND user_id = ?")
		args = append(args, req.UserID)
	}
	b.WriteString(`
GROUP BY user_id, local_path
HAVING COUNT(*) > 1
ORDER BY cnt DESC
LIMIT ?`)
	args = append(args, req.MaxDuplicateGroups)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("query duplicate local_path groups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID, localPath string
		var cnt int64
		if err := rows.Scan(&userID, &localPath, &cnt); err != nil {
			return fmt.Errorf("scan duplicate local_path group: %w", err)
		}

		res.DuplicatesByLocalPath.Groups++
		res.DuplicatesByLocalPath.Rows += int(cnt)
		if cnt > 0 {
			res.DuplicatesByLocalPath.ToDelete += int(cnt - 1)
		}

		var keepID int64
		if err := s.db.QueryRowContext(ctx, `
SELECT id
FROM media_file
WHERE user_id = ? AND local_path = ? AND (file_md5 IS NULL OR file_md5 = '')
ORDER BY
  (CASE WHEN remote_url IS NULL OR remote_url = '' THEN 0 ELSE 1 END) DESC,
  upload_time DESC,
  id DESC
LIMIT 1`, userID, localPath).Scan(&keepID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return fmt.Errorf("select keep row for local_path dedup: %w", err)
		}

		if len(res.Samples) < req.SampleLimit {
			res.Samples = append(res.Samples, RepairMediaHistorySample{
				Kind:        "localPath",
				UserID:      userID,
				KeepID:      keepID,
				DeleteCount: int(max64(cnt-1, 0)),
			})
		}

		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, `
DELETE FROM media_file
WHERE user_id = ? AND local_path = ? AND (file_md5 IS NULL OR file_md5 = '') AND id <> ?`, userID, localPath, keepID)
		if err != nil {
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("local_path dedup delete failed user_id=%s: %v", userID, err))
			}
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.DuplicatesByLocalPath.Deleted += int(n)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate duplicate local_path groups: %w", err)
	}
	return nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

