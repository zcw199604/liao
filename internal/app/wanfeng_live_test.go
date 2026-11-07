package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"liao/internal/config"
	"liao/internal/upstream/wanfeng"
)

// TestLiveWanfengWebSocketProxy 是「前端 → 我方 /ws(provider=wanfeng) → 晚风上游」的联调冒烟测试（默认跳过）。
//
// 运行方式：
//
//	WANFENG_LIVE_TEST=1 go test ./internal/app/ -run TestLiveWanfengWebSocketProxy -v
//
// 说明：只验证「连接 + 登录 + 协议翻译」链路（期望收到 code=12 连接成功），
// 不进入匹配队列，避免打扰对方真实用户。
func TestLiveWanfengWebSocketProxy(t *testing.T) {
	if os.Getenv("WANFENG_LIVE_TEST") != "1" {
		t.Skip("未开启联调测试（设置 WANFENG_LIVE_TEST=1 启用）")
	}

	baseURL := os.Getenv("WANFENG_BASE_URL")
	if baseURL == "" {
		baseURL = "https://wanfengniming.store"
	}

	jwtService := NewJWTService("live-test-secret-at-least-256-bits-long-0123456789", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{
		BaseURL:  baseURL,
		Protocol: wanfeng.DefaultProtocol(),
	})
	manager := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)
	defer manager.CloseAll()

	app := &App{
		cfg:     config.Config{WanfengEnabled: true, WanfengBaseURL: baseURL},
		jwt:     jwtService,
		wanfeng: manager,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", app.handleWebSocket)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=" + token + "&provider=wanfeng"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("连接本地 /ws 失败: %v", err)
	}
	defer conn.Close()

	identityID := "live-" + time.Now().Format("150405")
	sign := map[string]any{
		"act":     "sign",
		"id":      identityID,
		"name":    "联调测试",
		"userSex": "男",
		"userAge": "0",
	}
	raw, _ := json.Marshal(sign)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("发送 sign 失败: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	var received []map[string]any
	for {
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			t.Fatalf("未收到连接成功消息，已收到: %+v（err=%v）", received, readErr)
		}
		var node map[string]any
		if err := json.Unmarshal(payload, &node); err != nil {
			continue
		}
		received = append(received, node)
		if code, ok := node["code"].(float64); ok && int(code) == 12 {
			t.Logf("收到连接成功消息，链路可用；全部消息: %+v", received)
			return
		}
	}
}
