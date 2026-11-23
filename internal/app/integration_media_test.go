package app

import (
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func integrationMediaColumns() []string {
	return []string{
		"source", "local_path", "local_filename", "original_filename",
		"file_size", "file_type", "file_extension", "file_md5", "media_width", "media_height",
		"sort_time", "sec_user_id",
	}
}

func TestParseIntegrationMediaQuery_DefaultsAndValidation(t *testing.T) {
	query, err := parseIntegrationMediaQuery(map[string][]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Page != 1 || query.PageSize != 60 || query.Source != "all" || query.Kind != "all" {
		t.Fatalf("unexpected defaults: %+v", query)
	}

	query, err = parseIntegrationMediaQuery(map[string][]string{
		"page":     {"2"},
		"pageSize": {"999"},
		"source":   {"Douyin"},
		"kind":     {"VIDEO"},
		"q":        {" Hello "},
		"from":     {"2026-01-02"},
		"to":       {"2026-01-03"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Page != 2 || query.PageSize != 200 || query.Source != "douyin" || query.Kind != "video" {
		t.Fatalf("unexpected parsed query: %+v", query)
	}
	if query.Keyword != "Hello" {
		t.Fatalf("keyword=%q", query.Keyword)
	}
	if query.From == nil || query.From.Format("2006-01-02") != "2026-01-02" {
		t.Fatalf("from=%v", query.From)
	}
	if query.To == nil || query.To.Format("2006-01-02") != "2026-01-04" {
		t.Fatalf("to should be exclusive next day, got %v", query.To)
	}

	bad := []map[string][]string{
		{"page": {"0"}},
		{"page": {"x"}},
		{"pageSize": {"0"}},
		{"source": {"other"}},
		{"kind": {"file"}},
		{"from": {"2026/01/02"}},
		{"to": {"bad"}},
		{"q": {strings.Repeat("a", integrationMediaMaxQueryLength+1)}},
	}
	for _, values := range bad {
		if _, err := parseIntegrationMediaQuery(values); err == nil {
			t.Fatalf("expected error for %+v", values)
		}
	}
}

func TestIntegrationMediaQueryWhere(t *testing.T) {
	from := time.Date(2026, 1, 2, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 1, 4, 0, 0, 0, 0, time.Local)
	query := integrationMediaQuery{
		Source:          "douyin",
		Kind:            "image",
		Keyword:         "Cat",
		From:            &from,
		To:              &to,
		DouyinSecUserID: "sec-1",
	}
	where, args := query.where()
	for _, fragment := range []string{
		"t.source = ?", "LOWER(t.file_extension) IN", "LOWER(t.original_filename) LIKE ?",
		"t.sort_time >= ?", "t.sort_time < ?", "t.sec_user_id = ?",
	} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("where=%q missing %q", where, fragment)
		}
	}
	wantArgs := []any{"douyin", "%cat%", from, to, "sec-1"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args=%v", args)
	}
	for i := range wantArgs {
		if args[i] != wantArgs[i] {
			t.Fatalf("args[%d]=%v want %v", i, args[i], wantArgs[i])
		}
	}

	allWhere, allArgs := (integrationMediaQuery{Source: "all", Kind: "all"}).where()
	if allWhere != " WHERE 1=1" || len(allArgs) != 0 {
		t.Fatalf("all where=%q args=%v", allWhere, allArgs)
	}
}

func TestIntegrationMediaPage_ListAndCount(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	base := t.TempDir()
	app := &App{db: wrapMySQLDB(sqlDB), fileStorage: &FileStorageService{baseUploadAbs: base}}

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM .*media_file.*douyin_media_file.* WHERE 1=1`).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT t\.source, t\.local_path.* FROM .*media_file.*douyin_media_file.* WHERE 1=1 ORDER BY .* LIMIT \? OFFSET \?`).
		WithArgs(60, 0).
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/images/a.jpg", "a.jpg", "A.jpg", 123, "image/jpeg", "jpg", "0123456789abcdef0123456789abcdef", 800, 600, time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local), nil))

	rows, total, err := app.integrationMediaPage(context.Background(), integrationMediaQuery{Page: 1, PageSize: 60, Source: "all", Kind: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("total=%d rows=%d", total, len(rows))
	}
	row := rows[0]
	if row.key() != "local:/images/a.jpg" || row.kind() != "image" {
		t.Fatalf("row=%+v", row)
	}
	if row.Width != 800 || row.Height != 600 {
		t.Fatalf("dimensions=%dx%d", row.Width, row.Height)
	}
	dto := app.integrationMediaDTO(row)
	if dto.Date != "2026-01-02T03:04:05" || !dto.Supported || dto.HasPoster {
		t.Fatalf("dto=%+v", dto)
	}
}

func TestIntegrationMediaRowByKey(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	app := &App{db: wrapMySQLDB(sqlDB), fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}

	mock.ExpectQuery(`(?s)SELECT 'local' AS source.*FROM media_file WHERE local_path = \? LIMIT 1`).
		WithArgs("/images/a.jpg").
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/images/a.jpg", "a.jpg", "A.jpg", 10, "image/jpeg", "jpg", "0123456789abcdef0123456789abcdef", nil, nil, time.Now(), nil))
	row, err := app.integrationMediaRowByKey(context.Background(), "local:/images/a.jpg")
	if err != nil || row.LocalPath != "/images/a.jpg" {
		t.Fatalf("row=%+v err=%v", row, err)
	}

	if _, err := app.integrationMediaRowByKey(context.Background(), "bad"); err == nil {
		t.Fatalf("expected error for bad key")
	}
	if _, err := app.integrationMediaRowByKey(context.Background(), "other:/x"); err == nil {
		t.Fatalf("expected error for bad source")
	}

	mock.ExpectQuery(`(?s)SELECT 'douyin' AS source.*FROM douyin_media_file WHERE local_path = \? LIMIT 1`).
		WithArgs("/douyin/a.mp4").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := app.integrationMediaRowByKey(context.Background(), "douyin:/douyin/a.mp4"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestHandleIntegrationMediaList(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	app := &App{db: wrapMySQLDB(sqlDB), fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM .* WHERE 1=1 AND t\.source = \?`).
		WithArgs("local").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT t\.source.* WHERE 1=1 AND t\.source = \? ORDER BY .* LIMIT \? OFFSET \?`).
		WithArgs("local", 60, 0).
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/images/a.jpg", "a.jpg", "A.jpg", 10, "image/jpeg", "jpg", "0123456789abcdef0123456789abcdef", 100, 50, time.Now(), nil))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media?source=local", nil)
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items      []integrationMediaItem `json:"items"`
		Page       int                    `json:"page"`
		PageSize   int                    `json:"pageSize"`
		Total      int                    `json:"total"`
		TotalPages int                    `json:"totalPages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "local:/images/a.jpg" || body.TotalPages != 1 {
		t.Fatalf("body=%+v", body)
	}
}

func TestHandleIntegrationMediaList_BadQuery(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media?source=bad", nil)
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaList(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandleIntegrationMediaItem(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	app := &App{db: wrapMySQLDB(sqlDB), fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}

	mock.ExpectQuery(`(?s)SELECT 'local' AS source.*FROM media_file WHERE local_path = \? LIMIT 1`).
		WithArgs("/images/a.jpg").
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/images/a.jpg", "a.jpg", "A.jpg", 10, "image/jpeg", "jpg", "0123456789abcdef0123456789abcdef", 100, 50, time.Now(), nil))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media/item?key=local:/images/a.jpg", nil)
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaItem(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleIntegrationMediaFile_Range(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "images", "a.jpg"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := &App{db: wrapMySQLDB(sqlDB), fileStorage: &FileStorageService{baseUploadAbs: base}}

	mock.ExpectQuery(`(?s)SELECT 'local' AS source.*FROM media_file WHERE local_path = \? LIMIT 1`).
		WithArgs("/images/a.jpg").
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/images/a.jpg", "a.jpg", "A.jpg", 10, "image/jpeg", "jpg", nil, nil, nil, time.Now(), nil))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media/file?key=local:/images/a.jpg", nil)
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaFile(rec, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "0123" {
		t.Fatalf("body=%q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Range"); !strings.HasPrefix(got, "bytes 0-3/") {
		t.Fatalf("content-range=%q", got)
	}
}

func TestHandleIntegrationMediaThumb_ImageAndETag(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJPEG(t, filepath.Join(base, "images", "a.jpg"), 40, 30)
	app := &App{
		db:                wrapMySQLDB(sqlDB),
		fileStorage:       &FileStorageService{baseUploadAbs: base},
		integrationThumbs: newIntegrationThumbCache(t.TempDir()),
	}

	expectLocalLookup(mock, "/images/a.jpg")
	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media/thumb?key=local:/images/a.jpg&size=s160", nil)
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaThumb(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("status=%d content-type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing etag")
	}

	expectLocalLookup(mock, "/images/a.jpg")
	req = httptest.NewRequest(http.MethodGet, "/api/integration/v1/media/thumb?key=local:/images/a.jpg&size=s160", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.handleIntegrationMediaThumb(rec, req)
	if rec.Code != http.StatusNotModified {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandleIntegrationMediaThumb_MissingPosterReturns404(t *testing.T) {
	sqlDB, mock, cleanup := newSQLMock(t)
	defer cleanup()
	app := &App{
		db:                wrapMySQLDB(sqlDB),
		fileStorage:       &FileStorageService{baseUploadAbs: t.TempDir()},
		integrationThumbs: newIntegrationThumbCache(t.TempDir()),
	}
	mock.ExpectQuery(`(?s)SELECT 'local' AS source.*FROM media_file WHERE local_path = \? LIMIT 1`).
		WithArgs("/videos/a.mp4").
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", "/videos/a.mp4", "a.mp4", "a.mp4", 10, "video/mp4", "mp4", nil, nil, nil, time.Now(), nil))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media/thumb?key=local:/videos/a.mp4", nil)
	rec := httptest.NewRecorder()
	app.handleIntegrationMediaThumb(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandleIntegrationMediaThumb_Validation(t *testing.T) {
	app := &App{integrationThumbs: newIntegrationThumbCache(t.TempDir())}
	cases := []string{
		"/api/integration/v1/media/thumb",
		"/api/integration/v1/media/thumb?key=local:/x.jpg&size=xxl",
	}
	for _, target := range cases {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		app.handleIntegrationMediaThumb(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", target, rec.Code)
		}
	}
}

func expectLocalLookup(mock sqlmock.Sqlmock, localPath string) {
	mock.ExpectQuery(`(?s)SELECT 'local' AS source.*FROM media_file WHERE local_path = \? LIMIT 1`).
		WithArgs(localPath).
		WillReturnRows(sqlmock.NewRows(integrationMediaColumns()).
			AddRow("local", localPath, filepath.Base(localPath), filepath.Base(localPath), 100, "image/jpeg", "jpg", nil, nil, nil, time.Now(), nil))
}

func writeTestJPEG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRegularUnderRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, _, err := openRegularUnder(root, link); err == nil {
		t.Fatalf("symlink escape accepted")
	}
	regular := filepath.Join(root, "ok.txt")
	if err := os.WriteFile(regular, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, info, err := openRegularUnder(root, regular)
	if err != nil || info.Size() != 2 {
		t.Fatalf("regular file rejected: %v", err)
	}
	_ = file.Close()
}

func TestOpenRegularUnderRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if _, _, err := openRegularUnder(root, root); err == nil {
		t.Fatalf("root directory accepted as file")
	}
	if _, _, err := openRegularUnder("", filepath.Join(root, "x")); err == nil {
		t.Fatalf("empty root accepted")
	}
}
