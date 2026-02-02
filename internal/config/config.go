package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config 表示服务运行所需配置。
// 设计目标：与现有 application.yml 的 env 覆盖规则保持一致。
type Config struct {
	ServerPort int

	DBURL      string
	DBUsername string
	DBPassword string

	// UpstreamHTTPTimeoutSeconds 表示调用上游 HTTP 接口的超时时间（秒）。
	// 默认 60 秒；可通过环境变量 UPSTREAM_HTTP_TIMEOUT_SECONDS 覆盖。
	UpstreamHTTPTimeoutSeconds int

	// TikTokDownloaderTimeoutSeconds 表示调用 TikTokDownloader Web API（抖音抓取/下载上游）的超时时间（秒）。
	// 默认与 UpstreamHTTPTimeoutSeconds 一致（两者默认均为 60 秒）；可通过环境变量 TIKTOKDOWNLOADER_TIMEOUT_SECONDS 单独覆盖。
	TikTokDownloaderTimeoutSeconds int

	// RedisURL 支持通过完整连接串配置（例如 Upstash 的 rediss://...）。
	// 优先级高于 REDIS_HOST/REDIS_PORT/REDIS_PASSWORD/REDIS_DB。
	RedisURL string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// RedisTimeoutSeconds 表示 Redis 连接/读写超时（秒）。
	// 默认 15 秒；可通过环境变量 REDIS_TIMEOUT_SECONDS 覆盖。
	RedisTimeoutSeconds int

	AuthAccessCode    string
	JWTSecret         string
	TokenExpireHours  int
	WebSocketFallback string

	CacheType                   string
	CacheRedisKeyPrefix         string
	CacheRedisLastMessagePrefix string
	CacheRedisExpireDays        int
	CacheRedisFlushIntervalSec  int
	CacheRedisLocalTTLSeconds   int

	CacheRedisChatHistoryPrefix     string
	CacheRedisChatHistoryExpireDays int

	ImageServerHost        string
	ImageServerPort        string
	ImageServerUpstreamURL string

	// LspRoot 表示 /lsp/* 静态文件映射的本地根目录。
	// 默认 /lsp（与现网 mtPhoto 返回的 filePath 前缀保持一致），便于在容器/本地开发时重定向到其他目录。
	LspRoot string

	// MtPhoto* 为 mtPhoto 相册系统对接配置（可选；未配置时相关 API 将返回错误）。
	MtPhotoBaseURL       string
	MtPhotoLoginUsername string
	MtPhotoLoginPassword string
	MtPhotoLoginOTP      string

	// TikTokDownloaderBaseURL 为 TikTokDownloader Web API（FastAPI）地址（可选；未配置时抖音相关 API 将返回错误）。
	// 参考知识库：helloagents/wiki/external/tiktokdownloader-web-api.md
	TikTokDownloaderBaseURL string
	// TikTokDownloaderToken 为上游 Web API 的 token Header（默认上游不校验，可为空；如你自行启用校验则配置）。
	TikTokDownloaderToken string
	// DouyinDefaultCookie/DouyinDefaultProxy 为抖音抓取的默认 Cookie/代理（可选；页面传入优先）。
	DouyinDefaultCookie string
	DouyinDefaultProxy  string

	// CookieCloud* 为 CookieCloud（浏览器 Cookie 同步）对接配置（可选；未配置时不启用）。
	// 参考知识库：helloagents/modules/external/cookiecloud.md
	CookieCloudBaseURL    string
	CookieCloudUUID       string
	CookieCloudPassword   string
	CookieCloudCryptoType string
	CookieCloudDomain     string
	// CookieCloudCookieExpireHours controls how long the decrypted CookieCloud cookie header value is cached.
	// Unit: hours. Default 72 hours (3 days).
	CookieCloudCookieExpireHours int

	// 视频抽帧（ffmpeg/ffprobe）配置。
	FFmpegPath              string
	FFprobePath             string
	VideoExtractWorkers     int
	VideoExtractQueueSize   int
	VideoExtractFramePageSz int
}

