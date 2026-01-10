package app

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSystemConfigService_Update_UpsertsAndCaches(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := NewSystemConfigService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortMode, "probe", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortFixed, "9006", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO system_config`).
		WithArgs(systemConfigKeyImagePortRealMinBytes, "2048", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	updated, err := svc.Update(context.Background(), SystemConfig{
		ImagePortMode:         ImagePortModeProbe,
		ImagePortFixed:        "9006",
		ImagePortRealMinBytes: 2048,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.ImagePortMode != ImagePortModeProbe {
		t.Fatalf("mode=%q, want %q", updated.ImagePortMode, ImagePortModeProbe)
	}
	if updated.ImagePortFixed != "9006" {
		t.Fatalf("fixed=%q, want %q", updated.ImagePortFixed, "9006")
	}
	if updated.ImagePortRealMinBytes != 2048 {
		t.Fatalf("minBytes=%d, want %d", updated.ImagePortRealMinBytes, 2048)
	}

	// 二次 Get 应直接命中内存缓存，不触发 DB 查询（因此无需追加 sqlmock 期望）。
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != updated {
		t.Fatalf("cached=%+v, want %+v", got, updated)
	}
}

func TestSystemConfigService_Get_LoadsFromDBWithDefaults(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
		WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes).
		WillReturnRows(sqlmock.NewRows([]string{"config_key", "config_value"}).
			AddRow(systemConfigKeyImagePortMode, "real").
			AddRow(systemConfigKeyImagePortRealMinBytes, "4096"),
		)

	svc := NewSystemConfigService(db)
	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if cfg.ImagePortMode != ImagePortModeReal {
		t.Fatalf("mode=%q, want %q", cfg.ImagePortMode, ImagePortModeReal)
	}
	// 未返回 fixed port 时应回落到默认值。
	if cfg.ImagePortFixed != defaultSystemConfig.ImagePortFixed {
		t.Fatalf("fixed=%q, want %q", cfg.ImagePortFixed, defaultSystemConfig.ImagePortFixed)
	}
	if cfg.ImagePortRealMinBytes != 4096 {
		t.Fatalf("minBytes=%d, want %d", cfg.ImagePortRealMinBytes, 4096)
	}
}

