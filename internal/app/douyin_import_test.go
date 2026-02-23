package app

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	expectInsertReturningID(
		mock,
		`(?s)INSERT INTO douyin_media_file`,
		1,
		"u1",
		sqlmock.AnyArg(),
		"d",
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
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
	)

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: wrapMySQLDB(db)},
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
	expectInsertReturningID(
		mock,
		`(?s)INSERT INTO douyin_media_file`,
		1,
		"u1",
		sqlmock.AnyArg(),
		"d",
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
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
	)

	uploadRoot := t.TempDir()
	app := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:1", "", "", "", 60*time.Second),
		fileStorage:      &FileStorageService{baseUploadAbs: uploadRoot},
		mediaUpload:      &MediaUploadService{db: wrapMySQLDB(db)},
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

func TestHandleDouyinImport_DefaultUserIDAndCrossHostRedirectDropsCookie(t *testing.T) {
	oldURL := "http://www.douyin.com/video/old.mp4"
	finalURL := "http://cdn.example.com/final.jpg"

	svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
	var firstCookie, finalCookie string
	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case oldURL:
			firstCookie = strings.TrimSpace(r.Header.Get("Cookie"))
			h := make(http.Header)
			h.Set("Location", finalURL)
			return &http.Response{StatusCode: http.StatusFound, Status: "302 Found", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		case finalURL:
			finalCookie = strings.TrimSpace(r.Header.Get("Cookie"))
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("img")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.ExpectQuery(`(?s)FROM media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	expectInsertReturningID(
		mock,
		`(?s)INSERT INTO douyin_media_file`,
		1,
		"pre_identity",
		sqlmock.AnyArg(),
		"d1",
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		"t.jpg",
		sqlmock.AnyArg(),
		"",
		"",
		sqlmock.AnyArg(),
		int64(3),
		"image/jpeg",
		"jpg",
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	)

	a := &App{
		douyinDownloader: svc,
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{db: wrapMySQLDB(db)},
	}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Downloads: []string{oldURL}})

	req := newDouyinImportFormRequest(t, url.Values{
		"key":   {key},
		"index": {"0"},
	})
	rec := httptest.NewRecorder()
	a.handleDouyinImport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if firstCookie != "sid=abc" {
		t.Fatalf("first cookie=%q", firstCookie)
	}
	if finalCookie != "" {
		t.Fatalf("final cookie should be dropped, got=%q", finalCookie)
	}

	resp := decodeJSONBody(t, rec.Body)
	if dedup, _ := resp["dedup"].(bool); dedup {
		t.Fatalf("unexpected dedup response: %v", resp)
	}
	if got, _ := resp["localPath"].(string); !strings.HasPrefix(got, "/douyin/images/") {
		t.Fatalf("localPath=%q", got)
	}
}

func TestHandleDouyinImport_ForbiddenRefreshBranches(t *testing.T) {
	t.Run("refresh detail returns error", func(t *testing.T) {
		oldURL := "http://www.douyin.com/video/old.mp4"
		svc := NewDouyinDownloaderService("http://upstream.local", "", "", "", 60*time.Second)
		svc.SetCookieProvider(cookieProviderFunc(func(ctx context.Context) (string, error) {
			return "", errors.New("cookie unavailable")
		}))

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{
			douyinDownloader: svc,
			fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload:      &MediaUploadService{},
		}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Downloads: []string{oldURL}})

		req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
		rec := httptest.NewRecorder()
		a.handleDouyinImport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		resp := decodeJSONBody(t, rec.Body)
		if !strings.Contains(asString(resp["error"]), "403") {
			t.Fatalf("resp=%v", resp)
		}
	})

	t.Run("refresh success but refreshed list does not contain requested index", func(t *testing.T) {
		oldURL0 := "http://www.douyin.com/image/old0.jpg"
		oldURL1 := "http://www.douyin.com/image/old1.jpg"
		newURL := "http://cdn.example.com/new-only.jpg"

		svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-index")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"message":"OK","data":{"desc":"t","type":"图集","downloads":["` + newURL + `"]}}`)),
					Request:    r,
				}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc, fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}, mediaUpload: &MediaUploadService{}}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d2", Title: "t", Downloads: []string{oldURL0, oldURL1}})

		req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"1"}})
		rec := httptest.NewRecorder()
		a.handleDouyinImport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		resp := decodeJSONBody(t, rec.Body)
		if !strings.Contains(asString(resp["error"]), "403") {
			t.Fatalf("resp=%v", resp)
		}
	})

	t.Run("refresh success but retry request returns error", func(t *testing.T) {
		oldURL := "http://www.douyin.com/video/old.mp4"
		newURL := "http://www.douyin.com/video/new.mp4"
		newCover := "http://www.douyin.com/image/new.jpg"

		svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(
						`{"message":"OK","data":{"desc":"t","type":"视频","sec_user_id":"sec-new","static_cover":"` + newCover + `","downloads":["` + newURL + `"]}}`,
					)),
					Request: r,
				}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL:
				return nil, io.EOF
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{
			douyinDownloader: svc,
			fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
			mediaUpload:      &MediaUploadService{},
			douyinFavorite:   &DouyinFavoriteService{},
		}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d3", Title: "t", SecUserID: "", Downloads: []string{oldURL}})

		req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
		rec := httptest.NewRecorder()
		a.handleDouyinImport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		resp := decodeJSONBody(t, rec.Body)
		if !strings.Contains(asString(resp["error"]), "EOF") {
			t.Fatalf("resp=%v", resp)
		}
	})

	t.Run("refresh success but retry response still non-2xx", func(t *testing.T) {
		oldURL := "http://www.douyin.com/video/old2.mp4"
		newURL := "http://www.douyin.com/video/new2.mp4"

		svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old2")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"message":"OK","data":{"desc":"t","type":"视频","downloads":["` + newURL + `"]}}`)),
					Request:    r,
				}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL:
				h := make(http.Header)
				h.Set("Content-Type", "text/plain")
				return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Header: h, Body: io.NopCloser(strings.NewReader("still-bad")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc, fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}, mediaUpload: &MediaUploadService{}}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d4", Title: "t", Downloads: []string{oldURL}})

		req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
		rec := httptest.NewRecorder()
		a.handleDouyinImport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		resp := decodeJSONBody(t, rec.Body)
		if !strings.Contains(asString(resp["error"]), "still-bad") {
			t.Fatalf("resp=%v", resp)
		}
	})
}

