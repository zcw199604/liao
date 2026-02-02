package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_SaveUploadRecord_FindExistingError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "md5").
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.SaveUploadRecord(context.Background(), UploadRecord{UserID: "u1", FileMD5: "md5"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_SaveUploadRecord_UpdateExisting_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "md5").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"o.png",
			"l.png",
			"r.png",
			"http://remote",
			"/images/x.png",
			int64(1),
			"image/png",
			"png",
			sql.NullString{String: "md5", Valid: true},
			uploadTime,
			sql.NullTime{Valid: false},
		))

	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	got, err := svc.SaveUploadRecord(context.Background(), UploadRecord{UserID: "u1", FileMD5: "md5"})
	if err != nil {
		t.Fatalf("SaveUploadRecord: %v", err)
	}
	if got == nil || got.ID != 1 || got.UpdateTime == "" {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_SaveUploadRecord_UpdateExisting_UpdateError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", "md5").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1),
			"u1",
			"o.png",
			"l.png",
			"r.png",
			"http://remote",
			"/images/x.png",
			int64(1),
			"image/png",
			"png",
			sql.NullString{String: "md5", Valid: true},
			uploadTime,
			sql.NullTime{Valid: false},
		))

	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(1)).
		WillReturnError(errors.New("update fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.SaveUploadRecord(context.Background(), UploadRecord{UserID: "u1", FileMD5: "md5"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_SaveUploadRecord_InsertError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	expectInsertReturningIDError(mock, `INSERT INTO media_file`, errors.New("insert fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.SaveUploadRecord(context.Background(), UploadRecord{UserID: "u1"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RecordImageSend_RemoteURL_GlobalFallback_InsertLog(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	remoteURL := "http://example.com/img/Upload/x.png"
	fromUserID := "u1"
	toUserID := "u2"

	uploadTime := time.Now()

	// remote_url 精确匹配：先 userId，再全局
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(remoteURL, fromUserID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*ORDER BY id LIMIT 1`).
		WithArgs(remoteURL).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(10), fromUserID, "o.png", "l.png", "r.png", remoteURL, "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "", Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	// send log miss
	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log`).
		WithArgs(remoteURL, fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "remote_url"}))

	mock.ExpectExec(`INSERT INTO media_send_log`).
		WithArgs(fromUserID, toUserID, "/images/x.png", remoteURL, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	got, err := svc.RecordImageSend(context.Background(), remoteURL, fromUserID, toUserID, "")
	if err != nil {
		t.Fatalf("RecordImageSend: %v", err)
	}
	if got == nil || got.ToUserID != toUserID || got.SendTime == "" || got.RemoteURL != remoteURL {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_RecordImageSend_RemoteFilename_ExistingLog(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	remoteURL := "http://example.com/img/Upload/2026/01/a.png"
	fromUserID := "u1"
	toUserID := "u2"

	uploadTime := time.Now()

	// remote_url 先 miss（userId + global），再走 remoteFilename
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(remoteURL, fromUserID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*ORDER BY id LIMIT 1`).
		WithArgs(remoteURL).
		WillReturnError(sql.ErrNoRows)

	remoteFilename := "2026/01/a.png"
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_filename = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(remoteFilename, fromUserID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_filename = \?.*ORDER BY id LIMIT 1`).
		WithArgs(remoteFilename).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(11), fromUserID, "o.png", "l.png", "a.png", remoteURL, "/images/a.png",
			int64(1), "image/png", "png", sql.NullString{String: "", Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log`).
		WithArgs(remoteURL, fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "remote_url"}).AddRow(int64(7), "http://cached"))

	mock.ExpectExec(`UPDATE media_send_log SET send_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	got, err := svc.RecordImageSend(context.Background(), remoteURL, fromUserID, toUserID, "")
	if err != nil {
		t.Fatalf("RecordImageSend: %v", err)
	}
	if got == nil || got.RemoteURL != "http://cached" {
		t.Fatalf("got=%+v", got)
	}
}

func TestMediaUploadService_RecordImageSend_ExistingLog_UpdateError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	remoteURL := "http://example.com/img/Upload/x.png"
	fromUserID := "u1"
	toUserID := "u2"

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_filename = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("l.png", fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(12), fromUserID, "o.png", "l.png", "r.png", remoteURL, "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "", Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log`).
		WithArgs(remoteURL, fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "remote_url"}).AddRow(int64(8), "http://cached"))

	mock.ExpectExec(`UPDATE media_send_log SET send_time = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), int64(8)).
		WillReturnError(errors.New("update fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.RecordImageSend(context.Background(), remoteURL, fromUserID, toUserID, "l.png"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RecordImageSend_Basename_InsertError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	remoteURL := "http://example.com/x.png"
	fromUserID := "u1"
	toUserID := "u2"

	// remote_url miss（userId + global）
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(remoteURL, fromUserID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*ORDER BY id LIMIT 1`).
		WithArgs(remoteURL).
		WillReturnError(sql.ErrNoRows)

	// basename -> remote_filename miss(user) then hit(global)
	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_filename = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("x.png", fromUserID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_filename = \?.*ORDER BY id LIMIT 1`).
		WithArgs("x.png").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(13), fromUserID, "o.png", "l.png", "x.png", remoteURL, "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "", Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log`).
		WithArgs(remoteURL, fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "remote_url"}))

	mock.ExpectExec(`INSERT INTO media_send_log`).
		WillReturnError(errors.New("insert fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.RecordImageSend(context.Background(), remoteURL, fromUserID, toUserID, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RecordImageSend_FindSendLogError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	remoteURL := "http://example.com/x.png"
	fromUserID := "u1"
	toUserID := "u2"

	uploadTime := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE remote_url = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs(remoteURL, fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(14), fromUserID, "o.png", "l.png", "x.png", remoteURL, "/images/x.png",
			int64(1), "image/png", "png", sql.NullString{String: "", Valid: false}, uploadTime, sql.NullTime{Valid: false},
		))

	mock.ExpectQuery(`SELECT id, remote_url FROM media_send_log`).
		WithArgs(remoteURL, fromUserID, toUserID).
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.RecordImageSend(context.Background(), remoteURL, fromUserID, toUserID, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RecordImageSend_NoMatch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// local_filename miss（userId + global）
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_filename = \?.*AND user_id = \?.*LIMIT 1`).
		WithArgs("l.png", "u1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_filename = \?.*ORDER BY id LIMIT 1`).
		WithArgs("l.png").
		WillReturnError(sql.ErrNoRows)

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	got, err := svc.RecordImageSend(context.Background(), "", "u1", "u2", "l.png")
	if err != nil {
		t.Fatalf("RecordImageSend: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMediaUploadService_ConvertPathsToLocalURLs_Empty(t *testing.T) {
	svc := &MediaUploadService{serverPort: 8080}
	if got := svc.ConvertPathsToLocalURLs(nil, ""); got == nil || len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}

func TestParseJSONStateOK_Branches(t *testing.T) {
	if parseJSONStateOK("not-json") {
		t.Fatalf("expected false")
	}
	if parseJSONStateOK(`{"state":"NO"}`) {
		t.Fatalf("expected false")
	}
	if !parseJSONStateOK(`{"state":"OK"}`) {
		t.Fatalf("expected true")
	}
}

func TestInferTypeFromExtension_Branches(t *testing.T) {
	if got := inferTypeFromExtension("jpg"); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := inferTypeFromExtension(".mp4"); got != "video" {
		t.Fatalf("got=%q", got)
	}
	if got := inferTypeFromExtension("unknown"); got != "file" {
		t.Fatalf("got=%q", got)
	}
}

func TestNormalizeUploadLocalPathInput_EmptyAfterTrim(t *testing.T) {
	if got := normalizeUploadLocalPathInput("?x=1"); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}
	if got := normalizeUploadLocalPathInput("#x"); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}
}

func TestMediaUploadService_ReuploadLocalFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tempDir}
	svc := &MediaUploadService{fileStore: fileStore}

	if err := os.WriteFile(filepath.Join(tempDir, "empty.bin"), nil, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := svc.ReuploadLocalFile(context.Background(), "u1", "/empty.bin", "", "", ""); err == nil {
		t.Fatalf("expected error")
	}
}
