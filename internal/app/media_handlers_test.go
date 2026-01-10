package app

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net"
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

type stringPrefixSuffix struct {
	prefix string
	suffix string
}

func (m stringPrefixSuffix) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(s, m.prefix) && strings.HasSuffix(s, m.suffix)
}

func TestHandleCheckDuplicateMedia_MD5Hit(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	content := []byte("image-bytes")
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}).AddRow(
			int64(1),
			"/images/2026/01/10/x.png",
			"x.png",
			"images/2026/01/10",
			md5Hex,
			int64(123),
			int64(10),
			time.Now(),
		))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["matchType"].(string); got != "md5" {
		t.Fatalf("matchType=%q, want %q", got, "md5")
	}
	if got, _ := data["md5"].(string); got != md5Hex {
		t.Fatalf("md5=%q, want %q", got, md5Hex)
	}
	items, _ := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len=%d, want 1", len(items))
	}
}

func TestHandleCheckDuplicateMedia_PHashUnsupported(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	content := []byte("not-an-image")
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.bin", "application/octet-stream", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["matchType"].(string); got != "none" {
		t.Fatalf("matchType=%q, want %q", got, "none")
	}
	if got, _ := data["reason"].(string); got == "" {
		t.Fatalf("expected non-empty reason")
	}
}

func TestHandleCheckDuplicateMedia_PHashMatch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 5), B: 0, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode png failed: %v", err)
	}
	content := pngBuf.Bytes()
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}))

	mock.ExpectQuery(`(?s)BIT_COUNT.*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at", "distance",
		}).AddRow(
			int64(2),
			"/images/2026/01/10/y.png",
			"y.png",
			"images/2026/01/10",
			"othermd5",
			int64(456),
			int64(1),
			time.Now(),
			3,
		))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["matchType"].(string); got != "phash" {
		t.Fatalf("matchType=%q, want %q", got, "phash")
	}
	items, _ := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len=%d, want 1", len(items))
	}
}

func TestHandleUploadMedia_SuccessEnhanced(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9006" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"OK","msg":"abc.jpg"}`))
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url failed: %v", err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}

	imageSrv := NewImageServerService(host, port)
	fileStore := &FileStorageService{db: db, baseUploadAbs: tempDir}
	mediaUpload := NewMediaUploadService(db, 8080, fileStore, imageSrv, upstream.Client())

	app := &App{
		httpClient:  upstream.Client(),
		fileStorage: fileStore,
		imageServer: imageSrv,
		imageCache:  NewImageCacheService(),
		mediaUpload: mediaUpload,
	}

	content := []byte("png-bytes-for-upload")
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs(md5Hex).
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}))

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", md5Hex).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))

	mock.ExpectExec(`INSERT INTO media_file`).
		WithArgs(
			"u1",
			"a.png",
			stringPrefixSuffix{prefix: "", suffix: ".png"},
			"abc.jpg",
			"http://"+host+":9006/img/Upload/abc.jpg",
			stringPrefixSuffix{prefix: "/images/", suffix: ".png"},
			int64(len(content)),
			"image/png",
			"png",
			md5Hex,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, _ := newMultipartRequest(
		t,
		"POST",
		"http://example.com/api/uploadMedia",
		"file",
		"a.png",
		"image/png",
		content,
		map[string]string{"userid": "u1"},
	)

	rr := httptest.NewRecorder()
	app.handleUploadMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v; body=%s", err, rr.Body.String())
	}
	if got, _ := resp["state"].(string); got != "OK" {
		t.Fatalf("state=%q, want %q", got, "OK")
	}
	if got, _ := resp["msg"].(string); got != "abc.jpg" {
		t.Fatalf("msg=%q, want %q", got, "abc.jpg")
	}
	if got, _ := resp["port"].(string); got != "9006" {
		t.Fatalf("port=%q, want %q", got, "9006")
	}
	localFilename, _ := resp["localFilename"].(string)
	if !strings.HasSuffix(localFilename, ".png") || localFilename == "" {
		t.Fatalf("localFilename=%q invalid", localFilename)
	}

	cached := app.imageCache.GetCachedImages("u1")
	if cached == nil || len(cached.ImageURLs) != 1 {
		t.Fatalf("expected cache updated, got %+v", cached)
	}
	localPath := cached.ImageURLs[0]

	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("expected file exists: %s err=%v", full, err)
	}

	// cleanup: 确保不会污染临时目录（可重复执行）
	_ = os.Remove(full)
	_ = os.RemoveAll(filepath.Dir(full))
}

func TestHandleUploadMedia_BadContentType(t *testing.T) {
	app := &App{fileStorage: &FileStorageService{}}

	req, _ := newMultipartRequest(
		t,
		"POST",
		"http://example.com/api/uploadMedia",
		"file",
		"a.txt",
		"text/plain",
		[]byte("x"),
		map[string]string{"userid": "u1"},
	)

	rr := httptest.NewRecorder()
	app.handleUploadMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleUploadMedia_FindLocalPathByMD5ErrorIgnored(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9006" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"OK","msg":"abc.jpg"}`))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	host, port, _ := net.SplitHostPort(u.Host)

	imageSrv := NewImageServerService(host, port)
	fileStore := &FileStorageService{db: db, baseUploadAbs: tempDir}
	mediaUpload := NewMediaUploadService(db, 8080, fileStore, imageSrv, upstream.Client())

	app := &App{
		httpClient:  upstream.Client(),
		fileStorage: fileStore,
		imageServer: imageSrv,
		imageCache:  NewImageCacheService(),
		mediaUpload: mediaUpload,
	}

	content := []byte("png-bytes-for-upload")
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	// 这里模拟 DB 异常：FindLocalPathByMD5 会返回 error，从而走 SaveFile 分支（错误被忽略）
	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs(md5Hex).
		WillReturnError(context.DeadlineExceeded)

	mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
		WithArgs("u1", md5Hex).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
			"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
		}))

	mock.ExpectExec(`INSERT INTO media_file`).
		WithArgs(
			"u1",
			"a.png",
			stringPrefixSuffix{prefix: "", suffix: ".png"},
			"abc.jpg",
			"http://"+host+":9006/img/Upload/abc.jpg",
			stringPrefixSuffix{prefix: "/images/", suffix: ".png"},
			int64(len(content)),
			"image/png",
			"png",
			md5Hex,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, _ := newMultipartRequest(
		t,
		"POST",
		"http://example.com/api/uploadMedia",
		"file",
		"a.png",
		"image/png",
		content,
		map[string]string{"userid": "u1"},
	)

	rr := httptest.NewRecorder()
	app.handleUploadMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}
