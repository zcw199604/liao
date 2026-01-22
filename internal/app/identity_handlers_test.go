package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func decodeJSON(t *testing.T, body *strings.Reader) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestHandleCreateIdentity_ValidatesInput(t *testing.T) {
	a := &App{identityService: &IdentityService{}}

	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/createIdentity", strings.NewReader("sex=男"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.handleCreateIdentity(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "http://api.local/api/createIdentity", strings.NewReader("name=a&sex=x"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	a.handleCreateIdentity(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec2.Code)
	}
}

func TestHandleCreateIdentity_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs(sqlmock.AnyArg(), "Alice", "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/createIdentity", strings.NewReader("name=Alice&sex=女"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.handleCreateIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(payload["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", payload["code"])
	}
	data := payload["data"].(map[string]any)
	if data["name"].(string) != "Alice" || data["sex"].(string) != "女" {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestHandleUpdateIdentity_NotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateIdentity", strings.NewReader("id=id1&name=Alice&sex=女"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.handleUpdateIdentity(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestHandleGetIdentityList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "name", "sex", "created_at", "last_used_at"}).
		AddRow("a", "A", "男", now, now).
		AddRow("b", "B", "女", now, sql.NullTime{Valid: false})
	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnRows(rows)

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getIdentityList", nil)
	rec := httptest.NewRecorder()
	a.handleGetIdentityList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(payload["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", payload["code"])
	}
	data, ok := payload["data"].([]any)
	if !ok || len(data) != 2 {
		t.Fatalf("data=%v, want 2 items", payload["data"])
	}
}

func TestHandleGetIdentityList_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnError(sql.ErrConnDone)

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getIdentityList", nil)
	rec := httptest.NewRecorder()
	a.handleGetIdentityList(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", rec.Code)
	}
}

func TestHandleQuickCreateIdentity_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/quickCreateIdentity", nil)
	rec := httptest.NewRecorder()
	a.handleQuickCreateIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(payload["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", payload["code"])
	}
	data := payload["data"].(map[string]any)
	if id, _ := data["id"].(string); id == "" {
		t.Fatalf("id should not be empty: %+v", data)
	}
	if name, _ := data["name"].(string); name == "" {
		t.Fatalf("name should not be empty: %+v", data)
	}
	sex, _ := data["sex"].(string)
	if sex != "男" && sex != "女" {
		t.Fatalf("sex=%q, want 男/女", sex)
	}
}

func TestHandleQuickCreateIdentity_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	a := &App{identityService: NewIdentityService(db)}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/quickCreateIdentity", nil)
	rec := httptest.NewRecorder()
	a.handleQuickCreateIdentity(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", rec.Code)
	}
}

func TestHandleUpdateIdentity_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "女", now, now))
	mock.ExpectExec(`UPDATE identity SET name = \?, sex = \?, last_used_at = \? WHERE id = \?`).
		WithArgs("New", "男", sqlmock.AnyArg(), "id1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("id", "id1")
	form.Set("name", "New")
	form.Set("sex", "男")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/updateIdentity", form)
	rec := httptest.NewRecorder()
	a.handleUpdateIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestHandleUpdateIdentity_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "女", now, now))
	mock.ExpectExec(`UPDATE identity SET name = \?, sex = \?, last_used_at = \? WHERE id = \?`).
		WithArgs("New", "男", sqlmock.AnyArg(), "id1").
		WillReturnError(sql.ErrConnDone)

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("id", "id1")
	form.Set("name", "New")
	form.Set("sex", "男")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/updateIdentity", form)
	rec := httptest.NewRecorder()
	a.handleUpdateIdentity(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", rec.Code)
	}
}

func TestHandleUpdateIdentityID_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createdAt := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	createdAtStr := createdAt.Format("2006-01-02 15:04:05")

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", createdAt, createdAt))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs("new", "New", "女", createdAtStr, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("oldId", "old")
	form.Set("newId", "new")
	form.Set("name", "New")
	form.Set("sex", "女")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/updateIdentityID", form)
	rec := httptest.NewRecorder()
	a.handleUpdateIdentityID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestHandleUpdateIdentityID_NewIDAlreadyUsed(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", now, now))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Existing", "女", now, now))

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("oldId", "old")
	form.Set("newId", "new")
	form.Set("name", "New")
	form.Set("sex", "女")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/updateIdentityID", form)
	rec := httptest.NewRecorder()
	a.handleUpdateIdentityID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestHandleUpdateIdentityID_TransactionInsertError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createdAt := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	createdAtStr := createdAt.Format("2006-01-02 15:04:05")

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", createdAt, createdAt))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs("new", "New", "女", createdAtStr, sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("oldId", "old")
	form.Set("newId", "new")
	form.Set("name", "New")
	form.Set("sex", "女")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/updateIdentityID", form)
	rec := httptest.NewRecorder()
	a.handleUpdateIdentityID(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", rec.Code)
	}
}

func TestHandleDeleteIdentity_EmptyID(t *testing.T) {
	a := &App{identityService: &IdentityService{}}

	form := url.Values{}
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/deleteIdentity", form)
	rec := httptest.NewRecorder()
	a.handleDeleteIdentity(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestHandleDeleteIdentity_NotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("id", "missing")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/deleteIdentity", form)
	rec := httptest.NewRecorder()
	a.handleDeleteIdentity(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestHandleDeleteIdentity_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("id", "id1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/deleteIdentity", form)
	rec := httptest.NewRecorder()
	a.handleDeleteIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestHandleSelectIdentity_UpdateLastUsedAtErrorStillOK(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Alice", "女", now, now))
	mock.ExpectExec(`UPDATE identity SET last_used_at = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), "id1").
		WillReturnError(sql.ErrConnDone)

	a := &App{identityService: NewIdentityService(db)}
	form := url.Values{}
	form.Set("id", "id1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://api.local/api/selectIdentity", form)
	rec := httptest.NewRecorder()
	a.handleSelectIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(payload["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", payload["code"])
	}
}
