package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type errReadCloserMediaUpload struct{}

func (errReadCloserMediaUpload) Read(_ []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserMediaUpload) Close() error               { return nil }

func buildTinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestMediaUploadHelpers_WebPBranches(t *testing.T) {
	t.Run("convertImageToJPEG decode error", func(t *testing.T) {
		if _, err := convertImageToJPEG([]byte("not-image")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("convertImageToJPEG success", func(t *testing.T) {
		jpegBytes, err := convertImageToJPEG(buildTinyPNG(t))
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(jpegBytes) == 0 {
			t.Fatalf("empty jpeg bytes")
		}
	})

	t.Run("shouldRetryWebPAsJPEG branches", func(t *testing.T) {
		if shouldRetryWebPAsJPEG("/images/a.png", "a.png", `{"code":-3}`) {
			t.Fatalf("non-webp should not retry")
		}
		if shouldRetryWebPAsJPEG("/images/a.webp", "a.webp", "") {
			t.Fatalf("empty upstream response should not retry")
		}
		if !shouldRetryWebPAsJPEG("/images/a.webp", "a.webp", `{"state":"fail","code":-3,"msg":"bitmap..ctor"}`) {
			t.Fatalf("expected retry for fail/code=-3")
		}
		if shouldRetryWebPAsJPEG("/images/a.webp", "a.webp", `{"state":"ok"}`) {
			t.Fatalf("state ok should not retry")
		}
		if !shouldRetryWebPAsJPEG("/images/a.webp", "a.webp", `xx "code":-3 xx`) {
			t.Fatalf("fallback text matching should retry")
		}
	})

	t.Run("isLikelyWebPFile and rewriteFilenameExt", func(t *testing.T) {
		if !isLikelyWebPFile("/images/a.webp?x=1", "") {
			t.Fatalf("expected true")
		}
		if !isLikelyWebPFile("", "a.webp") {
			t.Fatalf("expected true")
		}
		if isLikelyWebPFile("/images/a.png", "a.png") {
			t.Fatalf("expected false")
		}

		if got := rewriteFilenameExt("", ""); got != "upload.jpg" {
			t.Fatalf("got=%q", got)
		}
		if got := rewriteFilenameExt("abc", ".png"); got != "abc.png" {
			t.Fatalf("got=%q", got)
		}
		if got := rewriteFilenameExt("a.webp", "jpg"); got != "a.jpg" {
			t.Fatalf("got=%q", got)
		}
	})
}

func TestMediaUploadService_UploadLocalFileToImageServer_Branches(t *testing.T) {
	ctx := context.Background()

	t.Run("Do error", func(t *testing.T) {
		svc := &MediaUploadService{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network fail")
		})}}
		if _, err := svc.uploadLocalFileToImageServer(ctx, "http://example.com/upload", "example.com:9003", "ref", "ua", "", "a.jpg", []byte("x")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read body error", func(t *testing.T) {
		svc := &MediaUploadService{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: errReadCloserMediaUpload{}, Header: make(http.Header)}, nil
		})}}
		if _, err := svc.uploadLocalFileToImageServer(ctx, "http://example.com/upload", "example.com:9003", "ref", "ua", "", "a.jpg", []byte("x")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("non-2xx response", func(t *testing.T) {
		svc := &MediaUploadService{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: io.NopCloser(strings.NewReader("bad")), Header: make(http.Header)}, nil
		})}}
		if _, err := svc.uploadLocalFileToImageServer(ctx, "http://example.com/upload", "example.com:9003", "ref", "ua", "", "a.jpg", []byte("x")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success with headers", func(t *testing.T) {
		svc := &MediaUploadService{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Host") != "example.com" {
				t.Fatalf("host header=%q", req.Header.Get("Host"))
			}
			if req.Header.Get("Cookie") != "c=1" {
				t.Fatalf("cookie=%q", req.Header.Get("Cookie"))
			}
			_, _ = io.ReadAll(req.Body)
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(`{"state":"OK"}`)), Header: make(http.Header)}, nil
		})}}
		got, err := svc.uploadLocalFileToImageServer(ctx, "http://example.com/upload", "example.com:9003", "ref", "ua", "c=1", "a.jpg", []byte("x"))
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if !strings.Contains(string(got), `"OK"`) {
			t.Fatalf("got=%q", string(got))
		}
	})
}

