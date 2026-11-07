// Package wanfeng 实现「晚风匿名聊天（wanfengniming.store）」上游线路的协议适配。
//
// 设计要点：
//  1. 与对方协议相关的一切易变参数（接口路径、帧动作名、加密信封版本、HKDF info 等）都集中在本文件，
//     对方升级改字段时，只需调整这里的默认值或通过 WANFENG_PROTOCOL_OVERRIDES(JSON) 覆盖，无需改动业务代码。
//  2. 读取对方字段时使用带候选键的 pick 函数（pickString/pickInt），对方新增/改名个别字段时仍可兼容。
//  3. 未知帧类型、解密失败等一律「降级跳过」而不是中断连接，避免对方小改动导致整条线路不可用。
package wanfeng

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Protocol 集中描述与对方站点协议相关的全部可配置项。
// 所有字段都可以用 WANFENG_PROTOCOL_OVERRIDES 中的同名 JSON 字段部分覆盖。
type Protocol struct {
	// —— HTTP 接口路径 ——
	HandshakePath        string `json:"handshakePath"`
	GenerateIdentityPath string `json:"generateIdentityPath"`
	IdentityCookiePath   string `json:"identityCookiePath"`
	ConfigAppInfoPath    string `json:"configAppInfoPath"`

	// —— HTTP 请求头 ——
	GetViaPostHeader string `json:"getViaPostHeader"`
	GetViaPostValue  string `json:"getViaPostValue"`
	SessionHeader    string `json:"sessionHeader"`
	UserAgent        string `json:"userAgent"`
	AcceptHeader     string `json:"acceptHeader"`

	// —— WebSocket ——
	WSPath           string `json:"wsPath"`
	WSTimestampParam string `json:"wsTimestampParam"`

	// —— 加密信封 ——
	// EnvelopeVersion 是请求体/帧里的版本号字段值（对方当前为 3）。
	EnvelopeVersion int `json:"envelopeVersion"`
	// EnvelopePrefix 是响应 data 与 WS 帧 data 的密文前缀（对方当前为 "v3."）。
	EnvelopePrefix string `json:"envelopePrefix"`
	// HKDFInfo 是会话密钥派生时的 HKDF info 常量。
	HKDFInfo string `json:"hkdfInfo"`

	// —— 帧结构字段名 ——
	FrameTypeField    string `json:"frameTypeField"`
	FrameDataField    string `json:"frameDataField"`
	FrameSIDField     string `json:"frameSidField"`
	FrameVersionField string `json:"frameVersionField"`

	// —— 下行动作名（我方 → 对方）——
	ActionLogin         string `json:"actionLogin"`
	ActionJoinMatch     string `json:"actionJoinMatch"`
	ActionCancelMatch   string `json:"actionCancelMatch"`
	ActionSendMessage   string `json:"actionSendMessage"`
	ActionExitChat      string `json:"actionExitChat"`
	ActionHeartbeat     string `json:"actionHeartbeat"`
	ActionUpdateProfile string `json:"actionUpdateProfile"`

	// —— 上行事件名（对方 → 我方）——
	EventConnected       string `json:"eventConnected"`
	EventUserInfo        string `json:"eventUserInfo"`
	EventMatchResult     string `json:"eventMatchResult"`
	EventReceiveMessage  string `json:"eventReceiveMessage"`
	EventMessageRecalled string `json:"eventMessageRecalled"`
	EventPartnerLeft     string `json:"eventPartnerLeft"`
	EventError           string `json:"eventError"`
	EventMessagesRead    string `json:"eventMessagesRead"`

	// —— 业务字段容错候选键（对方改字段名时在此追加，不需改代码）——
	TargetUIDKeys     []string `json:"targetUidKeys"`
	PartnerKeys       []string `json:"partnerKeys"`
	ContentKeys       []string `json:"contentKeys"`
	NicknameKeys      []string `json:"nicknameKeys"`
	GenderKeys        []string `json:"genderKeys"`
	AgeKeys           []string `json:"ageKeys"`
	CityKeys          []string `json:"cityKeys"`
	TimeKeys          []string `json:"timeKeys"`
	MessageIDKeys     []string `json:"messageIdKeys"`
	FromUIDKeys       []string `json:"fromUidKeys"`
	TypeKeys          []string `json:"typeKeys"`
	StateKeys         []string `json:"stateKeys"`
	UIDKeys           []string `json:"uidKeys"`
	IdentityTokenKeys []string `json:"identityTokenKeys"`
	ErrorCodeKeys     []string `json:"errorCodeKeys"`

	// —— 匹配状态常量（对方状态机值）——
	StateIdle     string `json:"stateIdle"`
	StateMatching string `json:"stateMatching"`
	StateInRoom   string `json:"stateInRoom"`

	// —— 行为开关 ——
	// DefaultAge 是我方身份没有年龄信息时上送的年龄（0 表示不展示）。
	DefaultAge int `json:"defaultAge"`
	// PingSeconds 是 WS 心跳间隔（秒）。
	PingSeconds int `json:"pingSeconds"`
	// LoginTimeoutSeconds 是等待 connected 帧的超时（秒）。
	LoginTimeoutSeconds int `json:"loginTimeoutSeconds"`
}

