package app

import (
	"context"
	"strings"
)

// ChatHistoryCacheService 用于缓存聊天历史（按会话维度，格式对齐上游 contents_list）。
// 说明：该缓存为 best-effort，Redis 异常不应影响主流程。
type ChatHistoryCacheService interface {
	SaveMessages(ctx context.Context, conversationKey string, messages []map[string]any)
	GetMessages(ctx context.Context, conversationKey string, beforeTid string, limit int) ([]map[string]any, error)
}

func extractHistoryMessageTid(message map[string]any) string {
	if message == nil {
		return ""
	}
	if v, ok := message["Tid"]; ok && v != nil {
		return strings.TrimSpace(toString(v))
	}
	if v, ok := message["tid"]; ok && v != nil {
		return strings.TrimSpace(toString(v))
	}
	return ""
}

func extractHistoryMessageDedupKey(message map[string]any) string {
	if message == nil {
		return ""
	}
	if tid := extractHistoryMessageTid(message); tid != "" {
		return "tid:" + tid
	}

	id := strings.TrimSpace(toString(message["id"]))
	toid := strings.TrimSpace(toString(message["toid"]))
	content := strings.TrimSpace(toString(message["content"]))
	tm := strings.TrimSpace(toString(message["time"]))
	if id == "" && toid == "" && content == "" && tm == "" {
		return ""
	}
	return "fallback:" + id + "|" + toid + "|" + tm + "|" + content
}

func mergeHistoryMessages(primary []map[string]any, secondary []map[string]any, limit int) []map[string]any {
	if limit <= 0 {
		limit = 0
	}

	seen := make(map[string]struct{}, len(primary)+len(secondary))
	out := make([]map[string]any, 0, len(primary)+len(secondary))

	appendList := func(list []map[string]any) {
		for _, msg := range list {
			if msg == nil {
				continue
			}
			key := extractHistoryMessageDedupKey(msg)
			if key != "" {
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
			}
			out = append(out, msg)
			if limit > 0 && len(out) >= limit {
				return
			}
		}
	}

	appendList(primary)
	if limit == 0 || len(out) < limit {
		appendList(secondary)
	}
	return out
}
