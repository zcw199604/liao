package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	integrationMediaDefaultPageSize = 60
	integrationMediaMaxPageSize     = 200
	integrationMediaMaxQueryLength  = 64
)

// integrationMediaRow 是 media_file 与 douyin_media_file 统一后的只读视图。
type integrationMediaRow struct {
	Source           string
	LocalPath        string
	LocalFilename    string
	OriginalFilename string
	FileSize         int64
	FileType         string
	FileExtension    string
	FileMD5          string
	Width            int
	Height           int
	SortTime         time.Time
	DouyinSecUserID  string
}

func (r integrationMediaRow) key() string {
	return r.Source + ":" + r.LocalPath
}

func (r integrationMediaRow) kind() string {
	return inferTypeFromExtension(r.FileExtension)
}

func (r integrationMediaRow) mimeType() string {
	if value := strings.TrimSpace(r.FileType); value != "" {
		return value
	}
	ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(r.FileExtension), "."))
	if ext == "" {
		return "application/octet-stream"
	}
	if value := mime.TypeByExtension("." + ext); value != "" {
		return value
	}
	return "application/octet-stream"
}

func (r integrationMediaRow) displayName() string {
	for _, candidate := range []string{r.OriginalFilename, r.LocalFilename, path.Base(r.LocalPath)} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return "media"
}

// integrationSendExtensions 与 AnonChat 的发送白名单保持一致。
var integrationSendExtensions = map[string]struct{}{
	"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "webp": {},
	"mp4": {}, "webm": {},
}

func integrationSupported(row integrationMediaRow) bool {
	ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(row.FileExtension), "."))
	_, ok := integrationSendExtensions[ext]
	return ok
}

const integrationMediaLocalSelect = `SELECT 'local' AS source, local_path, local_filename, original_filename,
	file_size, file_type, file_extension, file_md5, media_width, media_height,
	COALESCE(update_time, upload_time) AS sort_time, NULL AS sec_user_id
	FROM media_file`

const integrationMediaDouyinSelect = `SELECT 'douyin' AS source, local_path, local_filename, original_filename,
	file_size, file_type, file_extension, file_md5, media_width, media_height,
	COALESCE(update_time, upload_time) AS sort_time, sec_user_id
	FROM douyin_media_file`

func integrationMediaUnion() string {
	return integrationMediaLocalSelect + "\nUNION ALL\n" + integrationMediaDouyinSelect
}

type integrationMediaQuery struct {
	Page            int
	PageSize        int
	Source          string
	Kind            string
	Keyword         string
	From            *time.Time
	To              *time.Time
	DouyinSecUserID string
}

func parseIntegrationMediaQuery(values map[string][]string) (integrationMediaQuery, error) {
	get := func(key string) string {
		if list := values[key]; len(list) > 0 {
			return strings.TrimSpace(list[0])
		}
		return ""
	}
	query := integrationMediaQuery{
		Page:            1,
		PageSize:        integrationMediaDefaultPageSize,
		Source:          strings.ToLower(get("source")),
		Kind:            strings.ToLower(get("kind")),
		Keyword:         get("q"),
		DouyinSecUserID: get("douyinSecUserId"),
	}
	if raw := get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100000 {
			return query, fmt.Errorf("page 参数非法")
		}
		query.Page = value
	}
	if raw := get("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return query, fmt.Errorf("pageSize 参数非法")
		}
		if value > integrationMediaMaxPageSize {
			value = integrationMediaMaxPageSize
		}
		query.PageSize = value
	}
	switch query.Source {
	case "", "all":
		query.Source = "all"
	case "local", "douyin":
	default:
		return query, fmt.Errorf("source 参数非法")
	}
	switch query.Kind {
	case "", "all":
		query.Kind = "all"
	case "image", "video":
	default:
		return query, fmt.Errorf("kind 参数非法")
	}
	if len([]rune(query.Keyword)) > integrationMediaMaxQueryLength {
		return query, fmt.Errorf("q 过长（最多 %d 个字符）", integrationMediaMaxQueryLength)
	}
	if raw := get("from"); raw != "" {
		value, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			return query, fmt.Errorf("from 日期非法")
		}
		query.From = &value
	}
	if raw := get("to"); raw != "" {
		value, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			return query, fmt.Errorf("to 日期非法")
		}
		next := value.AddDate(0, 0, 1)
		query.To = &next
	}
	return query, nil
}

