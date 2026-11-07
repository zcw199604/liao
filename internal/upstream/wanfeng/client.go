package wanfeng

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// outbox 容量（未连接时的待发动作）
	clientOutboxSize = 32
	// 单条媒体搬运的最长等待时间
	clientMediaTimeout = 60 * time.Second
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
	// Media 是媒体搬运桥（二期图片/视频）。为 nil 时媒体消息会降级为提示文案。
	Media  MediaBridge
	Logger *slog.Logger
	Dialer *websocket.Dialer
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
	partnerUID      string
	partnerNickname string
	state           string
	decryptFailures int
	egressID        string

	// outbox 是「待下发给对方」的动作队列（保序，媒体搬运在队列内完成）
	outbox chan OutgoingCommand

	// outboundMedia 记录「对方媒体 fileId → chat2019 远程文件名」，
	// 用于自己消息回显时复用同一路径（前端按媒体路径合并乐观消息）。
	mediaCfgMu    sync.RWMutex
	mediaMu       sync.Mutex
	outboundMedia map[string]string

	writeMu   sync.Mutex
	startOnce sync.Once
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
		// HTTP 和 WS 必须使用同一出口；Clone 的身份客户端也保留此 Transport。
		if cfg.API != nil && cfg.API.http != nil {
			if transport, ok := cfg.API.http.Transport.(*http.Transport); ok {
				copy := *websocket.DefaultDialer
				copy.Proxy = transport.Proxy
				copy.NetDialContext = transport.DialContext
				copy.NetDialTLSContext = transport.DialTLSContext
				if transport.TLSClientConfig != nil {
					copy.TLSClientConfig = transport.TLSClientConfig.Clone()
				}
				dialer = &copy
			}
		}
	}
	return &Client{
		cfg:           cfg,
		log:           logger.With("mod", "wanfeng-client", "identityID", cfg.IdentityID),
		dialer:        dialer,
		done:          make(chan struct{}),
		outbox:        make(chan OutgoingCommand, clientOutboxSize),
		outboundMedia: make(map[string]string),
	}
}

// SetMediaBridge 运行期替换媒体搬运桥（管理器在装配后才创建时使用）。
func (c *Client) SetMediaBridge(bridge MediaBridge) {
	c.mediaCfgMu.Lock()
	c.cfg.Media = bridge
	c.mediaCfgMu.Unlock()
}

// MediaBridge 返回当前的媒体搬运桥（可能为 nil）。
func (c *Client) MediaBridge() MediaBridge { return c.mediaBridge() }

// mediaBridge 安全读取当前的媒体搬运桥（可能被运行期替换）。
func (c *Client) mediaBridge() MediaBridge {
	c.mediaCfgMu.RLock()
	defer c.mediaCfgMu.RUnlock()
	return c.cfg.Media
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
	c.startOnce.Do(func() {
		go c.run(ctx)
	})
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

// SendCommand 把动作放进下发队列（保序；未就绪时在队列里等待登录完成）。
func (c *Client) SendCommand(cmd OutgoingCommand) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return
	}
	select {
	case c.outbox <- cmd:
	default:
		// 队列满时丢弃最旧的一条，避免阻塞前端
		select {
		case <-c.outbox:
		default:
		}
		select {
		case c.outbox <- cmd:
		default:
			c.log.Warn("下发队列已满，丢弃动作", "action", cmd.Action)
		}
	}
}

// dispatchLoop 串行消费下发队列：保证消息顺序，并在媒体动作里完成「上传/搬运」。
func (c *Client) dispatchLoop(ctx context.Context) {
	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		case cmd := <-c.outbox:
			if !c.waitReady(ctx) {
				// 区分「客户端已关闭」与「本等待超时」：后者只丢本条，继续服务后续消息
				select {
				case <-c.done:
					return
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			if err := c.deliver(ctx, cmd); err != nil {
				c.log.Warn("下发动作失败", "action", cmd.Action, "error", err)
			}
		}
	}
}

