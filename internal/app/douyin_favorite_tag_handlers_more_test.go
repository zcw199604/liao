package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestHandleDouyinFavoriteUserTagList_Uninitialized(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/tag/list", nil)
	(&App{}).handleDouyinFavoriteUserTagList(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteUserTagList_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
		WillReturnError(errors.New("query failed"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/tag/list", nil)
	app.handleDouyinFavoriteUserTagList(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteUserTagAdd_BadJSONAndEmptyName(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", strings.NewReader("{"))
		app.handleDouyinFavoriteUserTagAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		rr := httptest.NewRecorder()
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", map[string]any{"name": "  "})
		app.handleDouyinFavoriteUserTagAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("empty name: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleDouyinFavoriteUserTagAdd_InternalErrorAndOutNil(t *testing.T) {
	// internal error (non-duplicate)
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_user_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_tag`).
			WithArgs("美食", int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", map[string]any{"name": "美食"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}

	// out == nil (find by id returns no rows)
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_user_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
		mock.ExpectExec(`INSERT INTO douyin_favorite_user_tag`).
			WithArgs("美食", int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(7, 1))
		mock.ExpectQuery(`FROM douyin_favorite_user_tag t`).
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			})) // no rows -> sql.ErrNoRows

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", map[string]any{"name": "美食"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteUserTagUpdate_ValidationAndNotFound(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	// id missing
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{"id": 0, "name": "x"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	// name empty
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{"id": 1, "name": " "})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	// not found: update OK but lookup returns nil
	{
		db2, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`UPDATE douyin_favorite_user_tag`).
			WithArgs("美食2", sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`FROM douyin_favorite_user_tag t`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			})) // no rows

		app2 := &App{douyinFavorite: NewDouyinFavoriteService(db2)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{"id": 1, "name": "美食2"})
		rr := httptest.NewRecorder()
		app2.handleDouyinFavoriteUserTagUpdate(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	}
}

func TestHandleDouyinFavoriteUserTagRemove_BadJSONAndValidation(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/remove", strings.NewReader("{"))
		app.handleDouyinFavoriteUserTagRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/remove", map[string]any{"id": 0})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing id: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleDouyinFavoriteAwemeTagAdd_Apply_AndMoreBranches(t *testing.T) {
	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)

	// add success
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
		mock.ExpectExec(`INSERT INTO douyin_favorite_aweme_tag`).
			WithArgs("教程", int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(5, 1))
		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			}).AddRow(
				int64(5),
				"教程",
				int64(1),
				int64(0),
				sql.NullTime{Time: createTime, Valid: true},
				sql.NullTime{Time: createTime, Valid: true},
			))

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", map[string]any{"name": "教程"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagAdd(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if got, _ := resp["id"].(float64); got != 5 {
			t.Fatalf("id=%v, want %v", resp["id"], 5)
		}
	}

	// add duplicate
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_aweme_tag`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))
		mock.ExpectExec(`INSERT INTO douyin_favorite_aweme_tag`).
			WithArgs("教程", int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(&mysql.MySQLError{Number: 1062, Message: "dup"})

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", map[string]any{"name": "教程"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	// add bad json + empty name
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()
		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", strings.NewReader("{"))
		app.handleDouyinFavoriteAwemeTagAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}

		rr2 := httptest.NewRecorder()
		req2 := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/add", map[string]any{"name": "  "})
		app.handleDouyinFavoriteAwemeTagAdd(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("empty name: status=%d, want %d", rr2.Code, http.StatusBadRequest)
		}
	}

	// apply invalid mode
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()
		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/apply", map[string]any{
			"awemeIds": []string{"a1"},
			"tagIds":   []int64{1},
			"mode":     "bad",
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagApply(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	// apply set success
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT IGNORE INTO douyin_favorite_aweme_tag_map`).
			WithArgs("a1", int64(1), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/apply", map[string]any{
			"awemeIds": []string{"a1"},
			"tagIds":   []int64{1},
			"mode":     "set",
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagApply(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	}
}

func TestHandleDouyinFavoriteAwemeTagList_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
		WillReturnError(errors.New("query failed"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteAweme/tag/list", nil)
	app.handleDouyinFavoriteAwemeTagList(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteAwemeTagUpdate_ValidationDuplicateNotFoundAndError(t *testing.T) {
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()
		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{"id": 0, "name": "x"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing id: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}

		req2 := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{"id": 1, "name": " "})
		rr2 := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagUpdate(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("empty name: status=%d, want %d", rr2.Code, http.StatusBadRequest)
		}
	}

	// duplicate
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag`).
			WithArgs("教程", sqlmock.AnyArg(), int64(1)).
			WillReturnError(&mysql.MySQLError{Number: 1062, Message: "dup"})

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{"id": 1, "name": "教程"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	// not found
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag`).
			WithArgs("教程2", sqlmock.AnyArg(), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`FROM douyin_favorite_aweme_tag t`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "sort_order", "cnt", "created_at", "updated_at",
			})) // no rows

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{"id": 1, "name": "教程2"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	}

	// internal error
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag`).
			WithArgs("教程2", sqlmock.AnyArg(), int64(1)).
			WillReturnError(errors.New("exec fail"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{"id": 1, "name": "教程2"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteAwemeTagRemove_BadJSONValidationAndError(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/remove", strings.NewReader("{"))
		app.handleDouyinFavoriteAwemeTagRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/remove", map[string]any{"id": 0})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeTagRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing id: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	// delete error -> 500
	{
		db2, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE tag_id = \?`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete failed"))
		mock.ExpectRollback()

		app2 := &App{douyinFavorite: NewDouyinFavoriteService(db2)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/remove", map[string]any{"id": 1})
		rr := httptest.NewRecorder()
		app2.handleDouyinFavoriteAwemeTagRemove(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteAwemeTagReorder_BadJSONAndError(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/reorder", strings.NewReader("{"))
		app.handleDouyinFavoriteAwemeTagReorder(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	{
		db2, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag SET sort_order = \? WHERE id = \?`).
			WithArgs(int64(1), int64(1)).
			WillReturnError(errors.New("update failed"))
		mock.ExpectRollback()

		app2 := &App{douyinFavorite: NewDouyinFavoriteService(db2)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/reorder", map[string]any{"tagIds": []int64{1}})
		rr := httptest.NewRecorder()
		app2.handleDouyinFavoriteAwemeTagReorder(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteAwemeTagApply_BadJSON(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/apply", strings.NewReader("{"))
	app.handleDouyinFavoriteAwemeTagApply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestDouyinFavoriteTagHandlers_Uninitialized_AllHandlers(t *testing.T) {
	app := &App{} // douyinFavorite is nil
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader("{}"))

	calls := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"UserTagList", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagList(w, r) }},
		{"UserTagAdd", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagAdd(w, r) }},
		{"UserTagUpdate", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagUpdate(w, r) }},
		{"UserTagRemove", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagRemove(w, r) }},
		{"UserTagApply", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagApply(w, r) }},
		{"UserTagReorder", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserTagReorder(w, r) }},
		{"AwemeTagList", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagList(w, r) }},
		{"AwemeTagAdd", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagAdd(w, r) }},
		{"AwemeTagUpdate", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagUpdate(w, r) }},
		{"AwemeTagRemove", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagRemove(w, r) }},
		{"AwemeTagReorder", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagReorder(w, r) }},
		{"AwemeTagApply", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeTagApply(w, r) }},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.fn(rr, req)
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
			}
		})
	}
}

