package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"liao/internal/upstream/wanfeng"
)

const (
	// 最后一个下游断开后，延迟多久关闭上游连接（保留极短窗口便于刷新重连）。
	wanfengCloseDelay = 5 * time.Second
)

// WanfengManager 管理「我方身份 → 晚风上游连接」的连接池与协议翻译。
//
// 与 chat2019 线路的差异：
//   - 对方是「服务端全局匹配池 + 加密帧」，身份由对方下发（需持久化绑定）；
//   - 同一身份的多端共享一条上游连接（与现有 UpstreamWebSocketManager 一致）。
type WanfengManager struct {
	api      *wanfeng.APIClient
	bindings wanfeng.BindingStore
	logger   *slog.Logger

	mu              sync.Mutex
	media           wanfeng.MediaBridge
	matchPreference MatchPreferenceProvider
	clients         map[string]*wanfeng.Client
	profiles        map[string]wanfeng.Profile
	sessions        map[string]map[*DownstreamSession]struct{}
	pendingClose    map[string]*time.Timer
}

// NewWanfengManager 创建晚风线路管理器。
func NewWanfengManager(api *wanfeng.APIClient, bindings wanfeng.BindingStore, logger *slog.Logger) *WanfengManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &WanfengManager{
		api:          api,
		bindings:     bindings,
		logger:       logger.With("mod", "wanfeng-manager"),
		clients:      make(map[string]*wanfeng.Client),
		profiles:     make(map[string]wanfeng.Profile),
		sessions:     make(map[string]map[*DownstreamSession]struct{}),
		pendingClose: make(map[string]*time.Timer),
	}
}

// RegisterDownstream 注册一个下游会话（对应前端一次 WS 连接）。
// signMessage 是前端首帧（act=sign），用于提取昵称/性别等展示资料。
func (m *WanfengManager) RegisterDownstream(userID string, session *DownstreamSession, signMessage string) {
	userID = strings.TrimSpace(userID)
	if m == nil || m.api == nil || userID == "" || session == nil {
		return
	}

	profile := m.profileFromSign(signMessage)
	// 本地匹配偏好要随登录帧上送，否则对方站点按默认偏好匹配
	profile = m.applyMatchPreference(userID, profile)

	var shouldCreate bool
	m.mu.Lock()
	if t := m.pendingClose[userID]; t != nil {
		t.Stop()
		delete(m.pendingClose, userID)
	}
	sessions := m.sessions[userID]
	if sessions == nil {
		sessions = make(map[*DownstreamSession]struct{})
		m.sessions[userID] = sessions
	}
	sessions[session] = struct{}{}
	if prof, ok := m.profiles[userID]; !ok || prof.Nickname == "" {
		m.profiles[userID] = profile
	} else if profile.Nickname != "" {
		m.profiles[userID] = profile
	}
	if _, ok := m.clients[userID]; !ok {
		shouldCreate = true
	}
	m.mu.Unlock()

	if shouldCreate {
		m.createClient(userID)
	}
}

// UnregisterDownstream 注销下游会话；最后一个会话断开后延迟关闭上游连接。
func (m *WanfengManager) UnregisterDownstream(userID string, session *DownstreamSession) {
	userID = strings.TrimSpace(userID)
	if m == nil || userID == "" || session == nil {
		return
	}

	m.mu.Lock()
	sessions := m.sessions[userID]
	if sessions != nil {
		delete(sessions, session)
	}
	if len(sessions) == 0 {
		delete(m.sessions, userID)
		if t := m.pendingClose[userID]; t != nil {
			t.Stop()
		}
		m.pendingClose[userID] = time.AfterFunc(wanfengCloseDelay, func() {
			m.mu.Lock()
			_, stillOpen := m.sessions[userID]
			if stillOpen {
				delete(m.pendingClose, userID)
				m.mu.Unlock()
				return
			}
			client := m.clients[userID]
			delete(m.clients, userID)
			delete(m.pendingClose, userID)
			m.mu.Unlock()
			if client != nil {
				client.Close()
			}
		})
	}
	m.mu.Unlock()
}

