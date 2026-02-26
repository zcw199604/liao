package app

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDouyinFavoriteTags_UserSide_MoreBranches(t *testing.T) {
	t.Run("ListUserTags scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "sort_order", "cnt", "created_at", "updated_at"}).
				AddRow(int64(1), "tag", "bad-sort", int64(1), nil, nil))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListUserTags(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ListUserTags rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"id", "name", "sort_order", "cnt", "created_at", "updated_at"}).
			AddRow(int64(1), "tag", int64(1), int64(0), nil, nil).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`FROM douyin_favorite_user_tag`).WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListUserTags(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("AddUserTag max sort query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_user_tag`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.AddUserTag(context.Background(), "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findUserTagByID non noRows error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM douyin_favorite_user_tag t`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findUserTagByID(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ApplyUserTags set insert error rollback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
			WithArgs("u1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
			WithArgs("u1", int64(1), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"u1"}, []int64{1}, "set"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ApplyUserTags add insert error rollback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
			WithArgs("u1", int64(2), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"u1"}, []int64{2}, "add"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listUserTagIDsBySecUserID query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM douyin_favorite_user_tag_map`).
			WithArgs("u1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listUserTagIDsBySecUserID(context.Background(), "u1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listUserTagIDsBySecUserID scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM douyin_favorite_user_tag_map`).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow("bad"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listUserTagIDsBySecUserID(context.Background(), "u1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("fillUserTagIDs empty items", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()
		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.fillUserTagIDs(context.Background(), []DouyinFavoriteUser{}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("fillUserTagIDs listAll error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.fillUserTagIDs(context.Background(), []DouyinFavoriteUser{{SecUserID: "u1"}}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllUserTagIDs query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllUserTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllUserTagIDs scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnRows(sqlmock.NewRows([]string{"sec_user_id", "tag_id"}).AddRow("u1", "bad"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllUserTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllUserTagIDs rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		rows := sqlmock.NewRows([]string{"sec_user_id", "tag_id"}).
			AddRow("u1", int64(1)).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllUserTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ReorderUserTags begin tx error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ReorderUserTags(context.Background(), []int64{1}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestDouyinFavoriteTags_AwemeSide_MoreBranches(t *testing.T) {
	t.Run("ListAwemeTags scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "sort_order", "cnt", "created_at", "updated_at"}).
				AddRow(int64(1), "tag", "bad-sort", int64(1), nil, nil))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListAwemeTags(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ListAwemeTags rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		rows := sqlmock.NewRows([]string{"id", "name", "sort_order", "cnt", "created_at", "updated_at"}).
			AddRow(int64(1), "tag", int64(1), int64(0), nil, nil).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListAwemeTags(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("AddAwemeTag max sort query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.AddAwemeTag(context.Background(), "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("AddAwemeTag insert non-dup error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"MAX(sort_order)"}).AddRow(int64(2)))
		expectInsertReturningIDError(mock, `INSERT INTO douyin_favorite_aweme_tag`, errors.New("insert fail"), "x", int64(3), sqlmock.AnyArg(), sqlmock.AnyArg())

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.AddAwemeTag(context.Background(), "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("RemoveAwemeTag begin tx error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.RemoveAwemeTag(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findAwemeTagByID non noRows error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag t`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("db fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findAwemeTagByID(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ApplyAwemeTags default mode(set) insert error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_aweme_tag_map`).
			WithArgs("a1", int64(1), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{1}, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ApplyAwemeTags add insert error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_aweme_tag_map`).
			WithArgs("a1", int64(2), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{2}, "add"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ApplyAwemeTags remove delete error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map`).
			WithArgs("a1", int64(2)).
			WillReturnError(errors.New("delete fail"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{2}, "remove"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ReorderAwemeTags begin tx error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ReorderAwemeTags(context.Background(), []int64{1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAwemeTagIDsByID query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WithArgs("a1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAwemeTagIDsByID(context.Background(), "a1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAwemeTagIDsByID scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WithArgs("a1").
			WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow("bad"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAwemeTagIDsByID(context.Background(), "a1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("fillAwemeTagIDs empty items", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()
		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.fillAwemeTagIDs(context.Background(), []DouyinFavoriteAweme{}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("fillAwemeTagIDs listAll error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT aweme_id, tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.fillAwemeTagIDs(context.Background(), []DouyinFavoriteAweme{{AwemeID: "a1"}}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllAwemeTagIDs query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT aweme_id, tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllAwemeTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllAwemeTagIDs scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT aweme_id, tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "tag_id"}).AddRow("a1", "bad"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllAwemeTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listAllAwemeTagIDs rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		rows := sqlmock.NewRows([]string{"aweme_id", "tag_id"}).
			AddRow("a1", int64(1)).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`SELECT aweme_id, tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.listAllAwemeTagIDs(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}
