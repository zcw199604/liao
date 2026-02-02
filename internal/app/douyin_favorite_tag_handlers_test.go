package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleDouyinFavoriteUserTagList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(1),
			"美食",
			int64(0),
			int64(2),
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/tag/list", nil)
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagList(rr, req)
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
}

func TestHandleDouyinFavoriteUserTagAdd_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_user_tag`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))

	expectInsertReturningID(
		mock,
		`INSERT INTO douyin_favorite_user_tag`,
		7,
		"美食",
		int64(1),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	)

	mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(7),
			"美食",
			int64(1),
			int64(0),
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", map[string]any{
		"name": "美食",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagAdd(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["id"].(float64); got != 7 {
		t.Fatalf("id=%v, want %v", resp["id"], 7)
	}
}

func TestHandleDouyinFavoriteUserTagAdd_Duplicate(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM douyin_favorite_user_tag`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(0)))

	expectInsertReturningIDError(
		mock,
		`INSERT INTO douyin_favorite_user_tag`,
		duplicateKeyErr(),
		"美食",
		int64(1),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	)

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/add", map[string]any{
		"name": "美食",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagAdd(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinFavoriteUserTagApply_InvalidMode(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", map[string]any{
		"secUserIds": []string{"MS4wLjABAAAA_x"},
		"tagIds":     []int64{1},
		"mode":       "bad",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagApply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinFavoriteUserTagUpdate_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	updateTime := time.Date(2026, 1, 24, 1, 2, 4, 0, time.Local)
	mock.ExpectExec(`UPDATE douyin_favorite_user_tag`).
		WithArgs(
			"美食2",
			sqlmock.AnyArg(),
			int64(1),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`FROM douyin_favorite_user_tag t`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(1),
			"美食2",
			int64(0),
			int64(0),
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: updateTime, Valid: true},
		))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{
		"id":   1,
		"name": "美食2",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["id"].(float64); got != 1 {
		t.Fatalf("id=%v, want %v", resp["id"], 1)
	}
	if got, _ := resp["name"].(string); got != "美食2" {
		t.Fatalf("name=%v, want %v", resp["name"], "美食2")
	}
}

func TestHandleDouyinFavoriteUserTagUpdate_Duplicate(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE douyin_favorite_user_tag`).
		WithArgs(
			"美食",
			sqlmock.AnyArg(),
			int64(1),
		).
		WillReturnError(duplicateKeyErr())

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/update", map[string]any{
		"id":   1,
		"name": "美食",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagUpdate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleDouyinFavoriteUserTagRemove_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE tag_id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag WHERE id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/remove", map[string]any{
		"id": 1,
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagRemove(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagApply_Set_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
		WithArgs("MS4wLjABAAAA_x").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("MS4wLjABAAAA_x", int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("MS4wLjABAAAA_x", int64(2), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", map[string]any{
		"secUserIds": []string{"MS4wLjABAAAA_x"},
		"tagIds":     []int64{1, 2},
		"mode":       "set",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagApply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagApply_Add_EmptyTagIDs(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", map[string]any{
		"secUserIds": []string{"MS4wLjABAAAA_x"},
		"tagIds":     []int64{},
		"mode":       "add",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagApply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagApply_Remove_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map\s+WHERE sec_user_id = \? AND tag_id = \?`).
		WithArgs("MS4wLjABAAAA_x", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/apply", map[string]any{
		"secUserIds": []string{"MS4wLjABAAAA_x"},
		"tagIds":     []int64{2},
		"mode":       "remove",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagApply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagReorder_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE douyin_favorite_user_tag SET sort_order = \? WHERE id = \?`).
		WithArgs(int64(1), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE douyin_favorite_user_tag SET sort_order = \? WHERE id = \?`).
		WithArgs(int64(2), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/reorder", map[string]any{
		"tagIds": []int64{2, 1},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagReorder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagReorder_EmptyTagIDs(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/reorder", map[string]any{
		"tagIds": []int64{},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagReorder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteUserTagReorder_UpdateError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE douyin_favorite_user_tag SET sort_order = \? WHERE id = \?`).
		WithArgs(int64(1), int64(2)).
		WillReturnError(errors.New("exec fail"))
	mock.ExpectRollback()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/tag/reorder", map[string]any{
		"tagIds": []int64{2},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserTagReorder(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleDouyinFavoriteAwemeTagList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(1),
			"教程",
			int64(0),
			int64(0),
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
		))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteAweme/tag/list", nil)
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeTagList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteAwemeTagUpdate_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	updateTime := time.Date(2026, 1, 24, 1, 2, 4, 0, time.Local)
	mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag`).
		WithArgs(
			"教程2",
			sqlmock.AnyArg(),
			int64(1),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag t`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(1),
			"教程2",
			int64(0),
			int64(0),
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: updateTime, Valid: true},
		))

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/update", map[string]any{
		"id":   1,
		"name": "教程2",
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeTagUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteAwemeTagRemove_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE tag_id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag WHERE id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/remove", map[string]any{
		"id": 1,
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeTagRemove(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleDouyinFavoriteAwemeTagReorder_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag SET sort_order = \? WHERE id = \?`).
		WithArgs(int64(1), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteAweme/tag/reorder", map[string]any{
		"tagIds": []int64{2},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteAwemeTagReorder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
}
