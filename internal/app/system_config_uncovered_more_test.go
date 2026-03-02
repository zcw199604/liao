package app

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSystemConfigService_ServiceDefaultsUncoveredBranches(t *testing.T) {
	var nilSvc *SystemConfigService
	if got := nilSvc.serviceDefaults(); got != defaultSystemConfig {
		t.Fatalf("nil service defaults=%+v, want %+v", got, defaultSystemConfig)
	}

	svc := &SystemConfigService{
		defaults: SystemConfig{
			ImagePortMode:                          ImagePortModeFixed,
			ImagePortFixed:                         "bad-port",
			ImagePortRealMinBytes:                  2048,
			MtPhotoTimelineDeferSubfolderThreshold: 10,
		},
	}
	if got := svc.serviceDefaults(); got != defaultSystemConfig {
		t.Fatalf("invalid defaults should fallback, got=%+v", got)
	}
}

func TestSystemConfigService_Get_LoadsMtPhotoTimelineThreshold(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT config_key, config_value FROM system_config WHERE config_key IN`).
		WithArgs(systemConfigKeyImagePortMode, systemConfigKeyImagePortFixed, systemConfigKeyImagePortRealMinBytes, systemConfigKeyMtPhotoTimelineDefer).
		WillReturnRows(sqlmock.NewRows([]string{"config_key", "config_value"}).
			AddRow(systemConfigKeyMtPhotoTimelineDefer, "123"),
		)

	svc := NewSystemConfigService(wrapMySQLDB(db))
	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get err=%v", err)
	}
	if cfg.MtPhotoTimelineDeferSubfolderThreshold != 123 {
		t.Fatalf("threshold=%d, want 123", cfg.MtPhotoTimelineDeferSubfolderThreshold)
	}
}

func TestNormalizeSystemConfigWithDefaults_DeferThresholdOutOfRange(t *testing.T) {
	_, err := normalizeSystemConfigWithDefaults(SystemConfig{
		ImagePortMode:                          ImagePortModeFixed,
		ImagePortFixed:                         "9006",
		ImagePortRealMinBytes:                  2048,
		MtPhotoTimelineDeferSubfolderThreshold: 999,
	}, defaultSystemConfig)
	if err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

