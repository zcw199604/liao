package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_FindMediaFileByLocalPath_Empty(t *testing.T) {
	svc := &MediaUploadService{}
	got, err := svc.findMediaFileByLocalPath(context.Background(), " ", "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMediaUploadService_FindMediaFileByLocalPath_QueryOnly(t *testing.T) {
	svc := &MediaUploadService{}
	got, err := svc.findMediaFileByLocalPath(context.Background(), "?x=1", "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMediaUploadService_FindMediaFileByLocalPath_PrefixUploadSlash_NoHit(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
	}

	svc := &MediaUploadService{db: db}
	got, err := svc.findMediaFileByLocalPath(context.Background(), "/upload/images/x.png", "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMediaUploadService_FindMediaFileByLocalPath_PrefixUploadNoSlash_NoHit(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
	}

	svc := &MediaUploadService{db: db}
	got, err := svc.findMediaFileByLocalPath(context.Background(), "upload/images/x.png", "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMediaUploadService_UpdateTimeByLocalPathIgnoreUser_ExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE local_path = \?`).
		WithArgs(sqlmock.AnyArg(), "/images/x.png").
		WillReturnError(errors.New("exec fail"))

	svc := &MediaUploadService{db: db}
	if got := svc.updateTimeByLocalPathIgnoreUser(context.Background(), "/images/x.png", time.Now()); got != 0 {
		t.Fatalf("got=%d, want 0", got)
	}
}

func TestMediaUploadService_ReuploadLocalFile_UsesBaseFilenameAndNewRequestError(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "images", "x.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// findMediaFileByLocalPath：3 candidates miss（order nondeterministic）
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*AND user_id = \?.*LIMIT 1`).
			WithArgs(sqlmock.AnyArg(), "bad\nuser").
			WillReturnError(sql.ErrNoRows)
	}

	svc := &MediaUploadService{
		db:         db,
		fileStore:  fileStore,
		imageSrv:   NewImageServerService("127.0.0.1", "9003"),
		httpClient: &http.Client{},
	}

	if _, err := svc.ReuploadLocalFile(context.Background(), "bad\nuser", "/images/x.png", "", "r", "ua"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_ReuploadLocalFile_DoError(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "images", "x.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "orig.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	svc := &MediaUploadService{
		db:         db,
		fileStore:  fileStore,
		imageSrv:   NewImageServerService("127.0.0.1", "9003"),
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("net down") })},
	}

	if _, err := svc.ReuploadLocalFile(context.Background(), "u1", "/images/x.png", "", "r", "ua"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_ReuploadLocalFile_ReadAllError(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "images", "x.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "orig.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	svc := &MediaUploadService{
		db:        db,
		fileStore: fileStore,
		imageSrv:  NewImageServerService("127.0.0.1", "9003"),
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       errReadCloser{},
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	if _, err := svc.ReuploadLocalFile(context.Background(), "u1", "/images/x.png", "", "r", "ua"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_ReuploadLocalFile_StatusNotOK(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "images", "x.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "orig.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	svc := &MediaUploadService{
		db:        db,
		fileStore: fileStore,
		imageSrv:  NewImageServerService("127.0.0.1", "9003"),
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       io.NopCloser(strings.NewReader("x")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	if _, err := svc.ReuploadLocalFile(context.Background(), "u1", "/images/x.png", "", "r", "ua"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_ReuploadLocalFile_AltPathWithoutLeadingSlash(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "images", "x.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// findMediaFileByLocalPath：3 candidates miss
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
	}

	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE local_path = \?`).
		WithArgs(sqlmock.AnyArg(), "images/x.png").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE local_path = \?`).
		WithArgs(sqlmock.AnyArg(), "/images/x.png").
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &MediaUploadService{
		db:        db,
		fileStore: fileStore,
		imageSrv:  NewImageServerService("127.0.0.1", "9003"),
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("OK")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	got, err := svc.ReuploadLocalFile(context.Background(), "", "images/x.png", "", "r", "ua")
	if err != nil || strings.TrimSpace(got) != "OK" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestMediaUploadService_DeleteMediaByPath_NotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
	}

	svc := &MediaUploadService{db: db}
	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png"); err != ErrDeleteForbidden {
		t.Fatalf("err=%v, want %v", err, ErrDeleteForbidden)
	}
}

func TestMediaUploadService_DeleteMediaByPath_FileMD5Empty_CoversCandidates(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}

	full := filepath.Join(tempDir, "images", "x.png")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", "images/x.png?x=1",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
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

	svc := &MediaUploadService{db: db, fileStore: fileStore}
	result, err := svc.DeleteMediaByPath(context.Background(), "u1", "/upload/images/x.png?x=1")
	if err != nil {
		t.Fatalf("DeleteMediaByPath: %v", err)
	}
	if !result.FileDeleted {
		t.Fatalf("result=%+v", result)
	}
	if _, statErr := os.Stat(full); statErr == nil {
		t.Fatalf("expected deleted: %s", full)
	}
}

func TestMediaUploadService_DeleteMediaByPath_CleanEmptyFromDBRow(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", "",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png"); err != ErrDeleteForbidden {
		t.Fatalf("err=%v, want %v", err, ErrDeleteForbidden)
	}
}

func TestMediaUploadService_DeleteMediaByPath_StoredLocalPathOnlyQuery_ReturnsForbidden(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", " ?x=1",
			int64(1), "image/png", "png", sql.NullString{Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png"); err != ErrDeleteForbidden {
		t.Fatalf("err=%v, want %v", err, ErrDeleteForbidden)
	}
}

func TestMediaUploadService_DeleteMediaByPath_DeleteSendLogError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "md5", Valid: true}, uploadTime, sql.NullTime{Valid: false},
		))

	mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(errors.New("delete fail"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_DeleteMediaByPath_DeleteMediaFileError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "md5", Valid: true}, uploadTime, sql.NullTime{Valid: false},
		))

	for i := 0; i < 4; i++ {
		mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	mock.ExpectExec(`DELETE FROM media_file WHERE local_path = \?`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(errors.New("delete fail"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_DeleteMediaByPath_FileMD5CountQueryError_DoesNotDeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	full := filepath.Join(tempDir, "images", "x.png")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.png", "l.png", "r.png", "http://x", "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "md5", Valid: true}, uploadTime, sql.NullTime{Valid: false},
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
		WillReturnError(errors.New("count fail"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: tempDir}}
	got, err := svc.DeleteMediaByPath(context.Background(), "u1", "/images/x.png")
	if err != nil {
		t.Fatalf("DeleteMediaByPath: %v", err)
	}
	if got.FileDeleted {
		t.Fatalf("expected FileDeleted=false, got=%+v", got)
	}
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("expected file still exists, err=%v", err)
	}
}
