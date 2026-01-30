package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"mime/multipart"
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

func TestHandleGetImgServer_CoversErrors(t *testing.T) {
	t.Run("http error", func(t *testing.T) {
		app := &App{httpClient: &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			}),
		}}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/getImgServer", nil)
		rr := httptest.NewRecorder()
		app.handleGetImgServer(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		app := &App{httpClient: &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Status:     "502 Bad Gateway",
					Body:       io.NopCloser(strings.NewReader("bad")),
					Header:     make(http.Header),
				}, nil
			}),
		}}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/getImgServer", nil)
		rr := httptest.NewRecorder()
		app.handleGetImgServer(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})
}

func TestHandleCancelFavorite_CoversErrorBranch(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return newTextResponse(http.StatusBadGateway, "bad"), nil
		}),
	}
	app := &App{httpClient: client}
	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/cancelFavorite", form)
	rr := httptest.NewRecorder()
	app.handleCancelFavorite(rr, req)
	if !strings.Contains(rr.Body.String(), `"state":"ERROR"`) {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestPostForm_CoversNewRequestAndReadAllErrors(t *testing.T) {
	app := &App{httpClient: &http.Client{Timeout: time.Second}}
	if _, _, err := app.postForm(context.Background(), "http://[::1", nil, nil); err == nil {
		t.Fatalf("expected error")
	}

	app = &App{httpClient: &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       errReadCloserUserHistory{},
				Header:     make(http.Header),
			}, nil
		}),
	}}
	if _, _, err := app.postForm(context.Background(), "http://example.com", url.Values{}, map[string]string{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUploadToUpstream_CoversBranches(t *testing.T) {
	// Prepare a real *multipart.FileHeader.
	req, fileHeader := newMultipartRequest(t, http.MethodPost, "http://example.com/api/upload", "file", "a.png", "image/png", []byte("x"), map[string]string{
		"userid": "u1",
	})
	_ = req

	tempDir := t.TempDir()
	uploadURL := "http://example.com/upload"

	t.Run("bad url", func(t *testing.T) {
		app := &App{httpClient: &http.Client{Timeout: time.Second}}
		if _, err := app.uploadToUpstream(context.Background(), "http://[::1", "example.com:1", fileHeader, "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("CreateFormFile error via early close", func(t *testing.T) {
		app := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_ = req.Body.Close()
				return nil, errors.New("boom")
			}),
		}}
		if _, err := app.uploadToUpstream(context.Background(), uploadURL, "example.com:1", fileHeader, "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("file open error", func(t *testing.T) {
		oldOpen := openMultipartFileHeaderFn
		openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
			return nil, errors.New("open err")
		}
		t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

		app := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := app.uploadToUpstream(context.Background(), uploadURL, "example.com:1", fileHeader, "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("io.Copy error", func(t *testing.T) {
		oldOpen := openMultipartFileHeaderFn
		openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
			return &errMultipartFile{r: bytes.NewReader([]byte("x"))}, nil
		}
		t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

		app := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := app.uploadToUpstream(context.Background(), uploadURL, "example.com:1", fileHeader, "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("cookie header + Do/read/status branches", func(t *testing.T) {
		app := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Host != "example.com" {
					t.Fatalf("host=%q", req.Host)
				}
				if got := req.Header.Get("Cookie"); got != "c=1" {
					t.Fatalf("cookie=%q", got)
				}
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				switch req.URL.Path {
				case "/upload-err":
					return nil, errors.New("net err")
				case "/upload-readerr":
					return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: errReadCloserUserHistory{}, Header: make(http.Header), Request: req}, nil
				case "/upload-bad":
					return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: io.NopCloser(strings.NewReader("bad")), Header: make(http.Header), Request: req}, nil
				default:
					return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("OK")), Header: make(http.Header), Request: req}, nil
				}
			}),
		}}

		if _, err := app.uploadToUpstream(context.Background(), "http://example.com/upload-err", "example.com:1", fileHeader, "c=1", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := app.uploadToUpstream(context.Background(), "http://example.com/upload-readerr", "example.com:1", fileHeader, "c=1", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := app.uploadToUpstream(context.Background(), "http://example.com/upload-bad", "example.com:1", fileHeader, "c=1", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
		if got, err := app.uploadToUpstream(context.Background(), uploadURL, "example.com:1", fileHeader, "c=1", "r", "ua"); err != nil || strings.TrimSpace(got) != "OK" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})

	_ = tempDir
}

