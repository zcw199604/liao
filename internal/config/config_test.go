package config

import (
	"net/url"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("X_GETENV", "")
	if got := getEnv("X_GETENV", "d"); got != "d" {
		t.Fatalf("got %q, want %q", got, "d")
	}

	t.Setenv("X_GETENV", "  ")
	if got := getEnv("X_GETENV", "d"); got != "d" {
		t.Fatalf("got %q, want %q", got, "d")
	}

	t.Setenv("X_GETENV", " v ")
	// getEnv 只用于空值判定，不会 trim 返回值
	if got := getEnv("X_GETENV", "d"); got != " v " {
		t.Fatalf("got %q, want %q", got, " v ")
	}
}

func TestGetEnvOptional2(t *testing.T) {
	t.Setenv("X_OPT1", "")
	t.Setenv("X_OPT2", "  b  ")
	if got := getEnvOptional2("X_OPT1", "X_OPT2"); got != "b" {
		t.Fatalf("got %q, want %q", got, "b")
	}

	t.Setenv("X_OPT1", "  a  ")
	t.Setenv("X_OPT2", "b")
	if got := getEnvOptional2("X_OPT1", "X_OPT2"); got != "a" {
		t.Fatalf("got %q, want %q", got, "a")
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("X_INT", "")
	if got := getEnvInt("X_INT", 7); got != 7 {
		t.Fatalf("got %d, want %d", got, 7)
	}

	t.Setenv("X_INT", "bad")
	if got := getEnvInt("X_INT", 7); got != 7 {
		t.Fatalf("got %d, want %d", got, 7)
	}

	t.Setenv("X_INT", " 12 ")
	if got := getEnvInt("X_INT", 7); got != 12 {
		t.Fatalf("got %d, want %d", got, 12)
	}
}

func TestGetEnvIntOptional2(t *testing.T) {
	t.Setenv("X_INT1", "")
	t.Setenv("X_INT2", "")
	if got := getEnvIntOptional2("X_INT1", "X_INT2", 7); got != 7 {
		t.Fatalf("got %d, want %d", got, 7)
	}

	t.Setenv("X_INT1", "bad")
	t.Setenv("X_INT2", "12")
	if got := getEnvIntOptional2("X_INT1", "X_INT2", 7); got != 7 {
		t.Fatalf("got %d, want %d", got, 7)
	}

	t.Setenv("X_INT1", " 11 ")
	t.Setenv("X_INT2", "12")
	if got := getEnvIntOptional2("X_INT1", "X_INT2", 7); got != 11 {
		t.Fatalf("got %d, want %d", got, 11)
	}

	t.Setenv("X_INT1", "")
	t.Setenv("X_INT2", " 12 ")
	if got := getEnvIntOptional2("X_INT1", "X_INT2", 7); got != 12 {
		t.Fatalf("got %d, want %d", got, 12)
	}
}

func TestParseJDBCMySQLURL(t *testing.T) {
	if _, _, _, _, err := ParseJDBCMySQLURL(""); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, _, _, err := ParseJDBCMySQLURL("://bad"); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, _, _, err := ParseJDBCMySQLURL("postgres://localhost:5432/db"); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, _, _, err := ParseJDBCMySQLURL("mysql://:3306/db"); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, _, _, err := ParseJDBCMySQLURL("mysql://localhost:bad/db"); err == nil {
		t.Fatalf("expected error")
	}

	// url.Parse 可能接受非常大的数字端口，但 strconv.Atoi 会溢出报错
	if _, _, _, _, err := ParseJDBCMySQLURL("mysql://localhost:999999999999999999999999/db"); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, _, _, err := ParseJDBCMySQLURL("mysql://localhost:3306/"); err == nil {
		t.Fatalf("expected error")
	}

	host, port, db, params, err := ParseJDBCMySQLURL("mysql://localhost/hot_img?useSSL=false")
	if err != nil {
		t.Fatalf("ParseJDBCMySQLURL: %v", err)
	}
	if host != "localhost" || port != 3306 || db != "hot_img" {
		t.Fatalf("host=%q port=%d db=%q", host, port, db)
	}
	if params.Get("useSSL") != "false" {
		t.Fatalf("params=%v", params)
	}

	host, port, db, params, err = ParseJDBCMySQLURL("jdbc:mysql://127.0.0.1:3307/hot_img?serverTimezone=Asia%2FShanghai")
	if err != nil {
		t.Fatalf("ParseJDBCMySQLURL: %v", err)
	}
	if host != "127.0.0.1" || port != 3307 || db != "hot_img" {
		t.Fatalf("host=%q port=%d db=%q", host, port, db)
	}
	if params.Get("serverTimezone") != "Asia/Shanghai" {
		t.Fatalf("params=%v", params)
	}
}

func TestParseJDBCURL_PostgresDefaultsAndJDBC(t *testing.T) {
	scheme, host, port, db, params, err := ParseJDBCURL("postgres://localhost/mydb?sslmode=disable")
	if err != nil {
		t.Fatalf("ParseJDBCURL: %v", err)
	}
	if scheme != "postgres" || host != "localhost" || port != 5432 || db != "mydb" {
		t.Fatalf("scheme=%q host=%q port=%d db=%q", scheme, host, port, db)
	}
	if params.Get("sslmode") != "disable" {
		t.Fatalf("params=%v", params)
	}

	scheme, host, port, db, params, err = ParseJDBCURL("jdbc:postgresql://127.0.0.1:5433/mydb")
	if err != nil {
		t.Fatalf("ParseJDBCURL: %v", err)
	}
	if scheme != "postgresql" || host != "127.0.0.1" || port != 5433 || db != "mydb" {
		t.Fatalf("scheme=%q host=%q port=%d db=%q", scheme, host, port, db)
	}
	if params == nil {
		t.Fatalf("params=nil")
	}
}

func TestLoad_SetsDefaultsAndValidates(t *testing.T) {
	t.Setenv("SERVER_PORT", "8081")
	t.Setenv("TOKEN_EXPIRE_HOURS", "1")
	t.Setenv("CACHE_TYPE", "redis")
	t.Setenv("CACHE_REDIS_FLUSH_INTERVAL_SECONDS", "0")   // 触发默认值回填
	t.Setenv("CACHE_REDIS_LOCAL_TTL_SECONDS", "0")        // 触发默认值回填
	t.Setenv("CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS", "0") // 触发默认值回填
	t.Setenv("UPSTREAM_HTTP_TIMEOUT_SECONDS", "0")        // 触发默认值回填
	t.Setenv("TIKTOKDOWNLOADER_TIMEOUT_SECONDS", "0")     // 触发默认值回填
	t.Setenv("REDIS_TIMEOUT_SECONDS", "0")                // 触发默认值回填
	t.Setenv("FFMPEG_PATH", "   ")
	t.Setenv("FFPROBE_PATH", "   ")
	t.Setenv("VIDEO_EXTRACT_WORKERS", "0")
	t.Setenv("VIDEO_EXTRACT_QUEUE_SIZE", "0")
	t.Setenv("VIDEO_EXTRACT_FRAME_PAGE_SIZE", "0")
	t.Setenv("TIKTOKDOWNLOADER_BASE_URL", "") // 不触发校验

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.ServerPort != 8081 {
		t.Fatalf("ServerPort=%d", cfg.ServerPort)
	}
	if cfg.ListenAddr() != ":8081" {
		t.Fatalf("ListenAddr=%q", cfg.ListenAddr())
	}
	if cfg.CacheType != "redis" {
		t.Fatalf("CacheType=%q", cfg.CacheType)
	}
	if cfg.CacheRedisFlushIntervalSec != 60 {
		t.Fatalf("CacheRedisFlushIntervalSec=%d", cfg.CacheRedisFlushIntervalSec)
	}
	if cfg.CacheRedisLocalTTLSeconds != 3600 {
		t.Fatalf("CacheRedisLocalTTLSeconds=%d", cfg.CacheRedisLocalTTLSeconds)
	}
	if cfg.CacheRedisChatHistoryExpireDays != 30 {
		t.Fatalf("CacheRedisChatHistoryExpireDays=%d", cfg.CacheRedisChatHistoryExpireDays)
	}
	if cfg.UpstreamHTTPTimeoutSeconds != 60 {
		t.Fatalf("UpstreamHTTPTimeoutSeconds=%d", cfg.UpstreamHTTPTimeoutSeconds)
	}
	if cfg.TikTokDownloaderTimeoutSeconds != cfg.UpstreamHTTPTimeoutSeconds {
		t.Fatalf("TikTokDownloaderTimeoutSeconds=%d", cfg.TikTokDownloaderTimeoutSeconds)
	}
	if cfg.RedisTimeoutSeconds != 15 {
		t.Fatalf("RedisTimeoutSeconds=%d", cfg.RedisTimeoutSeconds)
	}
	if cfg.FFmpegPath != "ffmpeg" || cfg.FFprobePath != "ffprobe" {
		t.Fatalf("ffmpeg=%q ffprobe=%q", cfg.FFmpegPath, cfg.FFprobePath)
	}
	if cfg.VideoExtractWorkers != 1 || cfg.VideoExtractQueueSize != 32 || cfg.VideoExtractFramePageSz != 120 {
		t.Fatalf("videoExtract defaults=%+v", cfg)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	t.Setenv("SERVER_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("TOKEN_EXPIRE_HOURS", "0")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	t.Setenv("TOKEN_EXPIRE_HOURS", "1")
	t.Setenv("CACHE_TYPE", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	// BASE_URL 配置后需要以 http(s):// 开头
	t.Setenv("CACHE_TYPE", "memory")
	t.Setenv("TIKTOKDOWNLOADER_BASE_URL", "ftp://x")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	// CookieCloud BASE_URL 配置后需要以 http(s):// 开头
	t.Setenv("TIKTOKDOWNLOADER_BASE_URL", "")
	t.Setenv("COOKIECLOUD_BASE_URL", "ftp://x")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	// CookieCloud 配置 BASE_URL 时必须配置 UUID/PASSWORD
	t.Setenv("COOKIECLOUD_BASE_URL", "http://127.0.0.1:8088")
	t.Setenv("COOKIECLOUD_UUID", "")
	t.Setenv("COOKIECLOUD_PASSWORD", "p")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	t.Setenv("COOKIECLOUD_UUID", "u")
	t.Setenv("COOKIECLOUD_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}

	// CookieCloud crypto_type 只支持 legacy/aes-128-cbc-fixed 或留空
	t.Setenv("COOKIECLOUD_PASSWORD", "p")
	t.Setenv("COOKIECLOUD_CRYPTO_TYPE", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseJDBCMySQLURL_ReturnsParams(t *testing.T) {
	host, port, db, params, err := ParseJDBCMySQLURL("mysql://localhost:3306/hot_img?a=1&b=2")
	if err != nil {
		t.Fatalf("ParseJDBCMySQLURL: %v", err)
	}
	if host != "localhost" || port != 3306 || db != "hot_img" {
		t.Fatalf("host=%q port=%d db=%q", host, port, db)
	}
	want := url.Values{"a": []string{"1"}, "b": []string{"2"}}
	if params.Encode() != want.Encode() {
		t.Fatalf("params=%v want=%v", params, want)
	}
}