// SendToUpstream 把前端一帧（act 协议）翻译成晚风动作并下发。
func (m *WanfengManager) SendToUpstream(userID string, message string) {
	userID = strings.TrimSpace(userID)
	if m == nil || m.api == nil || userID == "" {
		return
	}

	frame, err := wanfeng.ParseClientFrame(message)
	if err != nil {
		m.logger.Warn("解析前端帧失败", "userID", userID, "error", err)
		return
	}

	// sign 帧用于更新展示资料（含匹配偏好），不当作聊天动作转发
	if strings.TrimSpace(frame.Act) == "sign" {
		profile := m.profileFromFrame(frame)
		m.mu.Lock()
		previous, hadPrevious := m.profiles[userID]
		m.profiles[userID] = profile
		client := m.clients[userID]
		m.mu.Unlock()

		// 登录帧已带过资料时，资料变更（尤其是匹配偏好）必须走 update_profile，
		// 否则对方站点不会生效。
		if client != nil && hadPrevious && !profileEqual(previous, profile) {
			client.SendCommand(wanfeng.OutgoingCommand{
				Action:  m.Protocol().ActionUpdateProfile,
				Payload: m.Protocol().UpdateProfilePayload(profile),
			})
		}
		return
	}

	proto := m.api.Protocol()
	cmd, ok := proto.BuildCommand(frame, wanfeng.TargetFromAct(frame.Act))
	if !ok {
		if strings.HasPrefix(strings.TrimSpace(frame.Act), "touser_") {
			m.logger.Warn("消息缺少目标用户，已忽略", "userID", userID, "act", frame.Act)
		}
		return
	}

	m.mu.Lock()
	client := m.clients[userID]
	m.mu.Unlock()
	if client == nil {
		m.logger.Warn("晚风连接不存在，忽略下发", "userID", userID)
		return
	}
	client.SendCommand(cmd)

	// 匹配时立即回执，让前端 UI 进入「匹配中」状态（对方后续也会推 user_info/MATCHING）
	if cmd.Action == proto.ActionJoinMatch {
		m.Emit(userID, proto.MatchingMessage())
	}
}

// SendAction 直接向该身份的上游连接下发一个动作（收藏/已读/匹配偏好等旁路操作）。
//
// 与 SendToUpstream 的区别：这里由后端主动触发，不经过前端帧翻译。
func (m *WanfengManager) SendAction(identityID, action string, payload map[string]any) error {
	identityID = strings.TrimSpace(identityID)
	action = strings.TrimSpace(action)
	if m == nil || identityID == "" || action == "" {
		return fmt.Errorf("缺少身份或动作")
	}
	if payload == nil {
		payload = map[string]any{}
	}

	m.mu.Lock()
	client := m.clients[identityID]
	m.mu.Unlock()
	if client == nil {
		return fmt.Errorf("晚风连接未就绪，请稍后重试")
	}
	client.SendCommand(wanfeng.OutgoingCommand{Action: action, Payload: payload})
	return nil
}

// Emit 实现 wanfeng.Emitter：把业务消息广播给该身份的所有下游会话。
func (m *WanfengManager) Emit(identityID string, payload map[string]any) {
	if m == nil || payload == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		m.logger.Warn("序列化下游消息失败", "identityID", identityID, "error", err)
		return
	}
	m.broadcast(identityID, string(raw))
}

// EmitFatal 实现 wanfeng.Emitter：发送致命错误后关闭该身份的上游与下游连接。
func (m *WanfengManager) EmitFatal(identityID string, payload map[string]any) {
	m.Emit(identityID, payload)
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return
	}
	time.AfterFunc(time.Second, func() {
		m.mu.Lock()
		client := m.clients[identityID]
		delete(m.clients, identityID)
		sessions := make([]*DownstreamSession, 0, len(m.sessions[identityID]))
		for s := range m.sessions[identityID] {
			sessions = append(sessions, s)
		}
		delete(m.sessions, identityID)
		if t := m.pendingClose[identityID]; t != nil {
			t.Stop()
			delete(m.pendingClose, identityID)
		}
		m.mu.Unlock()

		if client != nil {
			client.Close()
		}
		for _, s := range sessions {
			_ = s.Close()
		}
	})
}

