package app

import (
	"sync"
	"time"
)

type ForceoutManager struct {
	mu    sync.Mutex
	users map[string]time.Time
}

func NewForceoutManager() *ForceoutManager {
	return &ForceoutManager{users: make(map[string]time.Time)}
}

const forceoutDuration = 5 * time.Minute

func (m *ForceoutManager) IsForbidden(userID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	expire, ok := m.users[userID]
	if !ok {
		return false
	}
	if time.Now().After(expire) {
		delete(m.users, userID)
		return false
	}
	return true
}

func (m *ForceoutManager) AddForceoutUser(userID string) {
	m.mu.Lock()
	m.users[userID] = time.Now().Add(forceoutDuration)
	m.mu.Unlock()
}

func (m *ForceoutManager) RemainingSeconds(userID string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	expire, ok := m.users[userID]
	if !ok {
		return 0
	}
	remaining := time.Until(expire).Seconds()
	if remaining <= 0 {
		delete(m.users, userID)
		return 0
	}
	return int64(remaining)
}

func (m *ForceoutManager) ClearAllForceout() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := len(m.users)
	m.users = make(map[string]time.Time)
	return count
}

func (m *ForceoutManager) GetForbiddenUserCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, exp := range m.users {
		if now.After(exp) {
			delete(m.users, k)
		}
	}
	return len(m.users)
}

