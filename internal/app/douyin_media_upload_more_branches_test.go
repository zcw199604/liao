package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func mediaHistoryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
		"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
	})
}

func TestDouyinMediaUpload_SaveRecord_MoreErrorBranches(t *testing.T) {
	t.Run("update existing exec error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("m1").
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(1), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m1", Valid: true}, now, sql.NullTime{Valid: false},
			))
		mock.ExpectExec(`UPDATE douyin_media_file`).
			WithArgs(sqlmock.AnyArg(), "", "", "", "", int64(1)).
			WillReturnError(errors.New("update fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.SaveDouyinUploadRecord(context.Background(), DouyinUploadRecord{FileMD5: "m1"}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("insert error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT INTO douyin_media_file`).WillReturnError(errors.New("insert fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.SaveDouyinUploadRecord(context.Background(), DouyinUploadRecord{UserID: "u1"}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestDouyinMediaUpload_FindByLocalPathAndStoredWrappers_Branches(t *testing.T) {
	t.Run("findDouyinMediaFileByLocalPath empty", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()
		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findDouyinMediaFileByLocalPath(context.Background(), "  ", "u1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("findDouyinMediaFileByLocalPath /upload prefix returns found", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE local_path = \?\s*AND user_id = \?\s*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg(), "u1").
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(7), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m7", Valid: true}, now, sql.NullTime{Valid: false},
			))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findDouyinMediaFileByLocalPath(context.Background(), "/upload/images/x.jpg?x=1", "u1")
		if err != nil || got == nil || got.ID != 7 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("findDouyinMediaFileByLocalPath upload/ prefix no rows", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		// candidateSet order is non-deterministic; each query uses AnyArg.
		for i := 0; i < 3; i++ {
			mock.ExpectQuery(`FROM douyin_media_file\s*WHERE local_path = \?\s*AND user_id = \?\s*ORDER BY id LIMIT 1`).
				WithArgs(sqlmock.AnyArg(), "u1").
				WillReturnError(sql.ErrNoRows)
		}

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findDouyinMediaFileByLocalPath(context.Background(), "upload/images/x.jpg", "u1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByUserAndMD5 local error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByUserAndMD5(context.Background(), "u1", "m1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByUserAndMD5 local hit", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()
		mock.ExpectQuery(`FROM media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(3), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m1", Valid: true}, now, sql.NullTime{Valid: false},
			))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByUserAndMD5(context.Background(), "u1", "m1")
		if err != nil || got == nil || got.Source != mediaFileSourceLocal {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByUserAndMD5 douyin error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByUserAndMD5(context.Background(), "u1", "m1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByMD5 douyin error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE file_md5 = \?`).
			WithArgs("m1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE file_md5 = \?`).
			WithArgs("m1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByMD5(context.Background(), "m1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByLocalFilename local error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE local_filename = \?`).
			WithArgs("a.jpg", "u1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByLocalFilename(context.Background(), "a.jpg", "u1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByRemoteURL local error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE remote_url = \?`).
			WithArgs("http://x", "u1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByRemoteURL(context.Background(), "http://x", "u1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByRemoteFilename local error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*WHERE remote_filename = \?`).
			WithArgs("r.jpg", "u1").
			WillReturnError(errors.New("query fail"))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if _, err := svc.findStoredMediaFileByRemoteFilename(context.Background(), "r.jpg", "u1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findStoredMediaFileByLocalPath fallback to douyin", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()
		// local media candidates (map order): return no rows for first 3 attempts.
		for i := 0; i < 3; i++ {
			mock.ExpectQuery(`FROM media_file\s*WHERE local_path = \?\s*ORDER BY id LIMIT 1`).
				WithArgs(sqlmock.AnyArg()).
				WillReturnError(sql.ErrNoRows)
		}
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE local_path = \?\s*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(9), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m9", Valid: true}, now, sql.NullTime{Valid: false},
			))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByLocalPath(context.Background(), "/images/x.jpg", "")
		if err != nil || got == nil || got.Source != mediaFileSourceDouyin {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("updateTimeByStoredMediaFile nil stored", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()
		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		if err := svc.updateTimeByStoredMediaFile(context.Background(), nil, time.Now()); err != nil {
			t.Fatalf("err=%v", err)
		}
	})
}
