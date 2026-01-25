package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var ErrDeleteForbidden = errors.New("文件不存在或无权删除")

type MediaUploadHistory struct {
	ID               int64  `json:"id"`
	UserID           string `json:"userId"`
	ToUserID         string `json:"toUserId,omitempty"`
	OriginalFilename string `json:"originalFilename,omitempty"`
	LocalFilename    string `json:"localFilename,omitempty"`
	RemoteFilename   string `json:"remoteFilename,omitempty"`
	RemoteURL        string `json:"remoteUrl,omitempty"`
	LocalPath        string `json:"localPath,omitempty"`
	FileSize         int64  `json:"fileSize,omitempty"`
	FileType         string `json:"fileType,omitempty"`
	FileExtension    string `json:"fileExtension,omitempty"`
	FileMD5          string `json:"fileMd5,omitempty"`
	UploadTime       string `json:"uploadTime,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
	SendTime         string `json:"sendTime,omitempty"`
}

type MediaFileDTO struct {
	URL              string `json:"url"`
	Type             string `json:"type"`
	LocalFilename    string `json:"localFilename,omitempty"`
	OriginalFilename string `json:"originalFilename,omitempty"`
	FileSize         int64  `json:"fileSize,omitempty"`
	FileType         string `json:"fileType,omitempty"`
	FileExtension    string `json:"fileExtension,omitempty"`
	UploadTime       string `json:"uploadTime,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
}

type MediaUploadService struct {
	db         *sql.DB
	serverPort int
	fileStore  *FileStorageService
	imageSrv   *ImageServerService
	httpClient *http.Client
}

func NewMediaUploadService(db *sql.DB, serverPort int, fileStore *FileStorageService, imageSrv *ImageServerService, httpClient *http.Client) *MediaUploadService {
	return &MediaUploadService{
		db:         db,
		serverPort: serverPort,
		fileStore:  fileStore,
		imageSrv:   imageSrv,
		httpClient: httpClient,
	}
}

