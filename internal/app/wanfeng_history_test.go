package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"liao/internal/upstream/wanfeng"
)

// TestBuildWanfengHistoryUsers 校验晚风会话列表 → 前端历史列表的字段映射。
func TestBuildWanfengHistoryUsers(t *testing.T) {
	relations := []wanfeng.Relation{
		{
			TargetUID:      "WFU2",
			TargetNickname: "小晚",
			TargetGender:   "female",
			TargetAge:      22,
			LastMessageAt:  time.Now().Add(-30 * time.Second).Format(time.RFC3339),
			UnreadCount:    2,
		},
		// 自己应被过滤（对方偶尔会把自聊会话塞进列表）
		{TargetUID: "SELF", TargetNickname: "自己"},
		// 缺昵称时兜底「匿名」
		{TargetUID: "WFU3", TargetGender: "male", TargetAge: 0},
	}

	users := buildWanfengHistoryUsers(relations, "SELF", "")
	if len(users) != 2 {
		t.Fatalf("应过滤掉自己，实际 %d 条", len(users))
	}

	first := users[0]
	if first.ID != "WFU2" || first.Nickname != "小晚" {
		t.Fatalf("基础字段错误: %+v", first)
	}
	// 前端按中文性别渲染图标，不能直接传 male/female
	if first.Sex != "女" {
		t.Fatalf("性别未翻译成中文: %+v", first)
	}
	if first.Age != "22" {
		t.Fatalf("年龄应转成字符串: %+v", first)
	}
	if first.UnreadCount != 2 {
		t.Fatalf("未读数丢失: %+v", first)
	}
	if first.LastTime != "刚刚" && !strings.HasSuffix(first.LastTime, "分钟前") {
		t.Fatalf("相对时间文案异常: %s", first.LastTime)
	}

	// 前端据此在列表里打「晚风」标签，缺了会被当成默认线路
	if first.Provider != "wanfeng" || users[1].Provider != "wanfeng" {
		t.Fatalf("历史用户应带线路标记: %+v", users)
	}

	second := users[1]
	if second.Nickname != "匿名" {
		t.Fatalf("缺昵称应兜底匿名: %+v", second)
	}
	if second.Sex != "男" || second.Age != "0" {
		t.Fatalf("性别/年龄兜底错误: %+v", second)
	}
}

// TestBuildWanfengFavoriteUsers 校验收藏列表映射。
func TestBuildWanfengFavoriteUsers(t *testing.T) {
	favorites := []wanfeng.Favorite{
		{UID: "WFU9", Nickname: "甲", Gender: "male", Age: 30, Pinned: true},
		{UID: ""}, // 脏数据丢弃
	}
	users := buildWanfengFavoriteUsers(favorites)
	if len(users) != 1 {
		t.Fatalf("脏数据应被丢弃，实际 %d 条", len(users))
	}
	if users[0].ID != "WFU9" || users[0].Sex != "男" || !users[0].Pinned {
		t.Fatalf("收藏字段错误: %+v", users[0])
	}
	if users[0].Provider != "wanfeng" {
		t.Fatalf("收藏用户应带线路标记: %+v", users[0])
	}
}