// DefaultProtocol 返回对方站点当前（2026-11）的协议参数。
func DefaultProtocol() Protocol {
	return Protocol{
		HandshakePath:        "/api/app/handshake",
		GenerateIdentityPath: "/api/tools/util/generate",
		IdentityCookiePath:   "/api/nimingChat/account/identity-cookie",
		ConfigAppInfoPath:    "/api/nimingChat/config/app-info",

		GetViaPostHeader: "X-Get-Via-Post",
		GetViaPostValue:  "1",
		SessionHeader:    "X-Enc-Sid",
		UserAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		AcceptHeader:     "application/json, text/plain, */*",

		WSPath:           "/ws",
		WSTimestampParam: "_t",

		EnvelopeVersion: 3,
		EnvelopePrefix:  "v3.",
		HKDFInfo:        "wf-enc-v3",

		FrameTypeField:    "type",
		FrameDataField:    "data",
		FrameSIDField:     "sid",
		FrameVersionField: "v",

		ActionLogin:         "login",
		ActionJoinMatch:     "join_match",
		ActionCancelMatch:   "cancel_match",
		ActionSendMessage:   "send_message",
		ActionExitChat:      "exit_chat",
		ActionHeartbeat:     "heartbeat",
		ActionUpdateProfile: "update_profile",

		EventConnected:       "connected",
		EventUserInfo:        "user_info",
		EventMatchResult:     "match_result",
		EventReceiveMessage:  "receive_message",
		EventMessageRecalled: "message_recalled",
		EventPartnerLeft:     "partner_left",
		EventError:           "error",
		EventMessagesRead:    "messages_read",

		TargetUIDKeys:     []string{"targetUid", "target_uid", "toUid"},
		PartnerKeys:       []string{"partner", "peer", "target"},
		ContentKeys:       []string{"content", "payload", "text", "msg"},
		NicknameKeys:      []string{"nickname", "name", "nick"},
		GenderKeys:        []string{"gender", "sex"},
		AgeKeys:           []string{"age"},
		CityKeys:          []string{"city", "address", "area"},
		TimeKeys:          []string{"ts", "time", "timestamp"},
		MessageIDKeys:     []string{"tid", "msgId", "id"},
		FromUIDKeys:       []string{"fromUid", "from_uid", "from"},
		TypeKeys:          []string{"type"},
		StateKeys:         []string{"state", "status"},
		ErrorCodeKeys:     []string{"code", "errorCode"},
		UIDKeys:           []string{"uid", "id"},
		IdentityTokenKeys: []string{"identityToken", "identity_token", "token"},

		StateIdle:     "IDLE",
		StateMatching: "MATCHING",
		StateInRoom:   "IN_ROOM",

		DefaultAge:          0,
		PingSeconds:         25,
		LoginTimeoutSeconds: 10,
	}
}

// ApplyOverrides 用 JSON 覆盖默认协议参数（只覆盖出现的字段）。
// 典型用法：对方改版后，通过环境变量 WANFENG_PROTOCOL_OVERRIDES 传入
// {"actionJoinMatch":"joinMatch","eventMatchResult":"matched","hkdfInfo":"wf-enc-v4"} 即可热修，无需重新发版。
func (p Protocol) ApplyOverrides(raw string) (Protocol, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return p, nil
	}
	out := p
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return p, fmt.Errorf("解析协议覆盖配置失败: %w", err)
	}
	out.applyDefaults()
	return out, nil
}