// waitReady 等待连接登录就绪（最多 30 秒）。
func (c *Client) waitReady(ctx context.Context) bool {
	deadline := time.Now().Add(30 * time.Second)
	for {
		select {
		case <-c.done:
			return false
		case <-ctx.Done():
			return false
		default:
		}
		if c.IsReady() {
			return true
		}
		if time.Now().After(deadline) {
			c.log.Warn("等待登录就绪超时，丢弃下发动作")
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// deliver 执行一条下发动作；媒体消息先搬运再发送。
func (c *Client) deliver(ctx context.Context, cmd OutgoingCommand) error {
	payload, err := c.resolvePayload(ctx, cmd)
	if err != nil {
		return err
	}

	conn := c.currentConn()
	if conn == nil {
		return errors.New("连接已断开")
	}
	return c.sendFrameOn(conn, cmd.Action, payload)
}

// resolvePayload 计算真正要发送的载荷：媒体消息会先把内容搬运到对方站点。
func (c *Client) resolvePayload(ctx context.Context, cmd OutgoingCommand) (map[string]any, error) {
	payload := cmd.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	if cmd.Media == nil {
		return payload, nil
	}

	mirrored, err := c.mirrorOutboundMedia(ctx, cmd)
	if err != nil {
		// 搬运失败：给前端一个可读提示，避免消息静默丢失
		c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
			"code":    codeSystemNotice,
			"content": "图片/视频发送失败：" + err.Error(),
		})
		return nil, err
	}
	return mirrored, nil
}

// mirrorOutboundMedia 把 chat2019 媒体文件名搬到对方站点，返回发送用载荷。
func (c *Client) mirrorOutboundMedia(ctx context.Context, cmd OutgoingCommand) (map[string]any, error) {
	media := cmd.Media
	if strings.TrimSpace(media.Path) == "" {
		return nil, errors.New("媒体路径为空")
	}

	targetUID := PickString(cmd.Payload, "targetUid")
	if targetUID == "" {
		return nil, errors.New("缺少目标用户")
	}

	bridge := c.mediaBridge()
	if bridge == nil {
		return nil, errors.New("当前线路未启用媒体转发")
	}

	// 搬运有独立超时（接口无 ctx）：避免对方慢上传把整个下发队列卡死。
	mirrorCtx, cancel := context.WithTimeout(ctx, clientMediaTimeout)
	defer cancel()

	started := time.Now()
	resultCh := make(chan mediaMirrorResult, 1)
	go func() {
		url, err := bridge.MirrorOutbound(c.cfg.IdentityID, media.Kind, media.Path)
		resultCh <- mediaMirrorResult{url: url, err: err}
	}()

	var mediaURL string
	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		mediaURL = res.url
	case <-mirrorCtx.Done():
		return nil, fmt.Errorf("媒体搬运超时（%s）", clientMediaTimeout)
	}

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		c.log.Warn("媒体搬运较慢，已阻塞后续消息", "kind", media.Kind, "elapsed", elapsed)
	}

	// 记录映射，供自己消息回显时复用同一聊天图片路径
	c.rememberOutboundMedia(mediaURL, media.Path)

	return c.proto().SendMediaPayload(targetUID, media.Kind, mediaURL), nil
}

// rememberOutboundMedia 记录对方媒体 URL → chat2019 文件名 的映射（有界）。
func (c *Client) rememberOutboundMedia(mediaURL, remoteFilename string) {
	key := MediaURLKey(mediaURL)
	if key == "" || strings.TrimSpace(remoteFilename) == "" {
		return
	}
	c.mediaMu.Lock()
	if len(c.outboundMedia) >= 256 {
		for k := range c.outboundMedia {
			delete(c.outboundMedia, k)
			break
		}
	}
	c.outboundMedia[key] = remoteFilename
	c.mediaMu.Unlock()
}

// lookupOutboundMedia 查询自己消息回显对应的 chat2019 文件名。
func (c *Client) lookupOutboundMedia(mediaURL string) string {
	key := MediaURLKey(mediaURL)
	if key == "" {
		return ""
	}
	c.mediaMu.Lock()
	defer c.mediaMu.Unlock()
	return c.outboundMedia[key]
}

// mediaMirrorResult 是媒体搬运的异步结果（便于加超时）。
type mediaMirrorResult struct {
	url string
	err error
}

