package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisUserInfoCacheService 对齐 Java 的 RedisUserInfoCacheService（L1 本地缓存 + L2 Redis）。
type RedisUserInfoCacheService struct {
	client  *redis.Client
	timeout time.Duration

	keyPrefix         string
	lastMessagePrefix string
	expire            time.Duration

	local *lruCache

	flushInterval time.Duration
	pendingMu     sync.Mutex
	pending       map[string][]byte
	stopCh        chan struct{}
	wg            sync.WaitGroup
	closeOnce     sync.Once
}

func NewRedisUserInfoCacheService(
	redisURL string,
	host string,
	port int,
	password string,
	db int,
	keyPrefix string,
	lastMessagePrefix string,
	expireDays int,
	flushIntervalSeconds int,
	localTTLSeconds int,
	timeoutSeconds int,
) (*RedisUserInfoCacheService, error) {
	if strings.TrimSpace(keyPrefix) == "" {
		keyPrefix = "user:info:"
	}
	if strings.TrimSpace(lastMessagePrefix) == "" {
		lastMessagePrefix = "user:lastmsg:"
	}
	if expireDays <= 0 {
		expireDays = 7
	}

	flushInterval := time.Duration(flushIntervalSeconds) * time.Second
	localTTL := time.Duration(localTTLSeconds) * time.Second
	if localTTL <= 0 {
		localTTL = time.Hour
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	opts, err := buildRedisOptions(redisURL, host, port, password, db, timeout)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	svc := &RedisUserInfoCacheService{
		client:            client,
		timeout:           timeout,
		keyPrefix:         keyPrefix,
		lastMessagePrefix: lastMessagePrefix,
		expire:            time.Duration(expireDays) * 24 * time.Hour,
		flushInterval:     flushInterval,
		local:             newLRUCache(10000, localTTL),
	}

	if svc.flushInterval > 0 {
		svc.pending = make(map[string][]byte, 1024)
		svc.stopCh = make(chan struct{})
		svc.wg.Add(1)
		go svc.flushLoop()
	}

	return svc, nil
}

func buildRedisOptions(redisURL string, host string, port int, password string, db int, timeout time.Duration) (*redis.Options, error) {
	candidate := strings.TrimSpace(redisURL)
	if candidate == "" {
		// 兼容：允许把 REDIS_HOST 直接设置为 redis:// 或 rediss:// 连接串。
		trimmedHost := strings.TrimSpace(host)
		if strings.HasPrefix(trimmedHost, "redis://") || strings.HasPrefix(trimmedHost, "rediss://") {
			candidate = trimmedHost
		}
	}

	var opts *redis.Options
	if candidate != "" {
		parsed, err := redis.ParseURL(candidate)
		if err != nil {
			// 解析错误可能包含原始 URL（含密码），这里避免泄露敏感信息。
			return nil, fmt.Errorf("解析 Redis URL 失败（请检查是否为 redis:// 或 rediss:// 格式）")
		}
		opts = parsed
	} else {
		addr := fmt.Sprintf("%s:%d", strings.TrimSpace(host), port)
		if strings.HasPrefix(addr, ":") {
			addr = "localhost" + addr
		}
		opts = &redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}
	}

	// 统一连接参数，避免不同来源（URL/host）导致行为不一致。
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	opts.DialTimeout = timeout
	opts.ReadTimeout = timeout
	opts.WriteTimeout = timeout
	opts.PoolSize = 8
	return opts, nil
}

func (s *RedisUserInfoCacheService) Close() error {
	if s == nil || s.client == nil {
		return nil
	}

	var err error
	s.closeOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
			s.wg.Wait()
		} else {
			s.flushOnce()
		}
		err = s.client.Close()
	})
	return err
}

func (s *RedisUserInfoCacheService) flushLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOnce()
		case <-s.stopCh:
			// 退出前尽量把最后一批写入刷入 Redis。
			s.flushOnce()
			return
		}
	}
}

func (s *RedisUserInfoCacheService) flushOnce() {
	if s == nil || s.client == nil {
		return
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = s.flushPending(ctx)
}

func (s *RedisUserInfoCacheService) flushPending(ctx context.Context) error {
	if s == nil || s.client == nil || ctx == nil {
		return nil
	}

	s.pendingMu.Lock()
	if len(s.pending) == 0 {
		s.pendingMu.Unlock()
		return nil
	}
	// 直接交换 map，避免复制与 clear 带来的额外 O(n) 开销。
	batch := s.pending
	s.pending = make(map[string][]byte, 1024)
	s.pendingMu.Unlock()

	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for key, raw := range batch {
			pipe.Set(ctx, key, raw, s.expire)
		}
		return nil
	})
	// 缓存写入为 best-effort：失败时不阻塞主流程，也不重试回灌（避免异常时内存无限增长）。
	return err
}

