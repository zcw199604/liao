package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
		}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/importMtPhotoMedia", strings.NewReader("userid=u1&md5="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		a.handleImportMtPhotoMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("existing record returns immediately (media_file)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(1), "u1", "o.jpg", "l.jpg", "remote.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: md5Value, Valid: true}, now, sql.NullTime{Valid: false},
			))
		mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE id = \?`).
			WithArgs(sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
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
		if got["dedup"] != true || got["uploaded"] != false {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("existing record returns immediately (douyin_media_file)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM douyin_media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(2), "u2", "o.jpg", "l.jpg", "remote.jpg", "http://x", "/images/x.jpg",
				int64(1), "image/jpeg", "jpg", sql.NullString{String: md5Value, Valid: true}, now, sql.NullTime{Valid: false},
			))
		mock.ExpectExec(`UPDATE douyin_media_file SET update_time = \? WHERE id = \?`).
			WithArgs(sqlmock.AnyArg(), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		a := &App{
			mtPhoto:     NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
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
		if got["dedup"] != true || got["uploaded"] != false {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("ResolveFilePath error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path,\s*file_size, file_type, file_extension, file_md5, upload_time, update_time\s*FROM douyin_media_file\s*WHERE file_md5 = \?\s*ORDER BY id\s*LIMIT 1`).
			WithArgs(md5Value).
			WillReturnError(sql.ErrNoRows)

		a := &App{
			mtPhoto: newConfiguredMtPhotoServiceForHandler(t, "http://example.com", &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, context.Canceled }),
			}),
			fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload: &MediaUploadService{db: wrapMySQLDB(db)},
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
}