func TestHandleDouyinImport_ForbiddenNeedCookieInferredFromFinalHost(t *testing.T) {
	oldURL := "http://cdn.example.com/import-old.mp4"
	finalURL := "http://www.douyin.com/video/protected.mp4"

	svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldURL:
			u, _ := url.Parse(finalURL)
			r2 := r.Clone(r.Context())
			r2.URL = u
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden")), Request: r2}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	a := &App{douyinDownloader: svc, fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}, mediaUpload: &MediaUploadService{}}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "i-final-host", Title: "t", Downloads: []string{oldURL}})

	req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
	rec := httptest.NewRecorder()
	a.handleDouyinImport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "403") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleDouyinImport_RefreshFallbackSkipsBlankAndRetrySuccess(t *testing.T) {
	oldURL := "http://www.douyin.com/video/import-old-fallback.mp4"
	newURL := "http://www.douyin.com/video/import-new-fallback.jpg"

	svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldURL:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"message":"OK","data":{"desc":"t","type":"视频","downloads":["","` + newURL + `"]}}`)),
				Request:    r,
			}, nil
		case r.Method == http.MethodGet && r.URL.String() == newURL:
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("img")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	db := mustNewSQLMockDB(t)
	t.Cleanup(func() { _ = db.Close() })

	a := &App{
		douyinDownloader: svc,
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{db: wrapMySQLDB(db)},
	}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "i-fallback", Title: "t", Downloads: []string{oldURL}})

	req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
	rec := httptest.NewRecorder()
	a.handleDouyinImport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeJSONBody(t, rec.Body)
	if dedup, _ := resp["dedup"].(bool); dedup {
		t.Fatalf("resp=%v", resp)
	}
	if strings.TrimSpace(asString(resp["localPath"])) == "" {
		t.Fatalf("resp=%v", resp)
	}
}

func TestHandleDouyinImport_EmptyContentTypeCannotInfer(t *testing.T) {
	oldURL := "http://cdn.example.com/noext"

	svc := NewDouyinDownloaderService("http://upstream.local", "", "", "", 60*time.Second)
	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet && r.URL.String() == oldURL {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}
		return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
	})}

	a := &App{douyinDownloader: svc, fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}, mediaUpload: &MediaUploadService{}}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "i-no-ct", Title: "t", Downloads: []string{oldURL}})

	req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"0"}})
	rec := httptest.NewRecorder()
	a.handleDouyinImport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "无法识别文件类型") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleDouyinImport_RefreshFallbackFirstVideoWhenIndexOutOfRange(t *testing.T) {
	oldURL0 := "http://img.example.com/import-old0.jpg"
	oldURL1 := "http://www.douyin.com/video/import-old1.mp4"
	newURL := "http://www.douyin.com/video/import-new1.mp4"

	svc := NewDouyinDownloaderService("http://upstream.local", "", "sid=abc", "", 60*time.Second)
	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldURL1:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"message":"OK","data":{"desc":"t","type":"视频","downloads":["` + newURL + `"]}}`)),
				Request:    r,
			}, nil
		case r.Method == http.MethodGet && r.URL.String() == newURL:
			h := make(http.Header)
			h.Set("Content-Type", "video/mp4")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("v")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	db := mustNewSQLMockDB(t)
	t.Cleanup(func() { _ = db.Close() })

	a := &App{
		douyinDownloader: svc,
		fileStorage:      &FileStorageService{baseUploadAbs: t.TempDir()},
		mediaUpload:      &MediaUploadService{db: wrapMySQLDB(db)},
	}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "i-fallback-index", Title: "t", Downloads: []string{oldURL0, oldURL1}})

	req := newDouyinImportFormRequest(t, url.Values{"userid": {"u1"}, "key": {key}, "index": {"1"}})
	rec := httptest.NewRecorder()
	a.handleDouyinImport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeJSONBody(t, rec.Body)
	if strings.TrimSpace(asString(resp["localPath"])) == "" {
		t.Fatalf("resp=%v", resp)
	}
}
