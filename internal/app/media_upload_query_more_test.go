package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_GetUserUploadHistory_PageLessThan1_UpdateTimeValid(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs("u1", 2, 0).
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
			nil,
			now,
			now,
		))

	svc := &MediaUploadService{db: db, serverPort: 8080}
	out, err := svc.GetUserUploadHistory(context.Background(), "u1", 0, 2, "")
	if err != nil {
		t.Fatalf("GetUserUploadHistory: %v", err)
	}
	if len(out) != 1 || out[0].RemoteURL != "http://localhost:8080/upload/images/x.png" || out[0].UpdateTime == "" {
		t.Fatalf("out=%v", out)
	}
}

func TestMediaUploadService_GetUserUploadHistory_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs("u1", 10, 0).
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetUserUploadHistory(context.Background(), "u1", 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetUserUploadHistory_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs("u1", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetUserUploadHistory(context.Background(), "u1", 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetUserSentImages_PageLessThan1_FileNil(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	fromUserID := "u1"
	toUserID := "u2"
	localPath := "images/x.png?x=1"
	sendTime := time.Now()

	mock.ExpectQuery(`(?s)FROM media_send_log.*WHERE user_id = \? AND to_user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs(fromUserID, toUserID, 1, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "remote_url", "send_time"}).
			AddRow(int64(1), localPath, "http://remote", sendTime))

	// findMediaFileByLocalPath: 3 candidates, all miss (order nondeterministic)
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*AND user_id = \?.*LIMIT 1`).
			WithArgs(sqlmock.AnyArg(), fromUserID).
			WillReturnError(sql.ErrNoRows)
	}

	svc := &MediaUploadService{db: db, serverPort: 8080}
	out, err := svc.GetUserSentImages(context.Background(), fromUserID, toUserID, 0, 1, "")
	if err != nil {
		t.Fatalf("GetUserSentImages: %v", err)
	}
	if len(out) != 1 || out[0].ToUserID != toUserID || out[0].SendTime == "" {
		t.Fatalf("out=%v", out)
	}
}

func TestMediaUploadService_GetUserSentImages_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_send_log.*WHERE user_id = \? AND to_user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs("u1", "u2", 10, 0).
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetUserSentImages(context.Background(), "u1", "u2", 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetUserSentImages_RowScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_send_log.*WHERE user_id = \? AND to_user_id = \?.*LIMIT \? OFFSET \?`).
		WithArgs("u1", "u2", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetUserSentImages(context.Background(), "u1", "u2", 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetChatImages_DefaultLimit(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_send_log.*GROUP BY local_path.*LIMIT \?`).
		WithArgs("u1", "u2", "u2", "u1", 20).
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow("images/x.png"))

	svc := &MediaUploadService{db: db, serverPort: 8080}
	out, err := svc.GetChatImages(context.Background(), "u1", "u2", 0, "")
	if err != nil {
		t.Fatalf("GetChatImages: %v", err)
	}
	if len(out) != 1 || out[0] != "http://localhost:8080/upload/images/x.png" {
		t.Fatalf("out=%v", out)
	}
}

func TestMediaUploadService_GetChatImages_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_send_log.*GROUP BY local_path.*LIMIT \?`).
		WithArgs("u1", "u2", "u2", "u1", 1).
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetChatImages(context.Background(), "u1", "u2", 1, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetChatImages_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_send_log.*GROUP BY local_path.*LIMIT \?`).
		WithArgs("u1", "u2", "u2", "u1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(nil))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetChatImages(context.Background(), "u1", "u2", 1, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetAllUploadImagesWithDetails_PageLessThan1_UpdateTimeValid(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`(?s)SELECT local_filename, original_filename, local_path.*FROM media_file.*LIMIT \? OFFSET \?`).
		WithArgs(1, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
		}).AddRow(
			"l.png",
			"o.png",
			"/images/x.png",
			int64(1),
			"image/png",
			"png",
			now,
			now,
		))

	svc := &MediaUploadService{db: db, serverPort: 8080}
	out, err := svc.GetAllUploadImagesWithDetails(context.Background(), 0, 1, "")
	if err != nil {
		t.Fatalf("GetAllUploadImagesWithDetails: %v", err)
	}
	if len(out) != 1 || out[0].UpdateTime == "" {
		t.Fatalf("out=%v", out)
	}
}

func TestMediaUploadService_GetAllUploadImagesWithDetails_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT local_filename, original_filename, local_path.*FROM media_file.*LIMIT \? OFFSET \?`).
		WithArgs(1, 0).
		WillReturnError(errors.New("query fail"))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetAllUploadImagesWithDetails(context.Background(), 1, 1, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_GetAllUploadImagesWithDetails_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT local_filename, original_filename, local_path.*FROM media_file.*LIMIT \? OFFSET \?`).
		WithArgs(1, 0).
		WillReturnRows(sqlmock.NewRows([]string{"local_filename"}).AddRow("x"))

	svc := &MediaUploadService{db: db}
	if _, err := svc.GetAllUploadImagesWithDetails(context.Background(), 1, 1, ""); err == nil {
		t.Fatalf("expected error")
	}
}
