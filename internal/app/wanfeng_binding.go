package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"liao/internal/database"
	"liao/internal/upstream/wanfeng"
)

// ProviderChat2019 / ProviderWanfeng 是前端可选的上游线路标识。
const (
	ProviderChat2019 = "chat2019"
	ProviderWanfeng  = "wanfeng"
)

// WanfengBindingService 持久化「我方身份 ↔ 晚风身份」映射。
// 说明：晚风会下发 uid 与 identityToken，必须落库，否则重连会变成新用户、
// 且同一设备多次申请身份可能触发对方风控。
type WanfengBindingService struct {
	db *database.DB
}

// NewWanfengBindingService 创建绑定服务。
func NewWanfengBindingService(db *database.DB) *WanfengBindingService {
	return &WanfengBindingService{db: db}
}

// Get 读取绑定；不存在时返回 (nil, nil)。
func (s *WanfengBindingService) Get(ctx context.Context, identityID string) (*wanfeng.Binding, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" || s.db == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row := s.db.QueryRowContext(ctx, `SELECT identity_id, upstream_uid, identity_token, device_id, nickname, gender, age
		FROM identity_provider_binding WHERE identity_id = ? AND provider = ?`, identityID, ProviderWanfeng)

	var (
		binding  wanfeng.Binding
		token    sql.NullString
		deviceID sql.NullString
		nickname sql.NullString
		gender   sql.NullString
		age      sql.NullInt64
	)
	if err := row.Scan(&binding.IdentityID, &binding.UpstreamUID, &token, &deviceID, &nickname, &gender, &age); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询晚风身份绑定失败: %w", err)
	}
	binding.IdentityToken = token.String
	binding.DeviceID = deviceID.String
	binding.Nickname = nickname.String
	binding.Gender = gender.String
	binding.Age = int(age.Int64)
	return &binding, nil
}

// Save 写入（或更新）绑定。
func (s *WanfengBindingService) Save(ctx context.Context, binding wanfeng.Binding) error {
	identityID := strings.TrimSpace(binding.IdentityID)
	if identityID == "" || s.db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	args := []any{
		identityID,
		ProviderWanfeng,
		strings.TrimSpace(binding.UpstreamUID),
		nullableStringValue(binding.IdentityToken),
		nullableStringValue(binding.DeviceID),
		nullableStringValue(binding.Nickname),
		nullableStringValue(binding.Gender),
		binding.Age,
		now,
		now,
	}

	baseQuery := `INSERT INTO identity_provider_binding (
		identity_id, provider, upstream_uid, identity_token, device_id, nickname, gender, age, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var query string
	switch s.db.Dialect().Name() {
	case "postgres":
		query = baseQuery + `
		ON CONFLICT (identity_id, provider) DO UPDATE SET
			upstream_uid = EXCLUDED.upstream_uid,
			identity_token = COALESCE(EXCLUDED.identity_token, identity_provider_binding.identity_token),
			device_id = COALESCE(EXCLUDED.device_id, identity_provider_binding.device_id),
			nickname = COALESCE(EXCLUDED.nickname, identity_provider_binding.nickname),
			gender = COALESCE(EXCLUDED.gender, identity_provider_binding.gender),
			age = EXCLUDED.age,
			updated_at = EXCLUDED.updated_at`
	default:
		query = baseQuery + `
		ON DUPLICATE KEY UPDATE
			upstream_uid = VALUES(upstream_uid),
			identity_token = COALESCE(VALUES(identity_token), identity_token),
			device_id = COALESCE(VALUES(device_id), device_id),
			nickname = COALESCE(VALUES(nickname), nickname),
			gender = COALESCE(VALUES(gender), gender),
			age = VALUES(age),
			updated_at = VALUES(updated_at)`
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("保存晚风身份绑定失败: %w", err)
	}
	return nil
}

// nullableStringValue 把空串转成 NULL，避免覆盖已有值。
func nullableStringValue(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}
