package app

import (
	"container/list"
	"sync"
	"time"
)

type lruCacheEntry struct {
	key       string
	value     any
	expiresAt time.Time
}

type lruCache struct {
	mu   sync.Mutex
	ll   *list.List
	data map[string]*list.Element

	maxEntries int
	ttl        time.Duration
}

func newLRUCache(maxEntries int, ttl time.Duration) *lruCache {
	if maxEntries <= 0 {
		maxEntries = 10000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &lruCache{
		ll:         list.New(),
		data:       make(map[string]*list.Element),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (c *lruCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el := c.data[key]
	if el == nil {
		return nil, false
	}
	entry, ok := el.Value.(lruCacheEntry)
	if !ok {
		c.ll.Remove(el)
		delete(c.data, key)
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.ll.Remove(el)
		delete(c.data, key)
		return nil, false
	}

	c.ll.MoveToFront(el)
	return entry.value, true
}

func (c *lruCache) Set(key string, value any) {
	if key == "" || value == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if el := c.data[key]; el != nil {
		c.ll.MoveToFront(el)
		el.Value = lruCacheEntry{key: key, value: value, expiresAt: time.Now().Add(c.ttl)}
		return
	}

	el := c.ll.PushFront(lruCacheEntry{key: key, value: value, expiresAt: time.Now().Add(c.ttl)})
	c.data[key] = el

	for c.maxEntries > 0 && c.ll.Len() > c.maxEntries {
		back := c.ll.Back()
		if back == nil {
			break
		}
		c.ll.Remove(back)
		if entry, ok := back.Value.(lruCacheEntry); ok {
			delete(c.data, entry.key)
		}
	}
}

func (c *lruCache) Delete(key string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	el := c.data[key]
	if el == nil {
		return
	}
	c.ll.Remove(el)
	delete(c.data, key)
}