// run 是连接主循环：断线自动重连，致命错误退出。
func (c *Client) run(ctx context.Context) {
	go c.dispatchLoop(ctx)
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
		select {
		case <-c.done:
			// 主动关闭（切换线路/进程退出）：不算异常，也不打致命日志
			return
		default:
		}
		if fatal {
			c.log.Warn("晚风连接致命错误，停止重连", "error", err)
			return
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
		return !errors.Is(err, ErrIPBanSwitched), err
	}

	sess, err := c.cfg.API.Session(ctx)
	if err != nil {
		var apiErr *APIError
		fatal := errors.As(err, &apiErr) && apiErr.Code == "IP_BANNED" && !errors.Is(err, ErrIPBanSwitched)
		return fatal, err
	}

	url := c.cfg.API.BuildWSURL()
	header := websocketHeader(c.cfg.API)
	dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	egress := c.cfg.API.egressID()
	conn, resp, err := c.dialer.DialContext(dialCtx, url, header)
	if err != nil {
		if resp != nil && resp.Body != nil {
			raw, readErr := io.ReadAll(io.LimitReader(resp.Body, apiBodyLimit))
			_ = resp.Body.Close()
			if readErr == nil {
				if banErr := c.cfg.API.checkIPBan(raw, egress); banErr != nil {
					return !errors.Is(banErr, ErrIPBanSwitched), banErr
				}
			}
		}
		return false, fmt.Errorf("连接晚风 WS 失败: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.session = sess
	c.binding = binding
	c.ready = false
	c.decryptFailures = 0
	c.egressID = egress
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
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) && isIPBan("", closeErr.Text) {
				if c.cfg.API.switchBannedEgress(egress) == nil {
					return false, ErrIPBanSwitched
				}
				return true, err
			}
			return false, err
		}
		_ = conn.SetReadDeadline(time.Now().Add(clientReadDeadline))

		fatal := c.handleFrame(payload)
		if fatal {
			return true, nil
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
		c.cfg.API.SetUploadUserID(existing.UpstreamUID)
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
		// 媒体消息要做下载+转存，放到独立 goroutine，避免阻塞读循环（心跳/其它事件）
		if _, isMedia := proto.ParseReceiveMedia(data); isMedia {
			go c.onReceiveMessage(data)
		} else {
			c.onReceiveMessage(data)
		}
	case proto.EventMessageRecalled:
		if msg, ok := c.translate(proto.RecalledMessage(data)); ok {
			c.cfg.Emitter.Emit(c.cfg.IdentityID, msg)
		}
	case proto.EventMessagesRead:
		// 对方已读回执：前端暂无已读 UI，这里只落日志与本地状态，不打扰用户
		c.markPartnerRead(data)
	case proto.EventReportUser:
		// 举报结果回执：失败才提示，成功由前端本地提示（避免重复弹窗）
		if ok, has := PickBool(data, "success"); has && !ok {
			c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
				"code":    codeSystemNotice,
				"content": "举报提交失败，请稍后重试",
			})
		}
	case proto.EventPartnerLeft:
		c.mu.Lock()
		c.partnerUID = ""
		c.partnerNickname = ""
		c.state = proto.StateIdle
		c.mu.Unlock()
		c.cfg.Emitter.Emit(c.cfg.IdentityID, proto.PartnerLeftMessage())
	case proto.EventError:
		if isIPBan(PickString(data, proto.ErrorCodeKeys...), PickString(data, "message", "clientMsg", "msg")) {
			c.mu.Lock()
			egress := c.egressID
			c.mu.Unlock()
			if c.cfg.API.switchBannedEgress(egress) == nil {
				c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
					"code": codeSystemNotice, "content": "当前出口 IP 被封禁，已切换代理，正在重新连接…",
				})
				if conn := c.currentConn(); conn != nil {
					_ = conn.Close()
				}
				return false
			}
			c.cfg.Emitter.EmitFatal(c.cfg.IdentityID, map[string]any{
				"code": codeForbidden, "forceout": true,
				"content": "当前出口 IP 已被封禁，代理切换失败，请检查备用节点和池文件写入权限",
			})
			return true
		}
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
	if token == "" && c.binding != nil {
		// 少数情况下 connected 不下发 token，沿用已保存的 token 绑定 Cookie
		token = c.binding.IdentityToken
	}
	c.mu.Unlock()

	if bindingCopy != nil {
		if err := c.cfg.Bindings.Save(context.Background(), *bindingCopy); err != nil {
			c.log.Warn("保存 identityToken 失败", "error", err)
		}
	}
	if token != "" {
		// 每次都绑定：Cookie Jar 是进程内的，重启后必须重新换取身份 Cookie，
		// 否则媒体上传等按 Cookie 鉴权的接口会失败。
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

	// 媒体消息（图片/视频/语音）需要先搬运到我方媒体服务器，前端才能渲染
	if info, ok := proto.ParseReceiveMedia(data); ok {
		c.handleReceiveMedia(info, ctx, data)
		return
	}

	msg, ok := proto.ReceiveMessage(data, ctx)
	if !ok {
		c.log.Debug("忽略非文本消息", "data", data)
		return
	}
	c.cfg.Emitter.Emit(c.cfg.IdentityID, msg)

	// 用户正在与对方聊天：顺手把该会话在对方侧标为已读，
	// 这样对方能看到已读回执（我方前端不显示，纯属对外体验对齐）。
	c.sendAutoRead(data, ctx)
}