// applyDefaults 补齐覆盖后可能为空的字段，避免空串导致请求异常。
func (p *Protocol) applyDefaults() {
	def := DefaultProtocol()
	if strings.TrimSpace(p.HandshakePath) == "" {
		p.HandshakePath = def.HandshakePath
	}
	if strings.TrimSpace(p.GenerateIdentityPath) == "" {
		p.GenerateIdentityPath = def.GenerateIdentityPath
	}
	if strings.TrimSpace(p.GetViaPostHeader) == "" {
		p.GetViaPostHeader = def.GetViaPostHeader
	}
	if strings.TrimSpace(p.WSPath) == "" {
		p.WSPath = def.WSPath
	}
	if strings.TrimSpace(p.WSTimestampParam) == "" {
		p.WSTimestampParam = def.WSTimestampParam
	}
	if p.EnvelopeVersion == 0 {
		p.EnvelopeVersion = def.EnvelopeVersion
	}
	if strings.TrimSpace(p.EnvelopePrefix) == "" {
		p.EnvelopePrefix = def.EnvelopePrefix
	}
	if strings.TrimSpace(p.HKDFInfo) == "" {
		p.HKDFInfo = def.HKDFInfo
	}
	if strings.TrimSpace(p.FrameTypeField) == "" {
		p.FrameTypeField = def.FrameTypeField
	}
	if strings.TrimSpace(p.FrameDataField) == "" {
		p.FrameDataField = def.FrameDataField
	}
	if strings.TrimSpace(p.FrameSIDField) == "" {
		p.FrameSIDField = def.FrameSIDField
	}
	if strings.TrimSpace(p.FrameVersionField) == "" {
		p.FrameVersionField = def.FrameVersionField
	}
	if strings.TrimSpace(p.ActionLogin) == "" {
		p.ActionLogin = def.ActionLogin
	}
	if strings.TrimSpace(p.ActionJoinMatch) == "" {
		p.ActionJoinMatch = def.ActionJoinMatch
	}
	if strings.TrimSpace(p.ActionCancelMatch) == "" {
		p.ActionCancelMatch = def.ActionCancelMatch
	}
	if strings.TrimSpace(p.ActionSendMessage) == "" {
		p.ActionSendMessage = def.ActionSendMessage
	}
	if strings.TrimSpace(p.ActionExitChat) == "" {
		p.ActionExitChat = def.ActionExitChat
	}
	if strings.TrimSpace(p.ActionHeartbeat) == "" {
		p.ActionHeartbeat = def.ActionHeartbeat
	}
	if strings.TrimSpace(p.EventConnected) == "" {
		p.EventConnected = def.EventConnected
	}
	if strings.TrimSpace(p.EventUserInfo) == "" {
		p.EventUserInfo = def.EventUserInfo
	}
	if strings.TrimSpace(p.EventMatchResult) == "" {
		p.EventMatchResult = def.EventMatchResult
	}
	if strings.TrimSpace(p.EventReceiveMessage) == "" {
		p.EventReceiveMessage = def.EventReceiveMessage
	}
	if strings.TrimSpace(p.EventMessageRecalled) == "" {
		p.EventMessageRecalled = def.EventMessageRecalled
	}
	if strings.TrimSpace(p.EventPartnerLeft) == "" {
		p.EventPartnerLeft = def.EventPartnerLeft
	}
	if strings.TrimSpace(p.EventError) == "" {
		p.EventError = def.EventError
	}
	if len(p.TargetUIDKeys) == 0 {
		p.TargetUIDKeys = def.TargetUIDKeys
	}
	if len(p.PartnerKeys) == 0 {
		p.PartnerKeys = def.PartnerKeys
	}
	if len(p.ContentKeys) == 0 {
		p.ContentKeys = def.ContentKeys
	}
	if len(p.NicknameKeys) == 0 {
		p.NicknameKeys = def.NicknameKeys
	}
	if len(p.GenderKeys) == 0 {
		p.GenderKeys = def.GenderKeys
	}
	if len(p.AgeKeys) == 0 {
		p.AgeKeys = def.AgeKeys
	}
	if len(p.CityKeys) == 0 {
		p.CityKeys = def.CityKeys
	}
	if len(p.TimeKeys) == 0 {
		p.TimeKeys = def.TimeKeys
	}
	if len(p.MessageIDKeys) == 0 {
		p.MessageIDKeys = def.MessageIDKeys
	}
	if len(p.FromUIDKeys) == 0 {
		p.FromUIDKeys = def.FromUIDKeys
	}
	if len(p.TypeKeys) == 0 {
		p.TypeKeys = def.TypeKeys
	}
	if len(p.ErrorCodeKeys) == 0 {
		p.ErrorCodeKeys = def.ErrorCodeKeys
	}
	if len(p.StateKeys) == 0 {
		p.StateKeys = def.StateKeys
	}
	if strings.TrimSpace(p.StateIdle) == "" {
		p.StateIdle = def.StateIdle
	}
	if strings.TrimSpace(p.StateMatching) == "" {
		p.StateMatching = def.StateMatching
	}
	if strings.TrimSpace(p.StateInRoom) == "" {
		p.StateInRoom = def.StateInRoom
	}
	if p.PingSeconds <= 0 {
		p.PingSeconds = def.PingSeconds
	}
	if p.LoginTimeoutSeconds <= 0 {
		p.LoginTimeoutSeconds = def.LoginTimeoutSeconds
	}
}

// —— 容错取值辅助 ——

// pickAny 依次尝试多个候选键，返回第一个非空值。
func pickAny(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

// PickString 依次尝试多个候选键并转成字符串（对方改字段名时由协议配置兜住）。
func PickString(m map[string]any, keys ...string) string {
	switch v := pickAny(m, keys...).(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strings.TrimSpace(formatFloat(v))
	case json.Number:
		return strings.TrimSpace(v.String())
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// PickInt 依次尝试多个候选键并转成 int。
func PickInt(m map[string]any, keys ...string) int {
	switch v := pickAny(m, keys...).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err == nil {
			return n
		}
	}
	return 0
}

// PickMap 依次尝试多个候选键并取子对象。
func PickMap(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if sub, ok := m[k].(map[string]any); ok && sub != nil {
			return sub
		}
	}
	return map[string]any{}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}
