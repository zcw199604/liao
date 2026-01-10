package app

// SystemConfigService 管理系统级全局配置（所有用户共用），以 DB Key-Value 的方式存储并提供默认值兜底。

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ImagePortMode string

const (
	ImagePortModeFixed ImagePortMode = "fixed"
	ImagePortModeProbe ImagePortMode = "probe"
	ImagePortModeReal  ImagePortMode = "real"
)

const (
	systemConfigKeyImagePortMode         = "image_port_mode"
	systemConfigKeyImagePortFixed        = "image_port_fixed"
	systemConfigKeyImagePortRealMinBytes = "image_port_real_min_bytes"
)

var defaultSystemConfig = SystemConfig{
	ImagePortMode:         ImagePortModeFixed,
	ImagePortFixed:        "9006",
	ImagePortRealMinBytes: 2048,
}

type SystemConfig struct {
	ImagePortMode         ImagePortMode `json:"imagePortMode"`
	ImagePortFixed        string        `json:"imagePortFixed"`
	ImagePortRealMinBytes int64         `json:"imagePortRealMinBytes"`
}

type SystemConfigService struct {
	db *sql.DB

	mu     sync.RWMutex
	loaded bool
	cached SystemConfig
}

func NewSystemConfigService(db *sql.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) EnsureDefaults(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}

	now := time.Now()
	defaults := map[string]string{
		systemConfigKeyImagePortMode:         string(defaultSystemConfig.ImagePortMode),
		systemConfigKeyImagePortFixed:        defaultSystemConfig.ImagePortFixed,
		systemConfigKeyImagePortRealMinBytes: fmt.Sprint(defaultSystemConfig.ImagePortRealMinBytes),
	}

	for k, v := range defaults {
		if _, err := s.db.ExecContext(ctx,
			"INSERT IGNORE INTO system_config (config_key, config_value, created_at, updated_at) VALUES (?, ?, ?, ?)",
			k, v, now, now,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SystemConfigService) Get(ctx context.Context) (SystemConfig, error) {
	if s == nil || s.db == nil {
		return defaultSystemConfig, nil
	}

	s.mu.RLock()
	if s.loaded {
		cfg := s.cached
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	cfg, err := s.load(ctx)
	if err != nil {
		return SystemConfig{}, err
	}

	s.mu.Lock()
	s.cached = cfg
	s.loaded = true
	s.mu.Unlock()

	return cfg, nil
}

func (s *SystemConfigService) load(ctx context.Context) (SystemConfig, error) {
	out := defaultSystemConfig

	rows, err := s.db.QueryContext(ctx,
		"SELECT config_key, config_value FROM system_config WHERE config_key IN (?, ?, ?)",
		systemConfigKeyImagePortMode,
		systemConfigKeyImagePortFixed,
		systemConfigKeyImagePortRealMinBytes,
	)
	if err != nil {
		return SystemConfig{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return SystemConfig{}, err
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case systemConfigKeyImagePortMode:
			mode := ImagePortMode(value)
			if mode == ImagePortModeFixed || mode == ImagePortModeProbe || mode == ImagePortModeReal {
				out.ImagePortMode = mode
			}
		case systemConfigKeyImagePortFixed:
			if value != "" {
				if _, err := parsePortString(value); err == nil {
					out.ImagePortFixed = value
				}
			}
		case systemConfigKeyImagePortRealMinBytes:
			if value != "" {
				if n, err := strconv.ParseInt(value, 10, 64); err == nil && n > 0 {
					out.ImagePortRealMinBytes = n
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return SystemConfig{}, err
	}

	return normalizeSystemConfig(out)
}

func (s *SystemConfigService) Update(ctx context.Context, next SystemConfig) (SystemConfig, error) {
	if s == nil || s.db == nil {
		return SystemConfig{}, fmt.Errorf("系统配置服务未初始化")
	}

	normalized, err := normalizeSystemConfig(next)
	if err != nil {
		return SystemConfig{}, err
	}

	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SystemConfig{}, err
	}
	defer func() { _ = tx.Rollback() }()

	upsert := func(key, value string) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO system_config (config_key, config_value, created_at, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE config_value = VALUES(config_value), updated_at = VALUES(updated_at)`,
			key, value, now, now,
		)
		return err
	}

	if err := upsert(systemConfigKeyImagePortMode, string(normalized.ImagePortMode)); err != nil {
		return SystemConfig{}, err
	}
	if err := upsert(systemConfigKeyImagePortFixed, normalized.ImagePortFixed); err != nil {
		return SystemConfig{}, err
	}
	if err := upsert(systemConfigKeyImagePortRealMinBytes, fmt.Sprint(normalized.ImagePortRealMinBytes)); err != nil {
		return SystemConfig{}, err
	}

	if err := tx.Commit(); err != nil {
		return SystemConfig{}, err
	}

	s.mu.Lock()
	s.cached = normalized
	s.loaded = true
	s.mu.Unlock()

	return normalized, nil
}

func normalizeSystemConfig(cfg SystemConfig) (SystemConfig, error) {
	mode := ImagePortMode(strings.TrimSpace(string(cfg.ImagePortMode)))
	if mode == "" {
		mode = defaultSystemConfig.ImagePortMode
	}
	switch mode {
	case ImagePortModeFixed, ImagePortModeProbe, ImagePortModeReal:
	default:
		return SystemConfig{}, fmt.Errorf("不支持的 imagePortMode: %s", mode)
	}

	fixedPort := strings.TrimSpace(cfg.ImagePortFixed)
	if fixedPort == "" {
		fixedPort = defaultSystemConfig.ImagePortFixed
	}
	if _, err := parsePortString(fixedPort); err != nil {
		return SystemConfig{}, err
	}

	minBytes := cfg.ImagePortRealMinBytes
	if minBytes <= 0 {
		minBytes = defaultSystemConfig.ImagePortRealMinBytes
	}
	if minBytes < 256 || minBytes > 64*1024 {
		return SystemConfig{}, fmt.Errorf("imagePortRealMinBytes 超出范围（256~65536）: %d", minBytes)
	}

	return SystemConfig{
		ImagePortMode:         mode,
		ImagePortFixed:        fixedPort,
		ImagePortRealMinBytes: minBytes,
	}, nil
}

func parsePortString(port string) (int, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return 0, fmt.Errorf("端口不能为空")
	}
	v, err := strconv.Atoi(port)
	if err != nil || v <= 0 || v > 65535 {
		return 0, fmt.Errorf("端口不合法: %s", port)
	}
	return v, nil
}
