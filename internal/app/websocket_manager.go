package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsCloseDelaySeconds       = 80
	wsMaxConcurrentIdentities = 2
	wsUpstreamConnectionLost  = 700 * time.Second
	wsUpstreamWriteDeadline   = 10 * time.Second
	wsDownstreamWriteDeadline = 5 * time.Second
	wsUpstreamConnectTimeout  = 15 * time.Second
	wsUpstreamCloseWait       = 5 * time.Second
	wsEvictionCloseDelay      = 1 * time.Second
	wsForceoutDownstreamDelay = 1 * time.Second
)

var (
	wsCloseDelay    = wsCloseDelaySeconds * time.Second
	wsEvictionDelay = wsEvictionCloseDelay
	wsForceoutDelay = wsForceoutDownstreamDelay

	wsUpstreamPingInterval     = 600 * time.Second
	wsWebServiceRandServerBase = "http://v1.chat2019.cn/Act/WebService.asmx/getRandServer?ServerInfo=serversdeskry&_="
)

type UpstreamWebSocketManager struct {
	httpClient *http.Client
	fallbackWS string

	forceout *ForceoutManager
	cache    UserInfoCacheService
	history  ChatHistoryCacheService

	mu                    sync.Mutex
	upstreamClients       map[string]*UpstreamWebSocketClient
	downstreamSessions    map[string]map[*DownstreamSession]struct{}
	pendingCloseTasks     map[string]*time.Timer
	connectionCreateMilli map[string]int64
}

func NewUpstreamWebSocketManager(httpClient *http.Client, fallbackWS string, forceout *ForceoutManager, cache UserInfoCacheService, history ChatHistoryCacheService) *UpstreamWebSocketManager {
	if strings.TrimSpace(fallbackWS) == "" {
		fallbackWS = "ws://localhost:9999"
	}
	return &UpstreamWebSocketManager{
		httpClient:            httpClient,
		fallbackWS:            fallbackWS,
		forceout:              forceout,
		cache:                 cache,
		history:               history,
		upstreamClients:       make(map[string]*UpstreamWebSocketClient),
		downstreamSessions:    make(map[string]map[*DownstreamSession]struct{}),
		pendingCloseTasks:     make(map[string]*time.Timer),
		connectionCreateMilli: make(map[string]int64),
	}
}

func (m *UpstreamWebSocketManager) RegisterDownstream(userID string, session *DownstreamSession, signMessage string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || session == nil {
		return
	}

	if m.forceout != nil && m.forceout.IsForbidden(userID) {
		remaining := int64(0)
		if m.forceout != nil {
			remaining = m.forceout.RemainingSeconds(userID)
		}
		reject := fmt.Sprintf("{\"code\":-4,\"content\":\"由于重复登录，您的连接被暂时禁止，请%d秒后再试\",\"forceout\":true}", remaining)
		_ = session.SendText(reject)
		_ = session.Close()
		return
	}

	var shouldCreate bool
	var evictUserID string

	m.mu.Lock()
	if t := m.pendingCloseTasks[userID]; t != nil {
		t.Stop()
		delete(m.pendingCloseTasks, userID)
	}

	sessions := m.downstreamSessions[userID]
	if sessions == nil {
		sessions = make(map[*DownstreamSession]struct{})
		m.downstreamSessions[userID] = sessions
	}
	sessions[session] = struct{}{}

	if _, ok := m.upstreamClients[userID]; !ok {
		shouldCreate = true

		if len(m.upstreamClients) >= wsMaxConcurrentIdentities {
			oldestUserID := ""
			oldestTs := int64(0)
			for uid, ts := range m.connectionCreateMilli {
				if uid == userID {
					continue
				}
				if oldestUserID == "" || ts < oldestTs {
					oldestUserID = uid
					oldestTs = ts
				}
			}
			if oldestUserID != "" {
				evictUserID = oldestUserID
			}
		}
	}
	m.mu.Unlock()

	if evictUserID != "" && evictUserID != userID {
		evictMessage := "{\"code\":-6,\"content\":\"由于新身份连接，您已被自动断开\",\"evicted\":true}"
		m.BroadcastToDownstream(evictUserID, evictMessage)
		time.AfterFunc(wsEvictionDelay, func() {
			m.CloseUpstreamConnection(evictUserID)

			sessions := m.snapshotDownstream(evictUserID)
			for _, s := range sessions {
				_ = s.Close()
			}

			m.mu.Lock()
			delete(m.downstreamSessions, evictUserID)
			if t := m.pendingCloseTasks[evictUserID]; t != nil {
				t.Stop()
				delete(m.pendingCloseTasks, evictUserID)
			}
			m.mu.Unlock()
		})
	}

	if shouldCreate {
		m.createUpstreamConnection(userID, signMessage)
	}
}

