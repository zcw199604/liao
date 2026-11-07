package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"liao/internal/database"
)

// wanfengMediaMapStore 是「晚风媒体键 → 我方媒体文件名」的解析接口。
//
// 之所以要持久化：晚风媒体只有约 30 分钟有效期，过期后下载接口直接 404。
// 只要我们把字节搬运过一次（实时收发），映射就必须永久可用，
// 否则拉历史时又要回头找晚风要文件 → 文件已过期 → 历史里的图片/视频消失。
type wanfengMediaMapStore interface {
	// Get 查询映射；不存在时返回 ("", false, nil)。
	Get(ctx context.Context, key string) (string, bool, error)
	// Put 写入/更新映射。
	Put(ctx context.Context, key, remoteFilename, identityID string) error
}

// wanfengMediaMapDB 是 wanfengMediaMapStore 的数据库实现（表 wanfeng_media_map）。
type wanfengMediaMapDB struct {
	db *database.DB
}

// NewWanfengMediaMapDB 创建映射存储（db 为空时返回 nil，调用方按「无持久化」降级处理）。
func NewWanfengMediaMapDB(db *database.DB) *wanfengMediaMapDB {
	if db == nil {
		return nil
	}
	return &wanfengMediaMapDB{db: db}
}

// Get 读取映射。
func (s *wanfengMediaMapDB) Get(ctx context.Context, key string) (string, bool, error) {
	key = strings.TrimSpace(key)
	if s == nil || s.db == nil || key == "" {
		return "", false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT remote_filename FROM wanfeng_media_map WHERE wf_key = ?`, key).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("查询晚风媒体映射失败: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, nil
	}
	return name, true, nil
}

// Put 写入映射（幂等 upsert）。
func (s *wanfengMediaMapDB) Put(ctx context.Context, key, remoteFilename, identityID string) error {
	key = strings.TrimSpace(key)
	remoteFilename = strings.TrimSpace(remoteFilename)
	if s == nil || s.db == nil || key == "" || remoteFilename == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	query := `INSERT INTO wanfeng_media_map (wf_key, remote_filename, identity_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`
	args := []any{key, remoteFilename, nullableStringValue(identityID), now, now}
	if s.db.Dialect().Name() == "postgres" {
		query += `
		ON CONFLICT (wf_key) DO UPDATE SET
			remote_filename = EXCLUDED.remote_filename,
			identity_id = COALESCE(EXCLUDED.identity_id, wanfeng_media_map.identity_id),
			updated_at = EXCLUDED.updated_at`
	} else {
		query += `
		ON DUPLICATE KEY UPDATE
			remote_filename = VALUES(remote_filename),
			identity_id = COALESCE(VALUES(identity_id), identity_id),
			updated_at = VALUES(updated_at)`
	}
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("保存晚风媒体映射失败: %w", err)
	}
	return nil
}
