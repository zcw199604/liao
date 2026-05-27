package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_RepairMediaDimensions_DryRunAndCommit(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "images", "a.png"), buildTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, &FileStorageService{baseUploadAbs: root}, nil, nil)

	mock.ExpectQuery(`(?s)FROM media_file.*media_width.*ORDER BY id ASC.*LIMIT \?`).
		WithArgs(int64(0), 3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), "/images/a.png", "image/png", "png").
			AddRow(int64(2), "/images/missing.png", "image/png", "png").
			AddRow(int64(3), "/images/a.png", "video/mp4", "mp4"))

	res, err := svc.RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{Limit: 2})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !res.HasMore || res.Scanned != 2 || res.NeedUpdate != 1 || res.FileMissing != 1 || res.Updated != 0 {
		t.Fatalf("dry run result=%+v", res)
	}

	mock.ExpectQuery(`(?s)FROM media_file.*media_width.*ORDER BY id ASC.*LIMIT \?`).
		WithArgs(int64(0), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), "/images/a.png", "image/png", "png"))
	mock.ExpectExec(`UPDATE media_file SET media_width = \?, media_height = \? WHERE id = \?`).
		WithArgs(1, 1, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	res, err = svc.RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{Commit: true, Limit: 1})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Scanned != 1 || res.NeedUpdate != 1 || res.Updated != 1 {
		t.Fatalf("commit result=%+v", res)
	}
}

func TestMediaUploadService_RepairMediaDimensions_InvalidPathDecodeUnsupportedAndDouyin(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "images", "bad.png"), []byte("not-image"), 0o644); err != nil {
		t.Fatalf("write bad image: %v", err)
	}

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, &FileStorageService{baseUploadAbs: root}, nil, nil)

	mock.ExpectQuery(`(?s)FROM douyin_media_file.*ORDER BY id ASC.*LIMIT \?`).
		WithArgs(int64(10), 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(11), "", "image/png", "png").
			AddRow(int64(12), "/images/bad.png", "image/png", "png").
			AddRow(int64(13), "/videos/a.mp4", "video/mp4", "mp4"))

	res, err := svc.RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{
		Source:       "douyin",
		StartAfterID: 10,
		Limit:        4,
		Force:        true,
	})
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if res.Source != "douyin" || !res.Force || res.InvalidPath != 1 || res.DecodeFailed != 1 || res.Unsupported != 1 {
		t.Fatalf("result=%+v", res)
	}
}

func TestHandleRepairMediaDimensions(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		app := &App{mediaUpload: &MediaUploadService{}, fileStorage: &FileStorageService{}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaDimensions", bytes.NewBufferString("{"))
		rr := httptest.NewRecorder()
		app.handleRepairMediaDimensions(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("success empty body", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		root := t.TempDir()
		app := &App{
			mediaUpload: NewMediaUploadService(wrapMySQLDB(db), 0, &FileStorageService{baseUploadAbs: root}, nil, nil),
			fileStorage: &FileStorageService{baseUploadAbs: root},
		}

		mock.ExpectQuery(`(?s)FROM media_file.*media_width.*ORDER BY id ASC.*LIMIT \?`).
			WithArgs(int64(0), 201).
			WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}))

		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaDimensions", nil)
		rr := httptest.NewRecorder()
		app.handleRepairMediaDimensions(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var res RepairMediaDimensionsResult
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if res.Source != "local" || res.Limit != 200 {
			t.Fatalf("res=%+v", res)
		}
	})
}

func TestMediaUploadService_RepairMediaDimensionsValidation(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, &FileStorageService{baseUploadAbs: t.TempDir()}, nil, nil)
	if _, err := svc.RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{Source: "bad"}); err == nil {
		t.Fatalf("expected bad source error")
	}
	if _, err := svc.RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{Limit: -1}); err == nil {
		t.Fatalf("expected negative limit error")
	}
	if _, err := (*MediaUploadService)(nil).RepairMediaDimensions(context.Background(), RepairMediaDimensionsRequest{}); err == nil {
		t.Fatalf("expected nil service error")
	}

}
