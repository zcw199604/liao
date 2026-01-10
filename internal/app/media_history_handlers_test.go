package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newURLEncodedRequest(t *testing.T, method, urlStr string, values url.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, urlStr, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestHandleGetAllUploadImages_Success(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9006" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local)
	rows := sqlmock.NewRows([]string{
		"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
	}).AddRow(
		"x.png",
		"orig.png",
		"/images/2026/01/10/x.png",
		int64(4),
		"image/png",
		"png",
		uploadTime,
		sql.NullTime{Valid: false},
	)

	mock.ExpectQuery(`(?s)SELECT local_filename, original_filename, local_path, file_size, file_type, file_extension, upload_time, update_time\s+FROM media_file\s+ORDER BY update_time DESC\s+LIMIT \? OFFSET \?`).
		WithArgs(20, 0).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	app := &App{
		mediaUpload: &MediaUploadService{db: db, serverPort: 8080},
		imageServer: NewImageServerService("img-host", "9003"),
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local:8080/api/getAllUploadImages?page=1&pageSize=20", nil)
	rr := httptest.NewRecorder()
	app.handleGetAllUploadImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["port"].(string); got != "9006" {
		t.Fatalf("port=%q, want %q", got, "9006")
	}
	if got, _ := resp["total"].(float64); int(got) != 1 {
		t.Fatalf("total=%v, want 1", resp["total"])
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data=%v, want single item", resp["data"])
	}
	item, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("data[0] type=%T, want map", data[0])
	}
	if got, _ := item["url"].(string); got != "http://api.local:8080/upload/images/2026/01/10/x.png" {
		t.Fatalf("url=%q, want %q", got, "http://api.local:8080/upload/images/2026/01/10/x.png")
	}
	if got, _ := item["type"].(string); got != "image" {
		t.Fatalf("type=%q, want %q", got, "image")
	}
}

func TestHandleGetUserUploadHistory_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	}).AddRow(
		int64(1),
		"u1",
		"orig.png",
		"x.png",
		"remote.png",
		"http://remote",
		"/images/2026/01/10/x.png",
		int64(4),
		"image/png",
		"png",
		sql.NullString{String: "md5", Valid: true},
		uploadTime,
		sql.NullTime{Valid: false},
	)

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE user_id = \?\s+ORDER BY update_time DESC\s+LIMIT \? OFFSET \?`).
		WithArgs("u1", 20, 0).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	app := &App{
		mediaUpload: &MediaUploadService{db: db, serverPort: 8080},
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local:8080/api/getUserUploadHistory?userId=u1&page=1&pageSize=20", nil)
	rr := httptest.NewRecorder()
	app.handleGetUserUploadHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["success"].(bool); !got {
		t.Fatalf("success=%v, want true", resp["success"])
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	list, _ := data["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("list len=%d, want 1", len(list))
	}
	item, _ := list[0].(map[string]any)
	if item == nil {
		t.Fatalf("list[0] invalid: %v", list[0])
	}
	if got, _ := item["remoteUrl"].(string); got != "http://api.local:8080/upload/images/2026/01/10/x.png" {
		t.Fatalf("remoteUrl=%q, want %q", got, "http://api.local:8080/upload/images/2026/01/10/x.png")
	}
}

func TestHandleGetUserSentImages_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	sendTime := time.Date(2026, 1, 10, 11, 0, 0, 0, time.Local)
	logRows := sqlmock.NewRows([]string{"id", "local_path", "remote_url", "send_time"}).
		AddRow(int64(1), "/images/2026/01/10/x.png", "http://remote", sendTime)

	mock.ExpectQuery(`(?s)FROM media_send_log\s+WHERE user_id = \? AND to_user_id = \?\s+ORDER BY send_time DESC\s+LIMIT \? OFFSET \?`).
		WithArgs("u1", "u2", 20, 0).
		WillReturnRows(logRows)

	fileRows := sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	}).AddRow(
		int64(1),
		"u1",
		"orig.png",
		"x.png",
		"remote.png",
		"http://remote",
		"/images/2026/01/10/x.png",
		int64(4),
		"image/png",
		"png",
		sql.NullString{String: "md5", Valid: true},
		time.Now(),
		sql.NullTime{Valid: false},
	)

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_path = \?\s+AND user_id = \?\s+ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(fileRows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_send_log WHERE user_id = \? AND to_user_id = \?`).
		WithArgs("u1", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	app := &App{
		mediaUpload: &MediaUploadService{db: db, serverPort: 8080},
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local:8080/api/getUserSentImages?fromUserId=u1&toUserId=u2&page=1&pageSize=20", nil)
	rr := httptest.NewRecorder()
	app.handleGetUserSentImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	list, _ := data["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("list len=%d, want 1", len(list))
	}
	item, _ := list[0].(map[string]any)
	if got, _ := item["toUserId"].(string); got != "u2" {
		t.Fatalf("toUserId=%q, want %q", got, "u2")
	}
	if got, _ := item["sendTime"].(string); got != sendTime.Format("2006-01-02 15:04:05") {
		t.Fatalf("sendTime=%q, want %q", got, sendTime.Format("2006-01-02 15:04:05"))
	}
	if got, _ := item["remoteUrl"].(string); got != "http://api.local:8080/upload/images/2026/01/10/x.png" {
		t.Fatalf("remoteUrl=%q, want %q", got, "http://api.local:8080/upload/images/2026/01/10/x.png")
	}
}

func TestHandleGetUserUploadStats_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(123))

	app := &App{mediaUpload: &MediaUploadService{db: db}}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/getUserUploadStats?userId=u1", nil)
	rr := httptest.NewRecorder()
	app.handleGetUserUploadStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["totalCount"].(float64); int(got) != 123 {
		t.Fatalf("totalCount=%v, want 123", data["totalCount"])
	}
}

