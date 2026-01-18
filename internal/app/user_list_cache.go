package app

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// userListCache 用于缓存历史/收藏用户列表的最终响应（JSON）。
// 设计目标：减少上游调用次数，并在必要时由 WS 写入同步刷新列表预览字段。
type userListCache struct {
	history  *lruCache
	favorite *lruCache
}

type cachedUserList struct {
	mu   sync.RWMutex
	list []map[string]any
	byID map[string]map[string]any

	jsonBytes []byte
	size      int
	dirty     bool
}

func newUserListCache(maxEntries int, ttl time.Duration) *userListCache {
	return &userListCache{
		history:  newLRUCache(maxEntries, ttl),
		favorite: newLRUCache(maxEntries, ttl),
	}
}

func (c *userListCache) GetHistory(myUserID string) (body []byte, size int, ok bool) {
	return c.get(c.history, myUserID)
}

func (c *userListCache) SetHistory(myUserID string, list []map[string]any, body []byte) {
	c.set(c.history, myUserID, list, body)
}

func (c *userListCache) GetFavorite(myUserID string) (body []byte, size int, ok bool) {
	return c.get(c.favorite, myUserID)
}

func (c *userListCache) SetFavorite(myUserID string, list []map[string]any, body []byte) {
	c.set(c.favorite, myUserID, list, body)
}

func (c *userListCache) UpdateLastMessage(message CachedLastMessage) {
	if c == nil {
		return
	}
	from := strings.TrimSpace(message.FromUserID)
	to := strings.TrimSpace(message.ToUserID)
	if from == "" || to == "" {
		return
	}

	// lastMsg/lastTime 的显示与 myUserID 有关，需要分别更新两侧用户的列表缓存。
	c.updateLastMessage(c.history, from, to, message)
	c.updateLastMessage(c.history, to, from, message)
	c.updateLastMessage(c.favorite, from, to, message)
	c.updateLastMessage(c.favorite, to, from, message)
}

func (c *userListCache) get(store *lruCache, myUserID string) (body []byte, size int, ok bool) {
	if c == nil || store == nil {
		return nil, 0, false
	}
	key := strings.TrimSpace(myUserID)
	if key == "" {
		return nil, 0, false
	}

	v, ok := store.Get(key)
	if !ok || v == nil {
		return nil, 0, false
	}
	entry, ok := v.(*cachedUserList)
	if !ok || entry == nil {
		return nil, 0, false
	}
	return entry.bytes()
}

func (c *userListCache) set(store *lruCache, myUserID string, list []map[string]any, body []byte) {
	if c == nil || store == nil {
		return
	}
	key := strings.TrimSpace(myUserID)
	if key == "" || list == nil {
		return
	}
	store.Set(key, newCachedUserList(list, body))
}

func (c *userListCache) updateLastMessage(store *lruCache, myUserID string, otherUserID string, message CachedLastMessage) {
	if c == nil || store == nil {
		return
	}
	myUserID = strings.TrimSpace(myUserID)
	otherUserID = strings.TrimSpace(otherUserID)
	if myUserID == "" || otherUserID == "" {
		return
	}
	v, ok := store.Get(myUserID)
	if !ok || v == nil {
		return
	}
	entry, ok := v.(*cachedUserList)
	if !ok || entry == nil {
		return
	}
	if entry.updateLastMessage(myUserID, otherUserID, message) {
		// 延长 TTL：只要持续有消息写入，就尽量保持缓存可用。
		store.Set(myUserID, entry)
	}
}

func newCachedUserList(list []map[string]any, body []byte) *cachedUserList {
	byID := make(map[string]map[string]any, len(list))
	for _, m := range list {
		uid := strings.TrimSpace(extractUserID(m))
		if uid == "" {
			continue
		}
		byID[uid] = m
	}

	entry := &cachedUserList{
		list: list,
		byID: byID,
		size: len(list),
	}
	if len(body) > 0 {
		entry.jsonBytes = body
		entry.dirty = false
	} else {
		entry.dirty = true
	}
	return entry
}

func (e *cachedUserList) bytes() (body []byte, size int, ok bool) {
	if e == nil {
		return nil, 0, false
	}

	e.mu.RLock()
	if !e.dirty && len(e.jsonBytes) > 0 {
		body = e.jsonBytes
		size = e.size
		e.mu.RUnlock()
		return body, size, true
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.dirty && len(e.jsonBytes) > 0 {
		return e.jsonBytes, e.size, true
	}
	b, err := json.Marshal(e.list)
	if err != nil {
		return nil, 0, false
	}
	e.jsonBytes = b
	e.size = len(e.list)
	e.dirty = false
	return b, e.size, true
}

func (e *cachedUserList) updateLastMessage(myUserID string, otherUserID string, message CachedLastMessage) bool {
	if e == nil {
		return false
	}
	otherUserID = strings.TrimSpace(otherUserID)
	if otherUserID == "" {
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	target := e.byID[otherUserID]
	if target == nil {
		return false
	}

	target["lastMsg"] = formatLastMessage(message, myUserID)
	target["lastTime"] = formatTime(message.Time)

	e.dirty = true
	e.jsonBytes = nil
	return true
}
