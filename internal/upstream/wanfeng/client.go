package wanfeng

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// ws 写超时
	clientWriteDeadline = 10 * time.Second
	// ws 读超时（对方会定时推送/心跳，超时视为断线）
	clientReadDeadline = 150 * time.Second
	// 未连接时最多缓存的下发动作数量
	clientMaxPending = 16
	// 重连退避序列
	clientReconnectMax = 5
	// 连续解密失败多少次后重建握手
	clientMaxDecryptFailures = 5
)

var clientReconnectDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
}

// Emitter 是客户端向「下游（我方前端）」投递消息的出口。
type Emitter interface {
	// Emit 把一帧业务消息发给该身份的所有下游会话。
	Emit(identityID string, payload map[string]any)
	// EmitFatal 发送致命错误（发送后连接会被关闭）。
	EmitFatal(identityID string, payload map[string]any)
}

// ClientConfig 是 Client 构造参数。
type ClientConfig struct {
	IdentityID string
	Profile    Profile
	API        *APIClient
	Bindings   BindingStore
	Emitter    Emitter
	Logger     *slog.Logger
	Dialer     *websocket.Dialer
}

// Client 维护「我方一个身份 ↔ 对方一条连接」的长连接与协议翻译。
type Client struct {
	cfg ClientConfig
	log *slog.Logger

	dialer *websocket.Dialer

	mu              sync.Mutex
	conn            *websocket.Conn
	session         *Session
	binding         *Binding
	ready           bool
	closed          bool
	expectedClose   bool
	pending         []OutgoingCommand
	partnerUID      string
	partnerNickname string
	state           string
	decryptFailures int

	writeMu   sync.Mutex
	closeOnce sync.Once
	done      chan struct{}
}

// NewClient 创建上游客户端。
func NewClient(cfg ClientConfig) *Client {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	return &Client{
		cfg:    cfg,
		log:    logger.With("mod", "wanfeng-client", "identityID", cfg.IdentityID),
		dialer: dialer,
		done:   make(chan struct{}),
	}
}

// PartnerUID 返回当前聊天对象 uid（无则空）。
func (c *Client) PartnerUID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.partnerUID
}

// IsReady 返回连接是否已登录就绪。
func (c *Client) IsReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready && !c.closed
}

// Start 启动连接（异步）。
func (c *Client) Start(ctx context.Context) {
	go c.run(ctx)
}

// Close 主动关闭（会尽力通知对方「退出会话/取消匹配」）。
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.expectedClose = true
		c.closed = true
		conn := c.conn
		c.conn = nil
		c.ready = false
		state := c.state
		c.mu.Unlock()

		close(c.done)

		if conn != nil {
			// 尽力通知对方，避免继续占用匹配队列
			switch state {
			case c.proto().StateInRoom:
				c.sendFrameOn(conn, c.proto().ActionExitChat, map[string]any{})
			case c.proto().StateMatching:
				c.sendFrameOn(conn, c.proto().ActionCancelMatch, map[string]any{})
			}
			c.writeMu.Lock()
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(3*time.Second))
			_ = conn.Close()
			c.writeMu.Unlock()
		}
	})
}

// SendCommand 下发动作为对方帧；未就绪时先入队。
func (c *Client) SendCommand(cmd OutgoingCommand) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	conn := c.conn
	ready := c.ready
	if !ready || conn == nil {
		if len(c.pending) >= clientMaxPending {
			c.pending = c.pending[1:]
		}
		c.pending = append(c.pending, cmd)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	c.sendFrameOn(conn, cmd.Action, cmd.Payload)
}

// run 是连接主循环：断线自动重连，致命错误退出。
func (c *Client) run(ctx context.Context) {
	attempt := 0
	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			c.Close()
			return
		default:
		}

		fatal, err := c.serveOnce(ctx)
		if fatal {
			c.log.Warn("晚风连接致命错误，停止重连", "error", err)
			return
		}
		select {
		case <-c.done:
			return
		default:
		}

		if err != nil {
			c.log.Warn("晚风连接中断", "error", err, "attempt", attempt+1)
		}
		if attempt >= clientReconnectMax {
			c.log.Warn("晚风连接重连次数已达上限，放弃")
			c.cfg.Emitter.EmitFatal(c.cfg.IdentityID, map[string]any{
				"code":     codeForbidden,
				"content":  "当前线路连接失败，请稍后重试或切换线路",
				"forceout": true,
			})
			return
		}
		delayIdx := attempt
		if delayIdx >= len(clientReconnectDelays) {
			delayIdx = len(clientReconnectDelays) - 1
		}
		delay := clientReconnectDelays[delayIdx]
		attempt++
		select {
		case <-c.done:
			return
		case <-time.After(delay):
		}
	}
}