func (m *UpstreamWebSocketManager) UnregisterDownstream(userID string, session *DownstreamSession) {
	userID = strings.TrimSpace(userID)
	if userID == "" || session == nil {
		return
	}

	m.mu.Lock()
	sessions := m.downstreamSessions[userID]
	if sessions != nil {
		delete(sessions, session)
		if len(sessions) == 0 {
			delete(m.downstreamSessions, userID)
			m.scheduleCloseUpstreamLocked(userID)
		}
	}
	m.mu.Unlock()
}

func (m *UpstreamWebSocketManager) SendToUpstream(userID string, message string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	m.mu.Lock()
	client := m.upstreamClients[userID]
	_, hasDownstream := m.downstreamSessions[userID]
	m.mu.Unlock()

	if client == nil || !client.IsOpen() {
		if !hasDownstream {
			return
		}
		client = m.createUpstreamConnection(userID, "")
	}
	if client != nil {
		client.SendMessage(message)
	}
}

func (m *UpstreamWebSocketManager) BroadcastToDownstream(userID string, message string) {
	sessions := m.snapshotDownstream(userID)
	for _, session := range sessions {
		if session == nil {
			continue
		}
		if err := session.SendText(message); err != nil {
			_ = session.Close()
			m.UnregisterDownstream(userID, session)
		}
	}
}

func (m *UpstreamWebSocketManager) HandleForceout(userID string, message string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	if m.forceout != nil {
		m.forceout.AddForceoutUser(userID)
	}

	m.BroadcastToDownstream(userID, message)
	m.CloseUpstreamConnection(userID)

	time.AfterFunc(wsForceoutDelay, func() {
		sessions := m.snapshotDownstream(userID)
		for _, s := range sessions {
			_ = s.Close()
		}
		m.mu.Lock()
		delete(m.downstreamSessions, userID)
		m.mu.Unlock()
	})
}

func (m *UpstreamWebSocketManager) HandleUpstreamDisconnect(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	m.mu.Lock()
	delete(m.upstreamClients, userID)
	delete(m.connectionCreateMilli, userID)
	sessions := m.snapshotDownstreamLocked(userID)
	delete(m.downstreamSessions, userID)
	m.mu.Unlock()

	if len(sessions) == 0 {
		return
	}
	for _, s := range sessions {
		_ = s.Close()
	}
}

func (m *UpstreamWebSocketManager) CloseAllConnections() {
	m.mu.Lock()
	upstream := make([]*UpstreamWebSocketClient, 0, len(m.upstreamClients))
	for _, c := range m.upstreamClients {
		upstream = append(upstream, c)
	}
	downstream := make([]*DownstreamSession, 0)
	for _, sessions := range m.downstreamSessions {
		for s := range sessions {
			downstream = append(downstream, s)
		}
	}
	for _, t := range m.pendingCloseTasks {
		t.Stop()
	}
	m.upstreamClients = make(map[string]*UpstreamWebSocketClient)
	m.downstreamSessions = make(map[string]map[*DownstreamSession]struct{})
	m.pendingCloseTasks = make(map[string]*time.Timer)
	m.connectionCreateMilli = make(map[string]int64)
	m.mu.Unlock()

	for _, c := range upstream {
		c.CloseExpected()
	}
	for _, s := range downstream {
		if s == nil {
			continue
		}
		_ = s.Close()
	}
}