func TestHandleDouyinFavoriteUserTagUpdate_BadJSONAndInternalError(t *testing.T) {
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", strings.NewReader("{"))
		app.handleDouyinFavoriteUserTagUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`UPDATE douyin_favorite_user_tag`).
			WithArgs("美食2", sqlmock.AnyArg(), int64(1)).
			WillReturnError(errors.New("exec fail"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		rr := httptest.NewRecorder()
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{"id": 1, "name": "美食2"})
		app.handleDouyinFavoriteUserTagUpdate(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("internal error: status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteUserTagRemove_ServiceError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/remove", map[string]any{"id": 1})
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteUserTagRemove(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteUserTagApply_BadJSONAndInternalError(t *testing.T) {
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", strings.NewReader("{"))
		app.handleDouyinFavoriteUserTagApply(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", map[string]any{
			"secUserIds": []string{"u1"},
			"tagIds":     []int64{1},
			"mode":       "set",
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserTagApply(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("internal error: status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteUserTagReorder_BadJSON(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/reorder", strings.NewReader("{"))
	app.handleDouyinFavoriteUserTagReorder(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinFavoriteAwemeTagApply_InternalError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/apply", map[string]any{
		"awemeIds": []string{"a1"},
		"tagIds":   []int64{1},
		"mode":     "set",
	})
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteAwemeTagApply(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