// serveOnce 建立一次连接并跑读循环，直到断开。
// 返回 fatal=true 表示不应重连。
func (c *Client) serveOnce(ctx context.Context) (bool, error) {
	binding, err := c.ensureBinding(ctx)
	if err != nil {
		return true, err
	}

	sess, err := c.cfg.API.Session(ctx)
	if err != nil {
		return false, err
	}

	url := c.cfg.API.BuildWSURL()
	header := websocketHeader(c.cfg.API)
	dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	conn, _, err := c.dialer.DialContext(dialCtx, url, header)
	if err != nil {
		return false, fmt.Errorf("连接晚风 WS 失败: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.session = sess
	c.binding = binding
	c.ready = false
	c.decryptFailures = 0
	pending := append([]OutgoingCommand(nil), c.pending...)
	c.pending = nil
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
			c.ready = false
		}
		c.mu.Unlock()
		_ = conn.Close()
	}()

	_ = conn.SetReadDeadline(time.Now().Add(clientReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(clientReadDeadline))
	})

	// 1) 登录
	resetState := true
	if st := c.currentState(); st == c.proto().StateMatching || st == c.proto().StateInRoom {
		resetState = false
	}
	if err := c.sendFrameOn(conn, c.proto().ActionLogin, c.proto().LoginPayload(*binding, c.cfg.Profile, resetState)); err != nil {
		return false, fmt.Errorf("发送登录帧失败: %w", err)
	}

	// 2) 心跳
	stopHeartbeat := make(chan struct{})
	go c.heartbeatLoop(conn, stopHeartbeat)
	defer close(stopHeartbeat)

	// 3) 登录超时看门狗
	loginTimer := time.AfterFunc(time.Duration(c.proto().LoginTimeoutSeconds)*time.Second, func() {
		if !c.IsReady() {
			c.log.Warn("晚风登录超时，主动断开重连")
			_ = conn.Close()
		}
	})
	defer loginTimer.Stop()

	// 4) 读循环
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			expected := c.expectedClose
			c.mu.Unlock()
			if expected {
				return true, nil
			}
			return false, err
		}
		_ = conn.SetReadDeadline(time.Now().Add(clientReadDeadline))

		fatal := c.handleFrame(payload)
		if fatal {
			return true, nil
		}

		// 首次就绪后补发排队动作
		c.mu.Lock()
		becameReady := c.ready
		c.mu.Unlock()
		if becameReady && len(pending) > 0 {
			for _, cmd := range pending {
				c.sendFrameOn(conn, cmd.Action, cmd.Payload)
			}
			pending = nil
		}
	}
}

// ensureBinding 获取（或申请）对方身份绑定。
func (c *Client) ensureBinding(ctx context.Context) (*Binding, error) {
	existing, err := c.cfg.Bindings.Get(ctx, c.cfg.IdentityID)
	if err != nil {
		return nil, fmt.Errorf("读取身份绑定失败: %w", err)
	}
	if existing != nil && strings.TrimSpace(existing.UpstreamUID) != "" {
		return existing, nil
	}

	identity, err := c.cfg.API.GenerateIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("申请晚风身份失败: %w", err)
	}

	deviceID := ""
	if existing != nil {
		deviceID = existing.DeviceID
	}
	if strings.TrimSpace(deviceID) == "" {
		deviceID = uuid.NewString()
	}

	binding := Binding{
		IdentityID:  c.cfg.IdentityID,
		UpstreamUID: identity.UID,
		DeviceID:    deviceID,
		Nickname:    c.cfg.Profile.Nickname,
		Gender:      c.cfg.Profile.Gender,
		Age:         c.cfg.Profile.Age,
	}
	if err := c.cfg.Bindings.Save(ctx, binding); err != nil {
		c.log.Warn("保存身份绑定失败", "error", err)
	}
	c.log.Info("已申请晚风身份", "upstreamUID", identity.UID)
	return &binding, nil
}

