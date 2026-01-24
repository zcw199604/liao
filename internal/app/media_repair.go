package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var calculateMD5FromLocalPathFn = func(fileStore *FileStorageService, localPath string) (string, error) {
	return fileStore.CalculateMD5FromLocalPath(localPath)
}

type RepairMediaHistoryRequest struct {
	// Commit=true 才会真实写入/删除；默认 false 为 dry-run。
	Commit bool `json:"commit"`

	// UserID 已废弃：不再按用户过滤（全局修复），仅为兼容旧调用保留。
	UserID string `json:"userId,omitempty"`

	// 兼容开关：默认会执行“补齐缺失 MD5 + 按 MD5 全局去重”；local_path 去重可按需开启。
	FixMissingMD5          bool `json:"fixMissingMd5"`
	DeduplicateByMD5       bool `json:"deduplicateByMd5"`
	DeduplicateByLocalPath bool `json:"deduplicateByLocalPath"`

	// 每次调用的处理上限，避免一次性跑太久。
	LimitMissingMD5    int `json:"limitMissingMd5,omitempty"`
	MaxDuplicateGroups int `json:"maxDuplicateGroups,omitempty"`
	SampleLimit        int `json:"sampleLimit,omitempty"`
}

type RepairMediaHistoryResult struct {
	Commit bool   `json:"commit"`
	UserID string `json:"userId,omitempty"`

	FixMissingMD5          bool `json:"fixMissingMd5"`
	DeduplicateByMD5       bool `json:"deduplicateByMd5"`
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

	if req.UserID != "" {
		res.Warnings = append(res.Warnings, "userId 参数已忽略：当前为全局修复，不按用户过滤")
		req.UserID = ""
	}

	// 该接口的核心用途是：补齐缺失 MD5 → 按 MD5 去重（全局仅保留 1 条）。
	// 仍保留开关以兼容旧调用，但默认强制开启上述两步。
	if !req.FixMissingMD5 {
		req.FixMissingMD5 = true
	}
	if !req.DeduplicateByMD5 {
		req.DeduplicateByMD5 = true
	}

	res.Commit = req.Commit
	res.UserID = req.UserID
	res.FixMissingMD5 = req.FixMissingMD5
	res.DeduplicateByMD5 = req.DeduplicateByMD5
	res.DeduplicateByLocalPath = req.DeduplicateByLocalPath

	if req.FixMissingMD5 {
		startAfterID := int64(0)
		for {
			scanned, nextAfterID, err := s.repairMissingMD5Batch(ctx, startAfterID, &req, &res)
			if err != nil {
				return res, err
			}
			if scanned == 0 {
				break
			}
			startAfterID = nextAfterID
			if !req.Commit {
				res.Warnings = append(res.Warnings, "dry-run 模式仅扫描一批缺失 MD5 记录；如需全量补齐请使用 commit=true")
				break
			}
		}
	}
	if req.DeduplicateByMD5 {
		for {
			prevDeleted := res.DuplicatesByMD5.Deleted
			groups, err := s.dedupByMD5Batch(ctx, &req, &res)
			if err != nil {
				return res, err
			}
			if groups == 0 {
				break
			}
			if !req.Commit {
				res.Warnings = append(res.Warnings, "dry-run 模式仅统计一批重复 MD5 分组；如需全量去重请使用 commit=true")
				break
			}
			if res.DuplicatesByMD5.Deleted == prevDeleted {
				res.Warnings = append(res.Warnings, "MD5 去重未产生删除进展，可能存在删除失败或约束问题，已停止循环以避免无限重试")
				break
			}
		}
	}
	if req.DeduplicateByLocalPath {
		for {
			prevDeleted := res.DuplicatesByLocalPath.Deleted
			groups, err := s.dedupByLocalPathBatch(ctx, &req, &res)
			if err != nil {
				return res, err
			}
			if groups == 0 {
				break
			}
			if !req.Commit {
				res.Warnings = append(res.Warnings, "dry-run 模式仅统计一批重复 local_path 分组；如需全量去重请使用 commit=true")
				break
			}
			if res.DuplicatesByLocalPath.Deleted == prevDeleted {
				res.Warnings = append(res.Warnings, "local_path 去重未产生删除进展，可能存在删除失败或约束问题，已停止循环以避免无限重试")
				break
			}
		}
	}

	return res, nil
}

func (s *MediaUploadService) repairMissingMD5Batch(ctx context.Context, startAfterID int64, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) (int, int64, error) {
	var b strings.Builder
	b.WriteString(`
SELECT id, user_id, local_path
FROM media_upload_history
WHERE (file_md5 IS NULL OR file_md5 = '')
  AND local_path IS NOT NULL AND local_path <> ''`)
	args := make([]any, 0, 3)
	b.WriteString("\n  AND id > ?")
	args = append(args, startAfterID)
	b.WriteString("\nORDER BY id ASC\nLIMIT ?")
	args = append(args, req.LimitMissingMD5)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return 0, startAfterID, fmt.Errorf("query missing md5 rows: %w", err)
	}
	defer rows.Close()

	var scanned int
	nextAfterID := startAfterID
	for rows.Next() {
		var id int64
		var userID, localPath string
		if err := rows.Scan(&id, &userID, &localPath); err != nil {
			return scanned, nextAfterID, fmt.Errorf("scan missing md5 row: %w", err)
		}
		nextAfterID = id
		scanned++
		res.MissingMD5.Scanned++

		md5Value, err := calculateMD5FromLocalPathFn(s.fileStore, localPath)
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
UPDATE media_upload_history
SET file_md5 = ?
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
		return scanned, nextAfterID, fmt.Errorf("iterate missing md5 rows: %w", err)
	}
	return scanned, nextAfterID, nil
}

