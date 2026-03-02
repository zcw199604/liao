package app

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestDouyinMediaUpload_Finders_UncoveredBranches(t *testing.T) {
	t.Run("findDouyinMediaFileByLocalPath candidate empty after trim", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findDouyinMediaFileByLocalPath(context.Background(), "?x=1", "u1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByUserAndMD5 both miss", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE user_id = \? AND file_md5 = \?`).
			WithArgs("u1", "m1").
			WillReturnError(sql.ErrNoRows)

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByUserAndMD5(context.Background(), "u1", "m1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByLocalFilename fallback to douyin", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()

		mock.ExpectQuery(`FROM media_file\s*WHERE local_filename = \?`).
			WithArgs("a.jpg", "u1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE local_filename = \?`).
			WithArgs("a.jpg", "u1").
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(12), "u1", "o.jpg", "a.jpg", "r.jpg", "http://x", "/images/a.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m12", Valid: true}, now, sql.NullTime{Valid: false},
			))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByLocalFilename(context.Background(), "a.jpg", "u1")
		if err != nil || got == nil || got.Source != mediaFileSourceDouyin {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByRemoteURL both miss", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM media_file\s*WHERE remote_url = \?`).
			WithArgs("http://x", "u1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE remote_url = \?`).
			WithArgs("http://x", "u1").
			WillReturnError(sql.ErrNoRows)

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByRemoteURL(context.Background(), "http://x", "u1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByRemoteFilename fallback to douyin", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()

		mock.ExpectQuery(`FROM media_file\s*WHERE remote_filename = \?`).
			WithArgs("r.jpg", "u1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE remote_filename = \?`).
			WithArgs("r.jpg", "u1").
			WillReturnRows(mediaHistoryRows().AddRow(
				int64(13), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/a.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: "m13", Valid: true}, now, sql.NullTime{Valid: false},
			))

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByRemoteFilename(context.Background(), "r.jpg", "u1")
		if err != nil || got == nil || got.Source != mediaFileSourceDouyin {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("findStoredMediaFileByRemoteFilename both miss", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM media_file\s*WHERE remote_filename = \?`).
			WithArgs("r.jpg", "u1").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`FROM douyin_media_file\s*WHERE remote_filename = \?`).
			WithArgs("r.jpg", "u1").
			WillReturnError(sql.ErrNoRows)

		svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
		got, err := svc.findStoredMediaFileByRemoteFilename(context.Background(), "r.jpg", "u1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})
}
