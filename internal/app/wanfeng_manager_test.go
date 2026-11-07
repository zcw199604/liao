package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liao/internal/config"
	"liao/internal/upstream/wanfeng"
)

// TestWanfengManagerProfileFromSign 校验 sign 帧能正确解析出展示资料。
func TestWanfengManagerProfileFromSign(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)

	sign := `{"act":"sign","id":"u1","name":"小明","userSex":"女","userAge":"22"}`
	profile := m.profileFromSign(sign)
	if profile.Nickname != "小明" || profile.Gender != "female" || profile.Age != 22 {
		t.Fatalf("资料解析不符: %+v", profile)
	}

	// 非法 JSON 时降级为 secret，不应 panic
	profile = m.profileFromSign("not-json")
	if profile.Gender != "secret" {
		t.Fatalf("非法帧应降级 secret，实际 %+v", profile)
	}

	// 性别未知时应上送 secret
	profile = m.profileFromSign(`{"act":"sign","id":"u1","name":"甲"}`)
	if profile.Gender != "secret" {
		t.Fatalf("未知性别应映射为 secret，实际 %+v", profile)
	}
}

// TestHandleProviderList 校验线路列表接口：未启用时不暴露晚风线路。
func TestHandleProviderList(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})

	app := &App{
		cfg:     config.Config{WanfengEnabled: true},
		wanfeng: NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil),
	}
	rec := httptest.NewRecorder()
	app.handleProviderList(rec, httptest.NewRequest(http.MethodGet, "/api/provider/list", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Providers []ProviderInfo `json:"providers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Providers) != 2 {
		t.Fatalf("启用晚风时应返回两条线路，实际 %+v", resp)
	}
	if resp.Data.Providers[0].ID != ProviderChat2019 || !resp.Data.Providers[0].Default {
		t.Fatalf("第一条应为默认的 chat2019 线路，实际 %+v", resp.Data.Providers[0])
	}
	if resp.Data.Providers[1].ID != ProviderWanfeng {
		t.Fatalf("第二条应为 wanfeng 线路，实际 %+v", resp.Data.Providers[1])
	}

	// 未启用时只保留经典线路
	app2 := &App{cfg: config.Config{WanfengEnabled: false}}
	rec2 := httptest.NewRecorder()
	app2.handleProviderList(rec2, httptest.NewRequest(http.MethodGet, "/api/provider/list", nil))
	var resp2 struct {
		Data struct {
			Providers []ProviderInfo `json:"providers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp2.Data.Providers) != 1 {
		t.Fatalf("未启用晚风时应只有一条线路，实际 %+v", resp2.Data.Providers)
	}
}

// TestWanfengManagerSendWithoutClient 校验没有上游连接时下发不会 panic。
func TestWanfengManagerSendWithoutClient(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)

	// 无 client 时下发应安全忽略
	m.SendToUpstream("u1", `{"act":"random","id":"u1"}`)
	// sign 帧只更新资料
	m.SendToUpstream("u1", `{"act":"sign","id":"u1","name":"小明","userSex":"男"}`)

	m.mu.Lock()
	profile := m.profiles["u1"]
	m.mu.Unlock()
	if profile.Nickname != "小明" {
		t.Fatalf("sign 帧应更新资料，实际 %+v", profile)
	}
}

// TestWanfengManagerStats 校验统计接口在未启用时也可安全调用。
func TestWanfengManagerStats(t *testing.T) {
	m := NewWanfengManager(wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()}), wanfeng.NewMemoryBindingStore(), nil)
	stats := m.Stats()
	if stats["enabled"] != true {
		t.Fatalf("统计应包含 enabled=true，实际 %+v", stats)
	}
	var nilManager *WanfengManager
	if got := nilManager.Stats(); got["enabled"] != false {
		t.Fatalf("nil 管理器应返回 enabled=false，实际 %+v", got)
	}
}