// CloseAll 关闭全部上游连接（服务优雅关闭时调用）。
func (m *WanfengManager) CloseAll() {
	if m == nil {
		return
	}
	m.mu.Lock()
	clients := make([]*wanfeng.Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	downstream := make([]*DownstreamSession, 0)
	for _, sessions := range m.sessions {
		for s := range sessions {
			downstream = append(downstream, s)
		}
	}
	for _, t := range m.pendingClose {
		t.Stop()
	}
	m.clients = make(map[string]*wanfeng.Client)
	m.sessions = make(map[string]map[*DownstreamSession]struct{})
	m.pendingClose = make(map[string]*time.Timer)
	m.profiles = make(map[string]wanfeng.Profile)
	m.mu.Unlock()

	for _, c := range clients {
		c.Close()
	}
	for _, s := range downstream {
		if s != nil {
			_ = s.Close()
		}
	}
}

// SetMatchPreferenceProvider 注入「读取本地匹配偏好」的能力（由 App 装配时调用）。
//
// 用函数注入而不是直接依赖服务，是为了让 manager 不反向依赖 App/数据库。
func (m *WanfengManager) SetMatchPreferenceProvider(fn MatchPreferenceProvider) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.matchPreference = fn
	m.mu.Unlock()
}

// applyMatchPreference 把本地偏好并入要上送的展示资料。
func (m *WanfengManager) applyMatchPreference(identityID string, profile wanfeng.Profile) wanfeng.Profile {
	m.mu.Lock()
	fn := m.matchPreference
	m.mu.Unlock()
	if fn == nil {
		return profile
	}
	pref, err := fn(identityID)
	if err != nil {
		return profile
	}
	// 全默认值时不带，避免覆盖对方侧已有偏好
	if pref.PreferGender == PreferGenderAny && pref.PreferAgeMin == 0 &&
		pref.PreferAgeMax == 0 && !pref.PreferSameProvince {
		return profile
	}
	profile.MatchPreference = pref.UpstreamPayload()
	return profile
}

// Supported 表示晚风线路是否已启用（api 已配置）。
func (m *WanfengManager) Supported() bool {
	return m != nil && m.api != nil
}

// API 返回底层 HTTP 客户端（只读，供历史/媒体等旁路调用）。
func (m *WanfengManager) API() *wanfeng.APIClient {
	if m == nil {
		return nil
	}
	return m.api
}

// Protocol 返回晚风协议参数（只读；含可热覆盖的动作名）。
func (m *WanfengManager) Protocol() wanfeng.Protocol {
	if m == nil || m.api == nil {
		return wanfeng.DefaultProtocol()
	}
	return m.api.Protocol()
}