func (s *MediaUploadService) dedupByMD5Batch(ctx context.Context, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) (int, error) {
	var b strings.Builder
	b.WriteString(`
SELECT file_md5, COUNT(*) AS cnt
FROM media_upload_history
WHERE file_md5 IS NOT NULL AND file_md5 <> ''`)
	args := make([]any, 0, 1)
	b.WriteString(`
GROUP BY file_md5
HAVING COUNT(*) > 1
ORDER BY cnt DESC
LIMIT ?`)
	args = append(args, req.MaxDuplicateGroups)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("query duplicate md5 groups: %w", err)
	}
	defer rows.Close()

	groups := 0
	for rows.Next() {
		var md5Value string
		var cnt int64
		if err := rows.Scan(&md5Value, &cnt); err != nil {
			return groups, fmt.Errorf("scan duplicate md5 group: %w", err)
		}

		res.DuplicatesByMD5.Groups++
		groups++
		res.DuplicatesByMD5.Rows += int(cnt)
		if cnt > 0 {
			res.DuplicatesByMD5.ToDelete += int(cnt - 1)
		}

		var keepID int64
		var keepUserID string
		if err := s.db.QueryRowContext(ctx, `
SELECT id, user_id
FROM media_upload_history
WHERE file_md5 = ?
ORDER BY
  (CASE WHEN remote_url IS NULL OR remote_url = '' THEN 0 ELSE 1 END) DESC,
  upload_time DESC,
  id DESC
LIMIT 1`, md5Value).Scan(&keepID, &keepUserID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return groups, fmt.Errorf("select keep row for md5 dedup: %w", err)
		}

		if len(res.Samples) < req.SampleLimit {
			res.Samples = append(res.Samples, RepairMediaHistorySample{
				Kind:        "md5",
				UserID:      keepUserID,
				MD5:         md5Value,
				KeepID:      keepID,
				DeleteCount: int(max64(cnt-1, 0)),
			})
		}

		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, `
DELETE FROM media_upload_history
WHERE file_md5 = ? AND id <> ?`, md5Value, keepID)
		if err != nil {
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("md5 dedup delete failed md5=%s: %v", md5Value, err))
			}
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.DuplicatesByMD5.Deleted += int(n)
		}
	}
	if err := rows.Err(); err != nil {
		return groups, fmt.Errorf("iterate duplicate md5 groups: %w", err)
	}
	return groups, nil
}

func (s *MediaUploadService) dedupByLocalPathBatch(ctx context.Context, req *RepairMediaHistoryRequest, res *RepairMediaHistoryResult) (int, error) {
	var b strings.Builder
	b.WriteString(`
SELECT local_path, COUNT(*) AS cnt
FROM media_upload_history
WHERE (file_md5 IS NULL OR file_md5 = '')
  AND local_path IS NOT NULL AND local_path <> ''`)
	args := make([]any, 0, 1)
	b.WriteString(`
GROUP BY local_path
HAVING COUNT(*) > 1
ORDER BY cnt DESC
LIMIT ?`)
	args = append(args, req.MaxDuplicateGroups)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("query duplicate local_path groups: %w", err)
	}
	defer rows.Close()

	groups := 0
	for rows.Next() {
		var localPath string
		var cnt int64
		if err := rows.Scan(&localPath, &cnt); err != nil {
			return groups, fmt.Errorf("scan duplicate local_path group: %w", err)
		}

		res.DuplicatesByLocalPath.Groups++
		groups++
		res.DuplicatesByLocalPath.Rows += int(cnt)
		if cnt > 0 {
			res.DuplicatesByLocalPath.ToDelete += int(cnt - 1)
		}

		var keepID int64
		var keepUserID string
		if err := s.db.QueryRowContext(ctx, `
SELECT id, user_id
FROM media_upload_history
WHERE local_path = ? AND (file_md5 IS NULL OR file_md5 = '')
ORDER BY
  (CASE WHEN remote_url IS NULL OR remote_url = '' THEN 0 ELSE 1 END) DESC,
  upload_time DESC,
  id DESC
LIMIT 1`, localPath).Scan(&keepID, &keepUserID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return groups, fmt.Errorf("select keep row for local_path dedup: %w", err)
		}

		if len(res.Samples) < req.SampleLimit {
			res.Samples = append(res.Samples, RepairMediaHistorySample{
				Kind:        "localPath",
				UserID:      keepUserID,
				KeepID:      keepID,
				DeleteCount: int(max64(cnt-1, 0)),
			})
		}

		if !req.Commit {
			continue
		}

		r, err := s.db.ExecContext(ctx, `
DELETE FROM media_upload_history
WHERE local_path = ? AND (file_md5 IS NULL OR file_md5 = '') AND id <> ?`, localPath, keepID)
		if err != nil {
			if len(res.Warnings) < 200 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("local_path dedup delete failed local_path=%s: %v", localPath, err))
			}
			continue
		}
		if n, _ := r.RowsAffected(); n > 0 {
			res.DuplicatesByLocalPath.Deleted += int(n)
		}
	}
	if err := rows.Err(); err != nil {
		return groups, fmt.Errorf("iterate duplicate local_path groups: %w", err)
	}
	return groups, nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
