package app

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type errReadCloser struct{}

func (errReadCloser) Read(p []byte) (int, error) { return 0, errors.New("read fail") }
func (errReadCloser) Close() error               { return nil }

func TestHandleGetSystemConfig_Default(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getSystemConfig", nil)
	rec := httptest.NewRecorder()
	a.handleGetSystemConfig(rec, req)

	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != 0 {
		t.Fatalf("code=%v", got["code"])
	}
	data := got["data"].(map[string]any)
	if data["imagePortMode"].(string) != string(defaultSystemConfig.ImagePortMode) {
		t.Fatalf("data=%v", data)
	}
}

func TestHandleUpdateSystemConfig_Errors(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateSystemConfig", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	a.handleUpdateSystemConfig(rec, req)
	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != -1 {
		t.Fatalf("got=%v", got)
	}

	a2 := &App{systemConfig: &SystemConfigService{db: nil}}
	req2 := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateSystemConfig", nil)
	req2.Body = errReadCloser{}
	rec2 := httptest.NewRecorder()
	a2.handleUpdateSystemConfig(rec2, req2)
	got2 := decodeJSONBody(t, rec2.Body)
	if int(got2["code"].(float64)) != -1 || got2["msg"].(string) != "读取请求失败" {
		t.Fatalf("got=%v", got2)
	}

	req3 := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateSystemConfig", bytes.NewBufferString(`not-json`))
	rec3 := httptest.NewRecorder()
	a2.handleUpdateSystemConfig(rec3, req3)
	got3 := decodeJSONBody(t, rec3.Body)
	if int(got3["code"].(float64)) != -1 || got3["msg"].(string) != "JSON解析失败" {
		t.Fatalf("got=%v", got3)
	}

	// normalize 失败（不触发 DB）
	db, _, cleanup := newSQLMock(t)
	defer cleanup()
	svc := NewSystemConfigService(db)
	svc.loaded = true
	svc.cached = defaultSystemConfig

	a3 := &App{systemConfig: svc}
	req4 := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateSystemConfig", bytes.NewBufferString(`{"imagePortMode":"bad"}`))
	rec4 := httptest.NewRecorder()
	a3.handleUpdateSystemConfig(rec4, req4)
	got4 := decodeJSONBody(t, rec4.Body)
	if int(got4["code"].(float64)) != -1 {
		t.Fatalf("got=%v", got4)
	}
}

func TestHandleUpdateSystemConfig_Success_ClearsResolverCache(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewSystemConfigService(db)
	svc.loaded = true
	svc.cached = defaultSystemConfig

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortMode, "probe", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortFixed, "9006", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortRealMinBytes, "4096", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	resolver := NewImagePortResolver(nil)
	resolver.cache["h"] = "9003"

	a := &App{
		systemConfig:      svc,
		imagePortResolver: resolver,
	}

	body := `{"imagePortMode":"probe","imagePortFixed":"9006","imagePortRealMinBytes":4096}`
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/updateSystemConfig", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	a.handleUpdateSystemConfig(rec, req)

	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != 0 {
		t.Fatalf("got=%v", got)
	}
	if resolver.GetCached("h") != "" {
		t.Fatalf("expected cache cleared")
	}
}

func TestHandleResolveImagePort_ErrorsAndSuccess(t *testing.T) {
	a := &App{}

	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/resolveImagePort", nil)
	req.Body = errReadCloser{}
	rec := httptest.NewRecorder()
	a.handleResolveImagePort(rec, req)
	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != -1 || got["msg"].(string) != "读取请求失败" {
		t.Fatalf("got=%v", got)
	}

	req2 := httptest.NewRequest(http.MethodPost, "http://api.local/api/resolveImagePort", bytes.NewBufferString(`bad`))
	rec2 := httptest.NewRecorder()
	a.handleResolveImagePort(rec2, req2)
	got2 := decodeJSONBody(t, rec2.Body)
	if int(got2["code"].(float64)) != -1 || got2["msg"].(string) != "JSON解析失败" {
		t.Fatalf("got=%v", got2)
	}

	req3 := httptest.NewRequest(http.MethodPost, "http://api.local/api/resolveImagePort", bytes.NewBufferString(`{"path":"a.jpg"}`))
	rec3 := httptest.NewRecorder()
	a.handleResolveImagePort(rec3, req3)
	got3 := decodeJSONBody(t, rec3.Body)
	if int(got3["code"].(float64)) != 0 {
		t.Fatalf("got=%v", got3)
	}
	port := got3["data"].(map[string]any)["port"].(string)
	if port != defaultSystemConfig.ImagePortFixed {
		t.Fatalf("port=%q", port)
	}
}

func TestGetSystemConfigOrDefault_ReturnsDefaultOnGetError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
		WillReturnError(errors.New("db fail"))

	svc := NewSystemConfigService(db)
	a := &App{systemConfig: svc}
	got := a.getSystemConfigOrDefault(context.Background())
	if got != defaultSystemConfig {
		t.Fatalf("got=%+v", got)
	}
}