func (m *UpstreamWebSocketManager) GetConnectionStats() map[string]any {
	m.mu.Lock()
	upstreamCount := len(m.upstreamClients)
	downstreamCount := 0
	for _, sessions := range m.downstreamSessions {
		downstreamCount += len(sessions)
	}
	m.mu.Unlock()

	return map[string]any{
		"active":         upstreamCount + downstreamCount,
		"upstream":       upstreamCount,
		"downstream":     downstreamCount,
		"maxIdentities":  wsMaxConcurrentIdentities,
		"availableSlots": wsMaxConcurrentIdentities - upstreamCount,
	}
}

func (m *UpstreamWebSocketManager) scheduleCloseUpstreamLocked(userID string) {
	if t := m.pendingCloseTasks[userID]; t != nil {
		t.Stop()
	}
	m.pendingCloseTasks[userID] = time.AfterFunc(wsCloseDelay, func() {
		m.mu.Lock()
		_, hasSessions := m.downstreamSessions[userID]
		if hasSessions {
			delete(m.pendingCloseTasks, userID)
			m.mu.Unlock()
			return
		}
		delete(m.pendingCloseTasks, userID)
		m.mu.Unlock()
		m.CloseUpstreamConnection(userID)
	})
}

func (m *UpstreamWebSocketManager) CloseUpstreamConnection(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	m.mu.Lock()
	client := m.upstreamClients[userID]
	delete(m.upstreamClients, userID)
	delete(m.connectionCreateMilli, userID)
	m.mu.Unlock()

	if client != nil {
		client.CloseExpected()
	}
}

func (m *UpstreamWebSocketManager) snapshotDownstream(userID string) []*DownstreamSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotDownstreamLocked(userID)
}

func (m *UpstreamWebSocketManager) snapshotDownstreamLocked(userID string) []*DownstreamSession {
	sessions := m.downstreamSessions[userID]
	if sessions == nil {
		return nil
	}
	out := make([]*DownstreamSession, 0, len(sessions))
	for s := range sessions {
		out = append(out, s)
	}
	return out
}

func (m *UpstreamWebSocketManager) createUpstreamConnection(userID string, signMessage string) *UpstreamWebSocketClient {
	m.mu.Lock()
	existing := m.upstreamClients[userID]
	m.mu.Unlock()

	if existing != nil {
		return existing
	}

	upstreamURL := m.getUpstreamWebSocketURL(context.Background())
	client := NewUpstreamWebSocketClient(userID, upstreamURL, m)
	if strings.TrimSpace(signMessage) != "" {
		client.SendMessage(signMessage)
	}

	m.mu.Lock()
	if existing := m.upstreamClients[userID]; existing != nil {
		m.mu.Unlock()
		client.CloseExpected()
		return existing
	}
	m.upstreamClients[userID] = client
	m.connectionCreateMilli[userID] = time.Now().UnixMilli()
	m.mu.Unlock()

	client.ConnectAsync()
	return client
}

func (m *UpstreamWebSocketManager) getUpstreamWebSocketURL(ctx context.Context) string {
	if m.httpClient == nil {
		return m.fallbackWS
	}
	reqURL := wsWebServiceRandServerBase + strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return m.fallbackWS
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return m.fallbackWS
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return m.fallbackWS
	}

	var node struct {
		State string `json:"state"`
		Msg   struct {
			Server string `json:"server"`
		} `json:"msg"`
	}
	if err := json.Unmarshal(body, &node); err != nil {
		return m.fallbackWS
	}
	if strings.TrimSpace(node.State) != "OK" {
		return m.fallbackWS
	}
	server := strings.TrimSpace(node.Msg.Server)
	if server == "" {
		return m.fallbackWS
	}
	return server
}