// handleFrame 处理一帧服务端消息；返回 fatal=true 表示不应重连。
func (c *Client) handleFrame(payload []byte) bool {
	proto := c.proto()

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.log.Warn("解析晚风帧失败", "error", err)
		return false
	}

	frameType := PickString(raw, proto.FrameTypeField)
	data, ok := c.parseFrameData(raw)
	if !ok {
		return false
	}

	switch frameType {
	case proto.EventConnected:
		c.onConnected(data)
	case proto.EventUserInfo:
		c.onUserInfo(data)
	case proto.EventMatchResult:
		c.onMatchResult(data)
	case proto.EventReceiveMessage:
		c.onReceiveMessage(data)
	case proto.EventMessageRecalled:
		if msg, ok := c.translate(proto.RecalledMessage(data)); ok {
			c.cfg.Emitter.Emit(c.cfg.IdentityID, msg)
		}
	case proto.EventPartnerLeft:
		c.mu.Lock()
		c.partnerUID = ""
		c.partnerNickname = ""
		c.state = proto.StateIdle
		c.mu.Unlock()
		c.cfg.Emitter.Emit(c.cfg.IdentityID, proto.PartnerLeftMessage())
	case proto.EventError:
		// 身份/设备类错误可自愈：清掉本地绑定后重连会自动申请新身份/新设备号，
		// 不必把用户当成「被封禁」踢回登录页。
		switch strings.ToUpper(PickString(data, proto.ErrorCodeKeys...)) {
		case "IDENTITY_INVALID":
			c.log.Warn("晚风身份失效，重新申请身份", "data", data)
			c.resetBinding(true, true)
			c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
				"code":    codeSystemNotice,
				"content": "正在重新建立身份，请稍候…",
			})
			if conn := c.currentConn(); conn != nil {
				_ = conn.Close()
			}
			return false
		case "DEVICE_DUPLICATE":
			c.log.Warn("晚风设备号重复，更换设备号后重连", "data", data)
			c.resetBinding(false, true)
			if conn := c.currentConn(); conn != nil {
				_ = conn.Close()
			}
			return false
		}

		messages, fatal := proto.ErrorMessages(data)
		for _, m := range messages {
			if fatal {
				c.cfg.Emitter.EmitFatal(c.cfg.IdentityID, m)
			} else {
				c.cfg.Emitter.Emit(c.cfg.IdentityID, m)
			}
		}
		if fatal {
			c.log.Warn("收到晚风致命错误", "data", data)
			return true
		}
	default:
		// 未知帧：降级忽略（对方新增事件时不影响主链路）
		c.log.Debug("忽略未处理的晚风帧", "type", frameType)
	}
	return false
}

// parseFrameData 解密（或直接读取）帧内 data。
func (c *Client) parseFrameData(raw map[string]any) (map[string]any, bool) {
	proto := c.proto()
	dataRaw, ok := raw[proto.FrameDataField]
	if !ok || dataRaw == nil {
		return map[string]any{}, true
	}

	switch v := dataRaw.(type) {
	case map[string]any:
		return v, true
	case string:
		c.mu.Lock()
		sess := c.session
		c.mu.Unlock()
		plain, err := c.cfg.API.OpenFrameData(sess, v)
		if err != nil {
			c.mu.Lock()
			c.decryptFailures++
			failures := c.decryptFailures
			c.mu.Unlock()
			c.log.Warn("解密晚风帧失败", "error", err, "failures", failures)
			if failures >= clientMaxDecryptFailures {
				// 会话可能已失效：作废并让主循环重连
				if sess != nil {
					c.sessionInvalidate(sess.ID)
				}
				if conn := c.currentConn(); conn != nil {
					_ = conn.Close()
				}
			}
			return map[string]any{}, false
		}
		if len(plain) == 0 {
			return map[string]any{}, true
		}
		var out map[string]any
		if err := json.Unmarshal(plain, &out); err != nil {
			c.log.Warn("解析晚风帧 data 失败", "error", err)
			return map[string]any{}, false
		}
		return out, true
	default:
		return map[string]any{}, true
	}
}

func (c *Client) onConnected(data map[string]any) {
	proto := c.proto()
	c.mu.Lock()
	c.ready = true
	if st := PickString(data, proto.StateKeys...); st != "" {
		c.state = st
	}
	// 保存对方下发的 identityToken（重连与身份恢复必须）
	var (
		token       string
		bindingCopy *Binding
	)
	if token = PickString(data, proto.IdentityTokenKeys...); token != "" && c.binding != nil && token != c.binding.IdentityToken {
		c.binding.IdentityToken = token
		snapshot := *c.binding
		bindingCopy = &snapshot
	}
	c.mu.Unlock()

	if bindingCopy != nil {
		if err := c.cfg.Bindings.Save(context.Background(), *bindingCopy); err != nil {
			c.log.Warn("保存 identityToken 失败", "error", err)
		}
		// 绑定 Cookie 失败不影响主流程（对方前端也允许失败）
		if err := c.cfg.API.BindIdentityCookie(context.Background(), token); err != nil {
			c.log.Debug("绑定身份 Cookie 失败", "error", err)
		}
	}

	c.log.Info("晚风登录成功", "state", c.currentState())
	for _, msg := range proto.ConnectedMessages(data) {
		c.cfg.Emitter.Emit(c.cfg.IdentityID, msg)
	}
}

func (c *Client) onUserInfo(data map[string]any) {
	proto := c.proto()
	c.mu.Lock()
	if st := PickString(data, proto.StateKeys...); st != "" {
		c.state = st
	}
	c.mu.Unlock()
}