// ProfileOf 返回某个身份最近一次上送的展示资料。
func (m *WanfengManager) ProfileOf(identityID string) (wanfeng.Profile, bool) {
	if m == nil {
		return wanfeng.Profile{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	profile, ok := m.profiles[identityID]
	return profile, ok
}

// BindingOf 返回某个身份的晚风绑定（含 uid 与 identityToken）。
func (m *WanfengManager) BindingOf(ctx context.Context, identityID string) (*wanfeng.Binding, error) {
	if m == nil || m.bindings == nil {
		return nil, nil
	}
	return m.bindings.Get(ctx, identityID)
}

// SetMediaBridge 设置媒体搬运桥（二期图片/视频），同时下发给已存在的客户端。
func (m *WanfengManager) SetMediaBridge(bridge wanfeng.MediaBridge) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.media = bridge
	clients := make([]*wanfeng.Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.mu.Unlock()

	for _, c := range clients {
		c.SetMediaBridge(bridge)
	}
}

// Stats 返回线路统计（供 /api/system/wsStats 之类的运维接口使用）。
func (m *WanfengManager) Stats() map[string]any {
	if m == nil {
		return map[string]any{"enabled": false}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	downstream := 0
	for _, sessions := range m.sessions {
		downstream += len(sessions)
	}
	return map[string]any{
		"enabled":    true,
		"upstream":   len(m.clients),
		"downstream": downstream,
	}
}

func (m *WanfengManager) broadcast(identityID string, message string) {
	m.mu.Lock()
	sessions := make([]*DownstreamSession, 0, len(m.sessions[identityID]))
	for s := range m.sessions[identityID] {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	for _, s := range sessions {
		if s == nil {
			continue
		}
		if err := s.SendText(message); err != nil {
			_ = s.Close()
			m.UnregisterDownstream(identityID, s)
		}
	}
}

func (m *WanfengManager) createClient(userID string) {
	m.mu.Lock()
	profile, ok := m.profiles[userID]
	if !ok {
		profile = wanfeng.Profile{Gender: "secret"}
	}
	m.mu.Unlock()

	profile = m.applyMatchPreference(userID, profile)

	m.mu.Lock()
	media := m.media
	m.mu.Unlock()

	client := wanfeng.NewClient(wanfeng.ClientConfig{
		IdentityID: userID,
		Profile:    profile,
		API:        m.api,
		Bindings:   m.bindings,
		Emitter:    m,
		Media:      media,
		Logger:     m.logger,
	})

	m.mu.Lock()
	if existing := m.clients[userID]; existing != nil {
		m.mu.Unlock()
		return
	}
	m.clients[userID] = client
	_, hasSessions := m.sessions[userID]
	m.mu.Unlock()

	if !hasSessions {
		// 极端场景：创建过程中下游已断开，直接回收
		client.Close()
		m.mu.Lock()
		delete(m.clients, userID)
		m.mu.Unlock()
		return
	}

	client.Start(context.Background())
	m.logger.Info("已创建晚风上游连接", "userID", userID, "profile", describeProfile(profile))
}

// profileFromSign 从 sign 帧解析展示资料。
func (m *WanfengManager) profileFromSign(signMessage string) wanfeng.Profile {
	frame, err := wanfeng.ParseClientFrame(signMessage)
	if err != nil {
		return wanfeng.Profile{Gender: "secret"}
	}
	return m.profileFromFrame(frame)
}

func (m *WanfengManager) profileFromFrame(frame wanfeng.ClientFrame) wanfeng.Profile {
	age, _ := strconv.Atoi(strings.TrimSpace(frame.UserAge))
	if age < 0 {
		age = 0
	}
	return wanfeng.Profile{
		Nickname:        strings.TrimSpace(frame.Name),
		Gender:          wanfeng.SexToUpstream(frame.UserSex),
		Age:             age,
		MatchPreference: frame.MatchPreference,
	}
}

// profileEqual 判断两份资料是否等价（匹配偏好按 JSON 比较，避免 map 直接比较 panic）。
func profileEqual(a, b wanfeng.Profile) bool {
	if a.Nickname != b.Nickname || a.Gender != b.Gender || a.Age != b.Age {
		return false
	}
	return matchPreferenceJSON(a.MatchPreference) == matchPreferenceJSON(b.MatchPreference)
}

// matchPreferenceJSON 把匹配偏好序列化成稳定字符串用于比较。
func matchPreferenceJSON(pref map[string]any) string {
	if len(pref) == 0 {
		return ""
	}
	raw, err := json.Marshal(pref)
	if err != nil {
		return fmt.Sprintf("%v", pref)
	}
	return string(raw)
}

// describeProfile 便于日志排查。
func describeProfile(p wanfeng.Profile) string {
	return fmt.Sprintf("nickname=%s gender=%s age=%d", p.Nickname, p.Gender, p.Age)
}
