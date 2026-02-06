package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_SaveDouyinUploadRecord_MD5HitUpdatesTime(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
	now := time.Now()

	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = \?.*LIMIT 1`).
		WithArgs("m1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(7),
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

	mock.ExpectExec(`UPDATE douyin_media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := svc.SaveDouyinUploadRecord(context.Background(), DouyinUploadRecord{
		UserID:    "u1",
		FileMD5:   "m1",
		LocalPath: "/images/x.jpg",
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil || got.ID != 7 || got.FileMD5 != "m1" || got.UpdateTime == "" {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_SaveDouyinUploadRecord_InsertWhenMD5Missing(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)

	expectInsertReturningID(mock, `INSERT INTO douyin_media_file`, 9,
		"u2",
		nil,
		nil,
		"orig.mp4",
		"l.mp4",
		"r.mp4",
		"http://remote/v1.mp4",
		"/videos/v1.mp4",
		int64(100),
		"video/mp4",
		"mp4",
		nil,
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	)

	got, err := svc.SaveDouyinUploadRecord(context.Background(), DouyinUploadRecord{
		UserID:           "u2",
		OriginalFilename: "orig.mp4",
		LocalFilename:    "l.mp4",
		RemoteFilename:   "r.mp4",
		RemoteURL:        "http://remote/v1.mp4",
		LocalPath:        "/videos/v1.mp4",
		FileSize:         100,
		FileType:         "video/mp4",
		FileExtension:    "mp4",
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil || got.ID != 9 {
		t.Fatalf("got=%+v", got)
	}
	if got.FileMD5 != "" {
		t.Fatalf("fileMD5=%q", got.FileMD5)
	}
}

func TestMediaUploadService_FindStoredMediaByLocalFilename_LocalFirst(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
	now := time.Now()

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_filename = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("a.png", "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"orig.png",
			"a.png",
			"r.png",
			"http://remote/a.png",
			"/images/a.png",
			int64(1),
			"image/png",
			"png",
			sql.NullString{String: "m1", Valid: true},
			now,
			sql.NullTime{Time: now, Valid: true},
		))

	got, err := svc.findStoredMediaFileByLocalFilename(context.Background(), "a.png", "u1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil || got.Source != mediaFileSourceLocal || got.File == nil || got.File.LocalFilename != "a.png" {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_FindStoredMediaByRemoteURL_FallbackToDouyin(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
	now := time.Now()

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("http://remote/fallback.jpg", "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))

	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("http://remote/fallback.jpg", "u1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(8),
			"u1",
			"orig.jpg",
			"f.jpg",
			"remote.jpg",
			"http://remote/fallback.jpg",
			"/images/f.jpg",
			int64(8),
			"image/jpeg",
			"jpg",
			sql.NullString{String: "m8", Valid: true},
			now,
			sql.NullTime{Time: now, Valid: true},
		))

	got, err := svc.findStoredMediaFileByRemoteURL(context.Background(), "http://remote/fallback.jpg", "u1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == nil || got.Source != mediaFileSourceDouyin || got.File == nil || got.File.ID != 8 {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_CountAnyMediaFileByMD5(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)

	if got, err := svc.countAnyMediaFileByMD5(context.Background(), " "); err != nil || got != 0 {
		t.Fatalf("empty md5 got=%d err=%v", got, err)
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file WHERE file_md5 = \?`).
		WithArgs("m9").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM douyin_media_file WHERE file_md5 = \?`).
		WithArgs("m9").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(3))

	got, err := svc.countAnyMediaFileByMD5(context.Background(), "m9")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != 5 {
		t.Fatalf("got=%d", got)
	}
}