// sendAutoRead 收到对端消息后，向对方回一个 read_messages。
//
// 只在「对方发来的消息」时触发；自己发出的消息会被对方侧标已读，不需要我们回。
func (c *Client) sendAutoRead(data map[string]any, ctx DownstreamContext) {
	if strings.EqualFold(PickString(data, "fromCid"), "self") {
		return
	}
	peer := ctx.PartnerUID
	if peer == "" {
		peer = PickString(data, c.proto().FromUIDKeys...)
	}
	if peer == "" || strings.EqualFold(peer, ctx.SelfUpstreamUID) {
		return
	}
	proto := c.proto()
	// 非阻塞：队列满时丢弃（已读只是锦上添花，不能拖慢消息下发）
	c.SendCommand(OutgoingCommand{
		Action:  proto.ActionReadMessages,
		Payload: map[string]any{"targetUid": peer},
	})
}

// markPartnerRead 处理对方推来的已读回执（messages_read）。
//
// 前端目前没有已读 UI，这里只做日志与状态记录，方便后续接入而不必再改协议层。
func (c *Client) markPartnerRead(data map[string]any) {
	proto := c.proto()
	peer := PickString(data, proto.TargetUIDKeys...)
	if peer == "" {
		peer = PickString(data, proto.FromUIDKeys...)
	}
	tids := PickStringSlice(data, "tids", "tidsRead", "readTids")
	c.log.Debug("收到对方已读回执", "peer", peer, "count", len(tids))
}

// handleReceiveMedia 把对方媒体搬到 chat2019 媒体服务器，再下发 code=7（前端零改动）。
func (c *Client) handleReceiveMedia(info ReceiveMediaInfo, ctx DownstreamContext, data map[string]any) {
	fromCid := PickString(data, "fromCid")
	fromUID := PickString(data, c.proto().FromUIDKeys...)
	isSelf := strings.EqualFold(fromCid, "self") ||
		(ctx.SelfUpstreamUID != "" && strings.EqualFold(fromUID, ctx.SelfUpstreamUID))

	// 自己发出的媒体：优先复用发送时记录的路径，保证前端能合并到乐观消息
	if isSelf {
		if existing := c.lookupOutboundMedia(info.URL); existing != "" {
			c.cfg.Emitter.Emit(c.cfg.IdentityID, c.proto().MediaMessage(info, existing, ctx, true))
			return
		}
	}

	bridge := c.mediaBridge()
	if bridge == nil {
		c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
			"code":    codeSystemNotice,
			"content": "收到一条图片/视频消息，当前线路未启用媒体转发",
		})
		return
	}

	remoteFilename, err := bridge.MirrorInbound(c.cfg.IdentityID, info.Kind, info.URL)
	if err != nil {
		c.log.Warn("媒体接收失败", "kind", info.Kind, "error", err)
		// 自己发的失败就不打扰用户（本地已有预览），对方发的则提示
		if !isSelf {
			c.cfg.Emitter.Emit(c.cfg.IdentityID, map[string]any{
				"code":    codeSystemNotice,
				"content": "对方发送的图片/视频加载失败",
			})
		}
		return
	}

	if isSelf {
		c.rememberOutboundMedia(info.URL, remoteFilename)
	}
	c.cfg.Emitter.Emit(c.cfg.IdentityID, c.proto().MediaMessage(info, remoteFilename, ctx, isSelf))
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
