package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type pendingChatHistoryMessage struct {
	conversationKey string
	tid             string
	score           float64
	member          string
}

// RedisChatHistoryCacheService 基于「单 key ZSET」的聊天记录缓存（每个会话一个 key）。
//
// Key:
// - {prefix}{conversationKey} -> ZSET（score=tid）
//
// Member:
// - "{tid}|{json}"（json 为上游 contents_list 单条消息对象）
type RedisChatHistoryCacheService struct {
	client    *redis.Client
	keyPrefix string
	expire    time.Duration
	timeout   time.Duration

	flushInterval time.Duration
	pendingMu     sync.Mutex
	pending       map[string]pendingChatHistoryMessage // key=conversationKey|tid，避免重复写入
	stopCh        chan struct{}
	wg            sync.WaitGroup
	closeOnce     sync.Once
}

func NewRedisChatHistoryCacheService(
	redisURL string,
	host string,
	port int,
	password string,
	db int,
	keyPrefix string,
	expireDays int,
	flushIntervalSeconds int,
	timeoutSeconds int,
) (*RedisChatHistoryCacheService, error) {
	if strings.TrimSpace(keyPrefix) == "" {
		keyPrefix = "user:chathistory:"
	}
	if expireDays <= 0 {
		expireDays = 30
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

	svc := &RedisChatHistoryCacheService{
		client:        client,
		keyPrefix:     keyPrefix,
		expire:        time.Duration(expireDays) * 24 * time.Hour,
		timeout:       timeout,
		flushInterval: time.Duration(flushIntervalSeconds) * time.Second,
	}

	if svc.flushInterval > 0 {
		svc.pending = make(map[string]pendingChatHistoryMessage, 2048)
		svc.stopCh = make(chan struct{})
		svc.wg.Add(1)
		go svc.flushLoop()
	}

	return svc, nil
}

func (s *RedisChatHistoryCacheService) Close() error {
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

func (s *RedisChatHistoryCacheService) flushLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOnce()
		case <-s.stopCh:
			s.flushOnce()
			return
		}
	}
}

func (s *RedisChatHistoryCacheService) flushOnce() {
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

func (s *RedisChatHistoryCacheService) flushPending(ctx context.Context) error {
	if s == nil || s.client == nil || ctx == nil {
		return nil
	}

	s.pendingMu.Lock()
	if len(s.pending) == 0 {
		s.pendingMu.Unlock()
		return nil
	}
	batch := s.pending
	s.pending = make(map[string]pendingChatHistoryMessage, 2048)
	s.pendingMu.Unlock()

	grouped := make(map[string][]pendingChatHistoryMessage, 32)
	for _, msg := range batch {
		grouped[msg.conversationKey] = append(grouped[msg.conversationKey], msg)
	}

	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for conv, msgs := range grouped {
			key := s.zsetKey(conv)
			for _, msg := range msgs {
				pipe.ZRemRangeByScore(ctx, key, msg.tid, msg.tid)
				pipe.ZAdd(ctx, key, redis.Z{Score: msg.score, Member: msg.member})
			}
			pipe.Expire(ctx, key, s.expire)
		}
		return nil
	})
	return err
}

func (s *RedisChatHistoryCacheService) SaveMessages(ctx context.Context, conversationKey string, messages []map[string]any) {
	conversationKey = strings.TrimSpace(conversationKey)
	if conversationKey == "" || s == nil || s.client == nil || len(messages) == 0 {
		return
	}

	var batch []pendingChatHistoryMessage
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		tid := strings.TrimSpace(extractHistoryMessageTid(msg))
		if tid == "" {
			continue
		}
		scoreInt, err := strconv.ParseInt(tid, 10, 64)
		if err != nil {
			continue
		}
		raw, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		member := tid + "|" + string(raw)
		batch = append(batch, pendingChatHistoryMessage{
			conversationKey: conversationKey,
			tid:             tid,
			score:           float64(scoreInt),
			member:          member,
		})
	}
	if len(batch) == 0 {
		return
	}

	if s.flushInterval <= 0 {
		writeCtx := ctx
		if writeCtx == nil {
			writeCtx = context.Background()
		}
		_ = s.writeBatch(writeCtx, batch)
		return
	}

	s.pendingMu.Lock()
	if s.pending != nil {
		for _, msg := range batch {
			s.pending[conversationKey+"|"+msg.tid] = msg
		}
	}
	s.pendingMu.Unlock()
}

func (s *RedisChatHistoryCacheService) writeBatch(ctx context.Context, msgs []pendingChatHistoryMessage) error {
	if s == nil || s.client == nil || ctx == nil || len(msgs) == 0 {
		return nil
	}

	grouped := make(map[string][]pendingChatHistoryMessage, 32)
	for _, msg := range msgs {
		grouped[msg.conversationKey] = append(grouped[msg.conversationKey], msg)
	}

	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for conv, items := range grouped {
			key := s.zsetKey(conv)
			for _, msg := range items {
				pipe.ZRemRangeByScore(ctx, key, msg.tid, msg.tid)
				pipe.ZAdd(ctx, key, redis.Z{Score: msg.score, Member: msg.member})
			}
			pipe.Expire(ctx, key, s.expire)
		}
		return nil
	})
	return err
}

func (s *RedisChatHistoryCacheService) GetMessages(ctx context.Context, conversationKey string, beforeTid string, limit int) ([]map[string]any, error) {
	conversationKey = strings.TrimSpace(conversationKey)
	if conversationKey == "" || s == nil || s.client == nil || limit <= 0 {
		return []map[string]any{}, nil
	}

	readCtx := ctx
	if readCtx == nil {
		readCtx = context.Background()
	}

	maxScore := "+inf"
	if trimmed := strings.TrimSpace(beforeTid); trimmed != "" && trimmed != "0" {
		if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil && n > 0 {
			maxScore = strconv.FormatInt(n-1, 10)
		}
	}

	key := s.zsetKey(conversationKey)
	members, err := s.client.ZRevRangeByScore(readCtx, key, &redis.ZRangeBy{
		Max:    maxScore,
		Min:    "-inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return []map[string]any{}, nil
	}

	out := make([]map[string]any, 0, len(members))
	for _, member := range members {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}

		payload := member
		if idx := strings.IndexByte(member, '|'); idx >= 0 && idx+1 < len(member) {
			payload = member[idx+1:]
		}

		var msg map[string]any
		if unmarshalErr := json.Unmarshal([]byte(payload), &msg); unmarshalErr != nil || msg == nil {
			continue
		}
		out = append(out, msg)
	}

	return out, nil
}

func (s *RedisChatHistoryCacheService) zsetKey(conversationKey string) string {
	return s.keyPrefix + conversationKey
}
