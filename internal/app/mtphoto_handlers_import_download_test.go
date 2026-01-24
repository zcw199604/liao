package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func newConfiguredMtPhotoServiceForHandler(t *testing.T, baseURL string, client *http.Client) *MtPhotoService {
	t.Helper()
	mt := NewMtPhotoService(baseURL, "u", "p", "", "/lsp", client)
	mt.token = "t"
	mt.authCode = "ac"
	mt.tokenExp = time.Now().Add(1 * time.Hour)
	return mt
}

func TestHandleDownloadMtPhotoOriginal_CoversBranches(t *testing.T) {
	md5Value := "0123456789abcdef0123456789abcdef"

	t.Run("not initialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		(&App{}).handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("bad params", func(t *testing.T) {
		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, "http://example.com", &http.Client{Timeout: time.Second})}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=0&md5=bad", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("GatewayFileDownload error", func(t *testing.T) {
		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, "http://example.com", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, context.Canceled
			}),
		})}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upstream non-2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/gateway/fileDownload/") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client())}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("success with Content-Disposition passthrough", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/gateway/fileDownload/") {
				w.Header().Set("Content-Type", "image/jpeg")
				w.Header().Set("Cache-Control", "max-age=60")
				w.Header().Set("Content-Length", "4")
				w.Header().Set("Accept-Ranges", "bytes")
				w.Header().Set("Content-Disposition", `attachment; filename="a.jpg"`)
				_, _ = w.Write([]byte("data"))
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client())}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		if rr.Body.String() != "data" {
			t.Fatalf("body=%q", rr.Body.String())
		}
		if strings.TrimSpace(rr.Header().Get("Content-Disposition")) == "" {
			t.Fatalf("missing content-disposition")
		}
	})

	t.Run("success without Content-Disposition, ResolveFilePath ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 9, "filePath": "/lsp/a/b.jpg"}})
				return
			default:
				if strings.HasPrefix(r.URL.Path, "/gateway/fileDownload/") {
					// 覆盖 handler 的 Content-Type 为空兜底分支：禁用 net/http 自动 sniff。
					w.Header().Set("Content-Type", "")
					_, _ = w.Write([]byte("bin"))
					return
				}
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client())}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		if strings.TrimSpace(rr.Header().Get("Content-Type")) != "application/octet-stream" {
			t.Fatalf("content-type=%q", rr.Header().Get("Content-Type"))
		}
		if disp := rr.Header().Get("Content-Disposition"); !strings.Contains(disp, "attachment;") {
			t.Fatalf("disp=%q", disp)
		}
	})

	t.Run("success without Content-Disposition, ResolveFilePath error uses contentType ext", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				w.WriteHeader(http.StatusInternalServerError)
				return
			default:
				if strings.HasPrefix(r.URL.Path, "/gateway/fileDownload/") {
					w.Header().Set("Content-Type", "image/jpeg")
					_, _ = w.Write([]byte("bin"))
					return
				}
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		a := &App{mtPhoto: newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client())}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
		rr := httptest.NewRecorder()
		a.handleDownloadMtPhotoOriginal(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		if disp := rr.Header().Get("Content-Disposition"); !strings.Contains(disp, md5Value+".jpg") {
			t.Fatalf("disp=%q", disp)
		}
	})
}

