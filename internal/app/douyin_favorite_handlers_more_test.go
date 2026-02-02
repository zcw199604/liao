package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDouyinFavoriteHandlers_Uninitialized_AllHandlers(t *testing.T) {
	app := &App{} // douyinFavorite is nil
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader("{}"))

	calls := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"UserList", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserList(w, r) }},
		{"UserAdd", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserAdd(w, r) }},
		{"UserRemove", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteUserRemove(w, r) }},
		{"AwemeList", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeList(w, r) }},
		{"AwemeAdd", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeAdd(w, r) }},
		{"AwemeRemove", func(w http.ResponseWriter, r *http.Request) { app.handleDouyinFavoriteAwemeRemove(w, r) }},
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

func TestHandleDouyinFavoriteUserList_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_user`).
		WillReturnError(errors.New("query failed"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/list", nil)
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteUserList(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteUserAdd_BadJSONAndValidation(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	{
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/add", strings.NewReader("{"))
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/add", map[string]any{"secUserId": " "})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing secUserId: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleDouyinFavoriteUserAdd_SaveErrorAndOutNil(t *testing.T) {
	// save error
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT INTO douyin_favorite_user`).
			WillReturnError(errors.New("exec failed"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/add", map[string]any{"secUserId": "u1"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}

	// out == nil (find returns no rows)
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT INTO douyin_favorite_user`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`FROM douyin_favorite_user`).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{
				"sec_user_id",
				"source_input",
				"display_name",
				"avatar_url",
				"profile_url",
				"last_parsed_at",
				"last_parsed_count",
				"last_parsed_raw",
				"created_at",
				"updated_at",
			})) // no rows -> sql.ErrNoRows

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/add", map[string]any{"secUserId": "u1"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteUserRemove_BadJSONAndValidation(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	{
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteUser/remove", strings.NewReader("{"))
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/remove", map[string]any{"secUserId": " "})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing secUserId: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleDouyinFavoriteAwemeList_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_aweme`).
		WillReturnError(errors.New("query failed"))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteAweme/list", nil)
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteAwemeList(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteAwemeAdd_BadJSONAndValidation(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	{
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/add", strings.NewReader("{"))
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/add", map[string]any{"awemeId": " "})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeAdd(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing awemeId: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleDouyinFavoriteAwemeAdd_SaveErrorAndOutNil(t *testing.T) {
	// save error
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT INTO douyin_favorite_aweme`).
			WillReturnError(errors.New("exec failed"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/add", map[string]any{"awemeId": "a1"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}

	// out == nil (find returns no rows)
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT INTO douyin_favorite_aweme`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`FROM douyin_favorite_aweme`).
			WithArgs("a1").
			WillReturnRows(sqlmock.NewRows([]string{
				"aweme_id",
				"sec_user_id",
				"type",
				"description",
				"cover_url",
				"created_at",
				"updated_at",
			})) // no rows -> sql.ErrNoRows

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/add", map[string]any{"awemeId": "a1"})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeAdd(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	}
}

func TestHandleDouyinFavoriteAwemeRemove_BadJSONAndValidation(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()
	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	{
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/favoriteAweme/remove", strings.NewReader("{"))
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
	{
		req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/remove", map[string]any{"awemeId": " "})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteAwemeRemove(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing awemeId: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	}
}