type UploadRecord struct {
	UserID           string
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

func (s *MediaUploadService) SaveUploadRecord(ctx context.Context, record UploadRecord) (*MediaUploadHistory, error) {
	if strings.TrimSpace(record.FileMD5) != "" {
		existing, err := s.findMediaFileByUserAndMD5(ctx, record.UserID, record.FileMD5)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			now := time.Now()
			if _, err := s.db.ExecContext(ctx, "UPDATE media_file SET update_time = ? WHERE id = ?", now, existing.ID); err != nil {
				return nil, err
			}
			existing.UpdateTime = now.Format("2006-01-02 15:04:05")
			return existing, nil
		}
	}

	now := time.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO media_file
		(user_id, original_filename, local_filename, remote_filename, remote_url, local_path, file_size, file_type, file_extension, file_md5, upload_time, update_time, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.UserID,
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
	id, _ := res.LastInsertId()

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

func (s *MediaUploadService) RecordImageSend(ctx context.Context, remoteURL, fromUserID, toUserID, localFilename string) (*MediaUploadHistory, error) {
	var original *storedMediaFile

	normalizedRemoteURL := strings.TrimSpace(remoteURL)

	// 1) localFilename 优先（先限定 userId，再全局兜底）
	if strings.TrimSpace(localFilename) != "" {
		normalizedLocal := strings.TrimSpace(localFilename)
		original, _ = s.findStoredMediaFileByLocalFilename(ctx, normalizedLocal, fromUserID)
		if original == nil {
			original, _ = s.findStoredMediaFileByLocalFilename(ctx, normalizedLocal, "")
		}
	}

	// 2) remoteUrl 精确匹配
	if original == nil && normalizedRemoteURL != "" {
		original, _ = s.findStoredMediaFileByRemoteURL(ctx, normalizedRemoteURL, fromUserID)
		if original == nil {
			original, _ = s.findStoredMediaFileByRemoteURL(ctx, normalizedRemoteURL, "")
		}
	}

	// 3) remoteFilename（/img/Upload/ 后面的相对路径）
	if original == nil && normalizedRemoteURL != "" {
		remoteFilename := extractRemoteFilenameFromURL(normalizedRemoteURL)
		if remoteFilename != "" {
			original, _ = s.findStoredMediaFileByRemoteFilename(ctx, remoteFilename, fromUserID)
			if original == nil {
				original, _ = s.findStoredMediaFileByRemoteFilename(ctx, remoteFilename, "")
			}
		}
	}

	// 4) 兜底：basename
	if original == nil && normalizedRemoteURL != "" {
		filename := extractFilenameFromURL(normalizedRemoteURL)
		if filename != "" {
			original, _ = s.findStoredMediaFileByRemoteFilename(ctx, filename, fromUserID)
			if original == nil {
				original, _ = s.findStoredMediaFileByRemoteFilename(ctx, filename, "")
			}
		}
	}

	if original == nil || original.File == nil {
		return nil, nil
	}

	// 去重：remoteUrl + from + to
	existingLog, err := s.findSendLog(ctx, remoteURL, fromUserID, toUserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if existingLog != nil {
		if _, err := s.db.ExecContext(ctx, "UPDATE media_send_log SET send_time = ? WHERE id = ?", now, existingLog.ID); err != nil {
			return nil, err
		}
		_ = s.updateTimeByStoredMediaFile(ctx, original, now)

		out := *original.File
		out.ToUserID = toUserID
		out.SendTime = now.Format("2006-01-02 15:04:05")
		out.RemoteURL = existingLog.RemoteURL
		return &out, nil
	}

	res, err := s.db.ExecContext(ctx,
		"INSERT INTO media_send_log (user_id, to_user_id, local_path, remote_url, send_time, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		fromUserID, toUserID, original.File.LocalPath, remoteURL, now, now)
	if err != nil {
		return nil, err
	}
	_, _ = res.LastInsertId()
	_ = s.updateTimeByStoredMediaFile(ctx, original, now)

	out := *original.File
	out.ToUserID = toUserID
	out.SendTime = now.Format("2006-01-02 15:04:05")
	out.RemoteURL = remoteURL
	return &out, nil
}

func (s *MediaUploadService) GetUserUploadHistory(ctx context.Context, userID string, page, pageSize int, hostHeader string) ([]MediaUploadHistory, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
		WHERE user_id = ?
		ORDER BY update_time DESC
		LIMIT ? OFFSET ?`, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MediaUploadHistory
	for rows.Next() {
		h, err := scanMediaFileHistory(rows)
		if err != nil {
			return nil, err
		}
		h.RemoteURL = s.convertToLocalURL(h.LocalPath, hostHeader)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *MediaUploadService) GetUserUploadCount(ctx context.Context, userID string) (int, error) {
	// 兼容现状：未按 user_id 过滤
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_file").Scan(&total)
	return total, err
}

func (s *MediaUploadService) GetUserSentImages(ctx context.Context, fromUserID, toUserID string, page, pageSize int, hostHeader string) ([]MediaUploadHistory, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	rows, err := s.db.QueryContext(ctx, `SELECT id, local_path, remote_url, send_time
		FROM media_send_log
		WHERE user_id = ? AND to_user_id = ?
		ORDER BY send_time DESC
		LIMIT ? OFFSET ?`, fromUserID, toUserID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MediaUploadHistory
	for rows.Next() {
		var logID int64
		var localPath, remoteURL string
		var sendTime time.Time
		if err := rows.Scan(&logID, &localPath, &remoteURL, &sendTime); err != nil {
			return nil, err
		}

		stored, _ := s.findStoredMediaFileByLocalPath(ctx, localPath, fromUserID)
		var file *MediaUploadHistory
		if stored != nil {
			file = stored.File
		}
		if file == nil {
			file = &MediaUploadHistory{}
		}

		item := *file
		item.ToUserID = toUserID
		item.SendTime = sendTime.Format("2006-01-02 15:04:05")
		item.RemoteURL = s.convertToLocalURL(localPath, hostHeader)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *MediaUploadService) GetUserSentCount(ctx context.Context, fromUserID, toUserID string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_send_log WHERE user_id = ? AND to_user_id = ?", fromUserID, toUserID).Scan(&total)
	return total, err
}

func (s *MediaUploadService) GetChatImages(ctx context.Context, userID1, userID2 string, limit int, hostHeader string) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `SELECT local_path
		FROM media_send_log
		WHERE ((user_id = ? AND to_user_id = ?) OR (user_id = ? AND to_user_id = ?))
		GROUP BY local_path
		ORDER BY MAX(send_time) DESC
		LIMIT ?`, userID1, userID2, userID2, userID1, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var localPath string
		if err := rows.Scan(&localPath); err != nil {
			return nil, err
		}
		urls = append(urls, s.convertToLocalURL(localPath, hostHeader))
	}
	return urls, rows.Err()
}

func (s *MediaUploadService) ConvertPathsToLocalURLs(localPaths []string, hostHeader string) []string {
	if len(localPaths) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(localPaths))
	for _, p := range localPaths {
		url := s.convertToLocalURL(p, hostHeader)
		if url != "" {
			out = append(out, url)
		}
	}
	return out
}

func (s *MediaUploadService) ReuploadLocalFile(ctx context.Context, userID, localPath, cookieData, referer, userAgent string) (string, error) {
	fileBytes, err := s.fileStore.ReadLocalFile(localPath)
	if err != nil {
		return "", err
	}
	if len(fileBytes) == 0 {
		return "", fmt.Errorf("本地文件不存在: %s", localPath)
	}

	stored, _ := s.findStoredMediaFileByLocalPath(ctx, localPath, userID)
	mediaFile := (*MediaUploadHistory)(nil)
	if stored != nil {
		mediaFile = stored.File
	}
	originalFilename := ""
	if mediaFile != nil && strings.TrimSpace(mediaFile.OriginalFilename) != "" {
		originalFilename = mediaFile.OriginalFilename
	} else {
		originalFilename = filepath.Base(strings.TrimPrefix(localPath, "/"))
	}

	imgServerHost := s.imageSrv.GetImgServerHost()
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userID)

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)
	part, _ := writer.CreateFormFile("upload_file", originalFilename)
	_, _ = io.Copy(part, bytes.NewReader(fileBytes))
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bodyBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Host", strings.Split(imgServerHost, ":")[0])
	req.Header.Set("Origin", "http://v1.chat2019.cn")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)
	// 兼容 Java：cookieData 为空字符串时也会设置 Cookie 头
	req.Header.Set("Cookie", cookieData)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("上游响应异常: %s", resp.Status)
	}

	// 更新时间：不按 user_id 限制，并兼容 local_path 可能无前导 "/" 的情况
	normalizedPath := strings.ReplaceAll(strings.TrimSpace(localPath), "\\", "/")
	updatedRows := 0
	if normalizedPath != "" {
		now := time.Now()
		updatedRows += s.updateTimeByLocalPathIgnoreUser(ctx, normalizedPath, now)
		altPath := normalizedPath
		if strings.HasPrefix(altPath, "/") {
			altPath = strings.TrimPrefix(altPath, "/")
		} else {
			altPath = "/" + altPath
		}
		if altPath != normalizedPath {
			updatedRows += s.updateTimeByLocalPathIgnoreUser(ctx, altPath, now)
		}
	}
	_ = updatedRows

	return string(respBody), nil
}

func (s *MediaUploadService) GetAllUploadImagesWithDetails(ctx context.Context, page, pageSize int, hostHeader string) ([]MediaFileDTO, error) {
	return s.GetAllUploadImagesWithDetailsBySource(ctx, page, pageSize, hostHeader, "all", "")
}

func (s *MediaUploadService) GetAllUploadImagesWithDetailsBySource(ctx context.Context, page, pageSize int, hostHeader string, source, douyinSecUserID string) ([]MediaFileDTO, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		source = "all"
	}

	var (
		rows *sql.Rows
		err  error
	)
	switch source {
	case "local":
		rows, err = s.db.QueryContext(ctx, `SELECT local_filename, original_filename, local_path, file_size, file_type, file_extension, upload_time, update_time
			FROM media_file
			ORDER BY update_time DESC
			LIMIT ? OFFSET ?`, pageSize, offset)
	case "douyin":
		args := make([]any, 0, 3)
		query := `SELECT local_filename, original_filename, local_path, file_size, file_type, file_extension, upload_time, update_time
			FROM douyin_media_file`
		if strings.TrimSpace(douyinSecUserID) != "" {
			query += " WHERE sec_user_id = ?"
			args = append(args, douyinSecUserID)
		}
		query += " ORDER BY update_time DESC LIMIT ? OFFSET ?"
		args = append(args, pageSize, offset)
		rows, err = s.db.QueryContext(ctx, query, args...)
	default:
		rows, err = s.db.QueryContext(ctx, `(
			SELECT local_filename, original_filename, local_path, file_size, file_type, file_extension, upload_time, update_time
			FROM media_file
		) UNION ALL (
			SELECT local_filename, original_filename, local_path, file_size, file_type, file_extension, upload_time, update_time
			FROM douyin_media_file
		)
		ORDER BY update_time DESC
		LIMIT ? OFFSET ?`, pageSize, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MediaFileDTO
	for rows.Next() {
		var localFilename, originalFilename, localPath, fileType, fileExtension string
		var fileSize int64
		var uploadTime time.Time
		var updateTime sql.NullTime
		if err := rows.Scan(&localFilename, &originalFilename, &localPath, &fileSize, &fileType, &fileExtension, &uploadTime, &updateTime); err != nil {
			return nil, err
		}

		dto := MediaFileDTO{
			URL:              s.convertToLocalURL(localPath, hostHeader),
			Type:             inferTypeFromExtension(fileExtension),
			LocalFilename:    localFilename,
			OriginalFilename: originalFilename,
			FileSize:         fileSize,
			FileType:         fileType,
			FileExtension:    fileExtension,
			UploadTime:       uploadTime.Format("2006-01-02T15:04:05"),
			UpdateTime:       "",
		}
		if updateTime.Valid {
			dto.UpdateTime = updateTime.Time.Format("2006-01-02T15:04:05")
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *MediaUploadService) GetAllUploadImagesCount(ctx context.Context) (int, error) {
	return s.GetAllUploadImagesCountBySource(ctx, "all", "")
}

func (s *MediaUploadService) GetAllUploadImagesCountBySource(ctx context.Context, source, douyinSecUserID string) (int, error) {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		source = "all"
	}

	switch source {
	case "local":
		var total int
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_file").Scan(&total)
		return total, err
	case "douyin":
		args := make([]any, 0, 1)
		query := "SELECT COUNT(*) FROM douyin_media_file"
		if strings.TrimSpace(douyinSecUserID) != "" {
			query += " WHERE sec_user_id = ?"
			args = append(args, douyinSecUserID)
		}
		var total int
		err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
		return total, err
	default:
		var totalLocal int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_file").Scan(&totalLocal); err != nil {
			return 0, err
		}
		var totalDouyin int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM douyin_media_file").Scan(&totalDouyin); err != nil {
			return 0, err
		}
		return totalLocal + totalDouyin, nil
	}
}

type DeleteResult struct {
	DeletedRecords int  `json:"deletedRecords"`
	FileDeleted    bool `json:"fileDeleted"`
}

func (s *MediaUploadService) DeleteMediaByPath(ctx context.Context, userID, localPath string) (DeleteResult, error) {
	normalizedPath := normalizeUploadLocalPathInput(localPath)
	if normalizedPath == "" {
		return DeleteResult{}, ErrDeleteForbidden
	}

	// 说明：全站图片库展示不按 userId 过滤，因此删除也不校验上传者归属（userID 参数仅用于兼容旧调用）。
	stored, err := s.findStoredMediaFileByLocalPath(ctx, normalizedPath, "")
	if err != nil {
		return DeleteResult{}, err
	}
	if stored == nil || stored.File == nil {
		return DeleteResult{}, ErrDeleteForbidden
	}

	file := stored.File
	storedNormalized := normalizeUploadLocalPathInput(file.LocalPath)

	candidateSet := make(map[string]struct{}, 8)
	addCandidate := func(p string) {
		p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
		if p == "" {
			return
		}
		if idx := strings.IndexAny(p, "?#"); idx >= 0 {
			p = p[:idx]
		}
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if strings.HasPrefix(p, "/") {
			candidateSet[p] = struct{}{}
			candidateSet[strings.TrimPrefix(p, "/")] = struct{}{}
		} else {
			candidateSet[p] = struct{}{}
			candidateSet["/"+p] = struct{}{}
		}
	}

	addCandidate(file.LocalPath)
	addCandidate(storedNormalized)
	addCandidate(normalizedPath)

	paths := make([]string, 0, len(candidateSet))
	for p := range candidateSet {
		paths = append(paths, p)
	}

	clean := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(storedNormalized), "\\", "/"), "/")
	if clean == "" {
		return DeleteResult{}, ErrDeleteForbidden
	}

	pathWithSlash := "/" + clean
	pathWithoutSlash := clean

	paths = append(paths, pathWithSlash, pathWithoutSlash)

	for _, p := range paths {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM media_send_log WHERE local_path = ?", p); err != nil {
			return DeleteResult{}, err
		}
	}

	deletedCount := 0
	for _, p := range paths {
		var (
			res sql.Result
			err error
		)
		if stored.Source == mediaFileSourceDouyin {
			res, err = s.db.ExecContext(ctx, "DELETE FROM douyin_media_file WHERE local_path = ?", p)
		} else {
			res, err = s.db.ExecContext(ctx, "DELETE FROM media_file WHERE local_path = ?", p)
		}
		if err != nil {
			return DeleteResult{}, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			deletedCount += int(n)
		}
	}

	fileDeleted := false
	if strings.TrimSpace(file.FileMD5) != "" {
		if remaining, err := s.countAnyMediaFileByMD5(ctx, file.FileMD5); err == nil {
			if remaining == 0 {
				fileDeleted = s.fileStore.DeleteFile(storedNormalized)
			}
		}
	} else {
		fileDeleted = s.fileStore.DeleteFile(storedNormalized)
	}

	return DeleteResult{DeletedRecords: deletedCount, FileDeleted: fileDeleted}, nil
}

type BatchDeleteResult struct {
	SuccessCount int                 `json:"successCount"`
	FailCount    int                 `json:"failCount"`
	FailedItems  []map[string]string `json:"failedItems"`
}

func (s *MediaUploadService) BatchDeleteMedia(ctx context.Context, userID string, localPaths []string) (BatchDeleteResult, error) {
	if s == nil || s.db == nil || s.fileStore == nil {
		return BatchDeleteResult{}, fmt.Errorf("服务未初始化")
	}

	result := BatchDeleteResult{
		FailedItems: make([]map[string]string, 0),
	}
	for _, localPath := range localPaths {
		if _, err := s.DeleteMediaByPath(ctx, userID, localPath); err != nil {
			result.FailCount++
			result.FailedItems = append(result.FailedItems, map[string]string{
				"localPath": localPath,
				"reason":    err.Error(),
			})
			continue
		}
		result.SuccessCount++
	}
	return result, nil
}

// --- internal helpers ---

func (s *MediaUploadService) convertToLocalURL(localPath string, hostHeader string) string {
	if strings.TrimSpace(localPath) == "" {
		return ""
	}
	path := localPath
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	host := strings.TrimSpace(hostHeader)
	if host == "" {
		host = fmt.Sprintf("localhost:%d", s.serverPort)
	}
	return "http://" + host + "/upload" + path
}

func normalizeUploadLocalPathInput(localPath string) string {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return ""
	}

	localPath = strings.ReplaceAll(localPath, "\\", "/")

	if strings.HasPrefix(localPath, "http://") || strings.HasPrefix(localPath, "https://") {
		if u, err := url.Parse(localPath); err == nil && u.Path != "" {
			localPath = u.Path
		}
	} else {
		if idx := strings.IndexAny(localPath, "?#"); idx >= 0 {
			localPath = localPath[:idx]
		}
	}

	// 兼容：部分调用方可能会将 localPath 作为普通字符串再做一次 encodeURIComponent，导致这里仍含 %2F 等编码。
	// 使用 PathUnescape 兜底解码一次（媒体文件名为 uuid+timestamp，不应包含 %）。
	if strings.Contains(localPath, "%") {
		if unescaped, err := url.PathUnescape(localPath); err == nil && unescaped != "" {
			localPath = unescaped
		}
	}

	// 兼容前端/历史数据：允许传入 /upload/images/... 或完整 URL
	if strings.HasPrefix(localPath, "/upload/") {
		localPath = strings.TrimPrefix(localPath, "/upload")
	} else if strings.HasPrefix(localPath, "upload/") {
		localPath = strings.TrimPrefix(localPath, "upload")
	}

	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return ""
	}
	if !strings.HasPrefix(localPath, "/") {
		localPath = "/" + localPath
	}
	return localPath
}

func scanMediaFileHistory(rows *sql.Rows) (MediaUploadHistory, error) {
	var h MediaUploadHistory
	var fileMD5 sql.NullString
	var uploadTime time.Time
	var updateTime sql.NullTime
	if err := rows.Scan(
		&h.ID,
		&h.UserID,
		&h.OriginalFilename,
		&h.LocalFilename,
		&h.RemoteFilename,
		&h.RemoteURL,
		&h.LocalPath,
		&h.FileSize,
		&h.FileType,
		&h.FileExtension,
		&fileMD5,
		&uploadTime,
		&updateTime,
	); err != nil {
		return MediaUploadHistory{}, err
	}
	if fileMD5.Valid {
		h.FileMD5 = fileMD5.String
	}
	h.UploadTime = uploadTime.Format("2006-01-02 15:04:05")
	if updateTime.Valid {
		h.UpdateTime = updateTime.Time.Format("2006-01-02 15:04:05")
	}
	return h, nil
}

func (s *MediaUploadService) findMediaFileByUserAndMD5(ctx context.Context, userID, md5 string) (*MediaUploadHistory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
		WHERE user_id = ? AND file_md5 = ?
		ORDER BY id
		LIMIT 1`, userID, md5)
	return scanMediaFileHistoryRow(row)
}

func (s *MediaUploadService) findMediaFileByLocalFilename(ctx context.Context, localFilename, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
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

func (s *MediaUploadService) findMediaFileByRemoteURL(ctx context.Context, remoteURL, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
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

func (s *MediaUploadService) findMediaFileByRemoteFilename(ctx context.Context, remoteFilename, userID string) (*MediaUploadHistory, error) {
	query := `SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,
		file_size, file_type, file_extension, file_md5, upload_time, update_time
		FROM media_file
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

func (s *MediaUploadService) findMediaFileByLocalPath(ctx context.Context, localPath, userID string) (*MediaUploadHistory, error) {
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
		FROM media_file
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

type sendLog struct {
	ID        int64
	RemoteURL string
}

func (s *MediaUploadService) findSendLog(ctx context.Context, remoteURL, fromUserID, toUserID string) (*sendLog, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, remote_url FROM media_send_log WHERE remote_url = ? AND user_id = ? AND to_user_id = ? ORDER BY id LIMIT 1",
		remoteURL, fromUserID, toUserID)
	var out sendLog
	if err := row.Scan(&out.ID, &out.RemoteURL); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (s *MediaUploadService) updateTimeByLocalPathIgnoreUser(ctx context.Context, localPath string, now time.Time) int {
	updated := 0

	res, err := s.db.ExecContext(ctx, "UPDATE media_file SET update_time = ? WHERE local_path = ?", now, localPath)
	if err == nil {
		affected, _ := res.RowsAffected()
		updated += int(affected)
	}

	res2, err2 := s.db.ExecContext(ctx, "UPDATE douyin_media_file SET update_time = ? WHERE local_path = ?", now, localPath)
	if err2 == nil {
		affected, _ := res2.RowsAffected()
		updated += int(affected)
	}

	return updated
}

func scanMediaFileHistoryRow(row *sql.Row) (*MediaUploadHistory, error) {
	var h MediaUploadHistory
	var fileMD5 sql.NullString
	var uploadTime time.Time
	var updateTime sql.NullTime
	err := row.Scan(
		&h.ID,
		&h.UserID,
		&h.OriginalFilename,
		&h.LocalFilename,
		&h.RemoteFilename,
		&h.RemoteURL,
		&h.LocalPath,
		&h.FileSize,
		&h.FileType,
		&h.FileExtension,
		&fileMD5,
		&uploadTime,
		&updateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if fileMD5.Valid {
		h.FileMD5 = fileMD5.String
	}
	h.UploadTime = uploadTime.Format("2006-01-02 15:04:05")
	if updateTime.Valid {
		h.UpdateTime = updateTime.Time.Format("2006-01-02 15:04:05")
	}
	return &h, nil
}

func nullStringIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func extractFilenameFromURL(url string) string {
	if url == "" {
		return ""
	}
	idx := strings.LastIndex(url, "/")
	if idx >= 0 && idx < len(url)-1 {
		return url[idx+1:]
	}
	return ""
}

func extractRemoteFilenameFromURL(url string) string {
	marker := "/img/Upload/"
	idx := strings.Index(url, marker)
	if idx >= 0 && idx+len(marker) < len(url) {
		return url[idx+len(marker):]
	}
	return ""
}

var (
	reImageExt = regexp.MustCompile(`(?i)^(jpg|jpeg|png|gif|webp|bmp)$`)
	reVideoExt = regexp.MustCompile(`(?i)^(mp4|mov|avi|mkv|webm)$`)
)

func inferTypeFromExtension(extension string) string {
	ext := strings.ToLower(strings.TrimSpace(extension))
	ext = strings.TrimPrefix(ext, ".")
	switch {
	case reImageExt.MatchString(ext):
		return "image"
	case reVideoExt.MatchString(ext):
		return "video"
	default:
		return "file"
	}
}

// parseJSONStateOK 用于 uploadMedia/reuploadHistoryImage 的返回解析。
func parseJSONStateOK(body string) bool {
	var v map[string]any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return false
	}
	state, _ := v["state"].(string)
	return state == "OK"
}
