package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"liao/internal/config"
)

// App 负责组装依赖并提供 HTTP Handler。
type App struct {
	cfg config.Config
	db  *sql.DB

	httpClient *http.Client
	jwt        *JWTService

	systemConfig      *SystemConfigService
	imagePortResolver *ImagePortResolver

	identityService *IdentityService
	favoriteService *FavoriteService
	fileStorage     *FileStorageService
	imageServer     *ImageServerService
	imageCache      *ImageCacheService
	imageHash       *ImageHashService
	mediaUpload     *MediaUploadService
	mtPhoto         *MtPhotoService
	userInfoCache   UserInfoCacheService
	forceoutManager *ForceoutManager
	wsManager       *UpstreamWebSocketManager

	staticDir string
	handler   http.Handler
}

func New(cfg config.Config) (*App, error) {
	db, err := openDB(cfg)
	if err != nil {
		return nil, err
	}

	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	staticDir := resolveStaticDir()
	if err := os.MkdirAll("upload", 0o755); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("创建 upload 目录失败: %w", err)
	}

	var userInfoCache UserInfoCacheService
	switch cfg.CacheType {
	case "redis":
		userInfoCache, err = NewRedisUserInfoCacheService(
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
		)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
	default:
		userInfoCache = NewMemoryUserInfoCacheService()
	}

	application := &App{
		cfg:             cfg,
		db:              db,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
		jwt:             NewJWTService(cfg.JWTSecret, cfg.TokenExpireHours),
		identityService: NewIdentityService(db),
		favoriteService: NewFavoriteService(db),
		fileStorage:     NewFileStorageService(db),
		imageServer:     NewImageServerService(cfg.ImageServerHost, cfg.ImageServerPort),
		imageCache:      NewImageCacheService(),
		imageHash:       NewImageHashService(db),
		userInfoCache:   userInfoCache,
		forceoutManager: NewForceoutManager(),
		staticDir:       staticDir,
	}
	application.systemConfig = NewSystemConfigService(db)
	application.imagePortResolver = NewImagePortResolver(application.httpClient)
	_ = application.systemConfig.EnsureDefaults(context.Background())
	application.wsManager = NewUpstreamWebSocketManager(application.httpClient, cfg.WebSocketFallback, application.forceoutManager, application.userInfoCache)
	application.mediaUpload = NewMediaUploadService(db, cfg.ServerPort, application.fileStorage, application.imageServer, application.httpClient)
	application.mtPhoto = NewMtPhotoService(cfg.MtPhotoBaseURL, cfg.MtPhotoLoginUsername, cfg.MtPhotoLoginPassword, cfg.MtPhotoLoginOTP, cfg.LspRoot, application.httpClient)

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
	if closer, ok := a.userInfoCache.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			slog.Error("关闭数据库失败", "error", err)
		}
	}
}

func openDB(cfg config.Config) (*sql.DB, error) {
	host, port, database, params, err := config.ParseJDBCMySQLURL(cfg.DBURL)
	if err != nil {
		return nil, err
	}

	loc := params.Get("serverTimezone")
	if loc == "" {
		loc = "Local"
	}

	// 说明：parseTime 与 loc 对应 Java 侧 DATETIME/LocalDateTime 的常用用法。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s&charset=utf8mb4,utf8&collation=utf8mb4_unicode_ci&timeout=15s&readTimeout=15s&writeTimeout=15s",
		cfg.DBUsername,
		cfg.DBPassword,
		host,
		port,
		database,
		urlQueryEscape(loc),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return db, nil
}

func urlQueryEscape(input string) string {
	// 避免在 config 包引入额外依赖导致循环引用。
	// 这里仅做最小处理：Go MySQL driver 要求 loc 值是 URL 编码。
	// 常见值如 Asia/Shanghai，需要将 / 转义为 %2F。
	replacer := strings.NewReplacer("/", "%2F", " ", "%20")
	return replacer.Replace(input)
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
