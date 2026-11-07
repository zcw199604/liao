package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"liao/internal/upstream/wanfeng"
)

// newClosedPortAPI 构造一个指向不可达地址的对方客户端（不发真实请求即可测缓存逻辑）。
func newClosedPortAPI() *wanfeng.APIClient {
	return wanfeng.NewAPIClient(wanfeng.APIClientConfig{
		// 127.0.0.1:1 必定连接失败，避免测试打到真实站点
		BaseURL:  "http://127.0.0.1:1",
		Protocol: wanfeng.DefaultProtocol(),
	})
}

// TestWanfengIdentityClientsReuseAndRebuild 校验按身份缓存客户端、token 换发后重建。
func TestWanfengIdentityClientsReuseAndRebuild(t *testing.T) {
	base := newClosedPortAPI()
	cache := newWanfengIdentityClients(base)
	ctx := context.Background()

	binding := &wanfeng.Binding{IdentityID: "u1", UpstreamUID: "WFU1", IdentityToken: "v1.t1"}
	first := cache.Get(ctx, "u1", binding)
	if first == nil || first == base {
		t.Fatal("已绑定身份应拿到独立客户端")
	}
	if again := cache.Get(ctx, "u1", binding); again != first {
		t.Fatal("同身份同 token 应复用客户端")
	}

	// token 换发（重新登录）后必须重建，否则用的还是旧身份 Cookie
	rotated := &wanfeng.Binding{IdentityID: "u1", UpstreamUID: "WFU1", IdentityToken: "v1.t2"}
	if got := cache.Get(ctx, "u1", rotated); got == first {
		t.Fatal("token 变化后应重建客户端")
	}

	// 主动丢弃（身份 Cookie 失效）后重建
	cache.Drop("u1")
	if got := cache.Get(ctx, "u1", rotated); got == nil {
		t.Fatal("丢弃后应能重建客户端")
	}

	// 空身份/空绑定不应崩
	if cache.Get(ctx, "  ", binding) != nil || cache.Get(ctx, "u1", nil) != nil {
		t.Fatal("非法入参应返回 nil")
	}
}

// TestRetryWanfengIdentityOnceOnlyForIdentityErrors 校验只有「身份未就绪」才重建 Cookie 重试。
func TestRetryWanfengIdentityOnceOnlyForIdentityErrors(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m, wanfengClients: newWanfengIdentityClients(m.API())}
	ctx := context.Background()

	identityErr := &wanfeng.APIError{Code: "40300", Status: http.StatusForbidden, Message: "请求错误，请刷新页面"}
	calls := 0
	err := a.retryWanfengIdentityOnce(ctx, "IDENT-1", identityErr, func(api *wanfeng.APIClient, selfUID string) error {
		calls++
		if api == nil || selfUID != "WFU1" {
			t.Fatalf("重试应带上专用客户端与 uid: api=%v uid=%q", api, selfUID)
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("身份类错误应重试一次: err=%v calls=%d", err, calls)
	}

	// 非身份类错误：不重试，原样返回
	otherErr := errors.New("网络异常")
	calls = 0
	if got := a.retryWanfengIdentityOnce(ctx, "IDENT-1", otherErr, func(*wanfeng.APIClient, string) error {
		calls++
		return nil
	}); got != otherErr || calls != 0 {
		t.Fatalf("非身份类错误不应重试: err=%v calls=%d", got, calls)
	}

	// 未绑定身份：重试回调不应被调用
	calls = 0
	if got := a.retryWanfengIdentityOnce(ctx, "unknown", identityErr, func(*wanfeng.APIClient, string) error {
		calls++
		return nil
	}); got != identityErr || calls != 0 {
		t.Fatalf("未绑定身份不应重试: err=%v calls=%d", got, calls)
	}
}

// TestServeWanfengHistoryFallsBackToSharedClient 校验未装配缓存时仍能走共享客户端。
func TestServeWanfengHistoryFallsBackToSharedClient(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	if !a.serveWanfengHistoryUsers(rec, req, "IDENT-1", false) {
		t.Fatal("已绑定身份即使没有客户端缓存也应由晚风分支处理")
	}
	// 上游不可达时必须给出错误状态（不能再用 200 + 空数组把前端列表覆盖掉）
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("上游不可达时应返回 502，实际 %d", rec.Code)
	}
}
