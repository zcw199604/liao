package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"liao/internal/config"
	"liao/internal/database"
)

var (
	openDBFn           = openDB
	ensureSchemaFn     = ensureSchema
	resolveStaticDirFn = resolveStaticDir
	mkdirAllFn         = os.MkdirAll
	sqlOpenFn          = sql.Open

	newRedisUserInfoCacheServiceFn = func(
		redisURL string,
		host string,
		port int,
		password string,
		db int,
		keyPrefix string,
		lastMessagePrefix string,
		expireDays int,
		flushIntervalSeconds int,
		localTTLSeconds int,
		timeoutSeconds int,
	) (UserInfoCacheService, error) {
		return NewRedisUserInfoCacheService(
			redisURL,
			host,
			port,
			password,
			db,
			keyPrefix,
			lastMessagePrefix,
			expireDays,
			flushIntervalSeconds,
			localTTLSeconds,
			timeoutSeconds,
		)
	}
	newRedisChatHistoryCacheServiceFn = func(
		redisURL string,
		host string,
		port int,
		password string,
		db int,
		keyPrefix string,
		expireDays int,
		flushIntervalSeconds int,
		timeoutSeconds int,
	) (ChatHistoryCacheService, error) {
		return NewRedisChatHistoryCacheService(
			redisURL,
			host,
			port,
			password,
			db,
			keyPrefix,
			expireDays,
			flushIntervalSeconds,
			timeoutSeconds,
		)
	}
)

// App 负责组装依赖并提供 HTTP Handler。
type App struct {
	cfg config.Config
	db  *database.DB

	httpClient *http.Client
	jwt        *JWTService

	systemConfig      *SystemConfigService
	imagePortResolver *ImagePortResolver

	identityService  *IdentityService
	favoriteService  *FavoriteService
	douyinFavorite   *DouyinFavoriteService
	fileStorage      *FileStorageService
	imageServer      *ImageServerService
	imageCache       *ImageCacheService
	imageHash        *ImageHashService
	mediaUpload      *MediaUploadService
	douyinDownloader *DouyinDownloaderService
	mtPhoto          *MtPhotoService
	videoExtract     *VideoExtractService
	userInfoCache    UserInfoCacheService
	chatHistoryCache ChatHistoryCacheService
	forceoutManager  *ForceoutManager
	wsManager        *UpstreamWebSocketManager

	staticDir string
	handler   http.Handler
}

func New(cfg config.Config) (*App, error) {
	db, err := openDBFn(cfg)
	if err != nil {
		return nil, err
	}

	if err := ensureSchemaFn(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	staticDir := resolveStaticDirFn()
	if err := mkdirAllFn("upload", 0o755); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("创建 upload 目录失败: %w", err)
	}

	var userInfoCache UserInfoCacheService
	var chatHistoryCache ChatHistoryCacheService
	switch cfg.CacheType {
	case "redis":
		userInfoCache, err = newRedisUserInfoCacheServiceFn(
			cfg.RedisURL,
			cfg.RedisHost,
			cfg.RedisPort,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.CacheRedisKeyPrefix,
			cfg.CacheRedisLastMessagePrefix,
			cfg.CacheRedisExpireDays,
			cfg.CacheRedisFlushIntervalSec,
			cfg.CacheRedisLocalTTLSeconds,
			cfg.RedisTimeoutSeconds,
		)
		if err != nil {
			_ = db.Close()
			return nil, err
		}

		chatHistoryCache, err = newRedisChatHistoryCacheServiceFn(
			cfg.RedisURL,
			cfg.RedisHost,
			cfg.RedisPort,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.CacheRedisChatHistoryPrefix,
			cfg.CacheRedisChatHistoryExpireDays,
			cfg.CacheRedisFlushIntervalSec,
			cfg.RedisTimeoutSeconds,
		)
		if err != nil {
			if closer, ok := userInfoCache.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
			_ = db.Close()
			return nil, err
		}
	default:
		userInfoCache = NewMemoryUserInfoCacheService()
	}

	application := &App{
		cfg:              cfg,
		db:               db,
		httpClient:       &http.Client{Timeout: time.Duration(cfg.UpstreamHTTPTimeoutSeconds) * time.Second},
		jwt:              NewJWTService(cfg.JWTSecret, cfg.TokenExpireHours),
		identityService:  NewIdentityService(db),
		favoriteService:  NewFavoriteService(db),
		douyinFavorite:   NewDouyinFavoriteService(db),
		fileStorage:      NewFileStorageService(db),
		imageServer:      NewImageServerService(cfg.ImageServerHost, cfg.ImageServerPort),
		imageCache:       NewImageCacheService(),
		imageHash:        NewImageHashService(db),
		userInfoCache:    userInfoCache,
		chatHistoryCache: chatHistoryCache,
		forceoutManager:  NewForceoutManager(),
		staticDir:        staticDir,
	}
	application.systemConfig = NewSystemConfigService(db)
	application.imagePortResolver = NewImagePortResolver(application.httpClient)
	_ = application.systemConfig.EnsureDefaults(context.Background())
	application.wsManager = NewUpstreamWebSocketManager(application.httpClient, cfg.WebSocketFallback, application.forceoutManager, application.userInfoCache, application.chatHistoryCache)
	application.mediaUpload = NewMediaUploadService(db, cfg.ServerPort, application.fileStorage, application.imageServer, application.httpClient)
	application.douyinDownloader = NewDouyinDownloaderService(cfg.TikTokDownloaderBaseURL, cfg.TikTokDownloaderToken, cfg.DouyinDefaultCookie, cfg.DouyinDefaultProxy, time.Duration(cfg.TikTokDownloaderTimeoutSeconds)*time.Second)
	if strings.TrimSpace(cfg.CookieCloudBaseURL) != "" {
		provider, err := NewDouyinCookieCloudProvider(cfg, application.httpClient)
		if err != nil {
			if closer, ok := userInfoCache.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
			if closer, ok := chatHistoryCache.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
			_ = db.Close()
			return nil, err
		}
		application.douyinDownloader.SetCookieProvider(provider)
	}
	application.mtPhoto = NewMtPhotoService(cfg.MtPhotoBaseURL, cfg.MtPhotoLoginUsername, cfg.MtPhotoLoginPassword, cfg.MtPhotoLoginOTP, cfg.LspRoot, application.httpClient)
	application.videoExtract = NewVideoExtractService(db, cfg, application.fileStorage, application.mtPhoto)

	application.handler = application.buildRouter()
	return application, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Shutdown(ctx context.Context) {
	if a.wsManager != nil {
		a.wsManager.CloseAllConnections()
	}
	if a.videoExtract != nil {
		a.videoExtract.Shutdown()
	}
	if a.douyinDownloader != nil {
		if closer, ok := a.douyinDownloader.cookieProvider.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	if closer, ok := a.userInfoCache.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	if closer, ok := a.chatHistoryCache.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			slog.Error("关闭数据库失败", "error", err)
		}
	}
}

