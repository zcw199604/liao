package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMtPhotoFolderFavoriteHelpers_MoreBranches(t *testing.T) {
	t.Run("normalize tags errors", func(t *testing.T) {
		tooMany := make([]string, mtPhotoFolderFavoriteMaxTags+1)
		for i := range tooMany {
			tooMany[i] = fmt.Sprintf("x%d", i)
		}
		if _, err := normalizeMtPhotoFolderFavoriteTags(tooMany); err == nil {
			t.Fatalf("expected too many tags error")
		}

		if _, err := normalizeMtPhotoFolderFavoriteTags([]string{strings.Repeat("中", mtPhotoFolderFavoriteMaxTagLength+1)}); err == nil {
			t.Fatalf("expected tag length error")
		}
	})

	t.Run("parse tags empty/invalid/empty-normalized", func(t *testing.T) {
		if got := parseMtPhotoFolderFavoriteTags(" "); len(got) != 0 {
			t.Fatalf("empty parse=%v", got)
		}
		if got := parseMtPhotoFolderFavoriteTags("{"); len(got) != 0 {
			t.Fatalf("invalid parse=%v", got)
		}
		if got := parseMtPhotoFolderFavoriteTags(`[" ",""]`); len(got) != 0 {
			t.Fatalf("normalized parse=%v", got)
		}
	})
}

func TestMtPhotoFolderFavoriteService_validateUpsertInput_MoreBranches(t *testing.T) {
	svc := NewMtPhotoFolderFavoriteService(nil)
	if _, _, err := svc.validateUpsertInput(MtPhotoFolderFavoriteUpsertInput{FolderID: 1, FolderName: "a", FolderPath: "/a"}); err == nil {
		t.Fatalf("expected db not initialized error")
	}

	db, _, cleanup := newSQLMock(t)
	defer cleanup()
	svc = NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))

	cases := []MtPhotoFolderFavoriteUpsertInput{
		{FolderID: 0, FolderName: "a", FolderPath: "/a"},
		{FolderID: 1, FolderName: " ", FolderPath: "/a"},
		{FolderID: 1, FolderName: "a", FolderPath: " "},
		{FolderID: 1, FolderName: "a", FolderPath: "/a", CoverMD5: "bad"},
		{FolderID: 1, FolderName: "a", FolderPath: "/a", Tags: make([]string, mtPhotoFolderFavoriteMaxTags+1)},
		{FolderID: 1, FolderName: "a", FolderPath: "/a", Note: strings.Repeat("中", mtPhotoFolderFavoriteMaxNoteRunes+1)},
	}
	for i := range cases {
		for j := range cases[i].Tags {
			cases[i].Tags[j] = fmt.Sprintf("t%d", j)
		}
		if _, _, err := svc.validateUpsertInput(cases[i]); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}

func TestMtPhotoFolderFavoriteService_ListWithOptions_ErrorBranches(t *testing.T) {
	t.Run("service nil db", func(t *testing.T) {
		svc := NewMtPhotoFolderFavoriteService(nil)
		if _, err := svc.ListWithOptions(context.Background(), MtPhotoFolderFavoriteListOptions{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WillReturnError(errors.New("boom"))
		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		if _, err := svc.ListWithOptions(context.Background(), MtPhotoFolderFavoriteListOptions{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		rows := sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).AddRow("bad-id", 1, "a", "/a", nil, nil, nil, nil, nil)
		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WillReturnRows(rows)
		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		if _, err := svc.ListWithOptions(context.Background(), MtPhotoFolderFavoriteListOptions{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("rows error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		rows := sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).
			AddRow(int64(1), int64(1), "a", "/a", nil, nil, nil, sql.NullTime{}, sql.NullTime{}).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WillReturnRows(rows)
		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		if _, err := svc.ListWithOptions(context.Background(), MtPhotoFolderFavoriteListOptions{}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestMtPhotoFolderFavoriteService_Upsert_FindByFolderID_MoreBranches(t *testing.T) {
	t.Run("upsert exec error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectExec(`INSERT INTO mtphoto_folder_favorite`).
			WillReturnError(errors.New("boom"))

		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		_, err := svc.Upsert(context.Background(), MtPhotoFolderFavoriteUpsertInput{
			FolderID:   1,
			FolderName: "a",
			FolderPath: "/a",
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findByFolderID guards and no rows", func(t *testing.T) {
		svc := NewMtPhotoFolderFavoriteService(nil)
		if _, err := svc.findByFolderID(context.Background(), 1); err == nil {
			t.Fatalf("expected db error")
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc = NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		if _, err := svc.findByFolderID(context.Background(), 0); err == nil {
			t.Fatalf("expected folderId error")
		}
		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WithArgs(int64(2)).
			WillReturnError(sql.ErrNoRows)
		item, err := svc.findByFolderID(context.Background(), 2)
		if err != nil || item != nil {
			t.Fatalf("item=%v err=%v", item, err)
		}
	})
}