type UpstreamWebSocketClient struct {
	userID  string
	wsURL   string
	manager *UpstreamWebSocketManager

	mu            sync.Mutex
	conn          *websocket.Conn
	connected     bool
	pending       []string
	expectedClose atomic.Bool

	writeMu   sync.Mutex
	closeOnce sync.Once
	done      chan struct{}
}

func NewUpstreamWebSocketClient(userID string, wsURL string, manager *UpstreamWebSocketManager) *UpstreamWebSocketClient {
	return &UpstreamWebSocketClient{
		userID:  userID,
		wsURL:   wsURL,
		manager: manager,
		done:    make(chan struct{}),
	}
}

func (c *UpstreamWebSocketClient) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected && c.conn != nil
}

func (c *UpstreamWebSocketClient) ConnectAsync() {
	go c.connect()
}

func (c *UpstreamWebSocketClient) connect() {
	ctx, cancel := context.WithTimeout(context.Background(), wsUpstreamConnectTimeout)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		if c.manager != nil {
			c.manager.HandleUpstreamDisconnect(c.userID)
		}
		return
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	_ = conn.SetReadDeadline(time.Now().Add(wsUpstreamConnectionLost))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsUpstreamConnectionLost))
	})

	go c.heartbeat()
	c.flushPending()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		c.onMessage(string(data))
	}

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	if c.expectedClose.Load() {
		return
	}
	if c.manager != nil {
		c.manager.HandleUpstreamDisconnect(c.userID)
	}
}

func (c *UpstreamWebSocketClient) heartbeat() {
	ticker := time.NewTicker(wsUpstreamPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()
			if conn == nil {
				continue
			}
			c.writeMu.Lock()
			_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsUpstreamWriteDeadline))
			c.writeMu.Unlock()
		case <-c.done:
			return
		}
	}
}

func (c *UpstreamWebSocketClient) SendMessage(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}

	c.mu.Lock()
	conn := c.conn
	connected := c.connected
	if !connected || conn == nil {
		c.pending = append(c.pending, message)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	c.writeMu.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(wsUpstreamWriteDeadline))
	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	c.writeMu.Unlock()
	if err != nil {
		c.CloseUnexpected()
	}
}

func (c *UpstreamWebSocketClient) flushPending() {
	c.mu.Lock()
	conn := c.conn
	queue := append([]string(nil), c.pending...)
	c.pending = nil
	c.mu.Unlock()
	if conn == nil {
		return
	}

	for _, msg := range queue {
		c.writeMu.Lock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsUpstreamWriteDeadline))
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		c.writeMu.Unlock()
		if err != nil {
			c.CloseUnexpected()
			return
		}
	}
}