// TestFormatHistoryTime 校验时间文案转换。
func TestFormatHistoryTime(t *testing.T) {
	if got := formatHistoryTime(""); got != "刚刚" {
		t.Fatalf("空值应显示刚刚: %s", got)
	}
	if got := formatHistoryTime(time.Now().Format(time.RFC3339)); got != "刚刚" {
		t.Fatalf("当前时间应显示刚刚: %s", got)
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if got := formatHistoryTime(yesterday.Format(time.RFC3339)); !strings.HasPrefix(got, "昨天") {
		t.Fatalf("昨天应显示昨天: %s", got)
	}
	// 无法解析的字符串原样返回（不吞掉信息）
	if got := formatHistoryTime("未知时间"); got != "未知时间" {
		t.Fatalf("无法解析时应原样返回: %s", got)
	}
}

// TestBuildWanfengHistoryContentsSelfFlag 校验自己发的消息用 md5(身份ID) 标记。
//
// 前端判定 isSelf 的依据是 content.id 是否等于「对方 id」，这里必须与一期实时消息一致。
func TestBuildWanfengHistoryContentsSelfFlag(t *testing.T) {
	a := &App{}
	page := &wanfeng.ChatHistoryPage{
		Messages: []wanfeng.HistoryMessage{
			{Tid: "t1", FromCid: "self", FromUID: "WFU1", Type: "text", Payload: "我发的", TS: 1699000000000},
			{Tid: "t2", FromCid: "peer", FromUID: "WFU2", Type: "text", Payload: "对方发的", TS: 1699000001000, Nickname: "小晚"},
			{Tid: "t3", FromCid: "peer", FromUID: "WFU2", Type: "text", Payload: "", TS: 1699000002000},
		},
		HasMore: true,
	}

	contents := a.buildWanfengHistoryContents(context.Background(), "IDENT-1", "WFU1", "WFU2", page)
	if !contents.HasMore {
		t.Fatal("hasMore 应透传")
	}
	if len(contents.ContentsList) != 2 {
		t.Fatalf("空内容应被丢弃，实际 %d 条", len(contents.ContentsList))
	}

	self := contents.ContentsList[0]
	if self.ID != md5HexLower("IDENT-1") {
		t.Fatalf("自己的消息应用 md5(身份ID) 标记: %+v", self)
	}
	if self.Content != "我发的" {
		t.Fatalf("文本内容错误: %+v", self)
	}

	peer := contents.ContentsList[1]
	if peer.ID != "WFU2" {
		t.Fatalf("对方消息应带对方 uid: %+v", peer)
	}
	if peer.Nickname != "小晚" {
		t.Fatalf("昵称丢失: %+v", peer)
	}
	// 时间要能被前端 Date 解析
	if _, err := time.Parse("2006-01-02 15:04:05.000", peer.Time); err != nil {
		t.Fatalf("时间格式前端无法解析: %s", peer.Time)
	}
}

// TestBuildWanfengHistoryContentsMediaMirror 校验媒体历史消息会搬回本地文件名。
func TestBuildWanfengHistoryContentsMediaMirror(t *testing.T) {
	bridge := &wanfengMediaBridge{
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
	}
	// 直接注入缓存，避免外呼
	bridge.inward[wanfeng.MediaURLKey("https://cdn.example.com/a.jpg")] = mediaCacheEntry{
		value:   "1700000000_abcd.jpg",
		expires: time.Now().Add(time.Hour),
	}

	a := &App{wanfengMedia: bridge}
	page := &wanfeng.ChatHistoryPage{
		Messages: []wanfeng.HistoryMessage{
			{Tid: "t1", FromCid: "peer", FromUID: "WFU2", Type: "image", Payload: "https://cdn.example.com/a.jpg", TS: 1699000000000},
		},
	}

	contents := a.buildWanfengHistoryContents(context.Background(), "IDENT-1", "WFU1", "WFU2", page)
	if len(contents.ContentsList) != 1 {
		t.Fatalf("应保留 1 条媒体消息，实际 %d", len(contents.ContentsList))
	}
	// 前端只认 "[文件名]" 编码，不能出现对方 URL
	if contents.ContentsList[0].Content != "[1700000000_abcd.jpg]" {
		t.Fatalf("媒体应转成本地文件名编码: %s", contents.ContentsList[0].Content)
	}
}

// TestFormatWanfengMessageTime 校验秒/毫秒时间戳都统一成毫秒再格式化。
func TestFormatWanfengMessageTime(t *testing.T) {
	// 1699000000 秒 与 1699000000000 毫秒应得到同一结果
	fromSec := formatWanfengMessageTime(1699000000)
	fromMs := formatWanfengMessageTime(1699000000000)
	if fromSec != fromMs {
		t.Fatalf("秒/毫秒结果不一致: %s vs %s", fromSec, fromMs)
	}
	if _, err := time.Parse("2006-01-02 15:04:05.000", fromMs); err != nil {
		t.Fatalf("时间格式不可解析: %s", fromMs)
	}
}

// newTestWanfengManager 用内存绑定库构造一个 manager（不连真实站点）。
func newTestWanfengManager(t *testing.T, bindings map[string]wanfeng.Binding) *WanfengManager {
	t.Helper()
	store := wanfeng.NewMemoryBindingStore()
	ctx := context.Background()
	for _, b := range bindings {
		if err := store.Save(ctx, b); err != nil {
			t.Fatalf("保存绑定失败: %v", err)
		}
	}
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{
		BaseURL:  "https://example.invalid",
		Protocol: wanfeng.DefaultProtocol(),
	})
	return NewWanfengManager(api, store, nil)
}

// TestServeWanfengHistoryUsersFallsBackForUnboundIdentity 校验未绑定晚风的身份仍走 chat2019。
func TestServeWanfengHistoryUsersFallsBackForUnboundIdentity(t *testing.T) {
	a := &App{wanfeng: newTestWanfengManager(t, nil)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)

	if a.serveWanfengHistoryUsers(rec, req, "chat2019-user", false) {
		t.Fatal("未绑定晚风的身份不应由晚风分支处理")
	}
	if a.serveWanfengMessageHistory(rec, req, "chat2019-user", "peer", "0") {
		t.Fatal("未绑定晚风的身份不应由晚风消息分支处理")
	}
	if a.serveWanfengFavorite(rec, req, "chat2019-user", "peer", true) {
		t.Fatal("未绑定晚风的身份不应由晚风收藏分支处理")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("回退时不应写响应: %s", rec.Body.String())
	}
}

