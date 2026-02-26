package app

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestWriteMtPhotoFolderError_Branches(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, nil)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, &mtPhotoStatusError{StatusCode: http.StatusUnauthorized, Status: "401 Unauthorized", Action: "x"})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, &mtPhotoStatusError{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Action: "x"})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, &mtPhotoStatusError{StatusCode: http.StatusNotFound, Status: "404 Not Found", Action: "x"})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("status error default", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, &mtPhotoStatusError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Action: "x"})
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("normal error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeMtPhotoFolderError(rr, context.Canceled)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})
}

func TestMtPhotoFolderHandlers_Branches(t *testing.T) {
	var timelineRequests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gateway/folders/root":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":       "/",
				"folderList": []map[string]any{{"id": 1, "name": "root"}},
				"fileList":   []map[string]any{},
				"trashNum":   0,
			})
			return
		case "/gateway/foldersV2/1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":       "/a",
				"folderList": []map[string]any{},
				"fileList": []map[string]any{{
					"id":       11,
					"fileType": "JPEG",
					"MD5":      "m1",
				}},
				"trashNum": 0,
			})
			return
		case "/gateway/folderFiles/1":
			timelineRequests++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{{
					"day": "2026-01-01",
					"list": []map[string]any{{
						"id":       11,
						"fileType": "JPEG",
						"MD5":      "m1",
					}},
				}},
				"totalCount": 1,
			})
			return
		case "/gateway/folderBreadcrumbs/1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":       "/a",
				"folderList": []map[string]any{{"id": 1, "name": "a"}},
				"fileList":   []map[string]any{},
				"trashNum":   0,
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	mt := newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client())
	app := &App{mtPhoto: mt}

	t.Run("root not init", func(t *testing.T) {
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoFolderRoot(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderRoot", nil))
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("root error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoFolderRoot(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderRoot", nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("root ok", func(t *testing.T) {
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderRoot(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderRoot", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("content not init", func(t *testing.T) {
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoFolderContent(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderContent?folderId=1", nil))
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("content folder id invalid", func(t *testing.T) {
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderContent?folderId=0", nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("content error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoFolderContent(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderContent?folderId=1", nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("content normalize page/pageSize + includeTimeline false", func(t *testing.T) {
		before := timelineRequests
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderContent?folderId=1&page=0&pageSize=0&includeTimeline=false", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if int(got["page"].(float64)) != 1 {
			t.Fatalf("page=%v", got["page"])
		}
		if int(got["pageSize"].(float64)) != 60 {
			t.Fatalf("pageSize=%v", got["pageSize"])
		}
		if timelineRequests != before {
			t.Fatalf("timeline should not be requested when includeTimeline=false")
		}
	})

	t.Run("content pageSize capped and includeTimeline parse error keeps true", func(t *testing.T) {
		before := timelineRequests
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderContent?folderId=1&page=2&pageSize=999&includeTimeline=bad", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderContent(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if int(got["pageSize"].(float64)) != 200 {
			t.Fatalf("pageSize=%v", got["pageSize"])
		}
		if timelineRequests <= before {
			t.Fatalf("timeline request expected when includeTimeline parse fails")
		}
	})

	t.Run("breadcrumbs not init", func(t *testing.T) {
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoFolderBreadcrumbs(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderBreadcrumbs?folderId=1", nil))
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("breadcrumbs invalid folder", func(t *testing.T) {
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderBreadcrumbs(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderBreadcrumbs?folderId=0", nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("breadcrumbs error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoFolderBreadcrumbs(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderBreadcrumbs?folderId=1", nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("breadcrumbs ok", func(t *testing.T) {
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderBreadcrumbs(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderBreadcrumbs?folderId=1", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
	})
}

func TestHandleImportMtPhotoMedia_DedupFallbackBranches(t *testing.T) {
	md5Input := "0123456789abcdef0123456789abcdef"
	now := time.Now()

	t.Run("existing record fallback localFilename", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Input).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(9), "u1", "o.jpg", "", "r.jpg", "http://x", "/images/fallback.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: md5Input, Valid: true}, now, sql.NullTime{Valid: false},
			))
		mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
			WithArgs(sqlmock.AnyArg(), int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Input)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["localFilename"] != "fallback.jpg" {
			t.Fatalf("localFilename=%v", got["localFilename"])
		}
	})

	t.Run("dedup after local save uses computed md5", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		lspRoot := t.TempDir()
		sourcePath := filepath.Join(lspRoot, "a.jpg")
		sourceBytes := []byte("dedup-after-save")
		if err := os.WriteFile(sourcePath, sourceBytes, 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		sum := md5.Sum(sourceBytes)
		computedMD5 := hex.EncodeToString(sum[:])

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Input).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM douyin_media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Input).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(computedMD5).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(77), "u1", "old.jpg", "", "", "", "/images/existing.jpg",
				int64(2), "image/jpeg", "jpg", sql.NullString{String: computedMD5, Valid: true}, now, sql.NullTime{Valid: false},
			))
		mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
			WithArgs(sqlmock.AnyArg(), int64(77)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		a := &App{
			cfg:         configForMtPhotoImport(lspRoot),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Input)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		got := decodeJSONBody(t, rr.Body)
		if got["dedup"] != true || got["localFilename"] != "existing.jpg" {
			t.Fatalf("got=%v", got)
		}
	})
}

func TestHandleImportMtPhotoMedia_PathReadSaveErrors(t *testing.T) {
	md5Input := "fedcba9876543210fedcba9876543210"

	newImportRequest := func() *http.Request {
		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Input)
		return newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
	}

	expectNoStoredByMD5 := func(mock sqlmock.Sqlmock, md5Value string) {
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM douyin_media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnError(sql.ErrNoRows)
	}

	t.Run("resolve lsp path error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		expectNoStoredByMD5(mock, md5Input)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/bad/a.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		a := &App{
			cfg:         configForMtPhotoImport(t.TempDir()),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
		}

		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, newImportRequest())
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("open local file error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		expectNoStoredByMD5(mock, md5Input)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/missing.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		a := &App{
			cfg:         configForMtPhotoImport(t.TempDir()),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
		}

		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, newImportRequest())
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("save local file error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		expectNoStoredByMD5(mock, md5Input)

		lspRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(lspRoot, "ok.jpg"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/ok.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		badBase := filepath.Join(t.TempDir(), "base-file")
		if err := os.WriteFile(badBase, []byte("not-dir"), 0o644); err != nil {
			t.Fatalf("write bad base: %v", err)
		}

		a := &App{
			cfg:         configForMtPhotoImport(lspRoot),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: badBase},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
		}

		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, newImportRequest())
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func configForMtPhotoImport(lspRoot string) config.Config {
	return config.Config{LspRoot: lspRoot}
}
