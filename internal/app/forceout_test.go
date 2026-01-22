package app

import (
	"testing"
	"time"
)

func TestForceoutManager_AddAndExpire(t *testing.T) {
	m := NewForceoutManager()
	if m.IsForbidden("u1") {
		t.Fatalf("expected not forbidden")
	}

	m.AddForceoutUser("u1")
	if !m.IsForbidden("u1") {
		t.Fatalf("expected forbidden")
	}
	if m.RemainingSeconds("u1") <= 0 {
		t.Fatalf("expected remaining seconds > 0")
	}

	// 手动构造过期，验证清理逻辑
	m.mu.Lock()
	m.users["u1"] = time.Now().Add(-1 * time.Second)
	m.mu.Unlock()

	if m.IsForbidden("u1") {
		t.Fatalf("expected expired -> not forbidden")
	}
}

func TestForceoutManager_ClearAndCount(t *testing.T) {
	m := NewForceoutManager()
	m.AddForceoutUser("u1")
	m.AddForceoutUser("u2")

	if got := m.GetForbiddenUserCount(); got != 2 {
		t.Fatalf("count=%d, want 2", got)
	}
	if cleared := m.ClearAllForceout(); cleared != 2 {
		t.Fatalf("cleared=%d, want 2", cleared)
	}
	if got := m.GetForbiddenUserCount(); got != 0 {
		t.Fatalf("count=%d, want 0", got)
	}
}