func TestHandleGetChatImages_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT local_path\s+FROM media_send_log\s+WHERE \(\(user_id = \? AND to_user_id = \?\) OR \(user_id = \? AND to_user_id = \?\)\)\s+GROUP BY local_path\s+ORDER BY MAX\(send_time\) DESC\s+LIMIT \?`).
		WithArgs("u1", "u2", "u2", "u1", 2).
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow("/images/2026/01/10/x.png"))

	app := &App{
		mediaUpload: &MediaUploadService{db: db, serverPort: 8080},
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local:8080/api/getChatImages?userId1=u1&userId2=u2&limit=2", nil)
	rr := httptest.NewRecorder()
	app.handleGetChatImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var out []string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(out) != 1 || out[0] != "http://api.local:8080/upload/images/2026/01/10/x.png" {
		t.Fatalf("out=%v", out)
	}
}

func TestHandleReuploadHistoryImage_SuccessWritesCache(t *testing.T) {
	tempDir := t.TempDir()

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"OK","msg":"abc.jpg"}`))
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url failed: %v", err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}

	localPath := "/images/2026/01/10/x.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("bytes"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	fileStore := &FileStorageService{db: db, baseUploadAbs: tempDir}
	imageSrv := NewImageServerService(host, port)
	mediaUpload := NewMediaUploadService(db, 8080, fileStore, imageSrv, upstream.Client())

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_path = \?\s+AND user_id = \?\s+ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"orig.png",
			"x.png",
			"remote.png",
			"http://remote",
			localPath,
			int64(4),
			"image/png",
			"png",
			sql.NullString{String: "md5", Valid: true},
			time.Now(),
			sql.NullTime{Valid: false},
		))

	mock.ExpectExec(`UPDATE media_file SET update_time = CURRENT_TIMESTAMP WHERE local_path = \?`).
		WithArgs(localPath).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE media_file SET update_time = CURRENT_TIMESTAMP WHERE local_path = \?`).
		WithArgs(strings.TrimPrefix(localPath, "/")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	app := &App{
		imageCache:  NewImageCacheService(),
		mediaUpload: mediaUpload,
	}

	form := url.Values{}
	form.Set("userId", "u1")
	form.Set("localPath", localPath)
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/reuploadHistoryImage", form)

	rr := httptest.NewRecorder()
	app.handleReuploadHistoryImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != `{"state":"OK","msg":"abc.jpg"}` {
		t.Fatalf("body=%q", rr.Body.String())
	}

	cached := app.imageCache.GetCachedImages("u1")
	if cached == nil || len(cached.ImageURLs) != 1 || cached.ImageURLs[0] != localPath {
		t.Fatalf("cache=%+v, want single localPath", cached)
	}
}

func TestHandleReuploadHistoryImage_FileMissing(t *testing.T) {
	app := &App{
		imageCache:  NewImageCacheService(),
		mediaUpload: &MediaUploadService{fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}},
	}

	form := url.Values{}
	form.Set("userId", "u1")
	form.Set("localPath", "/images/2026/01/10/missing.png")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/reuploadHistoryImage", form)

	rr := httptest.NewRecorder()
	app.handleReuploadHistoryImage(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), `"state":"ERROR"`) {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestHandleDeleteMedia_Success(t *testing.T) {
	tempDir := t.TempDir()

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	fileStore := &FileStorageService{db: db, baseUploadAbs: tempDir}
	svc := &MediaUploadService{db: db, fileStore: fileStore}
	app := &App{mediaUpload: svc}

	localPath := "/images/2026/01/10/x.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_path = \?\s+ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"orig.png",
			"x.png",
			"remote.png",
			"http://remote",
			localPath,
			int64(4),
			"image/png",
			"png",
			sql.NullString{String: "md5", Valid: true},
			uploadTime,
			sql.NullTime{Time: uploadTime, Valid: true},
		))

	for i := 0; i < 4; i++ {
		mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < 4; i++ {
		mock.ExpectExec(`DELETE FROM media_file WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file WHERE file_md5 = \?`).
		WithArgs("md5").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	form := url.Values{}
	form.Set("localPath", "http://example.com/upload"+localPath+"?x=1")
	form.Set("userId", "u1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/deleteMedia", form)

	rr := httptest.NewRecorder()
	app.handleDeleteMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); int(got) != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["fileDeleted"].(bool); !got {
		t.Fatalf("fileDeleted=%v, want true", data["fileDeleted"])
	}
	if _, err := os.Stat(full); err == nil {
		t.Fatalf("expected file deleted: %s", full)
	}
}

func TestHandleDeleteMedia_Forbidden(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{fileStore: &FileStorageService{}}}

	form := url.Values{}
	form.Set("localPath", "")
	form.Set("userId", "u1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/deleteMedia", form)

	rr := httptest.NewRecorder()
	app.handleDeleteMedia(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestHandleDeleteMedia_InternalError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{mediaUpload: &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}}

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_path = \?\s+ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(context.DeadlineExceeded)

	form := url.Values{}
	form.Set("localPath", "/images/2026/01/10/x.png")
	form.Set("userId", "u1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/deleteMedia", form)

	rr := httptest.NewRecorder()
	app.handleDeleteMedia(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleBatchDeleteMedia_Validation(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/batchDeleteMedia", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	app.handleBatchDeleteMedia(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "http://example.com/api/batchDeleteMedia", strings.NewReader(`{"userId":"","localPaths":["/a"]}`))
	req.Header.Set("Content-Type", "application/json")
	app.handleBatchDeleteMedia(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty userId status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "http://example.com/api/batchDeleteMedia", strings.NewReader(`{"userId":"u1","localPaths":[]}`))
	req.Header.Set("Content-Type", "application/json")
	app.handleBatchDeleteMedia(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty localPaths status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleBatchDeleteMedia_PartialFail(t *testing.T) {
	tempDir := t.TempDir()

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	fileStore := &FileStorageService{db: db, baseUploadAbs: tempDir}
	svc := &MediaUploadService{db: db, fileStore: fileStore}
	app := &App{mediaUpload: svc}

	localPath := "/images/2026/01/10/x.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_path = \?\s+ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"orig.png",
			"x.png",
			"remote.png",
			"http://remote",
			localPath,
			int64(4),
			"image/png",
			"png",
			sql.NullString{String: "md5", Valid: true},
			uploadTime,
			sql.NullTime{Time: uploadTime, Valid: true},
		))

	for i := 0; i < 4; i++ {
		mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < 4; i++ {
		mock.ExpectExec(`DELETE FROM media_file WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file WHERE file_md5 = \?`).
		WithArgs("md5").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	body := bytes.NewBufferString(`{"userId":"u1","localPaths":["` + localPath + `",""]}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/batchDeleteMedia", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	app.handleBatchDeleteMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); int(got) != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["successCount"].(float64); int(got) != 1 {
		t.Fatalf("successCount=%v, want 1", data["successCount"])
	}
	if got, _ := data["failCount"].(float64); int(got) != 1 {
		t.Fatalf("failCount=%v, want 1", data["failCount"])
	}
}

