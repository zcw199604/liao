package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_findDouyinMediaFileByUserAndMD5(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "m1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"orig.jpg",
			"l.jpg",
			"r.jpg",
			"http://remote",
			"/images/x.jpg",
			int64(12),
			"image/jpeg",
			"jpg",
			sql.NullString{String: "m1", Valid: true},
			now,
			sql.NullTime{Time: now, Valid: true},
		))

	got, err := svc.findDouyinMediaFileByUserAndMD5(context.Background(), "u1", "m1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil {
		t.Fatalf("expected result")
	}
	if got.UserID != "u1" || got.FileMD5 != "m1" {
		t.Fatalf("got=%+v", *got)
	}
}

func TestMediaUploadService_findDouyinMediaFileBySecUserAndMD5(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE sec_user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("sec1", "m1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(2),
			"u1",
			"orig2.jpg",
			"l2.jpg",
			"r2.jpg",
			"http://remote2",
			"/images/y.jpg",
			int64(13),
			"image/jpeg",
			"jpg",
			sql.NullString{String: "m1", Valid: true},
			now,
			sql.NullTime{Time: now, Valid: true},
		))

	got, err := svc.findDouyinMediaFileBySecUserAndMD5(context.Background(), "sec1", "m1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil {
		t.Fatalf("expected result")
	}
	if got.FileMD5 != "m1" {
		t.Fatalf("got=%+v", *got)
	}
}

func TestMediaUploadService_findStoredMediaFileByUserAndMD5_prefersDouyinWhenLocalMissing(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "m1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "m1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(3),
			"u1",
			"orig3.jpg",
			"l3.jpg",
			"r3.jpg",
			"http://remote3",
			"/images/z.jpg",
			int64(14),
			"image/jpeg",
			"jpg",
			sql.NullString{String: "m1", Valid: true},
			now,
			sql.NullTime{Time: now, Valid: true},
		))

	got, err := svc.findStoredMediaFileByUserAndMD5(context.Background(), "u1", "m1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil || got.File == nil {
		t.Fatalf("expected stored file")
	}
	if got.Source != mediaFileSourceDouyin {
		t.Fatalf("Source=%q, want %q", got.Source, mediaFileSourceDouyin)
	}
	if got.File.UserID != "u1" || got.File.FileMD5 != "m1" {
		t.Fatalf("got=%+v", *got.File)
	}
}