func (q integrationMediaQuery) where() (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}
	if q.Source != "all" {
		clauses = append(clauses, "t.source = ?")
		args = append(args, q.Source)
	}
	switch q.Kind {
	case "image":
		clauses = append(clauses, "LOWER(t.file_extension) IN ('jpg','jpeg','png','gif','webp','bmp')")
	case "video":
		clauses = append(clauses, "LOWER(t.file_extension) IN ('mp4','mov','avi','mkv','webm')")
	}
	if q.Keyword != "" {
		clauses = append(clauses, "LOWER(t.original_filename) LIKE ?")
		args = append(args, "%"+strings.ToLower(q.Keyword)+"%")
	}
	if q.From != nil {
		clauses = append(clauses, "t.sort_time >= ?")
		args = append(args, *q.From)
	}
	if q.To != nil {
		clauses = append(clauses, "t.sort_time < ?")
		args = append(args, *q.To)
	}
	if q.DouyinSecUserID != "" {
		clauses = append(clauses, "t.sec_user_id = ?")
		args = append(args, q.DouyinSecUserID)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanIntegrationMediaRow(scan func(dest ...any) error) (integrationMediaRow, error) {
	var row integrationMediaRow
	var md5, sec sql.NullString
	var width, height sql.NullInt64
	if err := scan(
		&row.Source, &row.LocalPath, &row.LocalFilename, &row.OriginalFilename,
		&row.FileSize, &row.FileType, &row.FileExtension, &md5, &width, &height,
		&row.SortTime, &sec,
	); err != nil {
		return integrationMediaRow{}, err
	}
	row.FileMD5 = strings.TrimSpace(md5.String)
	row.DouyinSecUserID = strings.TrimSpace(sec.String)
	if width.Valid {
		row.Width = int(width.Int64)
	}
	if height.Valid {
		row.Height = int(height.Int64)
	}
	return row, nil
}

func (a *App) integrationMediaPage(ctx context.Context, query integrationMediaQuery) ([]integrationMediaRow, int, error) {
	union := integrationMediaUnion()
	where, args := query.where()

	var total int
	countSQL := "SELECT COUNT(*) FROM (" + union + ") t" + where
	if err := a.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	listArgs := append(append([]any{}, args...), query.PageSize, offset)
	rows, err := a.db.QueryContext(ctx,
		"SELECT t.source, t.local_path, t.local_filename, t.original_filename, t.file_size, t.file_type, t.file_extension, t.file_md5, t.media_width, t.media_height, t.sort_time, t.sec_user_id FROM ("+union+") t"+where+" ORDER BY t.sort_time DESC, t.local_path DESC LIMIT ? OFFSET ?",
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]integrationMediaRow, 0, query.PageSize)
	for rows.Next() {
		row, scanErr := scanIntegrationMediaRow(rows.Scan)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (a *App) integrationMediaRowByKey(ctx context.Context, key string) (integrationMediaRow, error) {
	source, localPath, ok := strings.Cut(strings.TrimSpace(key), ":")
	if !ok || strings.TrimSpace(localPath) == "" {
		return integrationMediaRow{}, fmt.Errorf("key 非法")
	}
	var query string
	switch source {
	case "local":
		query = integrationMediaLocalSelect + " WHERE local_path = ? LIMIT 1"
	case "douyin":
		query = integrationMediaDouyinSelect + " WHERE local_path = ? LIMIT 1"
	default:
		return integrationMediaRow{}, fmt.Errorf("key 来源非法")
	}
	row, err := scanIntegrationMediaRow(a.db.QueryRowContext(ctx, query, localPath).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return integrationMediaRow{}, os.ErrNotExist
	}
	if err != nil {
		return integrationMediaRow{}, err
	}
	return row, nil
}

type integrationMediaItem struct {
	Key       string `json:"key"`
	Source    string `json:"source"`
	Kind      string `json:"kind"`
	Mime      string `json:"mime"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Date      string `json:"date"`
	MD5       string `json:"md5,omitempty"`
	HasPoster bool   `json:"hasPoster"`
	Supported bool   `json:"supported"`
}

func (a *App) integrationMediaDTO(row integrationMediaRow) integrationMediaItem {
	item := integrationMediaItem{
		Key:       row.key(),
		Source:    row.Source,
		Kind:      row.kind(),
		Mime:      row.mimeType(),
		Name:      row.displayName(),
		SizeBytes: row.FileSize,
		Width:     row.Width,
		Height:    row.Height,
		Date:      row.SortTime.Format("2006-01-02T15:04:05"),
		MD5:       row.FileMD5,
		HasPoster: a.integrationMediaPosterAvailable(row),
		Supported: integrationSupported(row),
	}
	return item
}

func (a *App) integrationMediaPosterAvailable(row integrationMediaRow) bool {
	if row.kind() != "video" {
		return false
	}
	posterPath := buildVideoPosterLocalPath(row.LocalPath)
	if posterPath == "" {
		return false
	}
	abs, err := a.fileStorage.resolveUploadAbsPath(posterPath)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func (a *App) handleIntegrationMediaList(w http.ResponseWriter, r *http.Request) {
	query, err := parseIntegrationMediaQuery(r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	rows, total, err := a.integrationMediaPage(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "查询媒体失败"})
		return
	}
	items := make([]integrationMediaItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, a.integrationMediaDTO(row))
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"page":       query.Page,
		"pageSize":   query.PageSize,
		"total":      total,
		"totalPages": totalPages,
	})
}

func (a *App) handleIntegrationMediaItem(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 不能为空"})
		return
	}
	row, err := a.integrationMediaRowByKey(r.Context(), key)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "媒体不存在"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a.integrationMediaDTO(row))
}

func (a *App) handleIntegrationMediaThumb(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 不能为空"})
		return
	}
	size := 320
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("size"))) {
	case "", "s320":
		size = 320
	case "s160":
		size = 160
	case "s640":
		size = 640
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "size 参数非法"})
		return
	}
	row, err := a.integrationMediaRowByKey(r.Context(), key)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	variant := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("variant")))
	kind := row.kind()
	if variant == "" {
		if kind == "video" {
			variant = "poster"
		} else {
			variant = "image"
		}
	}
	if variant != "image" && variant != "poster" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "variant 参数非法"})
		return
	}
	if (variant == "image" && kind != "image") || (variant == "poster" && kind != "video") {
		http.NotFound(w, r)
		return
	}

	sourcePath := row.LocalPath
	if variant == "poster" {
		sourcePath = buildVideoPosterLocalPath(row.LocalPath)
		if sourcePath == "" {
			http.NotFound(w, r)
			return
		}
	}
	abs, err := a.fileStorage.resolveUploadAbsPath(sourcePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "媒体路径非法"})
		return
	}
	data, etag, modTime, err := a.integrationThumbs.thumbnail(r.Context(), a.fileStorage.baseUploadAbs, row.key()+"|"+variant, abs, size)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, errIntegrationThumbUnsupported) {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]any{"error": "无法生成缩略图"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "生成缩略图失败"})
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if match := strings.TrimSpace(r.Header.Get("If-None-Match")); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	http.ServeContent(w, r, "", modTime, bytes.NewReader(data))
}

func (a *App) handleIntegrationMediaFile(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 不能为空"})
		return
	}
	row, err := a.integrationMediaRowByKey(r.Context(), key)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	abs, err := a.fileStorage.resolveUploadAbsPath(row.LocalPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "媒体路径非法"})
		return
	}
	file, info, err := openRegularUnder(a.fileStorage.baseUploadAbs, abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	name := row.displayName()
	w.Header().Set("Content-Type", row.mimeType())
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if disposition := mime.FormatMediaType("inline", map[string]string{"filename": name}); disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	http.ServeContent(w, r, filepath.Base(name), info.ModTime(), file)
}

// openRegularUnder 通过 os.Root 打开 upload 根目录内的普通文件，防止符号链接越界和特殊文件阻塞。
func openRegularUnder(rootDir, absPath string) (*os.File, os.FileInfo, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return nil, nil, os.ErrPermission
	}
	rel, err := filepath.Rel(rootDir, absPath)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return nil, nil, os.ErrPermission
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	file, err := root.Open(rel)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, nil, os.ErrNotExist
	}
	return file, info, nil
}
