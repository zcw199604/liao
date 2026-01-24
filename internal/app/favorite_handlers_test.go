package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleFavoriteAdd_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	mock.ExpectExec(`INSERT INTO chat_favorites \(identity_id, target_user_id, target_user_name, create_time\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs("i1", "u1", "Bob", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(9, 1))

	app := &App{favoriteService: NewFavoriteService(db)}

	form := url.Values{}
	form.Set("identityId", "i1")
	form.Set("targetUserId", "u1")
	form.Set("targetUserName", "Bob")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/add", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteAdd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); got != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["id"].(float64); got != 9 {
		t.Fatalf("id=%v, want 9", data["id"])
	}
	if got, _ := data["identityId"].(string); got != "i1" {
		t.Fatalf("identityId=%q, want %q", got, "i1")
	}
	if got, _ := data["targetUserId"].(string); got != "u1" {
		t.Fatalf("targetUserId=%q, want %q", got, "u1")
	}
}

func TestHandleFavoriteAdd_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	mock.ExpectExec(`INSERT INTO chat_favorites \(identity_id, target_user_id, target_user_name, create_time\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs("i1", "u1", "Bob", sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	app := &App{favoriteService: NewFavoriteService(db)}

	form := url.Values{}
	form.Set("identityId", "i1")
	form.Set("targetUserId", "u1")
	form.Set("targetUserName", "Bob")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/add", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteAdd(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "保存失败" {
		t.Fatalf("msg=%q, want %q", got, "保存失败")
	}
}

func TestHandleFavoriteListAll_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local)
	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}).
			AddRow(int64(1), "i1", "u1", sql.NullString{String: "Bob", Valid: true}, sql.NullTime{Time: createTime, Valid: true}))

	app := &App{favoriteService: NewFavoriteService(db)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/favorite/list", nil)
	rr := httptest.NewRecorder()

	app.handleFavoriteListAll(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data=%v, want 1 item", resp["data"])
	}
}

func TestHandleFavoriteCheck_NotFavorite(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT 1 FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	app := &App{favoriteService: NewFavoriteService(db)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/favorite/check?identityId=i1&targetUserId=u1", nil)
	rr := httptest.NewRecorder()

	app.handleFavoriteCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["isFavorite"].(bool); got {
		t.Fatalf("isFavorite=true, want false")
	}
}

func TestHandleFavoriteRemove_IgnoresDBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM chat_favorites WHERE identity_id = \? AND target_user_id = \?`).
		WithArgs("i1", "u1").
		WillReturnError(sql.ErrConnDone)

	app := &App{favoriteService: NewFavoriteService(db)}

	form := url.Values{}
	form.Set("identityId", "i1")
	form.Set("targetUserId", "u1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/remove", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteRemove(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); got != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}
}

func TestHandleFavoriteRemoveByID_InvalidID_ReturnsBadRequest(t *testing.T) {
	app := &App{}

	form := url.Values{}
	form.Set("id", "not-int")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/removeById", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteRemoveByID(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "id无效" {
		t.Fatalf("msg=%q, want %q", got, "id无效")
	}
}

func TestHandleFavoriteRemoveByID_EmptyID_ReturnsBadRequest(t *testing.T) {
	app := &App{}

	form := url.Values{}
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/removeById", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteRemoveByID(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "id不能为空" {
		t.Fatalf("msg=%q, want %q", got, "id不能为空")
	}
}

func TestHandleFavoriteRemoveByID_NonPositiveID_ReturnsBadRequest(t *testing.T) {
	app := &App{}

	form := url.Values{}
	form.Set("id", "0")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/removeById", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteRemoveByID(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleFavoriteRemoveByID_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM chat_favorites WHERE id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	app := &App{favoriteService: NewFavoriteService(db)}

	form := url.Values{}
	form.Set("id", "1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/favorite/removeById", form)
	rr := httptest.NewRecorder()

	app.handleFavoriteRemoveByID(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleFavoriteListAll_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC`).
		WillReturnError(sql.ErrConnDone)

	app := &App{favoriteService: NewFavoriteService(db)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/favorite/list", nil)
	rr := httptest.NewRecorder()

	app.handleFavoriteListAll(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleFavoriteCheck_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT 1 FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnError(sql.ErrConnDone)

	app := &App{favoriteService: NewFavoriteService(db)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/favorite/check?identityId=i1&targetUserId=u1", nil)
	rr := httptest.NewRecorder()

	app.handleFavoriteCheck(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
