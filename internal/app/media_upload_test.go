package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeUploadLocalPathInput(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plainNoSlash", in: "images/2026/01/10/a.png", want: "/images/2026/01/10/a.png"},
		{name: "leadingSlash", in: "/images/2026/01/10/a.png", want: "/images/2026/01/10/a.png"},
		{name: "uploadPrefixSlash", in: "/upload/images/2026/01/10/a.png", want: "/images/2026/01/10/a.png"},
		{name: "uploadPrefixNoSlash", in: "upload/images/2026/01/10/a.png", want: "/images/2026/01/10/a.png"},
		{name: "fullURL", in: "http://example.com/upload/images/2026/01/10/a.png", want: "/images/2026/01/10/a.png"},
		{name: "withQuery", in: "/upload/images/2026/01/10/a.png?x=1", want: "/images/2026/01/10/a.png"},
		{name: "percentEscaped", in: "%2Fimages%2F2026%2F01%2F10%2Fa.png", want: "/images/2026/01/10/a.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeUploadLocalPathInput(tc.in); got != tc.want {
				t.Fatalf("normalizeUploadLocalPathInput(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestConvertToLocalURL(t *testing.T) {
	svc := &MediaUploadService{serverPort: 8080}

	if got := svc.convertToLocalURL("", "example.com"); got != "" {
		t.Fatalf("convertToLocalURL(empty)=%q, want empty", got)
	}

	got := svc.convertToLocalURL("images/a.png", "")
	if got != "http://localhost:8080/upload/images/a.png" {
		t.Fatalf("got=%q", got)
	}

	got = svc.convertToLocalURL("/images/a.png", "example.com:123")
	if got != "http://example.com:123/upload/images/a.png" {
		t.Fatalf("got=%q", got)
	}

	out := svc.ConvertPathsToLocalURLs([]string{"images/a.png", ""}, "example.com:123")
	if len(out) != 1 || out[0] != "http://example.com:123/upload/images/a.png" {
		t.Fatalf("ConvertPathsToLocalURLs unexpected: %+v", out)
	}
}

func TestMediaUploadService_DeleteMediaByPath_Success(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	fileStore := &FileStorageService{baseUploadAbs: tempDir}
	svc := &MediaUploadService{db: db, fileStore: fileStore}

	localPath := "/images/2026/01/10/x.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
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

	// candidateSet 固定为“带/不带前导/”两种 + 追加两次，共 4 次删除
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

	got, err := svc.DeleteMediaByPath(context.Background(), "ignored", "http://example.com/upload"+localPath+"?x=1")
	if err != nil {
		t.Fatalf("DeleteMediaByPath failed: %v", err)
	}
	if got.DeletedRecords != 4 {
		t.Fatalf("DeletedRecords=%d, want 4", got.DeletedRecords)
	}
	if !got.FileDeleted {
		t.Fatalf("FileDeleted=false, want true")
	}
	if _, err := os.Stat(full); err == nil {
		t.Fatalf("expected file deleted: %s", full)
	}
}

func TestMediaUploadService_DeleteMediaByPath_FileMD5StillUsed_DoesNotDeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	fileStore := &FileStorageService{baseUploadAbs: tempDir}
	svc := &MediaUploadService{db: db, fileStore: fileStore}

	localPath := "/images/2026/01/10/x.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
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
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	got, err := svc.DeleteMediaByPath(context.Background(), "ignored", localPath)
	if err != nil {
		t.Fatalf("DeleteMediaByPath failed: %v", err)
	}
	if got.DeletedRecords != 4 {
		t.Fatalf("DeletedRecords=%d, want 4", got.DeletedRecords)
	}
	if got.FileDeleted {
		t.Fatalf("FileDeleted=true, want false")
	}
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("expected file still exists: %v", err)
	}
}

func TestMediaUploadService_DeleteMediaByPath_Forbidden(t *testing.T) {
	svc := &MediaUploadService{fileStore: &FileStorageService{}}

	if _, err := svc.DeleteMediaByPath(context.Background(), "u1", ""); err != ErrDeleteForbidden {
		t.Fatalf("expected ErrDeleteForbidden, got %v", err)
	}
}

func TestExtractFilenameFromURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "noSlash", in: "a.png", want: ""},
		{name: "trailingSlash", in: "http://example.com/a/", want: ""},
		{name: "simple", in: "http://example.com/a/b.png", want: "b.png"},
		{name: "withQuery", in: "http://example.com/a/b.png?x=1", want: "b.png?x=1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractFilenameFromURL(tc.in); got != tc.want {
				t.Fatalf("extractFilenameFromURL(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractRemoteFilenameFromURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "noMarker", in: "http://example.com/upload/a.png", want: ""},
		{name: "markerAtEnd", in: "http://example.com/img/Upload/", want: ""},
		{name: "simple", in: "http://example.com/img/Upload/2026/01/a.png", want: "2026/01/a.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractRemoteFilenameFromURL(tc.in); got != tc.want {
				t.Fatalf("extractRemoteFilenameFromURL(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMediaUploadService_FindByRemoteURLAndFilename(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	}).AddRow(
		int64(1),
		"u1",
		"orig.png",
		"l.png",
		"r.png",
		"http://remote",
		"/images/x.png",
		int64(1),
		"image/png",
		"png",
		sql.NullString{String: "md5", Valid: true},
		uploadTime,
		sql.NullTime{Time: uploadTime, Valid: true},
	)

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*ORDER BY id LIMIT 1`).
		WithArgs("http://remote", "u1").
		WillReturnRows(rows)

	svc := &MediaUploadService{db: db}
	got, err := svc.findMediaFileByRemoteURL(context.Background(), "http://remote", "u1")
	if err != nil || got == nil || got.ID != 1 {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	rows2 := sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	}).AddRow(
		int64(2),
		"u2",
		"orig.png",
		"l.png",
		"r.png",
		"http://remote2",
		"/images/y.png",
		int64(1),
		"image/png",
		"png",
		sql.NullString{String: "", Valid: false},
		uploadTime,
		sql.NullTime{Time: uploadTime, Valid: true},
	)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_filename = \?.*ORDER BY id LIMIT 1`).
		WithArgs("r.png").
		WillReturnRows(rows2)

	got2, err := svc.findMediaFileByRemoteFilename(context.Background(), "r.png", "")
	if err != nil || got2 == nil || got2.ID != 2 {
		t.Fatalf("got=%+v err=%v", got2, err)
	}
}
