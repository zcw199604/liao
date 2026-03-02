package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDouyinFavoriteTagHandlers_UncoveredBranches(t *testing.T) {
	t.Run("user tag list nil slice fallback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			}))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/tag/list", nil)
		app.handleDouyinFavoriteUserTagList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"items":[]`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("aweme tag list nil slice fallback", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			}))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteAweme/tag/list", nil)
		app.handleDouyinFavoriteAwemeTagList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"items":[]`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("aweme tag add internal error and out nil", func(t *testing.T) {
		{
			db, mock, cleanup := newSQLMock(t)
			defer cleanup()

			mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
				WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
			expectInsertReturningIDError(
				mock,
				`INSERT INTO douyin_favorite_aweme_tag`,
				errors.New("insert failed"),
				"教程",
				int64(1),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			)

			app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
			req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", map[string]any{"name": "教程"})
			rr := httptest.NewRecorder()
			app.handleDouyinFavoriteAwemeTagAdd(rr, req)
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		}

		{
			db, mock, cleanup := newSQLMock(t)
			defer cleanup()

			mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
				WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
			expectInsertReturningID(
				mock,
				`INSERT INTO douyin_favorite_aweme_tag`,
				6,
				"教程",
				int64(1),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			)
			mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
				WithArgs(int64(6)).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "name", "sort_order", "cnt", "created_at", "updated_at",
				})) // no rows -> nil result

			app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
			req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", map[string]any{"name": "教程"})
			rr := httptest.NewRecorder()
			app.handleDouyinFavoriteAwemeTagAdd(rr, req)
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		}
	})

	t.Run("aweme tag update bad json", func(t *testing.T) {
		db := mustNewSQLMockDB(t)
		defer db.Close()
		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", strings.NewReader("{"))
		app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}