func Load() (Config, error) {
	cfg := Config{
		ServerPort: getEnvInt("SERVER_PORT", 8080),

		DBURL:      getEnv("DB_URL", "jdbc:mysql://10.10.10.90:3306/hot_img?useSSL=false&serverTimezone=Asia/Shanghai&characterEncoding=utf8&allowPublicKeyRetrieval=true"),
		DBUsername: getEnv("DB_USERNAME", "root"),
		DBPassword: getEnv("DB_PASSWORD", "123456"),

		UpstreamHTTPTimeoutSeconds: getEnvInt("UPSTREAM_HTTP_TIMEOUT_SECONDS", 60),
		TikTokDownloaderTimeoutSeconds: getEnvInt(
			"TIKTOKDOWNLOADER_TIMEOUT_SECONDS",
			0, // 默认后置处理：跟随 UpstreamHTTPTimeoutSeconds
		),

		RedisURL: getEnvOptional2("UPSTASH_REDIS_URL", "REDIS_URL"),

		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnvInt("REDIS_PORT", 6379),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvInt("REDIS_DB", 0),
		RedisTimeoutSeconds: getEnvInt("REDIS_TIMEOUT_SECONDS", 15),

		AuthAccessCode:   getEnv("AUTH_ACCESS_CODE", "Aa305512775."),
		JWTSecret:        getEnv("JWT_SECRET", "your-jwt-secret-key-at-least-256-bits-long-please-change-this-to-random-string"),
		TokenExpireHours: getEnvInt("TOKEN_EXPIRE_HOURS", 24),

		WebSocketFallback: getEnv("WEBSOCKET_UPSTREAM_URL", "ws://localhost:9999"),

		CacheType:                   getEnv("CACHE_TYPE", "memory"),
		CacheRedisKeyPrefix:         getEnv("CACHE_REDIS_PREFIX", "user:info:"),
		CacheRedisLastMessagePrefix: getEnv("CACHE_REDIS_LASTMSG_PREFIX", "user:lastmsg:"),
		CacheRedisExpireDays:        getEnvInt("CACHE_REDIS_EXPIRE_DAYS", 7),
		CacheRedisFlushIntervalSec:  getEnvInt("CACHE_REDIS_FLUSH_INTERVAL_SECONDS", 60),
		CacheRedisLocalTTLSeconds:   getEnvInt("CACHE_REDIS_LOCAL_TTL_SECONDS", 3600),

		CacheRedisChatHistoryPrefix:     getEnv("CACHE_REDIS_CHAT_HISTORY_PREFIX", "user:chathistory:"),
		CacheRedisChatHistoryExpireDays: getEnvInt("CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS", 30),

		ImageServerHost:        getEnv("IMG_SERVER_HOST", "149.88.79.98"),
		ImageServerPort:        getEnv("IMG_SERVER_PORT", "9003"),
		ImageServerUpstreamURL: getEnv("IMG_SERVER_UPSTREAM_URL", "http://v1.chat2019.cn/asmx/method.asmx/getImgServer"),

		LspRoot: getEnv("LSP_ROOT", "/lsp"),

		MtPhotoBaseURL:       getEnv("MTPHOTO_BASE_URL", ""),
		MtPhotoLoginUsername: getEnv("MTPHOTO_LOGIN_USERNAME", ""),
		MtPhotoLoginPassword: getEnv("MTPHOTO_LOGIN_PASSWORD", ""),
		MtPhotoLoginOTP:      getEnv("MTPHOTO_LOGIN_OTP", ""),

		TikTokDownloaderBaseURL: getEnvOptional2("TIKTOKDOWNLOADER_BASE_URL", "TIKTOK_DOWNLOADER_BASE_URL"),
		TikTokDownloaderToken:   getEnvOptional2("TIKTOKDOWNLOADER_TOKEN", "TIKTOK_DOWNLOADER_TOKEN"),
		DouyinDefaultCookie:     getEnvOptional2("DOUYIN_COOKIE", "TIKTOKDOWNLOADER_DOUYIN_COOKIE"),
		DouyinDefaultProxy:      getEnvOptional2("DOUYIN_PROXY", "TIKTOKDOWNLOADER_DOUYIN_PROXY"),

		CookieCloudBaseURL:    getEnvOptional2("COOKIECLOUD_BASE_URL", "COOKIE_CLOUD_BASE_URL"),
		CookieCloudUUID:       getEnvOptional2("COOKIECLOUD_UUID", "COOKIE_CLOUD_UUID"),
		CookieCloudPassword:   getEnvOptional2("COOKIECLOUD_PASSWORD", "COOKIE_CLOUD_PASSWORD"),
		CookieCloudCryptoType: getEnvOptional2("COOKIECLOUD_CRYPTO_TYPE", "COOKIE_CLOUD_CRYPTO_TYPE"),
		CookieCloudDomain:     getEnvOptional2("COOKIECLOUD_DOMAIN", "COOKIE_CLOUD_DOMAIN"),
		CookieCloudCookieExpireHours: getEnvIntOptional2(
			"COOKIECLOUD_COOKIE_EXPIRE_HOURS",
			"COOKIE_CLOUD_COOKIE_EXPIRE_HOURS",
			72,
		),

		FFmpegPath:              getEnv("FFMPEG_PATH", "ffmpeg"),
		FFprobePath:             getEnv("FFPROBE_PATH", "ffprobe"),
		VideoExtractWorkers:     getEnvInt("VIDEO_EXTRACT_WORKERS", 1),
		VideoExtractQueueSize:   getEnvInt("VIDEO_EXTRACT_QUEUE_SIZE", 32),
		VideoExtractFramePageSz: getEnvInt("VIDEO_EXTRACT_FRAME_PAGE_SIZE", 120),
	}

	if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
		return Config{}, fmt.Errorf("SERVER_PORT 非法: %d", cfg.ServerPort)
	}

	if cfg.TokenExpireHours <= 0 {
		return Config{}, fmt.Errorf("TOKEN_EXPIRE_HOURS 非法: %d", cfg.TokenExpireHours)
	}

	switch cfg.CacheType {
	case "memory", "redis":
	default:
		return Config{}, fmt.Errorf("CACHE_TYPE 非法: %s（仅支持 memory/redis）", cfg.CacheType)
	}

	if cfg.CacheRedisFlushIntervalSec <= 0 {
		cfg.CacheRedisFlushIntervalSec = 60
	}

	if cfg.CacheRedisLocalTTLSeconds <= 0 {
		cfg.CacheRedisLocalTTLSeconds = 3600
	}

	if cfg.CacheRedisChatHistoryExpireDays <= 0 {
		cfg.CacheRedisChatHistoryExpireDays = 30
	}

	if cfg.UpstreamHTTPTimeoutSeconds <= 0 {
		cfg.UpstreamHTTPTimeoutSeconds = 60
	}

	if cfg.TikTokDownloaderTimeoutSeconds <= 0 {
		cfg.TikTokDownloaderTimeoutSeconds = cfg.UpstreamHTTPTimeoutSeconds
	}

	if cfg.RedisTimeoutSeconds <= 0 {
		cfg.RedisTimeoutSeconds = 15
	}

	if cfg.VideoExtractWorkers <= 0 {
		cfg.VideoExtractWorkers = 1
	}
	if cfg.VideoExtractQueueSize <= 0 {
		cfg.VideoExtractQueueSize = 32
	}
	if cfg.VideoExtractFramePageSz <= 0 {
		cfg.VideoExtractFramePageSz = 120
	}

	if v := strings.TrimSpace(cfg.TikTokDownloaderBaseURL); v != "" {
		// 轻量校验，避免误配导致运行时难以排查
		if !(strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")) {
			return Config{}, fmt.Errorf("TIKTOKDOWNLOADER_BASE_URL 非法: %s（需以 http(s):// 开头）", v)
		}
	}

	if v := strings.TrimSpace(cfg.CookieCloudBaseURL); v != "" {
		if !(strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")) {
			return Config{}, fmt.Errorf("COOKIECLOUD_BASE_URL 非法: %s（需以 http(s):// 开头）", v)
		}
		if strings.TrimSpace(cfg.CookieCloudUUID) == "" {
			return Config{}, fmt.Errorf("COOKIECLOUD_UUID 为空（已配置 COOKIECLOUD_BASE_URL 时必须提供）")
		}
		if strings.TrimSpace(cfg.CookieCloudPassword) == "" {
			return Config{}, fmt.Errorf("COOKIECLOUD_PASSWORD 为空（已配置 COOKIECLOUD_BASE_URL 时必须提供）")
		}
		if v := strings.TrimSpace(cfg.CookieCloudCryptoType); v != "" {
			switch v {
			case "legacy", "aes-128-cbc-fixed":
			default:
				return Config{}, fmt.Errorf("COOKIECLOUD_CRYPTO_TYPE 非法: %s（仅支持 legacy/aes-128-cbc-fixed 或留空）", v)
			}
		}
	}

	if cfg.CookieCloudCookieExpireHours <= 0 {
		cfg.CookieCloudCookieExpireHours = 72
	}

	return cfg, nil
}

