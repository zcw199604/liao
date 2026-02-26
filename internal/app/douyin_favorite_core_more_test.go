package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestApplyDouyinFavoriteUserMetaFromRaw_InvalidInputs(t *testing.T) {
	applyDouyinFavoriteUserMetaFromRaw(nil, sql.NullString{Valid: true, String: `{"signature":"x"}`})

	u := &DouyinFavoriteUser{}
	applyDouyinFavoriteUserMetaFromRaw(u, sql.NullString{Valid: false, String: `{"signature":"x"}`})
	applyDouyinFavoriteUserMetaFromRaw(u, sql.NullString{Valid: true, String: "   "})
	applyDouyinFavoriteUserMetaFromRaw(u, sql.NullString{Valid: true, String: "not-json"})

	if u.Signature != "" {
		t.Fatalf("signature should remain empty")
	}
}

func TestDouyinFavoriteService_ListUsers_MoreBranches(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_user`).
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
				"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
			}).AddRow("sec1", nil, nil, nil, nil, nil, "bad", nil, nil, nil))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListUsers(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
			"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
		}).AddRow("sec1", nil, nil, nil, nil, nil, nil, nil, nil, nil).RowError(0, errors.New("row err"))
		mock.ExpectQuery(`FROM douyin_favorite_user`).WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListUsers(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("fillUserTagIDs error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_user`).
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
				"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
			}).AddRow("sec1", nil, nil, nil, nil, now, nil, nil, now, now))
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListUsers(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success sets avatar/profile", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_user`).
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
				"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
			}).AddRow("sec1", "src", "nick", "http://avatar", "http://profile", now, int64(3), sql.NullString{Valid: true, String: `{"signature":"s"}`}, now, now))
		mock.ExpectQuery(`SELECT sec_user_id, tag_id\s*FROM douyin_favorite_user_tag_map`).
			WillReturnRows(sqlmock.NewRows([]string{"sec_user_id", "tag_id"}))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		out, err := s.ListUsers(context.Background())
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(out) != 1 || out[0].AvatarURL == "" || out[0].ProfileURL == "" {
			t.Fatalf("out=%+v", out)
		}
	})
}

func TestDouyinFavoriteService_FindUserBySecUserID_MoreBranches(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_user\s*WHERE sec_user_id = \?`).
			WithArgs("sec1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findUserBySecUserID(context.Background(), "sec1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("listUserTagIDs error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_user\s*WHERE sec_user_id = \?`).
			WithArgs("sec1").
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
				"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
			}).AddRow("sec1", "src", "nick", "http://avatar", "http://profile", now, int64(1), nil, now, now))
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_user_tag_map\s*WHERE sec_user_id = \?`).
			WithArgs("sec1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findUserBySecUserID(context.Background(), "sec1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("tagIDs nil fallback to empty slice", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_user\s*WHERE sec_user_id = \?`).
			WithArgs("sec1").
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id", "source_input", "display_name", "avatar_url", "profile_url",
				"last_parsed_at", "last_parsed_count", "last_parsed_raw", "created_at", "updated_at",
			}).AddRow("sec1", "src", "nick", "http://avatar", "http://profile", now, int64(1), nil, now, now))
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_user_tag_map\s*WHERE sec_user_id = \?`).
			WithArgs("sec1").
			WillReturnRows(sqlmock.NewRows([]string{"tag_id"}))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		out, err := s.findUserBySecUserID(context.Background(), "sec1")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if out == nil || out.DisplayName == "" || out.AvatarURL == "" || out.ProfileURL == "" {
			t.Fatalf("out=%+v", out)
		}
		if out.TagIDs == nil || len(out.TagIDs) != 0 {
			t.Fatalf("tagIDs=%v", out.TagIDs)
		}
	})
}

func TestDouyinFavoriteService_AwemeQueries_MoreBranches(t *testing.T) {
	t.Run("ListAwemes scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_aweme`).
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at"}).
				AddRow("a1", nil, nil, nil, nil, "bad", nil))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListAwemes(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ListAwemes rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at"}).
			AddRow("a1", nil, nil, nil, nil, nil, nil).
			RowError(0, errors.New("row err"))
		mock.ExpectQuery(`FROM douyin_favorite_aweme`).WillReturnRows(rows)

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListAwemes(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ListAwemes fillAwemeTagIDs error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_aweme`).
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at"}).
				AddRow("a1", "sec1", "video", "desc", "cover", now, now))
		mock.ExpectQuery(`SELECT aweme_id, tag_id\s*FROM douyin_favorite_aweme_tag_map`).
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.ListAwemes(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findAwemeByID query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_aweme\s*WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findAwemeByID(context.Background(), "a1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findAwemeByID list tags error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_aweme\s*WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at"}).
				AddRow("a1", "sec1", "video", "desc", "cover", now, now))
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_aweme_tag_map\s*WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnError(errors.New("query fail"))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if _, err := s.findAwemeByID(context.Background(), "a1"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("findAwemeByID tagIDs nil fallback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := sql.NullTime{Valid: false}
		mock.ExpectQuery(`FROM douyin_favorite_aweme\s*WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "sec_user_id", "type", "description", "cover_url", "created_at", "updated_at"}).
				AddRow("a1", "sec1", "video", "desc", "cover", now, now))
		mock.ExpectQuery(`SELECT tag_id\s*FROM douyin_favorite_aweme_tag_map\s*WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnRows(sqlmock.NewRows([]string{"tag_id"}))

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		out, err := s.findAwemeByID(context.Background(), "a1")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if out == nil || out.TagIDs == nil || len(out.TagIDs) != 0 {
			t.Fatalf("out=%+v", out)
		}
	})
}