func (c *Client) onMatchResult(data map[string]any) {
	proto := c.proto()
	result, ok := proto.ParseMatchResult(data)
	if !ok {
		c.log.Debug("忽略无有效对象的匹配帧", "data", data)
		return
	}

	c.mu.Lock()
	c.partnerUID = result.TargetUID
	c.partnerNickname = result.Nickname
	c.state = proto.StateInRoom
	c.mu.Unlock()

	c.cfg.Emitter.Emit(c.cfg.IdentityID, proto.MatchedMessage(result))
}

func (c *Client) onReceiveMessage(data map[string]any) {
	proto := c.proto()
	c.mu.Lock()
	ctx := DownstreamContext{
		SelfID:          c.cfg.IdentityID,
		SelfUpstreamUID: c.bindingUIDLocked(),
		SelfNickname:    c.cfg.Profile.Nickname,
		SelfSex:         SexFromUpstream(c.cfg.Profile.Gender),
		PartnerUID:      c.partnerUID,
		PartnerNickname: c.partnerNickname,
	}
	c.mu.Unlock()

	msg, ok := proto.ReceiveMessage(data, ctx)
	if !ok {
		c.log.Debug("忽略非文本消息", "data", data)
		return
	}
	c.cfg.Emitter.Emit(c.cfg.IdentityID, msg)
}

// resetBinding 清空本地绑定字段（uid/token 与 deviceID 可分别清除），下次重连会自动重建。
func (c *Client) resetBinding(clearIdentity, clearDevice bool) {
	c.mu.Lock()
	if c.binding == nil {
		c.mu.Unlock()
		return
	}
	if clearIdentity {
		c.binding.UpstreamUID = ""
		c.binding.IdentityToken = ""
	}
	if clearDevice {
		c.binding.DeviceID = ""
	}
	snapshot := *c.binding
	c.mu.Unlock()

	if err := c.cfg.Bindings.Save(context.Background(), snapshot); err != nil {
		c.log.Warn("重置身份绑定失败", "error", err)
	}
}

func (c *Client) bindingUIDLocked() string {
	if c.binding == nil {
		return ""
	}
	return c.binding.UpstreamUID
}

// heartbeatLoop 定时发送 ping 与心跳帧，保持连接与在线状态。
func (c *Client) heartbeatLoop(conn *websocket.Conn, stop chan struct{}) {
	interval := time.Duration(c.proto().PingSeconds) * time.Second
	if interval <= 0 {
		interval = 25 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(clientWriteDeadline))
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(clientWriteDeadline))
			c.writeMu.Unlock()
			if err != nil {
				return
			}
			// 业务心跳帧：对方若忽略也不影响
			if action := c.proto().ActionHeartbeat; action != "" {
				_ = c.sendFrameOn(conn, action, map[string]any{})
			}
		}
	}
}

// sendFrameOn 把动作加密后写入指定连接。
func (c *Client) sendFrameOn(conn *websocket.Conn, action string, payload map[string]any) error {
	if conn == nil || strings.TrimSpace(action) == "" {
		return nil
	}
	proto := c.proto()

	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	if sess == nil {
		return errors.New("会话未就绪")
	}

	if payload == nil {
		payload = map[string]any{}
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化动作失败: %w", err)
	}
	sealed, err := c.cfg.API.SealFrameData(sess, plain)
	if err != nil {
		return fmt.Errorf("加密动作失败: %w", err)
	}

	frame := map[string]any{
		proto.FrameTypeField:    action,
		proto.FrameDataField:    sealed,
		proto.FrameVersionField: proto.EnvelopeVersion,
		proto.FrameSIDField:     sess.ID,
	}
	raw, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("序列化帧失败: %w", err)
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(clientWriteDeadline))
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return err
	}
	return nil
}

func (c *Client) proto() Protocol { return c.cfg.API.Protocol() }

func (c *Client) currentState() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Client) currentConn() *websocket.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

func (c *Client) sessionInvalidate(id string) {
	c.cfg.API.sessions.invalidate(id)
}

func (c *Client) translate(msg map[string]any) (map[string]any, bool) {
	if len(msg) == 0 {
		return nil, false
	}
	return msg, true
}

// websocketHeader 构造 WS 握手所需的浏览器风格请求头（对方按同源校验 Origin/Referer）。
func websocketHeader(api *APIClient) http.Header {
	h := http.Header{}
	if api == nil {
		return h
	}
	if api.proto.UserAgent != "" {
		h.Set("User-Agent", api.proto.UserAgent)
	}
	if api.origin != "" {
		h.Set("Origin", api.origin)
	}
	if api.referer != "" {
		h.Set("Referer", api.referer)
	}
	h.Set("Accept-Language", "zh-CN,zh;q=0.9")
	return h
}
