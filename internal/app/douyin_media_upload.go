package app

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"liao/internal/database"
)

type mediaFileSource string

const (
	mediaFileSourceLocal  mediaFileSource = "local"
	mediaFileSourceDouyin mediaFileSource = "douyin"
)

type storedMediaFile struct {
	Source mediaFileSource
	File   *MediaUploadHistory
}

type DouyinUploadRecord struct {
	UserID           string
	SecUserID        string
	DetailID         string
	OriginalFilename string
	LocalFilename    string
	RemoteFilename   string
	RemoteURL        string
	LocalPath        string
	FileSize         int64
	FileType         string
	FileExtension    string
	FileMD5          string
}

func (s *MediaUploadService) SaveDouyinUploadRecord(ctx context.Context, record DouyinUploadRecord) (*MediaUploadHistory, error) {
	if strings.TrimSpace(record.FileMD5) != "" {
		// 说明：抖音导入媒体库按 MD5 全局去重（不按 user_id/sec_user_id 分桶）。
		existing, err := s.findDouyinMediaFileByMD5(ctx, record.FileMD5)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			now := time.Now()
			if _, err := s.db.ExecContext(ctx, "UPDATE douyin_media_file SET update_time = ? WHERE id = ?", now, existing.ID); err != nil {
				return nil, err
			}
			existing.UpdateTime = now.Format("2006-01-02 15:04:05")
			return existing, nil
		}
	}

	now := time.Now()
	id, err := database.InsertReturningID(ctx, s.db, `INSERT INTO douyin_media_file
		(user_id, sec_user_id, detail_id, original_filename, local_filename, remote_filename, remote_url, local_path, file_size, file_type, file_extension, file_md5, upload_time, update_time, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.UserID,
		nullStringIfEmpty(record.SecUserID),
		nullStringIfEmpty(record.DetailID),
		record.OriginalFilename,
		record.LocalFilename,
		record.RemoteFilename,
		record.RemoteURL,
		record.LocalPath,
		record.FileSize,
		record.FileType,
		record.FileExtension,
		nullStringIfEmpty(record.FileMD5),
		now,
		now,
		now,
	)
	if err != nil {
		return nil, err
	}

	return &MediaUploadHistory{
		ID:               id,
		UserID:           record.UserID,
		OriginalFilename: record.OriginalFilename,
		LocalFilename:    record.LocalFilename,
		RemoteFilename:   record.RemoteFilename,
		RemoteURL:        record.RemoteURL,
		LocalPath:        record.LocalPath,
		FileSize:         record.FileSize,
		FileType:         record.FileType,
		FileExtension:    record.FileExtension,
		FileMD5:          record.FileMD5,
		UploadTime:       now.Format("2006-01-02 15:04:05"),
		UpdateTime:       now.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *MediaUploadService) findDouyinMediaFileByUserAndMD5(ctx context.Context, userID, md5 string) (*MediaUploadHistory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE user_id = ? AND file_md5 = ?
		ORDER BY id
		LIMIT 1`, userID, md5)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileBySecUserAndMD5(ctx context.Context, secUserID, md5 string) (*MediaUploadHistory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE sec_user_id = ? AND file_md5 = ?
		ORDER BY id
		LIMIT 1`, secUserID, md5)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileByLocalFilename(ctx context.Context, localFilename, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE local_filename = ?`
	args := []any{localFilename}
	if strings.TrimSpace(userID) != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	query += " ORDER BY id LIMIT 1"
	row := s.db.QueryRowContext(ctx, query, args...)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileByRemoteURL(ctx context.Context, remoteURL, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE remote_url = ?`
	args := []any{remoteURL}
	if strings.TrimSpace(userID) != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	query += " ORDER BY id LIMIT 1"
	row := s.db.QueryRowContext(ctx, query, args...)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileByRemoteFilename(ctx context.Context, remoteFilename, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE remote_filename = ?`
	args := []any{remoteFilename}
	if strings.TrimSpace(userID) != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	query += " ORDER BY id LIMIT 1"
	row := s.db.QueryRowContext(ctx, query, args...)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileByLocalPath(ctx context.Context, localPath, userID string) (*MediaUploadHistory, error) {
	localPath = strings.ReplaceAll(strings.TrimSpace(localPath), "\\", "/")
	if localPath == "" {
		return nil, nil
	}

	candidateSet := make(map[string]struct{}, 6)
	addCandidate := func(p string) {
		p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")

		// 兼容 localPath 可能包含 query/hash
		if idx := strings.IndexAny(p, "?#"); idx >= 0 {
			p = p[:idx]
		}
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}

		// 兼容：保存时可能写成 /upload/images/...（历史数据/异常写入），查询时同时尝试保留与去除 /upload 前缀。
		if strings.HasPrefix(p, "/upload/") {
			candidateSet[p] = struct{}{}
			candidateSet[strings.TrimPrefix(p, "/upload")] = struct{}{}
		} else if strings.HasPrefix(p, "upload/") {
			candidateSet[p] = struct{}{}
			candidateSet[strings.TrimPrefix(p, "upload")] = struct{}{}
		} else {
			candidateSet[p] = struct{}{}
			if strings.HasPrefix(p, "/") {
				candidateSet["/upload"+p] = struct{}{}
			} else {
				candidateSet["upload/"+p] = struct{}{}
			}
		}

		// 兼容：local_path 有时会缺少前导 /。
		if strings.HasPrefix(p, "/") {
			candidateSet[strings.TrimPrefix(p, "/")] = struct{}{}
		} else {
			candidateSet["/"+p] = struct{}{}
		}
	}

	addCandidate(localPath)

	candidates := make([]string, 0, len(candidateSet))
	for candidate := range candidateSet {
		candidates = append(candidates, candidate)
	}

	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE local_path = ?`
	for _, candidate := range candidates {
		args := []any{candidate}
		q := query
		if strings.TrimSpace(userID) != "" {
			q += " AND user_id = ?"
			args = append(args, userID)
		}
		q += " ORDER BY id LIMIT 1"
		row := s.db.QueryRowContext(ctx, q, args...)
		file, err := scanMediaFileHistoryRow(row)
		if err != nil {
			return nil, err
		}
		if file != nil {
			return file, nil
		}
	}
	return nil, nil
}

func (s *MediaUploadService) findStoredMediaFileByUserAndMD5(ctx context.Context, userID, md5 string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByUserAndMD5(ctx, userID, md5)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByUserAndMD5(ctx, userID, md5)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) findMediaFileByMD5(ctx context.Context, md5 string) (*MediaUploadHistory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
		WHERE file_md5 = ?
		ORDER BY id
		LIMIT 1`, md5)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findDouyinMediaFileByMD5(ctx context.Context, md5 string) (*MediaUploadHistory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM douyin_media_file
		WHERE file_md5 = ?
		ORDER BY id
		LIMIT 1`, md5)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findStoredMediaFileByMD5(ctx context.Context, md5 string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByMD5(ctx, md5)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByMD5(ctx, md5)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) findStoredMediaFileByLocalFilename(ctx context.Context, localFilename, userID string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByLocalFilename(ctx, localFilename, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByLocalFilename(ctx, localFilename, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) findStoredMediaFileByRemoteURL(ctx context.Context, remoteURL, userID string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByRemoteURL(ctx, remoteURL, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByRemoteURL(ctx, remoteURL, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) findStoredMediaFileByRemoteFilename(ctx context.Context, remoteFilename, userID string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByRemoteFilename(ctx, remoteFilename, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByRemoteFilename(ctx, remoteFilename, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) findStoredMediaFileByLocalPath(ctx context.Context, localPath, userID string) (*storedMediaFile, error) {
	found, err := s.findMediaFileByLocalPath(ctx, localPath, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceLocal, File: found}, nil
	}

	found, err = s.findDouyinMediaFileByLocalPath(ctx, localPath, userID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return &storedMediaFile{Source: mediaFileSourceDouyin, File: found}, nil
	}
	return nil, nil
}

func (s *MediaUploadService) updateTimeByStoredMediaFile(ctx context.Context, stored *storedMediaFile, now time.Time) error {
	if stored == nil || stored.File == nil {
		return nil
	}
	switch stored.Source {
	case mediaFileSourceDouyin:
		_, err := s.db.ExecContext(ctx, "UPDATE douyin_media_file SET update_time = ? WHERE id = ?", now, stored.File.ID)
		return err
	default:
		_, err := s.db.ExecContext(ctx, "UPDATE media_file SET update_time = ? WHERE id = ?", now, stored.File.ID)
		return err
	}
}

func (s *MediaUploadService) countAnyMediaFileByMD5(ctx context.Context, md5 string) (int, error) {
	md5 = strings.TrimSpace(md5)
	if md5 == "" {
		return 0, nil
	}

	var c1 int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_file WHERE file_md5 = ?", md5).Scan(&c1); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	var c2 int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM douyin_media_file WHERE file_md5 = ?", md5).Scan(&c2); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return c1 + c2, nil
}
