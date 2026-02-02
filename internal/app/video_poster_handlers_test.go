package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestHandleRepairVideoPosters_InvalidJSON(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}, fileStorage: &FileStorageService{}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairVideoPosters", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairVideoPosters(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleRepairVideoPosters_UnknownField(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}, fileStorage: &FileStorageService{}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairVideoPosters", strings.NewReader(`{"unknown":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairVideoPosters(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleRepairVideoPosters_EmptyBody_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, local_path, file_type, file_extension\s+FROM media_file\s+WHERE .*id > \?\s+ORDER BY id ASC\s+LIMIT \?`).
		WithArgs(int64(0), 201).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}))

	app := &App{
		cfg:         config.Config{},
		fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload: &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}},
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairVideoPosters", strings.NewReader(""))
	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairVideoPosters(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["source"] != "local" {
		t.Fatalf("source=%v", out["source"])
	}
}
