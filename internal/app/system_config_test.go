package app

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSystemConfigService_EnsureDefaults_NilOrNoDB(t *testing.T) {
	var svc *SystemConfigService
	if err := svc.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	svc = &SystemConfigService{}
	if err := svc.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestSystemConfigService_EnsureDefaults_InsertsAndErrors(t *testing.T) {
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		mock.ExpectExec(`INSERT IGNORE INTO system_config`).
			WithArgs(systemConfigKeyImagePortMode, string(defaultSystemConfig.ImagePortMode), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT IGNORE INTO system_config`).
			WithArgs(systemConfigKeyImagePortFixed, defaultSystemConfig.ImagePortFixed, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT IGNORE INTO system_config`).
			WithArgs(systemConfigKeyImagePortRealMinBytes, "2048", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		svc := NewSystemConfigService(db)
		if err := svc.EnsureDefaults(context.Background()); err != nil {
			t.Fatalf("EnsureDefaults: %v", err)
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`INSERT IGNORE INTO system_config`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))

		svc := NewSystemConfigService(db)
		if err := svc.EnsureDefaults(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestSystemConfigService_Get_NilOrNoDB(t *testing.T) {
	var svc *SystemConfigService
	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if cfg != defaultSystemConfig {
		t.Fatalf("cfg=%+v, want %+v", cfg, defaultSystemConfig)
	}

	svc = &SystemConfigService{}
	cfg, err = svc.Get(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if cfg != defaultSystemConfig {
		t.Fatalf("cfg=%+v, want %+v", cfg, defaultSystemConfig)
	}
}

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

func TestSystemConfigService_Get_LoadError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
		WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes).
		WillReturnError(errors.New("query fail"))

	svc := NewSystemConfigService(db)
	if _, err := svc.Get(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSystemConfigService_Get_Load_IgnoresInvalidValues(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
		WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes).
		WillReturnRows(sqlmock.NewRows([]string{"config_key", "config_value"}).
			AddRow(systemConfigKeyImagePortMode, "bad").
			AddRow(systemConfigKeyImagePortMode, "probe").
			AddRow(systemConfigKeyImagePortFixed, "99999").
			AddRow(systemConfigKeyImagePortFixed, "8080").
			AddRow(systemConfigKeyImagePortRealMinBytes, "abc").
			AddRow(systemConfigKeyImagePortRealMinBytes, "-1").
			AddRow(systemConfigKeyImagePortRealMinBytes, "4096"),
		)

	svc := NewSystemConfigService(db)
	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg.ImagePortMode != ImagePortModeProbe {
		t.Fatalf("mode=%q, want %q", cfg.ImagePortMode, ImagePortModeProbe)
	}
	if cfg.ImagePortFixed != "8080" {
		t.Fatalf("fixed=%q, want %q", cfg.ImagePortFixed, "8080")
	}
	if cfg.ImagePortRealMinBytes != 4096 {
		t.Fatalf("minBytes=%d, want %d", cfg.ImagePortRealMinBytes, 4096)
	}
}

func TestSystemConfigService_Get_Load_ScanErrorAndRowsErr(t *testing.T) {
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
			WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes).
			WillReturnRows(sqlmock.NewRows([]string{"config_key", "config_value"}).
				AddRow(systemConfigKeyImagePortMode, nil),
			)

		svc := NewSystemConfigService(db)
		if _, err := svc.Get(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
			WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes).
			WillReturnRows(sqlmock.NewRows([]string{"config_key", "config_value"}).
				AddRow(systemConfigKeyImagePortMode, "probe").
				RowError(0, errors.New("next fail")),
			)

		svc := NewSystemConfigService(db)
		if _, err := svc.Get(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestSystemConfigService_Update_Errors(t *testing.T) {
	{
		var svc *SystemConfigService
		if _, err := svc.Update(context.Background(), SystemConfig{}); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), SystemConfig{ImagePortMode: ImagePortMode("bad")}); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), defaultSystemConfig); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortMode, string(defaultSystemConfig.ImagePortMode), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("upsert fail"))
		mock.ExpectRollback()

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), defaultSystemConfig); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortMode, string(defaultSystemConfig.ImagePortMode), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortFixed, defaultSystemConfig.ImagePortFixed, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("upsert fail"))
		mock.ExpectRollback()

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), defaultSystemConfig); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortMode, string(defaultSystemConfig.ImagePortMode), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortFixed, defaultSystemConfig.ImagePortFixed, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortRealMinBytes, "2048", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("upsert fail"))
		mock.ExpectRollback()

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), defaultSystemConfig); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortMode, string(defaultSystemConfig.ImagePortMode), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortFixed, defaultSystemConfig.ImagePortFixed, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO system_config`).
			WithArgs(systemConfigKeyImagePortRealMinBytes, "2048", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit fail"))

		svc := NewSystemConfigService(db)
		if _, err := svc.Update(context.Background(), defaultSystemConfig); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestNormalizeSystemConfig(t *testing.T) {
	if _, err := normalizeSystemConfig(SystemConfig{ImagePortMode: ImagePortMode("bad")}); err == nil {
		t.Fatalf("expected error")
	}

	cfg, err := normalizeSystemConfig(SystemConfig{ImagePortMode: "", ImagePortFixed: "", ImagePortRealMinBytes: 0})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg != defaultSystemConfig {
		t.Fatalf("cfg=%+v, want %+v", cfg, defaultSystemConfig)
	}

	if _, err := normalizeSystemConfig(SystemConfig{ImagePortMode: ImagePortModeFixed, ImagePortFixed: "9006", ImagePortRealMinBytes: 1}); err == nil {
		t.Fatalf("expected error")
	}

	if _, err := normalizeSystemConfig(SystemConfig{ImagePortMode: ImagePortModeFixed, ImagePortFixed: "0", ImagePortRealMinBytes: 2048}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePortString(t *testing.T) {
	if _, err := parsePortString(""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parsePortString("0"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parsePortString("65536"); err == nil {
		t.Fatalf("expected error")
	}
	if got, err := parsePortString("8080"); err != nil || got != 8080 {
		t.Fatalf("got=%d err=%v", got, err)
	}
}
