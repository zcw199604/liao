package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMtPhotoFolderFavoriteService_UncoveredBranches(t *testing.T) {
	t.Run("ListWithOptions tags_json null uses empty slice", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).AddRow(
			int64(1), int64(1), "n", "/p", nil, nil, nil, sql.NullTime{}, sql.NullTime{},
		)
		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WillReturnRows(rows)

		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		out, err := svc.ListWithOptions(context.Background(), MtPhotoFolderFavoriteListOptions{})
		if err != nil {
			t.Fatalf("ListWithOptions err=%v", err)
		}
		if len(out) != 1 || len(out[0].Tags) != 0 {
			t.Fatalf("out=%+v", out)
		}
	})

	t.Run("findByFolderID non-ErrNoRows returns error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM mtphoto_folder_favorite`).
			WithArgs(int64(9)).
			WillReturnError(errors.New("query failed"))

		svc := NewMtPhotoFolderFavoriteService(wrapMySQLDB(db))
		if _, err := svc.findByFolderID(context.Background(), 9); err == nil {
			t.Fatalf("expected query error")
		}
	})
}