// TestServeWanfengHistoryUsersWithoutManager 校验完全没启用晚风时保持原逻辑。
func TestServeWanfengHistoryUsersWithoutManager(t *testing.T) {
	a := &App{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	if a.serveWanfengHistoryUsers(rec, req, "u1", false) {
		t.Fatal("未启用晚风时应回退 chat2019")
	}
}

// TestServeWanfengHistoryUsersBoundReturnsEmptyOnError 校验上游失败时返回空列表而不是 5xx。
//
// 前端 loadHistoryUsers 只处理数组，返回 5xx 会让整块列表报错；空数组是更安全的降级。
func TestServeWanfengHistoryUsersBoundReturnsEmptyOnError(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList", nil)
	if !a.serveWanfengHistoryUsers(rec, req, "IDENT-1", false) {
		t.Fatal("已绑定晚风的身份应由晚风分支处理")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	var list []wanfengHistoryUser
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("响应应为数组: %v（body=%s）", err, rec.Body.String())
	}
	if len(list) != 0 {
		t.Fatalf("上游不可达时应返回空列表，实际 %d 条", len(list))
	}
}

// TestServeWanfengMessageHistoryBoundReturnsEmpty 校验历史消息上游失败时返回 code=0 空表。
func TestServeWanfengMessageHistoryBoundReturnsEmpty(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/getMessageHistory", nil)
	if !a.serveWanfengMessageHistory(rec, req, "IDENT-1", "WFU2", "0") {
		t.Fatal("已绑定晚风的身份应由晚风分支处理")
	}
	var resp wanfengHistoryContents
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("code 应为 0: %+v", resp)
	}
	if len(resp.ContentsList) != 0 {
		t.Fatalf("上游不可达时应返回空列表: %+v", resp)
	}
}

// TestServeWanfengFavoriteReturnsSuccessShape 校验收藏返回码与前端判断一致。
//
// 前端判定条件是 res.code === '0' || res.status === 'true'，两者都要满足其一。
func TestServeWanfengFavoriteReturnsSuccessShape(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/toggleFavorite", nil)
	if !a.serveWanfengFavorite(rec, req, "IDENT-1", "WFU2", true) {
		t.Fatal("已绑定晚风的身份应由晚风分支处理")
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if resp["code"] != "0" || resp["status"] != "true" {
		t.Fatalf("收藏成功返回码不符合前端判断: %+v", resp)
	}
}

// TestHandleGetHistoryUserListUsesWanfengBranch 校验老接口在晚风身份上不再打 chat2019 上游。
func TestHandleGetHistoryUserListUsesWanfengBranch(t *testing.T) {
	m := newTestWanfengManager(t, map[string]wanfeng.Binding{
		"IDENT-1": {IdentityID: "IDENT-1", UpstreamUID: "WFU1", IdentityToken: "v1.t", DeviceID: "d1"},
	})
	a := &App{wanfeng: m}

	form := url.Values{}
	form.Set("myUserID", "IDENT-1")
	req := httptest.NewRequest(http.MethodPost, "/api/getHistoryUserList",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	a.handleGetHistoryUserList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d（body=%s）", rec.Code, rec.Body.String())
	}
	// 走 chat2019 分支时会返回 {"error": ...}，这里必须是数组
	if strings.Contains(rec.Body.String(), "error") {
		t.Fatalf("不应回落到 chat2019 上游: %s", rec.Body.String())
	}
}

// TestBuildWanfengHistoryContentsExpiredMediaPlaceholder 校验过期媒体保留占位文字而不是整条消失。
func TestBuildWanfengHistoryContentsExpiredMediaPlaceholder(t *testing.T) {
	bridge := &wanfengMediaBridge{
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
		failed:  make(map[string]time.Time),
	}
	imageURL := "https://cdn.example.com/old.jpg"
	videoURL := "https://cdn.example.com/old.mp4"
	// 模拟「已判定过期」（对方媒体约 30 分钟就失效，救不回来）
	bridge.markFailed(wanfeng.MediaURLKey(imageURL))
	bridge.markFailed(wanfeng.MediaURLKey(videoURL))

	a := &App{wanfengMedia: bridge}
	page := &wanfeng.ChatHistoryPage{
		Messages: []wanfeng.HistoryMessage{
			{Tid: "t1", FromCid: "peer", FromUID: "WFU2", Type: "image", Payload: imageURL, TS: 1699000000000},
			{Tid: "t2", FromCid: "peer", FromUID: "WFU2", Type: "video", Payload: videoURL, TS: 1699000001000},
			{Tid: "t3", FromCid: "peer", FromUID: "WFU2", Type: "text", Payload: "还在吗", TS: 1699000002000},
		},
	}

	contents := a.buildWanfengHistoryContents(context.Background(), "IDENT-1", "WFU1", "WFU2", page)
	if len(contents.ContentsList) != 3 {
		t.Fatalf("过期媒体不应导致消息消失，实际 %d 条", len(contents.ContentsList))
	}
	if got := contents.ContentsList[0].Content; got != "[图片已过期]" {
		t.Fatalf("图片占位不符: %q", got)
	}
	if got := contents.ContentsList[1].Content; got != "[视频已过期]" {
		t.Fatalf("视频占位不符: %q", got)
	}
	if got := contents.ContentsList[2].Content; got != "还在吗" {
		t.Fatalf("文本消息被污染: %q", got)
	}
}
