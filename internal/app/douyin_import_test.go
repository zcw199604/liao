package app

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
)

func newDouyinImportFormRequest(t *testing.T, form url.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/import", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestHandleDouyinImport_ServiceNotInit(t *testing.T) {
	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{})

	(&App{}).handleDouyinImport(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinImport_DownloaderNotConfigured(t *testing.T) {
	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{})

	app := &App{
		douyinDownloader: NewDouyinDownloaderService("", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinImport_ParseFormError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/import", strings.NewReader("a=%ZZ"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinImport_ValidateParamsAndCache(t *testing.T) {
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}

	t.Run("invalid params", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {"k"},
			"index":  {"-1"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("expired key", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {"missing"},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("index out of range", func(t *testing.T) {
		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{"http://example.com/a.mp4"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"1"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty remote url", func(t *testing.T) {
		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{""},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleDouyinImport_BadRemoteURLAndDownloadFailures(t *testing.T) {
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}

	t.Run("bad remote url", func(t *testing.T) {
		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{"://bad"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("download http client error", func(t *testing.T) {
		d := NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second)
		d.api.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network down")
		})}

		app2 := *app
		app2.douyinDownloader = d

		key := app2.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{"http://example.com/a.mp4"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app2.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("download non-2xx", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		t.Cleanup(down.Close)

		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{down.URL + "/a.mp4"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "下载失败") {
			t.Fatalf("body=%q", rr.Body.String())
		}
	})
}

func TestHandleDouyinImport_ContentTypeValidationAndSaveFail(t *testing.T) {
	tmp := t.TempDir()

	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: tmp},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}

	t.Run("cannot infer content type", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("x"))
		}))
		t.Cleanup(down.Close)

		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{down.URL + "/noext"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("unsupported media type", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("x"))
		}))
		t.Cleanup(down.Close)

		key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{down.URL + "/a.zip"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app.handleDouyinImport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("save file fails", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("x"))
		}))
		t.Cleanup(down.Close)

		baseFile := filepath.Join(t.TempDir(), "base")
		if err := os.WriteFile(baseFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write base: %v", err)
		}
		app2 := *app
		app2.fileStorage = &FileStorageService{baseUploadAbs: baseFile}

		key := app2.douyinDownloader.CacheDetail(&douyinCachedDetail{
			DetailID:  "d",
			Title:     "t",
			Downloads: []string{down.URL + "/a.jpg"},
		})

		rr := httptest.NewRecorder()
		req := newDouyinImportFormRequest(t, url.Values{
			"userid": {"u1"},
			"key":    {key},
			"index":  {"0"},
		})
		app2.handleDouyinImport(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleDouyinImport_DedupExisting(t *testing.T) {
	fileBytes := []byte("dup-bytes")
	sum := md5.Sum(fileBytes)
	md5Value := hex.EncodeToString(sum[:])

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.mp4", "l.mp4", "r.mp4", "http://x", "/videos/x.mp4",
			int64(4), "video/mp4", "mp4", md5Value, time.Now(), time.Now(),
		))
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE user_id = [?] AND file_md5 = [?].*LIMIT 1`).
		WithArgs("u1", md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			"t.mp4",
			"l.mp4",
			"r.mp4",
			"http://x",
			"/videos/x.mp4",
			int64(4),
			"video/mp4",
			"mp4",
			md5Value,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uploadRoot := t.TempDir()

	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: db},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
		imageCache:       NewImageCacheService(),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.mp4"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["dedup"].(bool); !got {
		t.Fatalf("resp=%v", resp)
	}
	if port, _ := resp["port"].(string); port != "8006" {
		t.Fatalf("port=%v, want 8006", resp["port"])
	}
}

func TestHandleDouyinImport_UploadSuccessAndFallback(t *testing.T) {
	fileBytes := []byte("img-bytes")

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	var uploadReceived bool
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/asmx/upload.asmx/ProcessRequest") {
			http.NotFound(w, r)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		uploadReceived = true
		_, _ = w.Write([]byte(`{"state":"OK","msg":"abc.jpg"}`))
	}))
	t.Cleanup(imgSrv.Close)

	u, err := url.Parse(imgSrv.URL)
	if err != nil {
		t.Fatalf("parse imgSrv url: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE user_id = [?] AND file_md5 = [?].*LIMIT 1`).
		WithArgs("u1", sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"abc.jpg",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"image/jpeg",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: db},
		imageServer:      NewImageServerService(u.Hostname(), u.Port()),
		imageCache:       NewImageCacheService(),
		httpClient:       imgSrv.Client(),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.jpg"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !uploadReceived {
		t.Fatalf("expected upload received")
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if port, _ := resp["port"].(string); port != defaultSystemConfig.ImagePortFixed {
		t.Fatalf("port=%v, want %s", resp["port"], defaultSystemConfig.ImagePortFixed)
	}

	// fallback: upstream returns non-json text
	imgSrv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte("RAW_OK"))
	}))
	t.Cleanup(imgSrv2.Close)

	u2, err := url.Parse(imgSrv2.URL)
	if err != nil {
		t.Fatalf("parse imgSrv2 url: %v", err)
	}
	app2 := *app
	app2.imageServer = NewImageServerService(u2.Hostname(), u2.Port())
	app2.httpClient = imgSrv2.Client()

	rr2 := httptest.NewRecorder()
	req2 := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app2.handleDouyinImport(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr2.Code, http.StatusOK)
	}
	if strings.TrimSpace(rr2.Body.String()) != "RAW_OK" {
		t.Fatalf("body=%q, want RAW_OK", rr2.Body.String())
	}
}