func TestHandleRecordImageSend_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.Local)
	fileRows := sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	}).AddRow(
		int64(1),
		"u1",
		"orig.png",
		"x.png",
		"remote.png",
		"http://remote",
		"/images/2026/01/10/x.png",
		int64(4),
		"image/png",
		"png",
		sql.NullString{String: "md5", Valid: true},
		uploadTime,
		sql.NullTime{Valid: false},
	)

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_filename = \?\s+AND user_id = \?\s+ORDER BY id LIMIT 1`).
		WithArgs("x.png", "u1").
		WillReturnRows(fileRows)

	// send log 不存在
	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log WHERE remote_url = \? AND user_id = \? AND to_user_id = \? ORDER BY id LIMIT 1`).
		WithArgs("http://remote/upload/abc.jpg", "u1", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "remote_url"}))

	mock.ExpectExec(`INSERT INTO media_send_log \(user_id, to_user_id, local_path, remote_url, send_time, created_at\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WithArgs("u1", "u2", "/images/2026/01/10/x.png", "http://remote/upload/abc.jpg", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	app := &App{
		mediaUpload: &MediaUploadService{db: db},
	}

	form := url.Values{}
	form.Set("remoteUrl", "http://remote/upload/abc.jpg")
	form.Set("fromUserId", "u1")
	form.Set("toUserId", "u2")
	form.Set("localFilename", "x.png")

	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/recordImageSend", form)
	rr := httptest.NewRecorder()
	app.handleRecordImageSend(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["success"].(bool); !got {
		t.Fatalf("success=%v, want true", resp["success"])
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["toUserId"].(string); got != "u2" {
		t.Fatalf("toUserId=%q, want %q", got, "u2")
	}
	if got, _ := data["remoteUrl"].(string); got != "http://remote/upload/abc.jpg" {
		t.Fatalf("remoteUrl=%q, want %q", got, "http://remote/upload/abc.jpg")
	}
	sendTime, _ := data["sendTime"].(string)
	if strings.TrimSpace(sendTime) == "" {
		t.Fatalf("sendTime empty")
	}
}

func TestHandleRecordImageSend_NotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// localFilename 找不到 -> 返回 nil
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_filename = \?\s+AND user_id = \?\s+ORDER BY id LIMIT 1`).
		WithArgs("x.png", "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE local_filename = \?\s+ORDER BY id LIMIT 1`).
		WithArgs("x.png").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))

	app := &App{
		mediaUpload: &MediaUploadService{db: db},
	}

	form := url.Values{}
	form.Set("remoteUrl", "")
	form.Set("fromUserId", "u1")
	form.Set("toUserId", "u2")
	form.Set("localFilename", "x.png")

	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/recordImageSend", form)
	rr := httptest.NewRecorder()
	app.handleRecordImageSend(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["success"].(bool); got {
		t.Fatalf("success=%v, want false", resp["success"])
	}
}
