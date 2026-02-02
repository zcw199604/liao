package app

import (
	"context"
	"fmt"

	"liao/internal/database"
)

func ensureSchema(db *database.DB) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	m := database.NewMigrator("sql")
	if err := m.Migrate(context.Background(), db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	return nil
}
