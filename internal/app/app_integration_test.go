package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
	"liao/internal/database"
)

func TestURLQueryEscape(t *testing.T) {
	if got := urlQueryEscape("Asia/Shanghai"); got != "Asia%2FShanghai" {
		t.Fatalf("got=%q", got)
	}
	if got := urlQueryEscape("Local Time"); got != "Local%20Time" {
		t.Fatalf("got=%q", got)
	}
}

func TestResolveStaticDir_PrefersFirstExistingCandidate(t *testing.T) {
	tmp := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	// only "static"
	if err := os.MkdirAll("static", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := resolveStaticDir(); got != "static" {
		t.Fatalf("got=%q, want %q", got, "static")
	}

	// create first candidate too
	first := filepath.FromSlash("src/main/resources/static")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := resolveStaticDir(); got != first {
		t.Fatalf("got=%q, want %q", got, first)
	}
}

func TestOpenDB_UsesEncodedLocAndPings(t *testing.T) {
	oldOpen := sqlOpenFn
	t.Cleanup(func() { sqlOpenFn = oldOpen })

	db, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
		sqlmock.MonitorPingsOption(true),
	)
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(nil)

	var gotDriver string
	var gotDSN string
	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) {
		gotDriver = driverName
		gotDSN = dsn
		return db, nil
	}

	_, err = openDB(config.Config{
		DBURL:      "jdbc:mysql://localhost:3306/hot_img?serverTimezone=Asia/Shanghai",
		DBUsername: "u",
		DBPassword: "p",
	})
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if gotDriver != "mysql" {
		t.Fatalf("driver=%q", gotDriver)
	}
	if !strings.Contains(gotDSN, "loc=Asia%2FShanghai") {
		t.Fatalf("dsn=%q", gotDSN)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOpenDB_Postgres_FiltersMySQLParamsAndMapsTimezone(t *testing.T) {
	oldOpen := sqlOpenFn
	t.Cleanup(func() { sqlOpenFn = oldOpen })

	db, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
		sqlmock.MonitorPingsOption(true),
	)
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(nil)

	var gotDriver string
	var gotDSN string
	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) {
		gotDriver = driverName
		gotDSN = dsn
		return db, nil
	}

	_, err = openDB(config.Config{
		DBURL:      "jdbc:postgres://localhost:5432/hot_img?useSSL=false&serverTimezone=Asia/Shanghai&characterEncoding=utf8&allowPublicKeyRetrieval=true",
		DBUsername: "u",
		DBPassword: "p",
	})
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if gotDriver != "pgx" {
		t.Fatalf("driver=%q", gotDriver)
	}
	if !strings.Contains(gotDSN, "sslmode=disable") {
		t.Fatalf("dsn=%q (missing sslmode=disable)", gotDSN)
	}
	if !strings.Contains(gotDSN, "timezone=Asia%2FShanghai") {
		t.Fatalf("dsn=%q (missing timezone=Asia%%2FShanghai)", gotDSN)
	}

	// Ensure MySQL-only params don't leak into Postgres runtime parameters.
	for _, bad := range []string{"serverTimezone", "useSSL", "characterEncoding", "allowPublicKeyRetrieval"} {
		if strings.Contains(gotDSN, bad) {
			t.Fatalf("dsn=%q (should not contain %q)", gotDSN, bad)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOpenDB_Errors(t *testing.T) {
	oldOpen := sqlOpenFn
	t.Cleanup(func() { sqlOpenFn = oldOpen })

	if _, err := openDB(config.Config{DBURL: ""}); err == nil {
		t.Fatalf("expected error")
	}

	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) {
		return nil, errors.New("open fail")
	}
	if _, err := openDB(config.Config{DBURL: "mysql://localhost:3306/hot_img"}); err == nil || !strings.Contains(err.Error(), "打开数据库失败") {
		t.Fatalf("err=%v", err)
	}

	db, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
		sqlmock.MonitorPingsOption(true),
	)
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(errors.New("ping fail"))
	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) { return db, nil }
	if _, err := openDB(config.Config{DBURL: "mysql://localhost:3306/hot_img"}); err == nil || !strings.Contains(err.Error(), "连接数据库失败") {
		t.Fatalf("err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

type closableUserInfoCacheStub struct {
	closed bool
}

func (s *closableUserInfoCacheStub) Close() error { s.closed = true; return nil }

func (s *closableUserInfoCacheStub) SaveUserInfo(info CachedUserInfo) {}
func (s *closableUserInfoCacheStub) GetUserInfo(userID string) *CachedUserInfo {
	return nil
}
func (s *closableUserInfoCacheStub) EnrichUserInfo(userID string, originalData map[string]any) map[string]any {
	return originalData
}
func (s *closableUserInfoCacheStub) BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any {
	return userList
}
func (s *closableUserInfoCacheStub) SaveLastMessage(message CachedLastMessage) {}
func (s *closableUserInfoCacheStub) GetLastMessage(myUserID, otherUserID string) *CachedLastMessage {
	return nil
}
func (s *closableUserInfoCacheStub) BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any {
	return userList
}

func TestNew_ErrorBranches(t *testing.T) {
	oldOpen := openDBFn
	oldEnsure := ensureSchemaFn
	oldStatic := resolveStaticDirFn
	oldMkdirAll := mkdirAllFn
	oldUIC := newRedisUserInfoCacheServiceFn
	oldCHC := newRedisChatHistoryCacheServiceFn
	t.Cleanup(func() {
		openDBFn = oldOpen
		ensureSchemaFn = oldEnsure
		resolveStaticDirFn = oldStatic
		mkdirAllFn = oldMkdirAll
		newRedisUserInfoCacheServiceFn = oldUIC
		newRedisChatHistoryCacheServiceFn = oldCHC
	})

	openDBFn = func(cfg config.Config) (*database.DB, error) { return nil, errors.New("db fail") }
	if _, err := New(config.Config{}); err == nil {
		t.Fatalf("expected error")
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	openDBFn = func(cfg config.Config) (*database.DB, error) { return database.Wrap(db, database.MySQLDialect{}), nil }
	ensureSchemaFn = func(db *database.DB) error { return errors.New("schema fail") }
	if _, err := New(config.Config{}); err == nil {
		t.Fatalf("expected error")
	}

	ensureSchemaFn = func(db *database.DB) error { return nil }
	resolveStaticDirFn = func() string { return "static" }
	mkdirAllFn = func(path string, perm os.FileMode) error { return errors.New("mkdir fail") }
	if _, err := New(config.Config{JWTSecret: "s", CacheType: "memory"}); err == nil || !strings.Contains(err.Error(), "创建 upload 目录失败") {
		t.Fatalf("err=%v", err)
	}

	// redis: userInfoCache 创建失败
	mkdirAllFn = func(path string, perm os.FileMode) error { return nil }
	newRedisUserInfoCacheServiceFn = func(redisURL, host string, port int, password string, dbn int, keyPrefix, lastMsgPrefix string, expireDays, flushIntervalSeconds, localTTLSeconds, timeoutSeconds int) (UserInfoCacheService, error) {
		return nil, errors.New("redis uic fail")
	}
	if _, err := New(config.Config{JWTSecret: "s", CacheType: "redis"}); err == nil {
		t.Fatalf("expected error")
	}

	// redis: chatHistory 创建失败时应关闭 userInfoCache
	stub := &closableUserInfoCacheStub{}
	newRedisUserInfoCacheServiceFn = func(redisURL, host string, port int, password string, dbn int, keyPrefix, lastMsgPrefix string, expireDays, flushIntervalSeconds, localTTLSeconds, timeoutSeconds int) (UserInfoCacheService, error) {
		return stub, nil
	}
	newRedisChatHistoryCacheServiceFn = func(redisURL, host string, port int, password string, dbn int, keyPrefix string, expireDays, flushIntervalSeconds, timeoutSeconds int) (ChatHistoryCacheService, error) {
		return nil, errors.New("redis chc fail")
	}
	if _, err := New(config.Config{JWTSecret: "s", CacheType: "redis"}); err == nil {
		t.Fatalf("expected error")
	}
	if !stub.closed {
		t.Fatalf("expected userInfoCache closed")
	}

	// EnsureDefaults 被 New 调用（INSERT IGNORE x3）
	_ = mock
}

func TestNew_Success_MemoryCacheAndShutdown(t *testing.T) {
	oldOpen := openDBFn
	oldEnsure := ensureSchemaFn
	oldStatic := resolveStaticDirFn
	oldMkdirAll := mkdirAllFn
	t.Cleanup(func() {
		openDBFn = oldOpen
		ensureSchemaFn = oldEnsure
		resolveStaticDirFn = oldStatic
		mkdirAllFn = oldMkdirAll
	})

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	openDBFn = func(cfg config.Config) (*database.DB, error) { return database.Wrap(db, database.MySQLDialect{}), nil }
	ensureSchemaFn = func(db *database.DB) error { return nil }
	resolveStaticDirFn = func() string { return "static" }
	mkdirAllFn = func(path string, perm os.FileMode) error { return nil }

	for i := 0; i < 3; i++ {
		mock.ExpectExec(`INSERT (IGNORE )?INTO system_config`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	application, err := New(config.Config{
		JWTSecret:                      "s",
		TokenExpireHours:               1,
		ServerPort:                     8080,
		UpstreamHTTPTimeoutSeconds:     1,
		CacheType:                      "memory",
		VideoExtractWorkers:            1,
		VideoExtractQueueSize:          1,
		VideoExtractFramePageSz:        1,
		TikTokDownloaderTimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if application.Handler() == nil {
		t.Fatalf("expected handler")
	}

	application.Shutdown(context.Background())
}
