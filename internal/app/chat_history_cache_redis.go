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
	payload         []byte
}

// RedisChatHistoryCacheService 基于 ZSET index + 单消息 key 的聊天记录缓存。
// - index: {prefix}{conversationKey}:index (score=tid, member=tid)
// - msg:   {prefix}{conversationKey}:msg:{tid} (value=JSON, ttl=expire)
type RedisChatHistoryCacheService struct {
	client    *redis.Client
	keyPrefix string
	expire    time.Duration

	flushInterval time.Duration
	pendingMu     sync.Mutex
	pending       map[string]pendingChatHistoryMessage // key=messageKey，避免重复写入
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
) (*RedisChatHistoryCacheService, error) {
	if strings.TrimSpace(keyPrefix) == "" {
		keyPrefix = "user:chathistory:"
	}
	if expireDays <= 0 {
		expireDays = 30
	}

	opts, err := buildRedisOptions(redisURL, host, port, password, db)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	svc := &RedisChatHistoryCacheService{
		client:        client,
		keyPrefix:     keyPrefix,
		expire:        time.Duration(expireDays) * 24 * time.Hour,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for messageKey, msg := range batch {
			indexKey := s.indexKey(msg.conversationKey)
			pipe.Set(ctx, messageKey, msg.payload, s.expire)
			pipe.ZAdd(ctx, indexKey, redis.Z{Score: msg.score, Member: msg.tid})
			pipe.Expire(ctx, indexKey, s.expire)
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
		batch = append(batch, pendingChatHistoryMessage{
			conversationKey: conversationKey,
			tid:             tid,
			score:           float64(scoreInt),
			payload:         raw,
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
			s.pending[s.messageKey(conversationKey, msg.tid)] = msg
		}
	}
	s.pendingMu.Unlock()
}

func (s *RedisChatHistoryCacheService) writeBatch(ctx context.Context, msgs []pendingChatHistoryMessage) error {
	if s == nil || s.client == nil || ctx == nil || len(msgs) == 0 {
		return nil
	}

	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, msg := range msgs {
			messageKey := s.messageKey(msg.conversationKey, msg.tid)
			indexKey := s.indexKey(msg.conversationKey)
			pipe.Set(ctx, messageKey, msg.payload, s.expire)
			pipe.ZAdd(ctx, indexKey, redis.Z{Score: msg.score, Member: msg.tid})
			pipe.Expire(ctx, indexKey, s.expire)
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

	indexKey := s.indexKey(conversationKey)
	tids, err := s.client.ZRevRangeByScore(readCtx, indexKey, &redis.ZRangeBy{
		Max:    maxScore,
		Min:    "-inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(tids) == 0 {
		return []map[string]any{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, 0, len(tids))
	orderedTids := make([]string, 0, len(tids))
	for _, tid := range tids {
		tid = strings.TrimSpace(tid)
		if tid == "" {
			continue
		}
		orderedTids = append(orderedTids, tid)
		cmds = append(cmds, pipe.Get(readCtx, s.messageKey(conversationKey, tid)))
	}
	_, _ = pipe.Exec(readCtx)

	out := make([]map[string]any, 0, len(cmds))
	missing := make([]string, 0, len(cmds))
	for i, cmd := range cmds {
		tid := ""
		if i < len(orderedTids) {
			tid = orderedTids[i]
		}

		raw, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				if tid != "" {
					missing = append(missing, tid)
				}
				continue
			}
			return nil, err
		}

		var msg map[string]any
		if unmarshalErr := json.Unmarshal([]byte(raw), &msg); unmarshalErr != nil || msg == nil {
			if tid != "" {
				missing = append(missing, tid)
			}
			continue
		}
		out = append(out, msg)
	}

	if len(missing) > 0 {
		members := make([]any, 0, len(missing))
		for _, tid := range missing {
			tid = strings.TrimSpace(tid)
			if tid == "" {
				continue
			}
			members = append(members, tid)
		}
		if len(members) > 0 {
			_ = s.client.ZRem(readCtx, indexKey, members...).Err()
		}
	}

	return out, nil
}

func (s *RedisChatHistoryCacheService) indexKey(conversationKey string) string {
	return s.keyPrefix + conversationKey + ":index"
}

func (s *RedisChatHistoryCacheService) messageKey(conversationKey string, tid string) string {
	return s.keyPrefix + conversationKey + ":msg:" + tid
}
