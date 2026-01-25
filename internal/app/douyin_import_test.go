package app

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}).AddRow(
			int64(1), "u1", "o.mp4", "l.mp4", "r.mp4", "http://x", "/videos/x.mp4",
			int64(4), "video/mp4", "mp4", md5Value, time.Now(), time.Now(),
		))
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			"t.mp4",
			"l.mp4",
			"",
			"",
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
	if dedup, _ := resp["dedup"].(bool); !dedup {
		t.Fatalf("resp=%v", resp)
	}
	if uploaded, _ := resp["uploaded"].(bool); uploaded {
		t.Fatalf("resp=%v", resp)
	}
	if got, _ := resp["localPath"].(string); got != "/videos/x.mp4" {
		t.Fatalf("localPath=%v, want %q", resp["localPath"], "/videos/x.mp4")
	}
	if got, _ := resp["localFilename"].(string); got != "l.mp4" {
		t.Fatalf("localFilename=%v, want %q", resp["localFilename"], "l.mp4")
	}

	// 新写入的 douyin 临时文件应已被删除（保留目录不影响）
	douyinDir := filepath.Join(uploadRoot, "douyin")
	files := 0
	_ = filepath.Walk(douyinDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.Mode().IsRegular() {
			files++
		}
		return nil
	})
	if files != 0 {
		t.Fatalf("expected no files under %s, got %d", douyinDir, files)
	}
}

func TestHandleDouyinImport_LocalOnlySuccess(t *testing.T) {
	fileBytes := []byte("img-bytes")
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

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`(?s)INSERT INTO douyin_media_file`).
		WithArgs(
			"u1",
			sqlmock.AnyArg(),
			"d",
			"t.jpg",
			sqlmock.AnyArg(),
			"",
			"",
			sqlmock.AnyArg(),
			int64(len(fileBytes)),
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
	if dedup, _ := resp["dedup"].(bool); dedup {
		t.Fatalf("resp=%v", resp)
	}
	if uploaded, _ := resp["uploaded"].(bool); uploaded {
		t.Fatalf("resp=%v", resp)
	}
	localPath, _ := resp["localPath"].(string)
	if !strings.HasPrefix(localPath, "/douyin/images/") {
		t.Fatalf("localPath=%q", localPath)
	}
	if localFilename, _ := resp["localFilename"].(string); strings.TrimSpace(localFilename) == "" {
		t.Fatalf("resp=%v", resp)
	}

	full := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if st, err := os.Stat(full); err != nil || st == nil || st.Size() != int64(len(fileBytes)) {
		t.Fatalf("file=%s stat=%v size=%v", full, err, st)
	}
}
