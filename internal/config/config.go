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

	// RedisURL 支持通过完整连接串配置（例如 Upstash 的 rediss://...）。
	// 优先级高于 REDIS_HOST/REDIS_PORT/REDIS_PASSWORD/REDIS_DB。
	RedisURL string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

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

		RedisURL: getEnvOptional2("UPSTASH_REDIS_URL", "REDIS_URL"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

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

		FFmpegPath:              getEnv("FFMPEG_PATH", "ffmpeg"),
		FFprobePath:             getEnv("FFPROBE_PATH", "ffprobe"),
		VideoExtractWorkers:     getEnvInt("VIDEO_EXTRACT_WORKERS", 1),
		VideoExtractQueueSize:   getEnvInt("VIDEO_EXTRACT_QUEUE_SIZE", 32),
		VideoExtractFramePageSz: getEnvInt("VIDEO_EXTRACT_FRAME_PAGE_SIZE", 120),
	}

	if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
		return Config{}, fmt.Errorf("SERVER_PORT 非法: %d", cfg.ServerPort)
	}

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, fmt.Errorf("JWT_SECRET 不能为空")
	}

	if cfg.TokenExpireHours <= 0 {
		return Config{}, fmt.Errorf("TOKEN_EXPIRE_HOURS 非法: %d", cfg.TokenExpireHours)
	}

	if strings.TrimSpace(cfg.CacheType) == "" {
		cfg.CacheType = "memory"
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

	if strings.TrimSpace(cfg.FFmpegPath) == "" {
		cfg.FFmpegPath = "ffmpeg"
	}
	if strings.TrimSpace(cfg.FFprobePath) == "" {
		cfg.FFprobePath = "ffprobe"
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

// ParseJDBCMySQLURL 将 JDBC MySQL URL（jdbc:mysql://host:port/db?x=y）解析为 host/port/db/params。
func ParseJDBCMySQLURL(jdbcURL string) (host string, port int, database string, params url.Values, err error) {
	raw := strings.TrimSpace(jdbcURL)
	if raw == "" {
		return "", 0, "", nil, fmt.Errorf("DB_URL 为空")
	}

	// 支持 jdbc:mysql://... 与 mysql://... 两种形式（以便本地灵活配置）。
	if strings.HasPrefix(raw, "jdbc:") {
		raw = strings.TrimPrefix(raw, "jdbc:")
	}

	u, parseErr := url.Parse(raw)
	if parseErr != nil {
		return "", 0, "", nil, fmt.Errorf("解析 DB_URL 失败: %w", parseErr)
	}

	if u.Scheme != "mysql" {
		return "", 0, "", nil, fmt.Errorf("DB_URL scheme 非法: %s", u.Scheme)
	}

	host = u.Hostname()
	if host == "" {
		return "", 0, "", nil, fmt.Errorf("DB_URL 缺少 host")
	}

	if u.Port() == "" {
		port = 3306
	} else {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return "", 0, "", nil, fmt.Errorf("DB_URL port 非法: %w", err)
		}
	}

	database = strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return "", 0, "", nil, fmt.Errorf("DB_URL 缺少数据库名")
	}

	return host, port, database, u.Query(), nil
}
