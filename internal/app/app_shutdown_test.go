package app

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
)

type closableUserInfoCache struct {
	closed bool
}

func (c *closableUserInfoCache) Close() error { c.closed = true; return nil }

func (c *closableUserInfoCache) SaveUserInfo(info CachedUserInfo) {}
func (c *closableUserInfoCache) GetUserInfo(userID string) *CachedUserInfo {
	return nil
}
func (c *closableUserInfoCache) EnrichUserInfo(userID string, originalData map[string]any) map[string]any {
	return originalData
}
func (c *closableUserInfoCache) BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any {
	return userList
}
func (c *closableUserInfoCache) SaveLastMessage(message CachedLastMessage) {}
func (c *closableUserInfoCache) GetLastMessage(myUserID, otherUserID string) *CachedLastMessage {
	return nil
}
func (c *closableUserInfoCache) BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any {
	return userList
}

type closableChatHistoryCache struct {
	closed bool
}

func (c *closableChatHistoryCache) Close() error { c.closed = true; return nil }

func (c *closableChatHistoryCache) SaveMessages(ctx context.Context, conversationKey string, messages []map[string]any) {
}
func (c *closableChatHistoryCache) GetMessages(ctx context.Context, conversationKey string, beforeTid string, limit int) ([]map[string]any, error) {
	return nil, nil
}

var errCloseDriverCounter uint64

type errCloseDriver struct{}

func (errCloseDriver) Open(name string) (driver.Conn, error) { return &errCloseConn{}, nil }

type errCloseConn struct{}

func (*errCloseConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (*errCloseConn) Close() error              { return errors.New("close fail") }
func (*errCloseConn) Begin() (driver.Tx, error) { return nil, errors.New("tx not supported") }

func TestResolveStaticDir_WhenNoCandidateExists_ReturnsFirst(t *testing.T) {
	tmp := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	if got := resolveStaticDir(); got != "src/main/resources/static" {
		t.Fatalf("got=%q", got)
	}
}

func TestAppShutdown_ClosesClosableCaches_AndLogsDBCloseError(t *testing.T) {
	driverName := fmt.Sprintf("errclose_%d", atomic.AddUint64(&errCloseDriverCounter, 1))
	sql.Register(driverName, errCloseDriver{})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	uic := &closableUserInfoCache{}
	chc := &closableChatHistoryCache{}

	application := &App{
		db:               db,
		userInfoCache:    uic,
		chatHistoryCache: chc,
	}
	application.Shutdown(context.Background())

	if !uic.closed {
		t.Fatalf("expected userInfoCache closed")
	}
	if !chc.closed {
		t.Fatalf("expected chatHistoryCache closed")
	}
}