func (s *RedisUserInfoCacheService) SaveUserInfo(info CachedUserInfo) {
	userID := strings.TrimSpace(info.UserID)
	if userID == "" || s == nil || s.client == nil {
		return
	}

	info.UserID = userID
	info.UpdateTime = time.Now().UnixMilli()

	s.local.Set(userID, info)

	raw, err := json.Marshal(info)
	if err != nil {
		return
	}

	if s.flushInterval <= 0 {
		ctx := context.Background()
		_ = s.client.Set(ctx, s.keyPrefix+userID, raw, s.expire).Err()
		return
	}

	s.pendingMu.Lock()
	if s.pending != nil {
		s.pending[s.keyPrefix+userID] = raw
	}
	s.pendingMu.Unlock()
}

func (s *RedisUserInfoCacheService) GetUserInfo(userID string) *CachedUserInfo {
	userID = strings.TrimSpace(userID)
	if userID == "" || s == nil || s.client == nil {
		return nil
	}

	if v, ok := s.local.Get(userID); ok {
		if info, ok := v.(CachedUserInfo); ok {
			cp := info
			return &cp
		}
	}

	ctx := context.Background()
	raw, err := s.client.Get(ctx, s.keyPrefix+userID).Bytes()
	if err != nil {
		return nil
	}
	var info CachedUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil
	}
	s.local.Set(userID, info)
	return &info
}

func (s *RedisUserInfoCacheService) EnrichUserInfo(userID string, originalData map[string]any) map[string]any {
	info := s.GetUserInfo(userID)
	if info == nil {
		return originalData
	}
	putIfAbsent(originalData, "nickname", info.Nickname)
	putIfAbsent(originalData, "sex", info.Gender)
	putIfAbsent(originalData, "age", info.Age)
	putIfAbsent(originalData, "address", info.Address)
	return originalData
}

func (s *RedisUserInfoCacheService) BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any {
	if userList == nil || len(userList) == 0 {
		return userList
	}

	userIDs := make([]string, 0, len(userList))
	seen := make(map[string]struct{}, len(userList))
	for _, m := range userList {
		if m == nil {
			continue
		}
		raw, ok := m[userIDKey]
		if !ok || raw == nil {
			continue
		}
		uid := strings.TrimSpace(toString(raw))
		if uid == "" {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		userIDs = append(userIDs, uid)
	}
	if len(userIDs) == 0 {
		return userList
	}

	infoMap := s.multiGetUserInfo(userIDs)
	for _, m := range userList {
		if m == nil {
			continue
		}
		raw, ok := m[userIDKey]
		if !ok || raw == nil {
			continue
		}
		uid := strings.TrimSpace(toString(raw))
		if uid == "" {
			continue
		}
		info, ok := infoMap[uid]
		if !ok {
			continue
		}
		putIfAbsent(m, "nickname", info.Nickname)
		putIfAbsent(m, "sex", info.Gender)
		putIfAbsent(m, "age", info.Age)
		putIfAbsent(m, "address", info.Address)
	}
	return userList
}

func (s *RedisUserInfoCacheService) batchGetUserInfo(userIDs []string) map[string]CachedUserInfo {
	return s.multiGetUserInfo(userIDs)
}

func (s *RedisUserInfoCacheService) multiGetUserInfo(userIDs []string) map[string]CachedUserInfo {
	result := make(map[string]CachedUserInfo, len(userIDs))
	missing := make([]string, 0, len(userIDs))

	for _, uid := range userIDs {
		if v, ok := s.local.Get(uid); ok {
			if info, ok := v.(CachedUserInfo); ok {
				result[uid] = info
				continue
			}
		}
		missing = append(missing, uid)
	}
	if len(missing) == 0 {
		return result
	}

	keys := make([]string, 0, len(missing))
	for _, uid := range missing {
		keys = append(keys, s.keyPrefix+uid)
	}

	ctx := context.Background()
	rawList, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return result
	}

	for i, raw := range rawList {
		if raw == nil {
			continue
		}
		var bytes []byte
		switch t := raw.(type) {
		case string:
			bytes = []byte(t)
		case []byte:
			bytes = t
		default:
			continue
		}

		var info CachedUserInfo
		if err := json.Unmarshal(bytes, &info); err != nil {
			continue
		}
		uid := missing[i]
		result[uid] = info
		s.local.Set(uid, info)
	}
	return result
}