func TestHandleDouyinImport_UploadError(t *testing.T) {
	fileBytes := []byte("x")
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("no"))
	}))
	t.Cleanup(imgSrv.Close)

	u, err := url.Parse(imgSrv.URL)
	if err != nil {
		t.Fatalf("parse imgSrv url: %v", err)
	}

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: mustNewSQLMockDB(t)},
		imageServer:      NewImageServerService(u.Hostname(), u.Port()),
		imageCache:       NewImageCacheService(),
		httpClient:       imgSrv.Client(),
	}
	t.Cleanup(func() { _ = app.mediaUpload.db.Close() })

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.mp4"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), "导入上传失败") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestHandleDouyinImport_DedupExisting_ImagePortByConfig(t *testing.T) {
	fileBytes := []byte("dup-img-bytes")
	sum := md5.Sum(fileBytes)
	md5Value := hex.EncodeToString(sum[:])

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.jpg", "l.jpg", "r.jpg", "http://x", "/images/x.jpg",
			int64(4), "image/jpeg", "jpg", md5Value, time.Now(), time.Now(),
		))
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE user_id = [?] AND file_md5 = [?].*LIMIT 1`).
		WithArgs("u1", md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			"t.jpg",
			"l.jpg",
			"r.jpg",
			"http://x",
			"/images/x.jpg",
			int64(4),
			"image/jpeg",
			"jpg",
			md5Value,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: db},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
		imageCache:       NewImageCacheService(),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.jpg"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["dedup"].(bool); !got {
		t.Fatalf("resp=%v", resp)
	}
	if port, _ := resp["port"].(string); port != defaultSystemConfig.ImagePortFixed {
		t.Fatalf("port=%v, want %s", resp["port"], defaultSystemConfig.ImagePortFixed)
	}
}

func TestHandleDouyinImport_UploadSuccess_VideoPort8006(t *testing.T) {
	fileBytes := []byte("dup-video-bytes")

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/asmx/upload.asmx/ProcessRequest") {
			http.NotFound(w, r)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"state":"OK","msg":"abc.mp4"}`))
	}))
	t.Cleanup(imgSrv.Close)

	u, err := url.Parse(imgSrv.URL)
	if err != nil {
		t.Fatalf("parse imgSrv url: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE user_id = [?] AND file_md5 = [?].*LIMIT 1`).
		WithArgs("u1", sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"abc.mp4",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"video/mp4",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: db},
		imageServer:      NewImageServerService(u.Hostname(), u.Port()),
		imageCache:       NewImageCacheService(),
		httpClient:       imgSrv.Client(),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.mp4"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if port, _ := resp["port"].(string); port != "8006" {
		t.Fatalf("port=%v, want 8006", resp["port"])
	}
	if dedup, _ := resp["dedup"].(bool); dedup {
		t.Fatalf("resp=%v", resp)
	}
}

func TestHandleDouyinImport_JSONNotEnhanced_FallsBackToText(t *testing.T) {
	fileBytes := []byte("img-bytes")

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileBytes)
	}))
	t.Cleanup(down.Close)

	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"state":"NO","msg":"x"}`))
	}))
	t.Cleanup(imgSrv.Close)

	u, err := url.Parse(imgSrv.URL)
	if err != nil {
		t.Fatalf("parse imgSrv url: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT id, user_id, original_filename, local_filename, remote_filename, remote_url, local_path.*FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: db},
		imageServer:      NewImageServerService(u.Hostname(), u.Port()),
		httpClient:       imgSrv.Client(),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/a.jpg"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != `{"state":"NO","msg":"x"}` {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestHandleDouyinImport_EmptyContentType_CannotInfer(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(down.Close)

	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{},
		imageServer:      NewImageServerService("127.0.0.1", "9003"),
	}

	key := app.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "d",
		Title:     "t",
		Downloads: []string{down.URL + "/noext"},
	})

	rr := httptest.NewRecorder()
	req := newDouyinImportFormRequest(t, url.Values{
		"userid": {"u1"},
		"key":    {key},
		"index":  {"0"},
	})
	app.handleDouyinImport(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "无法识别文件类型") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func mustNewSQLMockDB(t *testing.T) *sql.DB {
	t.Helper()
	db, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return db
}