func (c *UpstreamWebSocketClient) onMessage(message string) {
	var node map[string]any
	if err := json.Unmarshal([]byte(message), &node); err == nil {
		code := toInt(node["code"])
		if code == -3 && toBool(node["forceout"]) {
			if c.manager != nil {
				c.manager.HandleForceout(c.userID, message)
			}
			return
		}

		if code == 15 {
			target := strings.TrimSpace(toString(node["sel_userid"]))
			if target != "" && c.manager != nil && c.manager.cache != nil {
				c.manager.cache.SaveUserInfo(CachedUserInfo{
					UserID:   target,
					Nickname: toString(node["sel_userNikename"]),
					Gender:   toString(node["sel_userSex"]),
					Age:      toString(node["sel_userAge"]),
					Address:  toString(node["sel_userAddress"]),
				})
			}
		}

		if code == 7 {
			fromUser := mapGetMap(node, "fromuser")
			toUser := mapGetMap(node, "touser")
			fromUserID := strings.TrimSpace(toString(fromUser["id"]))
			toUserID := strings.TrimSpace(toString(toUser["id"]))
			content := strings.TrimSpace(toString(firstNonNil(fromUser["content"], node["content"])))
			tm := strings.TrimSpace(toString(firstNonNil(fromUser["time"], node["time"])))
			msgType := strings.TrimSpace(toString(firstNonNil(fromUser["type"], node["type"])))
			tid := strings.TrimSpace(toString(firstNonNil(fromUser["Tid"], fromUser["tid"], node["Tid"], node["tid"])))
			if msgType == "" {
				msgType = "text"
			}

			if fromUserID != "" && toUserID != "" && content != "" && tm != "" && c.manager != nil && c.manager.cache != nil {
				last := CachedLastMessage{
					FromUserID: fromUserID,
					ToUserID:   toUserID,
					Content:    content,
					Type:       msgType,
					Time:       tm,
				}
				c.manager.cache.SaveLastMessage(last)

				// 兼容：上游返回的 toUserId 有时不是本地身份 userId，补写会话 key，保证列表可命中。
				if c.userID != fromUserID && c.userID != toUserID {
					c.manager.cache.SaveLastMessage(CachedLastMessage{
						FromUserID: fromUserID,
						ToUserID:   c.userID,
						Content:    content,
						Type:       msgType,
						Time:       tm,
					})
					c.manager.cache.SaveLastMessage(CachedLastMessage{
						FromUserID: c.userID,
						ToUserID:   toUserID,
						Content:    content,
						Type:       msgType,
						Time:       tm,
					})
				}
			}

			if tid != "" && content != "" && tm != "" && c.manager != nil && c.manager.history != nil {
				normalizedFrom := fromUserID
				normalizedTo := toUserID
				localMD5 := md5HexLower(c.userID)
				if localMD5 != "" {
					if strings.EqualFold(normalizedFrom, localMD5) {
						normalizedFrom = c.userID
					}
					if strings.EqualFold(normalizedTo, localMD5) {
						normalizedTo = c.userID
					}
				}

				historyMsg := map[string]any{
					"Tid":     tid,
					"id":      normalizedFrom,
					"toid":    normalizedTo,
					"content": content,
					"time":    tm,
				}
				if nickname := strings.TrimSpace(toString(firstNonNil(fromUser["nickname"], fromUser["name"], node["nickname"], node["name"]))); nickname != "" {
					historyMsg["nickname"] = nickname
				}

				conversationKeys := make(map[string]struct{}, 3)
				if key := generateConversationKey(normalizedFrom, normalizedTo); key != "" {
					conversationKeys[key] = struct{}{}
				}

				// 兼容：上游返回的 toUserId 有时不是本地身份 userId，补写会话 key，保证历史查询可命中。
				if strings.TrimSpace(c.userID) != "" && c.userID != normalizedFrom && c.userID != normalizedTo {
					if key := generateConversationKey(normalizedFrom, c.userID); key != "" {
						conversationKeys[key] = struct{}{}
					}
					if key := generateConversationKey(c.userID, normalizedTo); key != "" {
						conversationKeys[key] = struct{}{}
					}
				}

				for key := range conversationKeys {
					c.manager.history.SaveMessages(context.Background(), key, []map[string]any{historyMsg})
				}
			}
		}
	}

	if c.manager != nil {
		c.manager.BroadcastToDownstream(c.userID, message)
	}
}

func (c *UpstreamWebSocketClient) CloseExpected() {
	c.expectedClose.Store(true)
	c.Close()
}

func (c *UpstreamWebSocketClient) CloseUnexpected() {
	c.expectedClose.Store(false)
	c.Close()
}

func (c *UpstreamWebSocketClient) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.mu.Lock()
		conn := c.conn
		c.conn = nil
		c.connected = false
		c.mu.Unlock()

		if conn == nil {
			return
		}
		c.writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(wsUpstreamCloseWait))
		_ = conn.Close()
		c.writeMu.Unlock()
	})
}

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	}
	return 0
}

func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		normalized := strings.TrimSpace(strings.ToLower(t))
		return normalized == "true" || normalized == "1"
	case float64:
		return t != 0
	}
	return false
}

func mapGetMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	v, ok := m[key]
	if !ok || v == nil {
		return map[string]any{}
	}
	if mm, ok := v.(map[string]any); ok {
		return mm
	}
	return map[string]any{}
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func md5HexLower(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	sum := md5.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}
