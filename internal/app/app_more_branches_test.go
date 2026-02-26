package app

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
	"liao/internal/database"
)

type closableChatHistoryCacheStub struct {
	closed bool
}

func (s *closableChatHistoryCacheStub) SaveMessages(context.Context, string, []map[string]any) {}
func (s *closableChatHistoryCacheStub) GetMessages(context.Context, string, string, int) ([]map[string]any, error) {
	return nil, nil
}
func (s *closableChatHistoryCacheStub) Close() error { s.closed = true; return nil }

type closableCookieProviderStub struct {
	closed bool
}

func (s *closableCookieProviderStub) GetCookie(context.Context) (string, error) { return "", nil }
func (s *closableCookieProviderStub) Close() error                                { s.closed = true; return nil }

func TestNew_CookieCloudProviderError_ClosesRedisCaches(t *testing.T) {
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

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	for i := 0; i < 4; i++ {
		mock.ExpectExec(`INSERT (IGNORE )?INTO system_config`).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	openDBFn = func(cfg config.Config) (*database.DB, error) { return database.Wrap(db, database.MySQLDialect{}), nil }
	ensureSchemaFn = func(db *database.DB) error { return nil }
	resolveStaticDirFn = func() string { return "static" }
	mkdirAllFn = func(path string, perm os.FileMode) error { return nil }

	uic := &closableUserInfoCacheStub{}
	chc := &closableChatHistoryCacheStub{}
	newRedisUserInfoCacheServiceFn = func(redisURL, host string, port int, password string, dbn int, keyPrefix, lastMsgPrefix string, expireDays, flushIntervalSeconds, localTTLSeconds, timeoutSeconds int) (UserInfoCacheService, error) {
		return uic, nil
	}
	newRedisChatHistoryCacheServiceFn = func(redisURL, host string, port int, password string, dbn int, keyPrefix string, expireDays, flushIntervalSeconds, timeoutSeconds int) (ChatHistoryCacheService, error) {
		return chc, nil
	}

	_, err := New(config.Config{
		JWTSecret:          "s",
		CacheType:          "redis",
		CookieCloudBaseURL: "http://cookie-cloud.example",
		// intentionally missing UUID/PASSWORD -> provider creation error
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !uic.closed || !chc.closed {
		t.Fatalf("expected redis cache closers called: uic=%v chc=%v", uic.closed, chc.closed)
	}
}

func TestNew_CookieCloudProviderSuccess_SetsProvider(t *testing.T) {
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
	for i := 0; i < 4; i++ {
		mock.ExpectExec(`INSERT (IGNORE )?INTO system_config`).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	openDBFn = func(cfg config.Config) (*database.DB, error) { return database.Wrap(db, database.MySQLDialect{}), nil }
	ensureSchemaFn = func(db *database.DB) error { return nil }
	resolveStaticDirFn = func() string { return "static" }
	mkdirAllFn = func(path string, perm os.FileMode) error { return nil }

	app, err := New(config.Config{
		JWTSecret:          "s",
		CacheType:          "memory",
		CookieCloudBaseURL: "http://cookie-cloud.example",
		CookieCloudUUID:    "uuid",
		CookieCloudPassword:"pwd",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if app.douyinDownloader == nil || app.douyinDownloader.cookieProvider == nil {
		t.Fatalf("cookie provider should be set")
	}
}

func TestApp_Shutdown_ClosesCookieProvider(t *testing.T) {
	provider := &closableCookieProviderStub{}
	a := &App{
		douyinDownloader: &DouyinDownloaderService{cookieProvider: provider},
	}
	a.Shutdown(context.Background())
	if !provider.closed {
		t.Fatalf("cookie provider should be closed")
	}
}

func TestOpenDB_PostgresAdditionalBranches(t *testing.T) {
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

	var gotDSN string
	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) {
		gotDSN = dsn
		return db, nil
	}

	_, err = openDB(config.Config{
		DBURL:      "jdbc:postgres://localhost:5432/hot_img?useSSL=true",
		DBUsername: "u",
		DBPassword: "",
	})
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if !strings.Contains(gotDSN, "sslmode=require") {
		t.Fatalf("dsn=%q", gotDSN)
	}
	if !strings.Contains(gotDSN, "u@") {
		t.Fatalf("dsn should contain username without password: %q", gotDSN)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOpenDB_PostgresNoUseSSL_DefaultDisable(t *testing.T) {
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

	var gotDSN string
	sqlOpenFn = func(driverName, dsn string) (*sql.DB, error) {
		gotDSN = dsn
		return db, nil
	}

	_, err = openDB(config.Config{DBURL: "jdbc:postgres://localhost:5432/hot_img"})
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	if !strings.Contains(gotDSN, "sslmode=disable") {
		t.Fatalf("dsn=%q", gotDSN)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOpenDB_DialectErrorAndGetQueryParamCI_EmptyKey(t *testing.T) {
	if _, err := openDB(config.Config{DBURL: "jdbc:sqlite://localhost:3306/hot_img"}); err == nil {
		t.Fatalf("expected dialect error")
	}

	if got := getQueryParamCI(map[string][]string{"k": {"v"}}, "   "); got != "" {
		t.Fatalf("got=%q", got)
	}
}
