package app

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
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

func TestHandleImportMtPhotoMedia_Success(t *testing.T) {
	// 1) 准备 LSP_ROOT（模拟 mtPhoto 返回的 /lsp/... 映射目录）
	lspRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(lspRoot, "tg"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	srcAbs := filepath.Join(lspRoot, "tg", "a.jpg")
	srcBytes := []byte("fake-jpeg")
	if err := os.WriteFile(srcAbs, srcBytes, 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	sum := md5.Sum(srcBytes)
	md5Value := hex.EncodeToString(sum[:])

	// 2) mock mtPhoto：login + filesInMD5
	mtSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/filesInMD5":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "filePath": "/lsp/tg/a.jpg"},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(mtSrv.Close)

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// import handler 会先全局查重（MD5，不按 user_id 分桶），覆盖 local + douyin。
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)

	// 本地落盘后，会再按“实际文件 MD5”全局查重一次，避免重复写文件导致孤儿文件。
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs(md5Value).
		WillReturnError(sql.ErrNoRows)

	// SaveUploadRecord 内部也会查重一次
	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE user_id = \? AND file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs("u1", md5Value).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec(`INSERT INTO media_file`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	fileStorage := NewFileStorageService(db)
	fileStorage.baseUploadAbs = t.TempDir()

	app := &App{
		cfg:         config.Config{LspRoot: lspRoot},
		fileStorage: fileStorage,
		mediaUpload: NewMediaUploadService(db, 8080, fileStorage, nil, mtSrv.Client()),
		mtPhoto:     NewMtPhotoService(mtSrv.URL, "u", "p", "", lspRoot, mtSrv.Client()),
	}

	form := url.Values{}
	form.Set("userid", "u1")
	form.Set("md5", md5Value)
	req := httptest.NewRequest(http.MethodPost, "http://api.local:8080/api/importMtPhotoMedia", bytes.NewReader([]byte(form.Encode())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	app.handleImportMtPhotoMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["state"] != "OK" {
		t.Fatalf("state=%v, want OK", resp["state"])
	}
	if resp["uploaded"] != false {
		t.Fatalf("uploaded=%v, want false", resp["uploaded"])
	}
	if strings.TrimSpace(toString(resp["localPath"])) == "" {
		t.Fatalf("localPath empty: %v", resp)
	}
	if strings.TrimSpace(toString(resp["localFilename"])) == "" {
		t.Fatalf("localFilename empty: %v", resp)
	}
}

func TestHandleImportMtPhotoMedia_PathTraversalRejected(t *testing.T) {
	lspRoot := t.TempDir()

	mtSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/filesInMD5":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "filePath": "/lsp/../etc/passwd"},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(mtSrv.Close)

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs("md5-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)FROM douyin_media_file\s+WHERE file_md5 = \?\s+ORDER BY id\s+LIMIT 1`).
		WithArgs("md5-1").
		WillReturnError(sql.ErrNoRows)

	fileStorage := NewFileStorageService(db)
	fileStorage.baseUploadAbs = t.TempDir()

	app := &App{
		cfg:         config.Config{LspRoot: lspRoot},
		fileStorage: fileStorage,
		mediaUpload: NewMediaUploadService(db, 8080, fileStorage, nil, mtSrv.Client()),
		mtPhoto:     NewMtPhotoService(mtSrv.URL, "u", "p", "", lspRoot, mtSrv.Client()),
	}

	form := url.Values{}
	form.Set("userid", "u1")
	form.Set("md5", "md5-1")
	req := httptest.NewRequest(http.MethodPost, "http://api.local:8080/api/importMtPhotoMedia", bytes.NewReader([]byte(form.Encode())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	app.handleImportMtPhotoMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestUploadAbsPathToUpstream_ReadsFile(t *testing.T) {
	upSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{"state": "OK", "msg": "x"})
	}))
	t.Cleanup(upSrv.Close)

	u, _ := url.Parse(upSrv.URL)

	app := &App{httpClient: upSrv.Client()}

	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "x.jpg")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	_, err := app.uploadAbsPathToUpstream(ctx, upSrv.URL, u.Host, p, "x.jpg", "", "http://ref", "ua")
	if err != nil {
		t.Fatalf("uploadAbsPathToUpstream error: %v", err)
	}
}
