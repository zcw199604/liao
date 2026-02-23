package app

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMtPhotoFolderFavoriteService_Upsert_Validate(t *testing.T) {
	svc := NewMtPhotoFolderFavoriteService(nil)

	_, err := svc.Upsert(context.Background(), MtPhotoFolderFavoriteUpsertInput{
		FolderID: 0,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestMtPhotoFolderFavoriteService_Upsert_OK(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	t.Cleanup(cleanup)

	db := wrapMySQLDB(rawDB)
	svc := NewMtPhotoFolderFavoriteService(db)

	now := time.Now()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO mtphoto_folder_favorite")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows := sqlmock.NewRows([]string{
		"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
	}).AddRow(1, 644, "我的照片", "/photo/我的照片", "e38c3a4e832e7e66538002287d9663b5", `["常用","人像"]`, "每周更新", now, now)
	mock.ExpectQuery(`SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at\s+FROM mtphoto_folder_favorite\s+WHERE folder_id = \?\s+LIMIT 1`).
		WithArgs(int64(644)).
		WillReturnRows(rows)

	item, err := svc.Upsert(context.Background(), MtPhotoFolderFavoriteUpsertInput{
		FolderID:   644,
		FolderName: " 我的照片 ",
		FolderPath: " /photo/我的照片 ",
		CoverMD5:   "e38c3a4e832e7e66538002287d9663b5",
		Tags:       []string{"常用", "人像", "常用"},
		Note:       " 每周更新 ",
	})
	if err != nil {
		t.Fatalf("Upsert error: %v", err)
	}
	if item == nil {
		t.Fatalf("item should not be nil")
	}
	if item.FolderID != 644 {
		t.Fatalf("folderId=%d, want 644", item.FolderID)
	}
	if len(item.Tags) != 2 {
		t.Fatalf("tags=%v, want 2 items", item.Tags)
	}
}

func TestMtPhotoFolderFavoriteService_List_OK(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	t.Cleanup(cleanup)

	db := wrapMySQLDB(rawDB)
	svc := NewMtPhotoFolderFavoriteService(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
	}).
		AddRow(1, 644, "我的照片", "/photo/我的照片", "e38c3a4e832e7e66538002287d9663b5", `["常用","人像"]`, "每周更新", now, now).
		AddRow(2, 545, "照片", "/photo/照片", nil, `invalid-json`, nil, now, now)
	mock.ExpectQuery(`SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at\s+FROM mtphoto_folder_favorite\s+ORDER BY updated_at DESC, id DESC`).WillReturnRows(rows)

	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	if strings.Join(items[0].Tags, ",") != "常用,人像" {
		t.Fatalf("tags=%v, want [常用 人像]", items[0].Tags)
	}
	if len(items[1].Tags) != 0 {
		t.Fatalf("invalid tags json should fallback empty, got %v", items[1].Tags)
	}
}

func TestMtPhotoFolderFavoriteService_Remove(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	t.Cleanup(cleanup)

	db := wrapMySQLDB(rawDB)
	svc := NewMtPhotoFolderFavoriteService(db)

	if err := svc.Remove(context.Background(), 0); err == nil {
		t.Fatalf("expected invalid folder id error")
	}

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM mtphoto_folder_favorite WHERE folder_id = ?")).
		WithArgs(int64(644)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.Remove(context.Background(), 644); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
}
