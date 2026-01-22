package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
