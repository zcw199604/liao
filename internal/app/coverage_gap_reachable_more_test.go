package app

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/database"
)

func TestUploadRequestBuilders_CreateFormFileErrorBranches(t *testing.T) {
	mediaSvc := &MediaUploadService{httpClient: &http.Client{Timeout: time.Second}}
	if _, err := mediaSvc.uploadLocalFileToImageServer(
		context.Background(),
		"http://example.com/upload",
		"example.com:80",
		"http://example.com",
		"ua",
		"",
		"bad\r\nname.jpg",
		[]byte("x"),
	); err == nil {
		t.Fatalf("expected create form file error for media upload")
	}

	app := &App{httpClient: &http.Client{Timeout: time.Second}}
	if _, err := app.uploadBytesToUpstream(
		context.Background(),
		"http://example.com/upload",
		"example.com:80",
		"bad\r\nname.jpg",
		[]byte("x"),
		"",
		"http://example.com",
		"ua",
	); err == nil {
		t.Fatalf("expected create form file error for upstream upload")
	}
}

func TestHandleDownloadImgUpload_RequestBuildErrorBranch(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()

	sys := NewSystemConfigService(wrapMySQLDB(db))
	sys.loaded = true
	sys.cached = SystemConfig{
		ImagePortMode:         ImagePortModeFixed,
		ImagePortFixed:        "9006",
		ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
	}

	app := &App{
		httpClient:   &http.Client{Timeout: time.Second},
		systemConfig: sys,
		imageServer:  NewImageServerService("bad host", "9006"),
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
	rr := httptest.NewRecorder()
	app.handleDownloadImgUpload(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestImageHashService_FindSimilarByPHash_PostgresQueryBranch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)length\(replace\(\(\(phash # \$1\)::bit\(64\)\)::text, '0', ''\)\) AS distance`).
		WithArgs(int64(7), int64(7), 3, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at", "distance",
		}))

	svc := NewImageHashService(database.Wrap(db, database.PostgresDialect{}))
	got, err := svc.FindSimilarByPHash(context.Background(), 7, 3, 10)
	if err != nil {
		t.Fatalf("FindSimilarByPHash err=%v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}

func TestDeleteMediaByPath_DouyinDeleteErrorAndVideoPosterCleanup(t *testing.T) {
	t.Run("douyin source delete error branch", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Now()
		// local table miss（/a -> 三个候选）
		for i := 0; i < 3; i++ {
			mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
				WithArgs(sqlmock.AnyArg()).
				WillReturnError(sql.ErrNoRows)
		}
		// douyin table 首次命中
		mock.ExpectQuery(`(?s)FROM douyin_media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(7), "", "a.jpg", "a.jpg", "", "", "/a",
				int64(1), "image/jpeg", "jpg", sql.NullString{Valid: false}, now, sql.NullTime{Valid: false},
			))

		for i := 0; i < 4; i++ {
			mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
				WithArgs(sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
		mock.ExpectExec(`DELETE FROM douyin_media_file WHERE local_path = \?`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(context.DeadlineExceeded)

		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, err := svc.DeleteMediaByPath(context.Background(), "", "/a"); err == nil {
			t.Fatalf("expected delete error")
		}
	})

	t.Run("video delete triggers poster cleanup", func(t *testing.T) {
		baseUpload := t.TempDir()
		videoLocalPath := "/videos/v.mp4"
		videoAbs := filepath.Join(baseUpload, "videos", "v.mp4")
		posterAbs := filepath.Join(baseUpload, "videos", "v.poster.jpg")
		if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
			t.Fatalf("mkdir err=%v", err)
		}
		if err := os.WriteFile(videoAbs, []byte("video"), 0o644); err != nil {
			t.Fatalf("write video err=%v", err)
		}
		if err := os.WriteFile(posterAbs, []byte("poster"), 0o644); err != nil {
			t.Fatalf("write poster err=%v", err)
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		now := time.Now()

		mock.ExpectQuery(`(?s)FROM media_file.*WHERE local_path = \?.*ORDER BY id LIMIT 1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "original_filename", "local_filename", "remote_filename", "remote_url", "local_path",
				"file_size", "file_type", "file_extension", "file_md5", "upload_time", "update_time",
			}).AddRow(
				int64(8), "u1", "v.mp4", "v.mp4", "", "", videoLocalPath,
				int64(10), "video/mp4", "mp4", sql.NullString{Valid: false}, now, sql.NullTime{Valid: false},
			))

		for i := 0; i < 4; i++ {
			mock.ExpectExec(`DELETE FROM media_send_log WHERE local_path = \?`).
				WithArgs(sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
		for i := 0; i < 4; i++ {
			mock.ExpectExec(`DELETE FROM media_file WHERE local_path = \?`).
				WithArgs(sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: baseUpload}}
		got, err := svc.DeleteMediaByPath(context.Background(), "u1", videoLocalPath)
		if err != nil {
			t.Fatalf("DeleteMediaByPath err=%v", err)
		}
		if !got.FileDeleted {
			t.Fatalf("expected FileDeleted=true, got=%+v", got)
		}
		if _, err := os.Stat(posterAbs); !os.IsNotExist(err) {
			t.Fatalf("poster should be deleted, stat err=%v", err)
		}
	})
}

func TestExtractDouyinAccountItems_LivePhotoTypeLabelFallbackBranch(t *testing.T) {
	downloader := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{
				"aweme_id":  "a1",
				"type":      "",
				"downloads": []any{"https://example.com/pic.jpg", "https://example.com/aweme/v1/play/?video_id=v1"},
			},
		},
	}

	items := extractDouyinAccountItems(downloader, "sec-1", data)
	if len(items) != 1 || strings.TrimSpace(items[0].Key) == "" {
		t.Fatalf("items=%+v", items)
	}
	cached, ok := downloader.GetCachedDetail(items[0].Key)
	if !ok || cached == nil {
		t.Fatalf("cache not found for key=%q", items[0].Key)
	}
	if cached.Type != "实况" {
		t.Fatalf("cached type=%q", cached.Type)
	}
}
