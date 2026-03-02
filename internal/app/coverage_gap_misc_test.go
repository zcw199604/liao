package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestFavoriteServiceAdd_NilDBBranch(t *testing.T) {
	var svc *FavoriteService
	if _, err := svc.Add(context.Background(), "i1", "u1", "n1"); err == nil {
		t.Fatalf("expected db-not-initialized error")
	}
}

func TestSaveFileFromReaderInSubdir_EmptySubdirUsesDefaultPath(t *testing.T) {
	s := &FileStorageService{baseUploadAbs: t.TempDir()}
	localPath, fileSize, md5Value, err := s.SaveFileFromReaderInSubdir("a.jpg", "image/jpeg", bytes.NewReader([]byte("abc")), "   ")
	if err != nil {
		t.Fatalf("SaveFileFromReaderInSubdir err=%v", err)
	}
	if strings.TrimSpace(localPath) == "" || !strings.Contains(localPath, "/images/") {
		t.Fatalf("localPath=%q", localPath)
	}
	if fileSize != 3 || strings.TrimSpace(md5Value) == "" {
		t.Fatalf("fileSize=%d md5=%q", fileSize, md5Value)
	}
}

func TestFindStoredMediaFileByLocalFilename_BothMissBranch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM media_file\s*WHERE local_filename = \?`).
		WithArgs("miss.jpg", "u1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM douyin_media_file\s*WHERE local_filename = \?`).
		WithArgs("miss.jpg", "u1").
		WillReturnError(sql.ErrNoRows)

	svc := NewMediaUploadService(wrapMySQLDB(db), 0, nil, nil, nil)
	got, err := svc.findStoredMediaFileByLocalFilename(context.Background(), "miss.jpg", "u1")
	if err != nil {
		t.Fatalf("findStoredMediaFileByLocalFilename err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}
}

func TestMtPhotoCalculateMD5FromSupportedLocalPath_ExtraErrorBranches(t *testing.T) {
	app := &App{
		cfg:         config.Config{LspRoot: t.TempDir()},
		fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()},
	}

	if _, err := app.calculateMD5FromSupportedLocalPath("/lsp/"); err == nil || !strings.Contains(err.Error(), "localPath 非法") {
		t.Fatalf("expect lsp invalid path error, got=%v", err)
	}
	if _, err := app.calculateMD5FromSupportedLocalPath("/upload"); err == nil || !strings.Contains(err.Error(), "仅支持本地 /upload/images 或 /upload/videos 文件") {
		t.Fatalf("expect /upload root path error, got=%v", err)
	}
}

func TestHandleDouyinFavoriteList_EmptySliceFallbacks(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_user`).
		WillReturnRows(sqlmock.NewRows([]string{
			"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
			"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
		}))
	mock.ExpectQuery(`FROM douyin_favorite_aweme`).
		WillReturnRows(sqlmock.NewRows([]string{
			"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at",
		}))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	reqUsers := httptest.NewRequest("GET", "http://example.com/api/douyin/favorite/user/list", nil)
	rrUsers := httptest.NewRecorder()
	app.handleDouyinFavoriteUserList(rrUsers, reqUsers)
	if rrUsers.Code != 200 {
		t.Fatalf("user list status=%d body=%s", rrUsers.Code, rrUsers.Body.String())
	}
	var userBody map[string]any
	if err := json.Unmarshal(rrUsers.Body.Bytes(), &userBody); err != nil {
		t.Fatalf("unmarshal user body err=%v", err)
	}
	if items, ok := userBody["items"].([]any); !ok || items == nil || len(items) != 0 {
		t.Fatalf("user items=%v", userBody["items"])
	}

	reqAweme := httptest.NewRequest("GET", "http://example.com/api/douyin/favorite/aweme/list", nil)
	rrAweme := httptest.NewRecorder()
	app.handleDouyinFavoriteAwemeList(rrAweme, reqAweme)
	if rrAweme.Code != 200 {
		t.Fatalf("aweme list status=%d body=%s", rrAweme.Code, rrAweme.Body.String())
	}
	var awemeBody map[string]any
	if err := json.Unmarshal(rrAweme.Body.Bytes(), &awemeBody); err != nil {
		t.Fatalf("unmarshal aweme body err=%v", err)
	}
	if items, ok := awemeBody["items"].([]any); !ok || items == nil || len(items) != 0 {
		t.Fatalf("aweme items=%v", awemeBody["items"])
	}
}

func TestMtPhotoFolderFavorite_FindByFolderID_NullTagsBranch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
		WithArgs(int64(101)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).AddRow(
			int64(1), int64(101), "相册", "/lsp/a", sql.NullString{Valid: false}, sql.NullString{Valid: false}, sql.NullString{Valid: false}, sql.NullTime{Valid: false}, sql.NullTime{Valid: false},
		))

	svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
	got, err := svc.findByFolderID(context.Background(), 101)
	if err != nil {
		t.Fatalf("findByFolderID err=%v", err)
	}
	if got == nil || got.Tags == nil || len(got.Tags) != 0 {
		t.Fatalf("got=%+v", got)
	}
}