func TestHandleImportMtPhotoMedia_CoversBranches(t *testing.T) {
	md5Value := "0123456789abcdef0123456789abcdef"

	t.Run("not initialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/importMtPhotoMedia", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("parse form error", func(t *testing.T) {
		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/importMtPhotoMedia", strings.NewReader("userid=%zz"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("missing params", func(t *testing.T) {
		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/importMtPhotoMedia", strings.NewReader("userid=u1&md5="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("existing record returns immediately (video port)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(1), "u1", "o.mp4", "l.mp4", "remote.mp4", "http://x", "/images/x.mp4",
				int64(1), "video/mp4", "mp4", sql.NullString{String: md5Value, Valid: true}, now, sql.NullTime{Valid: false},
			))

		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["port"] != "8006" {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("existing record returns immediately (image port)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(1), "u1", "o.jpg", "l.jpg", "remote.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: md5Value, Valid: true}, now, sql.NullTime{Valid: false},
			))

		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["port"] != "9006" {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("ResolveFilePath error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			mtPhoto:     NewMtPhotoService("", "", "", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("resolveLspLocalPath error", func(t *testing.T) {
		lspRoot := t.TempDir()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/bad"}})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("unsupported file type", func(t *testing.T) {
		lspRoot := t.TempDir()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.txt"}})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("openLocalFileForRead error (missing file)", func(t *testing.T) {
		lspRoot := t.TempDir()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/missing.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("SaveFileFromReader error", func(t *testing.T) {
		lspRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(lspRoot, "a"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lspRoot, "a", "b.jpg"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		uploadRootFile := filepath.Join(t.TempDir(), "uploadRootFile")
		if err := os.WriteFile(uploadRootFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/gateway/filesInMD5" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.jpg"}})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: uploadRootFile},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("uploadAbsPathToUpstream error", func(t *testing.T) {
		lspRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(lspRoot, "a"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lspRoot, "a", "b.jpg"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.jpg"}})
				return
			case "/asmx/upload.asmx/ProcessRequest":
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte("bad"))
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		hostPort := strings.TrimPrefix(srv.URL, "http://")
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			httpClient:  srv.Client(),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService(strings.Split(hostPort, ":")[0], strings.Split(hostPort, ":")[1]),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("success enhanced (image) + mismatch md5 warning", func(t *testing.T) {
		lspRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(lspRoot, "a"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lspRoot, "a", "b.jpg"), []byte("hello"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.jpg"}})
				return
			case "/asmx/upload.asmx/ProcessRequest":
				_, _ = io.ReadAll(r.Body)
				_, _ = w.Write([]byte(`{"state":"OK","msg":"remote.jpg"}`))
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		hostPort := strings.TrimPrefix(srv.URL, "http://")
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(`INSERT INTO media_file`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "remote.jpg", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "image/jpeg", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			httpClient:  srv.Client(),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService(strings.Split(hostPort, ":")[0], strings.Split(hostPort, ":")[1]),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value) // mismatch with actual file md5
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["port"] != "9006" {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("success enhanced (video)", func(t *testing.T) {
		md5Video := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

		lspRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(lspRoot, "a"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lspRoot, "a", "b.mp4"), []byte("hello"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.mp4"}})
				return
			case "/asmx/upload.asmx/ProcessRequest":
				_, _ = io.ReadAll(r.Body)
				_, _ = w.Write([]byte(`{"state":"OK","msg":"remote.mp4"}`))
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		hostPort := strings.TrimPrefix(srv.URL, "http://")
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Video).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Video).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(`INSERT INTO media_file`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "remote.mp4", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "video/mp4", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			httpClient:  srv.Client(),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService(strings.Split(hostPort, ":")[0], strings.Split(hostPort, ":")[1]),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Video)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["port"] != "8006" {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("fallback (response not enhanced)", func(t *testing.T) {
		lspRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(lspRoot, "a"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lspRoot, "a", "b.jpg"), []byte("hello"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "filePath": "/lsp/a/b.jpg"}})
				return
			case "/asmx/upload.asmx/ProcessRequest":
				_, _ = io.ReadAll(r.Body)
				_, _ = w.Write([]byte("OK"))
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		hostPort := strings.TrimPrefix(srv.URL, "http://")
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE user_id = \? AND file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs("u1", md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			cfg:         config.Config{LspRoot: lspRoot},
			httpClient:  srv.Client(),
			mtPhoto:     newConfiguredMtPhotoServiceForHandler(t, srv.URL, srv.Client()),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: db},
			imageServer: NewImageServerService(strings.Split(hostPort, ":")[0], strings.Split(hostPort, ":")[1]),
			imageCache:  NewImageCacheService(),
		}

		form := url.Values{}
		form.Set("userid", "u1")
		form.Set("md5", md5Value)
		req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/importMtPhotoMedia", form)
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		if strings.TrimSpace(rr.Body.String()) != "OK" {
			t.Fatalf("body=%q", rr.Body.String())
		}
	})
}