func TestMediaUploadService_GetAllUploadImagesWithDetailsBySource_Branches(t *testing.T) {
	t.Run("source local with extended columns and video poster", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		localPath := "/videos/a.mp4"
		mock.ExpectQuery(`FROM media_file\s*ORDER BY update_time DESC\s*LIMIT \? OFFSET \?`).
			WithArgs(10, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
				"sec_user_id", "detail_id", "author_unique_id", "author_name", "source",
			}).AddRow(
				"a.mp4", "a.mp4", localPath, int64(11), "video/mp4", "mp4", now, sql.NullTime{Time: now, Valid: true},
				sql.NullString{String: "sec1", Valid: true},
				sql.NullString{String: "d1", Valid: true},
				sql.NullString{String: "author_u", Valid: true},
				sql.NullString{String: "author", Valid: true},
				sql.NullString{String: "local", Valid: true},
			))

		uploadRoot := t.TempDir()
		posterLocal := buildVideoPosterLocalPath(localPath)
		posterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocal, "/")))
		if err := os.MkdirAll(filepath.Dir(posterAbs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(posterAbs, []byte("poster"), 0o644); err != nil {
			t.Fatalf("write poster: %v", err)
		}

		svc := &MediaUploadService{db: wrapMySQLDB(db), serverPort: 8080, fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
		out, err := svc.GetAllUploadImagesWithDetailsBySource(context.Background(), 1, 10, "", "local", "")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(out) != 1 {
			t.Fatalf("len=%d", len(out))
		}
		if out[0].Source != "local" || out[0].DouyinSecUserID != "sec1" || out[0].PosterURL == "" {
			t.Fatalf("out=%+v", out[0])
		}
	})

	t.Run("source douyin with sec_user_id filter", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		mock.ExpectQuery(`FROM douyin_media_file WHERE sec_user_id = \? ORDER BY update_time DESC LIMIT \? OFFSET \?`).
			WithArgs("sec2", 5, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
				"sec_user_id", "detail_id", "author_unique_id", "author_name", "source",
			}).AddRow(
				"b.jpg", "b.jpg", "/images/b.jpg", int64(2), "image/jpeg", "jpg", now, sql.NullTime{Valid: false},
				sql.NullString{String: "sec2", Valid: true},
				sql.NullString{String: "detail2", Valid: true},
				sql.NullString{String: "au2", Valid: true},
				sql.NullString{String: "name2", Valid: true},
				sql.NullString{String: "douyin", Valid: true},
			))

		svc := &MediaUploadService{db: wrapMySQLDB(db), serverPort: 8080}
		out, err := svc.GetAllUploadImagesWithDetailsBySource(context.Background(), 1, 5, "", "douyin", "sec2")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(out) != 1 || out[0].Source != "douyin" || out[0].DouyinAuthorName != "name2" {
			t.Fatalf("out=%+v", out)
		}
	})

	t.Run("source all postgres branch", func(t *testing.T) {
		t.Setenv("TEST_DB_DIALECT", "postgres")
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()
		mock.ExpectQuery(`NULL::varchar\(128\) AS sec_user_id`).
			WithArgs(3, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
			}).AddRow("c.png", "c.png", "/images/c.png", int64(1), "image/png", "png", now, sql.NullTime{Valid: false}))

		svc := &MediaUploadService{db: wrapMySQLDB(db), serverPort: 8080}
		out, err := svc.GetAllUploadImagesWithDetailsBySource(context.Background(), 1, 3, "", "all", "")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(out) != 1 || out[0].LocalFilename != "c.png" {
			t.Fatalf("out=%+v", out)
		}
	})

	t.Run("extended scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM media_file\s*ORDER BY update_time DESC\s*LIMIT \? OFFSET \?`).
			WithArgs(10, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"local_filename", "original_filename", "local_path", "file_size", "file_type", "file_extension", "upload_time", "update_time",
				"sec_user_id", "detail_id", "author_unique_id", "author_name", "source",
			}).AddRow(
				"x", "x", "/images/x", "bad-size", "image/jpeg", "jpg", time.Now(), sql.NullTime{Valid: false},
				sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{},
			))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if _, err := svc.GetAllUploadImagesWithDetailsBySource(context.Background(), 1, 10, "", "local", ""); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestMediaUploadService_MoreCountAndUpdateBranches(t *testing.T) {
	t.Run("GetAllUploadImagesCountBySource all second query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_file`).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM douyin_media_file`).
			WillReturnError(errors.New("count fail"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if _, err := svc.GetAllUploadImagesCountBySource(context.Background(), "all", ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("updateTimeByLocalPathIgnoreUser includes douyin update", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectExec(`UPDATE media_file SET update_time = \? WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg(), "/images/x.png").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE douyin_media_file SET update_time = \? WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg(), "/images/x.png").
			WillReturnResult(sqlmock.NewResult(0, 2))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if got := svc.updateTimeByLocalPathIgnoreUser(context.Background(), "/images/x.png", time.Now()); got != 3 {
			t.Fatalf("got=%d", got)
		}
	})
}
