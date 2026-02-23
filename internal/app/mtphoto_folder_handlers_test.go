package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMtPhotoFolderHandlers_RootAndContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/folders/root":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path": "/",
				"folderList": []map[string]any{
					{"id": 518, "name": "photo", "subFolderNum": 1, "subFileNum": 0, "fileType": "folder"},
				},
				"fileList": []any{},
			})
			return
		case "/gateway/foldersV2/518":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path": "/photo",
				"folderList": []map[string]any{
					{"id": 644, "name": "我的照片", "subFolderNum": 0, "subFileNum": 2, "fileType": "folder"},
				},
				"fileList": []map[string]any{
					{"id": 1, "fileName": "a.jpg", "fileType": "JPEG", "size": "1", "tokenAt": "t", "MD5": "m1", "status": 2},
					{"id": 2, "fileName": "b.jpg", "fileType": "JPEG", "size": "2", "tokenAt": "t", "MD5": "m2", "status": 2},
				},
				"trashNum": 0,
			})
			return
		case "/gateway/foldersV2/403":
			w.WriteHeader(http.StatusForbidden)
			return
		case "/gateway/foldersV2/404":
			w.WriteHeader(http.StatusNotFound)
			return
		case "/gateway/folderBreadcrumbs/518":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path": "/photo",
				"folderList": []map[string]any{
					{"id": 644, "name": "我的照片", "subFolderNum": 0, "subFileNum": 2, "fileType": "folder"},
				},
				"fileList": []any{},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	app := &App{mtPhoto: NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())}

	t.Run("root not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderRoot", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoFolderRoot(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("root ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderRoot", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderRoot(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), `"folderList"`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("content invalid folderId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderContent?folderId=0", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("content ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderContent?folderId=518&page=1&pageSize=1", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), `"total":2`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("content forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderContent?folderId=403&page=1&pageSize=60", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("content not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderContent?folderId=404&page=1&pageSize=60", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("breadcrumbs invalid folderId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderBreadcrumbs?folderId=0", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderBreadcrumbs(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("breadcrumbs ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderBreadcrumbs?folderId=518", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderBreadcrumbs(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	})
}
