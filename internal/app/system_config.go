package app

// SystemConfigService 管理系统级全局配置（所有用户共用），以 DB Key-Value 的方式存储并提供默认值兜底。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"liao/internal/database"
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
	systemConfigKeyMtPhotoTimelineDefer  = "mtphoto_timeline_defer_subfolder_threshold"
)

var defaultSystemConfig = SystemConfig{
	ImagePortMode:                          ImagePortModeFixed,
	ImagePortFixed:                         "9006",
	ImagePortRealMinBytes:                  2048,
	MtPhotoTimelineDeferSubfolderThreshold: 10,
}

type SystemConfig struct {
	ImagePortMode                          ImagePortMode `json:"imagePortMode"`
	ImagePortFixed                         string        `json:"imagePortFixed"`
	ImagePortRealMinBytes                  int64         `json:"imagePortRealMinBytes"`
	MtPhotoTimelineDeferSubfolderThreshold int           `json:"mtPhotoTimelineDeferSubfolderThreshold"`
}

type SystemConfigService struct {
	db *database.DB
	// defaults 由环境变量注入（若未注入则使用 defaultSystemConfig）
	defaults SystemConfig

	mu     sync.RWMutex
	loaded bool
	cached SystemConfig
}

func NewSystemConfigService(db *database.DB, defaults ...SystemConfig) *SystemConfigService {
	base := defaultSystemConfig
	if len(defaults) > 0 {
		if normalized, err := normalizeSystemConfigWithDefaults(defaults[0], defaultSystemConfig); err == nil {
			base = normalized
		}
	}
	return &SystemConfigService{
		db:       db,
		defaults: base,
	}
}

func (s *SystemConfigService) serviceDefaults() SystemConfig {
	if s == nil {
		return defaultSystemConfig
	}
	defaults := s.defaults
	if normalized, err := normalizeSystemConfigWithDefaults(defaults, defaultSystemConfig); err == nil {
		return normalized
	}
	return defaultSystemConfig
}

func (s *SystemConfigService) EnsureDefaults(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}

	now := time.Now()
	defaultsCfg := s.serviceDefaults()
	defaults := map[string]string{
		systemConfigKeyImagePortMode:         string(defaultsCfg.ImagePortMode),
		systemConfigKeyImagePortFixed:        defaultsCfg.ImagePortFixed,
		systemConfigKeyImagePortRealMinBytes: fmt.Sprint(defaultsCfg.ImagePortRealMinBytes),
		systemConfigKeyMtPhotoTimelineDefer:  fmt.Sprint(defaultsCfg.MtPhotoTimelineDeferSubfolderThreshold),
	}

	for k, v := range defaults {
		if _, err := database.ExecInsertIgnore(
			ctx,
			s.db,
			"system_config",
			[]string{"config_key", "config_value", "created_at", "updated_at"},
			[]string{"config_key"},
			k,
			v,
			now,
			now,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SystemConfigService) Get(ctx context.Context) (SystemConfig, error) {
	if s == nil {
		return defaultSystemConfig, nil
	}
	if s.db == nil {
		return s.serviceDefaults(), nil
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
	out := s.serviceDefaults()

	rows, err := s.db.QueryContext(ctx,
		"SELECT config_key, config_value FROM system_config WHERE config_key IN (?, ?, ?, ?)",
		systemConfigKeyImagePortMode,
		systemConfigKeyImagePortFixed,
		systemConfigKeyImagePortRealMinBytes,
		systemConfigKeyMtPhotoTimelineDefer,
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
		case systemConfigKeyMtPhotoTimelineDefer:
			if value != "" {
				if n, err := strconv.Atoi(value); err == nil && n > 0 {
					out.MtPhotoTimelineDeferSubfolderThreshold = n
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return SystemConfig{}, err
	}

	return normalizeSystemConfigWithDefaults(out, s.serviceDefaults())
}

func (s *SystemConfigService) Update(ctx context.Context, next SystemConfig) (SystemConfig, error) {
	if s == nil || s.db == nil {
		return SystemConfig{}, fmt.Errorf("系统配置服务未初始化")
	}

	normalized, err := normalizeSystemConfigWithDefaults(next, s.serviceDefaults())
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
		_, err := database.ExecUpsert(
			ctx,
			tx,
			"system_config",
			[]string{"config_key", "config_value", "created_at", "updated_at"},
			[]string{"config_key"},
			[]string{"config_value", "updated_at"},
			nil,
			key,
			value,
			now,
			now,
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
	if err := upsert(systemConfigKeyMtPhotoTimelineDefer, fmt.Sprint(normalized.MtPhotoTimelineDeferSubfolderThreshold)); err != nil {
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
	return normalizeSystemConfigWithDefaults(cfg, defaultSystemConfig)
}

func normalizeSystemConfigWithDefaults(cfg, defaults SystemConfig) (SystemConfig, error) {
	mode := ImagePortMode(strings.TrimSpace(string(cfg.ImagePortMode)))
	if mode == "" {
		mode = defaults.ImagePortMode
	}
	switch mode {
	case ImagePortModeFixed, ImagePortModeProbe, ImagePortModeReal:
	default:
		return SystemConfig{}, fmt.Errorf("不支持的 imagePortMode: %s", mode)
	}

	fixedPort := strings.TrimSpace(cfg.ImagePortFixed)
	if fixedPort == "" {
		fixedPort = defaults.ImagePortFixed
	}
	if _, err := parsePortString(fixedPort); err != nil {
		return SystemConfig{}, err
	}

	minBytes := cfg.ImagePortRealMinBytes
	if minBytes <= 0 {
		minBytes = defaults.ImagePortRealMinBytes
	}
	if minBytes < 256 || minBytes > 64*1024 {
		return SystemConfig{}, fmt.Errorf("imagePortRealMinBytes 超出范围（256~65536）: %d", minBytes)
	}

	deferThreshold := cfg.MtPhotoTimelineDeferSubfolderThreshold
	if deferThreshold <= 0 {
		deferThreshold = defaults.MtPhotoTimelineDeferSubfolderThreshold
	}
	if deferThreshold < 1 || deferThreshold > 500 {
		return SystemConfig{}, fmt.Errorf("mtPhotoTimelineDeferSubfolderThreshold 超出范围（1~500）: %d", deferThreshold)
	}

	return SystemConfig{
		ImagePortMode:                          mode,
		ImagePortFixed:                         fixedPort,
		ImagePortRealMinBytes:                  minBytes,
		MtPhotoTimelineDeferSubfolderThreshold: deferThreshold,
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