func (c Config) ListenAddr() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return defaultValue
	}
	return val
}

func getEnvOptional2(key1, key2 string) string {
	if v := strings.TrimSpace(os.Getenv(key1)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(key2))
}

func getEnvInt(key string, defaultValue int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvIntOptional2(key1, key2 string, defaultValue int) int {
	if v := strings.TrimSpace(os.Getenv(key1)); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
		return defaultValue
	}
	return getEnvInt(key2, defaultValue)
}

// ParseJDBCURL parses a JDBC-like DB URL (supports "jdbc:" prefix) and returns its components.
// This function does not perform driver selection; callers should use the returned scheme.
//
// Supported schemes:
// - mysql
// - postgres
// - postgresql
//
// Examples:
// - jdbc:mysql://127.0.0.1:3306/db?serverTimezone=Asia%2FShanghai
// - mysql://127.0.0.1:3306/db
// - postgres://127.0.0.1:5432/db?sslmode=disable
// - jdbc:postgresql://127.0.0.1:5432/db
func ParseJDBCURL(jdbcURL string) (scheme string, host string, port int, database string, params url.Values, err error) {
	raw := strings.TrimSpace(jdbcURL)
	if raw == "" {
		return "", "", 0, "", nil, fmt.Errorf("DB_URL 为空")
	}

	// Support jdbc:<scheme>://... and <scheme>://...
	if strings.HasPrefix(raw, "jdbc:") {
		raw = strings.TrimPrefix(raw, "jdbc:")
	}

	u, parseErr := url.Parse(raw)
	if parseErr != nil {
		return "", "", 0, "", nil, fmt.Errorf("解析 DB_URL 失败: %w", parseErr)
	}

	scheme = strings.ToLower(strings.TrimSpace(u.Scheme))
	switch scheme {
	case "mysql", "postgres", "postgresql":
	default:
		return "", "", 0, "", nil, fmt.Errorf("DB_URL scheme 非法: %s", u.Scheme)
	}

	host = u.Hostname()
	if host == "" {
		return "", "", 0, "", nil, fmt.Errorf("DB_URL 缺少 host")
	}

	if u.Port() == "" {
		switch scheme {
		case "mysql":
			port = 3306
		default:
			port = 5432
		}
	} else {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return "", "", 0, "", nil, fmt.Errorf("DB_URL port 非法: %w", err)
		}
	}

	database = strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return "", "", 0, "", nil, fmt.Errorf("DB_URL 缺少数据库名")
	}

	return scheme, host, port, database, u.Query(), nil
}

// ParseJDBCMySQLURL parses JDBC MySQL URL (jdbc:mysql://host:port/db?x=y) into host/port/db/params.
// Deprecated: prefer ParseJDBCURL and use the returned scheme for dialect selection.
func ParseJDBCMySQLURL(jdbcURL string) (host string, port int, database string, params url.Values, err error) {
	scheme, host, port, database, params, err := ParseJDBCURL(jdbcURL)
	if err != nil {
		return "", 0, "", nil, err
	}
	if scheme != "mysql" {
		return "", 0, "", nil, fmt.Errorf("DB_URL scheme 非法: %s", scheme)
	}
	return host, port, database, params, nil
}
