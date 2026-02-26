package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestParseJSONStringArray_MoreBranches(t *testing.T) {
	if got := parseJSONStringArray(sql.NullString{Valid: true, String: " "}); got != nil {
		t.Fatalf("empty should be nil: %v", got)
	}
	if got := parseJSONStringArray(sql.NullString{Valid: true, String: "{"}); got != nil {
		t.Fatalf("invalid json should be nil: %v", got)
	}
}

func TestDouyinFavoriteService_UpsertUserAwemes_MoreBranches(t *testing.T) {
	t.Run("service/db guards and empty inputs", func(t *testing.T) {
		svc := NewDouyinFavoriteService(nil)
		if _, err := svc.UpsertUserAwemes(context.Background(), "u", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a"}}); err == nil {
			t.Fatalf("expected db error")
		}

		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc = NewDouyinFavoriteService(wrapMySQLDB(db))

		if added, err := svc.UpsertUserAwemes(context.Background(), " ", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a"}}); err != nil || added != 0 {
			t.Fatalf("added=%d err=%v", added, err)
		}
		if added, err := svc.UpsertUserAwemes(context.Background(), "u", nil); err != nil || added != 0 {
			t.Fatalf("added=%d err=%v", added, err)
		}
		if added, err := svc.UpsertUserAwemes(context.Background(), "u", []DouyinFavoriteUserAwemeUpsert{{AwemeID: " "}}); err != nil || added != 0 {
			t.Fatalf("added=%d err=%v", added, err)
		}
	})

	t.Run("query rows and begin/commit branches", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		now := time.Now()
		rank := 3
		pinnedAt := now.Add(-time.Minute)
		publishAt := now.Add(-time.Hour)

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u1", "a1", "a2").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}).
				AddRow(" a1 "))
		mock.ExpectQuery(`SELECT MIN\(sort_order\)\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(int64(10)))
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE douyin_favorite_user_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE douyin_favorite_user_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().
			WillReturnError(errors.New("commit failed"))

		_, err := svc.UpsertUserAwemes(context.Background(), "u1", []DouyinFavoriteUserAwemeUpsert{
			{
				AwemeID:        "a1",
				Type:           " video ",
				Desc:           " d1 ",
				CoverURL:       " c1 ",
				Downloads:      []string{"https://example.com/v1.mp4"},
				IsPinned:       true,
				PinnedRank:     &rank,
				PinnedAt:       &pinnedAt,
				PublishAt:      &publishAt,
				Status:         " ",
				AuthorUniqueID: " u ",
				AuthorName:     " n ",
			},
			{
				AwemeID:    "a2",
				Type:       "image",
				Desc:       "d2",
				CoverURL:   "c2",
				Downloads:  []string{"https://example.com/i2.jpg"},
				IsPinned:   false,
				PinnedRank: &rank,
				Status:     "ok",
			},
		})
		if err == nil {
			t.Fatalf("expected commit error")
		}
	})

	t.Run("existing rows scan/rowsErr and begin error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u1", "a1").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}).AddRow("a1").RowError(0, errors.New("row err")))
		if _, err := svc.UpsertUserAwemes(context.Background(), "u1", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a1"}}); err == nil {
			t.Fatalf("expected rows error")
		}

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u2", "a2").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))
		mock.ExpectQuery(`SELECT MIN\(sort_order\)\s+FROM douyin_favorite_user_aweme`).
			WithArgs("u2").
			WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
		mock.ExpectBegin().WillReturnError(errors.New("begin err"))
		if _, err := svc.UpsertUserAwemes(context.Background(), "u2", []DouyinFavoriteUserAwemeUpsert{{AwemeID: "a2"}}); err == nil {
			t.Fatalf("expected begin error")
		}
	})
}

func TestDouyinFavoriteService_ListUserAwemes_MoreBranches(t *testing.T) {
	t.Run("guards", func(t *testing.T) {
		svc := NewDouyinFavoriteService(nil)
		if _, _, _, err := svc.ListUserAwemes(context.Background(), "u", 0, 1); err == nil {
			t.Fatalf("expected db error")
		}

		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc = NewDouyinFavoriteService(wrapMySQLDB(db))
		out, next, more, err := svc.ListUserAwemes(context.Background(), " ", 0, 1)
		if err != nil || out != nil || next != 0 || more {
			t.Fatalf("out=%v next=%d more=%v err=%v", out, next, more, err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
			WithArgs("u1", 51, 0).
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}).AddRow("a1"))

		if _, _, _, err := svc.ListUserAwemes(context.Background(), "u1", -1, 100); err == nil {
			t.Fatalf("expected scan error")
		}
	})

	t.Run("rowsErr and hasMore", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDouyinFavoriteService(wrapMySQLDB(db))

		now := time.Now()
		rowsWithErr := sqlmock.NewRows([]string{
			"aweme_id", "type", "description", "cover_url", "downloads",
			"is_pinned", "pinned_rank", "pinned_at", "publish_at", "crawled_at", "last_seen_at",
			"status", "author_unique_id", "author_name", "created_at", "updated_at",
		}).
			AddRow("a1", "video", "d", "c", `["u"]`, true, int64(1), now, now, now, now, "normal", "uid", "name", now, now).
			RowError(0, errors.New("row err"))

		mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
			WithArgs("u2", 2, 0).
			WillReturnRows(rowsWithErr)
		if _, _, _, err := svc.ListUserAwemes(context.Background(), "u2", 0, 1); err == nil {
			t.Fatalf("expected rowsErr")
		}

		rows := sqlmock.NewRows([]string{
			"aweme_id", "type", "description", "cover_url", "downloads",
			"is_pinned", "pinned_rank", "pinned_at", "publish_at", "crawled_at", "last_seen_at",
			"status", "author_unique_id", "author_name", "created_at", "updated_at",
		}).
			AddRow("a1", "video", "d1", "c1", `["u1"]`, true, int64(1), now, now, now, now, "normal", "uid1", "name1", now, now).
			AddRow("a2", "image", "d2", "c2", `["u2"]`, false, nil, nil, nil, now, now, "normal", "uid2", "name2", now, now)

		mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
			WithArgs("u3", 2, 0).
			WillReturnRows(rows)

		out, next, more, err := svc.ListUserAwemes(context.Background(), "u3", 0, 1)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(out) != 1 || out[0].AwemeID != "a1" || !more || next != 1 {
			t.Fatalf("out=%v next=%d more=%v", out, next, more)
		}
	})
}

func TestDouyinFavoriteService_UpdateUserAwemeDownloadsCover_EmptyIDs(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()
	svc := NewDouyinFavoriteService(wrapMySQLDB(db))

	if err := svc.UpdateUserAwemeDownloadsCover(context.Background(), " ", "a1", []string{"u"}, "c"); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := svc.UpdateUserAwemeDownloadsCover(context.Background(), "u1", " ", []string{"u"}, "c"); err != nil {
		t.Fatalf("err=%v", err)
	}
}