func TestHandleUploadMedia_CoversRemainingBranches(t *testing.T) {
	t.Run("ParseMultipartForm error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/uploadMedia", strings.NewReader("x"))
		req.Header.Set("Content-Type", "text/plain")

		app := &App{}
		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("invalid media type", func(t *testing.T) {
		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.pdf", "application/pdf", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{fileStorage: &FileStorageService{}}
		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("missing file field", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		_ = w.WriteField("userid", "u1")
		_ = w.Close()

		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/uploadMedia", body)
		req.Header.Set("Content-Type", w.FormDataContentType())

		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}
		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("CalculateMD5 error", func(t *testing.T) {
		oldOpen := openMultipartFileHeaderFn
		openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
			return nil, errors.New("open err")
		}
		t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.jpg", "image/jpeg", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}
		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("FindLocalPathByMD5 hit + upload not enhanced", func(t *testing.T) {
		uploadRoot := t.TempDir()
		localPath := "/images/exist.jpg"
		full := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(localPath))

		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.jpg", "image/jpeg", []byte("x"), map[string]string{
			"userid":     "u1",
			"cookieData": "c=1",
		})

		app := &App{
			fileStorage: &FileStorageService{db: db, baseUploadAbs: uploadRoot},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
			mediaUpload: &MediaUploadService{db: db},
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					_, _ = io.ReadAll(req.Body)
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader("OK")),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				}),
			},
		}

		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "OK" {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("uploadToUpstream error includes localPath", func(t *testing.T) {
		uploadRoot := t.TempDir()
		localPath := "/images/exist.jpg"
		full := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(localPath))

		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.jpg", "image/jpeg", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{
			fileStorage: &FileStorageService{db: db, baseUploadAbs: uploadRoot},
			imageServer: NewImageServerService("example.com", "9003"),
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					_, _ = io.ReadAll(req.Body)
					return nil, errors.New("net err")
				}),
			},
		}

		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
		got := decodeJSONBody(t, rr.Body)
		if got["localPath"] != localPath {
			t.Fatalf("localPath=%v", got["localPath"])
		}
		if _, ok := got["error"].(string); !ok {
			t.Fatalf("error=%v", got["error"])
		}
	})

	t.Run("enhanced response (image)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
			WithArgs("u1", sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(`INSERT INTO media_file`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		uploadRoot := t.TempDir()
		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.jpg", "image/jpeg", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{
			fileStorage: &FileStorageService{db: db, baseUploadAbs: uploadRoot},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
			mediaUpload: &MediaUploadService{db: db},
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					_, err := io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(`{"state":"OK","msg":"remote.jpg"}`)),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				}),
			},
		}

		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
		got := decodeJSONBody(t, rr.Body)
		if got["state"] != "OK" || got["msg"] != "remote.jpg" || got["port"] != "9006" {
			t.Fatalf("resp=%v", got)
		}
		if name, _ := got["localFilename"].(string); !strings.HasSuffix(name, ".jpg") {
			t.Fatalf("localFilename=%v", got["localFilename"])
		}
	})

	t.Run("enhanced response (video)", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`(?s)FROM media_file.*WHERE user_id = \? AND file_md5 = \?.*LIMIT 1`).
			WithArgs("u1", sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(`INSERT INTO media_file`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		uploadRoot := t.TempDir()
		ffmpegOK := writeExecutable(t, "ffmpeg-ok", `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
echo poster > "$out"
exit 0
`)
		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.mp4", "video/mp4", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{
			fileStorage: &FileStorageService{db: db, baseUploadAbs: uploadRoot},
			imageServer: NewImageServerService("example.com", "9003"),
			imageCache:  NewImageCacheService(),
			mediaUpload: &MediaUploadService{db: db},
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					_, err := io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(`{"state":"OK","msg":"remote.mp4"}`)),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				}),
			},
		}
		app.cfg.FFmpegPath = ffmpegOK

		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
		got := decodeJSONBody(t, rr.Body)
		if got["state"] != "OK" || got["msg"] != "remote.mp4" || got["port"] != "8006" {
			t.Fatalf("resp=%v", got)
		}
		if name, _ := got["localFilename"].(string); !strings.HasSuffix(name, ".mp4") {
			t.Fatalf("localFilename=%v", got["localFilename"])
		}
		if posterURL, _ := got["posterUrl"].(string); posterURL == "" || !strings.HasPrefix(posterURL, "/upload/videos/") || !strings.HasSuffix(posterURL, ".poster.jpg") {
			t.Fatalf("posterUrl=%v", got["posterUrl"])
		}
		if posterLocalPath, _ := got["posterLocalPath"].(string); posterLocalPath == "" || !strings.HasPrefix(posterLocalPath, "/videos/") || !strings.HasSuffix(posterLocalPath, ".poster.jpg") {
			t.Fatalf("posterLocalPath=%v", got["posterLocalPath"])
		} else {
			full := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocalPath, "/")))
			if fi, err := os.Stat(full); err != nil || fi.IsDir() || fi.Size() == 0 {
				t.Fatalf("expected poster file exists: %s err=%v", full, err)
			}
		}
	})

	t.Run("SaveFile error", func(t *testing.T) {
		uploadRootFile := filepath.Join(t.TempDir(), "uploadRootFile")
		if err := os.WriteFile(uploadRootFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/uploadMedia", "file", "a.jpg", "image/jpeg", []byte("x"), map[string]string{
			"userid": "u1",
		})

		app := &App{fileStorage: &FileStorageService{db: db, baseUploadAbs: uploadRootFile}}
		rr := httptest.NewRecorder()
		app.handleUploadMedia(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})
}
