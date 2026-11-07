package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"liao/internal/upstream/wanfeng"
)

// TestProviderContextMiddleware 校验请求头/query 里的线路标识会进入 context。
func TestProviderContextMiddleware(t *testing.T) {
	a := &App{}

	var got string
	handler := a.providerContextMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = providerFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	req.Header.Set(ProviderHeaderName, "WanFeng")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got != ProviderWanfeng {
		t.Fatalf("请求头应归一成 %q，实际 %q", ProviderWanfeng, got)
	}

	// query 兜底
	got = ""
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/getHistoryUserList?provider=chat2019", nil))
	if got != ProviderChat2019 {
		t.Fatalf("query 应归一成 %q，实际 %q", ProviderChat2019, got)
	}

	// 未声明线路时不写 context（保持旧行为）
	got = "sentinel"
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/getHistoryUserList", nil))
	if got != "" {
		t.Fatalf("未声明线路时 context 应为空，实际 %q", got)
	}
}

// TestWanfengHistorySkipsWanfengWhenRequestIsChat2019 是本期回归测试的核心：
// 身份绑过晚风、但当前请求声明走 chat2019 时，历史接口必须让路，
// 否则刷新页面会被晚风侧数据整体替换，chat2019 的会话（含刚匹配到的人）全部消失。
func TestWanfengHistorySkipsWanfengWhenRequestIsChat2019(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	req.Header.Set(ProviderHeaderName, ProviderChat2019)

	if a.serveWanfengHistoryUsers(httptest.NewRecorder(), req, "IDENT-1", false) {
		t.Fatal("chat2019 线路下历史会话不应由晚风分支处理")
	}
	if a.serveWanfengHistoryUsers(httptest.NewRecorder(), req, "IDENT-1", true) {
		t.Fatal("chat2019 线路下收藏列表不应由晚风分支处理")
	}
	if a.serveWanfengMessageHistory(httptest.NewRecorder(), req, "IDENT-1", "WFU2", "0") {
		t.Fatal("chat2019 线路下聊天记录不应由晚风分支处理")
	}
	if a.serveWanfengFavorite(httptest.NewRecorder(), req, "IDENT-1", "WFU2", true) {
		t.Fatal("chat2019 线路下收藏动作不应由晚风分支处理")
	}
	if _, ok := a.wanfengIdentityUID(requestProviderContext(req), "IDENT-1"); ok {
		t.Fatal("chat2019 线路下不应返回晚风 uid")
	}
	if a.matchPreferenceSupported(requestProviderContext(req), "IDENT-1") {
		t.Fatal("chat2019 线路下匹配偏好应为不支持")
	}
	if a.locationSettingSupported(requestProviderContext(req), "IDENT-1") {
		t.Fatal("chat2019 线路下位置公开应为不支持")
	}
}

// TestWanfengHistoryRoutesWhenRequestIsWanfeng 校验晚风线路下仍然走晚风分支（行为不变）。
func TestWanfengHistoryRoutesWhenRequestIsWanfeng(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	req.Header.Set(ProviderHeaderName, ProviderWanfeng)

	if !a.serveWanfengHistoryUsers(httptest.NewRecorder(), req, "IDENT-1", false) {
		t.Fatal("晚风线路下已绑定身份应由晚风分支处理")
	}
	if uid, ok := a.wanfengIdentityUID(requestProviderContext(req), "IDENT-1"); !ok || uid != "WFU1" {
		t.Fatalf("晚风线路下应返回绑定 uid，实际 ok=%v uid=%q", ok, uid)
	}
}
