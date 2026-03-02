package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDouyinFavoriteService_UpsertUserAwemes_UncoveredBranches(t *testing.T) {
	t.Run("existing rows scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u1", "a1").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}).AddRow(nil))

		if _, err := svc.UpsertUserAwemes(context.Background(), "u1", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a1"}}); err == nil {
			t.Fatalf("expected scan error")
		}
	})

	t.Run("min sort query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u2", "a2").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))
		mock.ExpectQuery(`SELECT MIN\(sort_order\)\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u2").
			WillReturnError(errors.New("min sort query failed"))

		if _, err := svc.UpsertUserAwemes(context.Background(), "u2", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a2"}}); err == nil {
			t.Fatalf("expected min sort query error")
		}
	})

	t.Run("upsert exec error rollback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u3", "a3").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))
		mock.ExpectQuery(`SELECT MIN\(sort_order\)\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u3").
			WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
			WillReturnError(errors.New("upsert failed"))
		mock.ExpectRollback()

		if _, err := svc.UpsertUserAwemes(context.Background(), "u3", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a3"}}); err == nil {
			t.Fatalf("expected upsert error")
		}
	})

	t.Run("post-upsert update error rollback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u4", "a4").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))
		mock.ExpectQuery(`SELECT MIN\(sort_order\)\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u4").
			WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE douyin_favorite_user_aweme`).
			WillReturnError(errors.New("update failed"))
		mock.ExpectRollback()

		if _, err := svc.UpsertUserAwemes(context.Background(), "u4", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a4"}}); err == nil {
			t.Fatalf("expected update error")
		}
	})
}

func TestDouyinFavoriteService_ListUserAwemes_CountDefaultBranch(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
		WithArgs("u1", 21, 0). // count<=0 -> defaults to 20, query uses count+1
		WillReturnRows(sqlmock.NewRows([]string{
			"aweme_id", "type", "description", "cover_url", "downloads",
			"is_pinned", "pinned_rank", "pinned_at", "publish_at", "crawled_at", "last_seen_at",
			"status", "author_unique_id", "author_name", "created_at", "updated_at",
		}).AddRow(
			"a1", "video", "desc", "cover", `["https://example.com/v.mp4"]`,
			false, nil, nil, nil, now, now, "normal", "uid", "name", now, now,
		))

	svc := NewDouyinFavoriteService(wrapMySQLDB(db))
	out, next, more, err := svc.ListUserAwemes(context.Background(), "u1", 0, 0)
	if err != nil {
		t.Fatalf("ListUserAwemes err=%v", err)
	}
	if len(out) != 1 || next != 1 || more {
		t.Fatalf("out=%v next=%d more=%v", out, next, more)
	}
}