func (s *RedisUserInfoCacheService) SaveLastMessage(message CachedLastMessage) {
	if s == nil || s.client == nil {
		return
	}

	if strings.TrimSpace(message.ConversationKey) == "" {
		message.ConversationKey = generateConversationKey(message.FromUserID, message.ToUserID)
	}
	if strings.TrimSpace(message.ConversationKey) == "" {
		return
	}
	message.UpdateTime = time.Now().UnixMilli()

	cacheKey := "lastmsg_" + message.ConversationKey
	s.local.Set(cacheKey, message)

	raw, err := json.Marshal(message)
	if err != nil {
		return
	}

	if s.flushInterval <= 0 {
		ctx := context.Background()
		_ = s.client.Set(ctx, s.lastMessagePrefix+message.ConversationKey, raw, s.expire).Err()
		return
	}

	s.pendingMu.Lock()
	if s.pending != nil {
		s.pending[s.lastMessagePrefix+message.ConversationKey] = raw
	}
	s.pendingMu.Unlock()
}

func (s *RedisUserInfoCacheService) GetLastMessage(myUserID, otherUserID string) *CachedLastMessage {
	if s == nil || s.client == nil {
		return nil
	}

	key := generateConversationKey(strings.TrimSpace(myUserID), strings.TrimSpace(otherUserID))
	if key == "" {
		return nil
	}
	cacheKey := "lastmsg_" + key

	if v, ok := s.local.Get(cacheKey); ok {
		if msg, ok := v.(CachedLastMessage); ok {
			cp := msg
			return &cp
		}
	}

	ctx := context.Background()
	raw, err := s.client.Get(ctx, s.lastMessagePrefix+key).Bytes()
	if err != nil {
		return nil
	}
	var msg CachedLastMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}
	s.local.Set(cacheKey, msg)
	return &msg
}

func (s *RedisUserInfoCacheService) BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any {
	if userList == nil || len(userList) == 0 {
		return userList
	}
	myUserID = strings.TrimSpace(myUserID)
	if myUserID == "" {
		return userList
	}

	conversationKeys := make([]string, 0, len(userList))
	for _, user := range userList {
		otherUserID := strings.TrimSpace(extractUserID(user))
		if otherUserID == "" {
			continue
		}
		key := generateConversationKey(myUserID, otherUserID)
		if key == "" {
			continue
		}
		conversationKeys = append(conversationKeys, key)
	}
	if len(conversationKeys) == 0 {
		return userList
	}

	messageMap := s.multiGetLastMessages(conversationKeys)
	for _, user := range userList {
		otherUserID := strings.TrimSpace(extractUserID(user))
		if otherUserID == "" {
			continue
		}
		key := generateConversationKey(myUserID, otherUserID)
		msg, ok := messageMap[key]
		if !ok {
			continue
		}
		user["lastMsg"] = formatLastMessage(msg, myUserID)
		user["lastTime"] = formatTime(msg.Time)
	}
	return userList
}

func (s *RedisUserInfoCacheService) batchGetLastMessages(conversationKeys []string) map[string]CachedLastMessage {
	return s.multiGetLastMessages(conversationKeys)
}

func (s *RedisUserInfoCacheService) multiGetLastMessages(conversationKeys []string) map[string]CachedLastMessage {
	result := make(map[string]CachedLastMessage, len(conversationKeys))
	missing := make([]string, 0, len(conversationKeys))

	for _, key := range conversationKeys {
		cacheKey := "lastmsg_" + key
		if v, ok := s.local.Get(cacheKey); ok {
			if msg, ok := v.(CachedLastMessage); ok {
				result[key] = msg
				continue
			}
		}
		missing = append(missing, key)
	}
	if len(missing) == 0 {
		return result
	}

	keys := make([]string, 0, len(missing))
	for _, key := range missing {
		keys = append(keys, s.lastMessagePrefix+key)
	}

	ctx := context.Background()
	rawList, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return result
	}

	for i, raw := range rawList {
		if raw == nil {
			continue
		}
		var bytes []byte
		switch t := raw.(type) {
		case string:
			bytes = []byte(t)
		case []byte:
			bytes = t
		default:
			continue
		}

		var msg CachedLastMessage
		if err := json.Unmarshal(bytes, &msg); err != nil {
			continue
		}
		key := missing[i]
		result[key] = msg
		s.local.Set("lastmsg_"+key, msg)
	}

	return result
}