func openDB(cfg config.Config) (*database.DB, error) {
	scheme, host, port, databaseName, params, err := config.ParseJDBCURL(cfg.DBURL)
	if err != nil {
		return nil, err
	}

	d, err := database.DialectFromScheme(scheme)
	if err != nil {
		return nil, err
	}

	s := strings.ToLower(strings.TrimSpace(scheme))
	driverName := d.DriverName()
	dsn := ""
	switch s {
	case "mysql":
		loc := getQueryParamCI(params, "serverTimezone")
		if loc == "" {
			loc = "Local"
		}

		// 说明：parseTime 与 loc 对应 Java 侧 DATETIME/LocalDateTime 的常用用法。
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s&charset=utf8mb4,utf8&collation=utf8mb4_unicode_ci&timeout=15s&readTimeout=15s&writeTimeout=15s",
			cfg.DBUsername,
			cfg.DBPassword,
			host,
			port,
			databaseName,
			urlQueryEscape(loc),
		)
	case "postgres", "postgresql":
		u := &url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", host, port),
			Path:   "/" + databaseName,
		}
		if strings.TrimSpace(cfg.DBUsername) != "" {
			if cfg.DBPassword != "" {
				u.User = url.UserPassword(cfg.DBUsername, cfg.DBPassword)
			} else {
				u.User = url.User(cfg.DBUsername)
			}
		}

		// Best-effort compatibility: allow users to switch DB by only changing DB_URL scheme,
		// while keeping some legacy MySQL query params.
		q := filterPostgresParams(params)

		// Keep local dev painless; production can override via DB_URL.
		if strings.TrimSpace(getQueryParamCI(q, "sslmode")) == "" {
			if useSSL := strings.TrimSpace(getQueryParamCI(params, "useSSL")); useSSL != "" {
				if isTruthy(useSSL) {
					q.Set("sslmode", "require")
				} else {
					q.Set("sslmode", "disable")
				}
			} else {
				q.Set("sslmode", "disable")
			}
		}

		// MySQL commonly uses serverTimezone=Asia/Shanghai. For Postgres, map it to timezone
		// (a run-time parameter) so "CURRENT_TIMESTAMP" and timestamptz formatting are consistent.
		if strings.TrimSpace(getQueryParamCI(q, "timezone")) == "" {
			if tz := strings.TrimSpace(getQueryParamCI(params, "serverTimezone")); tz != "" {
				q.Set("timezone", tz)
			}
		}

		u.RawQuery = q.Encode()
		dsn = u.String()
	default:
		return nil, fmt.Errorf("unsupported DB_URL scheme: %s", scheme)
	}

	sqlDB, err := sqlOpenFn(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return database.Wrap(sqlDB, d), nil
}

func urlQueryEscape(input string) string {
	// 避免在 config 包引入额外依赖导致循环引用。
	// 这里仅做最小处理：Go MySQL driver 要求 loc 值是 URL 编码。
	// 常见值如 Asia/Shanghai，需要将 / 转义为 %2F。
	replacer := strings.NewReplacer("/", "%2F", " ", "%20")
	return replacer.Replace(input)
}

func getQueryParamCI(values url.Values, key string) string {
	if values == nil {
		return ""
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if v := strings.TrimSpace(values.Get(key)); v != "" {
		return v
	}
	for k, vs := range values {
		if !strings.EqualFold(k, key) {
			continue
		}
		for _, v := range vs {
			if vv := strings.TrimSpace(v); vv != "" {
				return vv
			}
		}
		if len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
	}
	return ""
}

func filterPostgresParams(params url.Values) url.Values {
	out := url.Values{}
	for k, vs := range params {
		// Drop common MySQL-only JDBC parameters to avoid "unrecognized configuration parameter"
		// errors when switching the scheme to postgres/postgresql.
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "usessl",
			"servertimezone",
			"characterencoding",
			"allowpublickeyretrieval",
			"useunicode",
			"usejdbccomplianttimezoneshift",
			"uselegacydatetimecode":
			continue
		}

		copied := make([]string, len(vs))
		copy(copied, vs)
		out[k] = copied
	}
	return out
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "t", "yes", "y", "on", "require", "required":
		return true
	default:
		return false
	}
}

func resolveStaticDir() string {
	candidates := []string{
		filepath.FromSlash("src/main/resources/static"),
		"static",
	}
	for _, dir := range candidates {
		fi, err := os.Stat(dir)
		if err == nil && fi.IsDir() {
			return dir
		}
	}
	return candidates[0]
}
