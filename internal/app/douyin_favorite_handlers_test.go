package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newJSONRequest(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode json failed: %v", err)
		}
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleDouyinFavoriteUserAdd_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO douyin_favorite_user`).
		WithArgs(
			"MS4wLjABAAAA_x",
			"https://www.douyin.com/user/MS4wLjABAAAA_x",
			nil,
			nil,
			nil,
			sqlmock.AnyArg(),
			2,
			`{"items":[1,2]}`,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	updateTime := time.Date(2026, 1, 24, 1, 2, 4, 0, time.Local)
	mock.ExpectQuery(`SELECT sec_user_id, source_input, display_name, avatar_url, profile_url,`).
		WithArgs("MS4wLjABAAAA_x").
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
		}).AddRow(
			"MS4wLjABAAAA_x",
			sql.NullString{String: "https://www.douyin.com/user/MS4wLjABAAAA_x", Valid: true},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			sql.NullTime{Time: updateTime, Valid: true},
			sql.NullInt64{Int64: 2, Valid: true},
			sql.NullString{String: `{"items":[1,2]}`, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: updateTime, Valid: true},
		))

	mock.ExpectQuery(`SELECT tag_id\s+FROM douyin_favorite_user_tag_map`).
		WithArgs("MS4wLjABAAAA_x").
		WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).
			AddRow(int64(1)).
			AddRow(int64(3)))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/add", map[string]any{
		"secUserId":       "MS4wLjABAAAA_x",
		"sourceInput":     "https://www.douyin.com/user/MS4wLjABAAAA_x",
		"lastParsedCount": 2,
		"lastParsedRaw":   map[string]any{"items": []any{1, 2}},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserAdd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["secUserId"].(string); got != "MS4wLjABAAAA_x" {
		t.Fatalf("secUserId=%q, want %q", got, "MS4wLjABAAAA_x")
	}
	tagIDs, ok := resp["tagIds"].([]any)
	if !ok || len(tagIDs) != 2 {
		t.Fatalf("tagIds=%v, want 2 items", resp["tagIds"])
	}
	if got, _ := tagIDs[0].(float64); got != 1 {
		t.Fatalf("tagIds[0]=%v, want %v", tagIDs[0], 1)
	}
	if got, _ := tagIDs[1].(float64); got != 3 {
		t.Fatalf("tagIds[1]=%v, want %v", tagIDs[1], 3)
	}
}

func TestHandleDouyinFavoriteUserList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`FROM douyin_favorite_user`).
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
		}).AddRow(
			"MS4wLjABAAAA_x",
			sql.NullString{String: "MS4wLjABAAAA_x", Valid: true},
			sql.NullString{String: "Alice", Valid: true},
			sql.NullString{},
			sql.NullString{},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullInt64{Int64: 18, Valid: true},
			sql.NullString{String: `{"signature":"hi"}`, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	mock.ExpectQuery(`FROM douyin_favorite_user_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"sec_user_id", "tag_id"}).
			AddRow("MS4wLjABAAAA_x", int64(7)).
			AddRow("MS4wLjABAAAA_x", int64(9)))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/list", nil)
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v, want 1 item", resp["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0]=%T, want object", items[0])
	}
	tagIDs, ok := first["tagIds"].([]any)
	if !ok || len(tagIDs) != 2 {
		t.Fatalf("tagIds=%v, want 2 items", first["tagIds"])
	}
}

func TestHandleDouyinFavoriteUserRemove_IgnoresDBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM douyin_favorite_user_aweme WHERE sec_user_id = \?`).
		WithArgs("MS4wLjABAAAA_x").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
		WithArgs("MS4wLjABAAAA_x").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`DELETE FROM douyin_favorite_user WHERE sec_user_id = \?`).
		WithArgs("MS4wLjABAAAA_x").
		WillReturnError(sql.ErrConnDone)

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/remove", map[string]any{
		"secUserId": "MS4wLjABAAAA_x",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserRemove(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteAwemeAdd_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO douyin_favorite_aweme`).
		WithArgs(
			"123456",
			"MS4wLjABAAAA_x",
			"video",
			"hi",
			"https://example.com/cover.jpg",
			`{"k":"v"}`,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`SELECT aweme_id, sec_user_id, type, description, cover_url, created_at, updated_at`).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows([]string{
			"aweme_id",
			"sec_user_id",
			"type",
			"description",
			"cover_url",
			"created_at",
			"updated_at",
		}).AddRow(
			"123456",
			sql.NullString{String: "MS4wLjABAAAA_x", Valid: true},
			sql.NullString{String: "video", Valid: true},
			sql.NullString{String: "hi", Valid: true},
			sql.NullString{String: "https://example.com/cover.jpg", Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	mock.ExpectQuery(`SELECT tag_id\s+FROM douyin_favorite_aweme_tag_map`).
		WithArgs("123456").
		WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow(int64(2)))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/add", map[string]any{
		"awemeId":   "123456",
		"secUserId": "MS4wLjABAAAA_x",
		"type":      "video",
		"desc":      "hi",
		"coverUrl":  "https://example.com/cover.jpg",
		"rawDetail": map[string]any{"k": "v"},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeAdd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["awemeId"].(string); got != "123456" {
		t.Fatalf("awemeId=%q, want %q", got, "123456")
	}
	tagIDs, ok := resp["tagIds"].([]any)
	if !ok || len(tagIDs) != 1 {
		t.Fatalf("tagIds=%v, want 1 item", resp["tagIds"])
	}
	if got, _ := tagIDs[0].(float64); got != 2 {
		t.Fatalf("tagIds[0]=%v, want %v", tagIDs[0], 2)
	}
}

func TestHandleDouyinFavoriteAwemeList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`FROM douyin_favorite_aweme`).
		WillReturnRows(sqlmock.NewRows([]string{
			"aweme_id",
			"sec_user_id",
			"type",
			"description",
			"cover_url",
			"created_at",
			"updated_at",
		}).AddRow(
			"123456",
			sql.NullString{String: "MS4wLjABAAAA_x", Valid: true},
			sql.NullString{String: "video", Valid: true},
			sql.NullString{String: "hi", Valid: true},
			sql.NullString{String: "https://example.com/cover.jpg", Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "tag_id"}).
			AddRow("123456", int64(2)))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteAweme/list", nil)
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v, want 1 item", resp["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0]=%T, want object", items[0])
	}
	tagIDs, ok := first["tagIds"].([]any)
	if !ok || len(tagIDs) != 1 {
		t.Fatalf("tagIds=%v, want 1 item", first["tagIds"])
	}
}

func TestHandleDouyinFavoriteAwemeRemove_IgnoresDBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = \?`).
		WithArgs("123456").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`DELETE FROM douyin_favorite_aweme WHERE aweme_id = \?`).
		WithArgs("123456").
		WillReturnError(sql.ErrConnDone)

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/remove", map[string]any{
		"awemeId": "123456",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeRemove(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}
