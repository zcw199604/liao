package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleRepairVideoPosters_UncoveredBranches(t *testing.T) {
	t.Run("uninitialized services", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairVideoPosters", nil)
		(&App{}).handleRepairVideoPosters(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("service returns error", func(t *testing.T) {
		app := &App{
			mediaUpload: NewMediaUploadService(nil, 0, nil, nil, nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
		}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairVideoPosters", nil)
		app.handleRepairVideoPosters(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

