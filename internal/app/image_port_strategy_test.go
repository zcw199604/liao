package app

import (
	"context"
	"net/http"
	"testing"
)

func TestResolveImagePortByConfig_RealFallsBackToProbe(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9001" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     db,
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeReal,
				ImagePortFixed:        "9006",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer:       NewImageServerService("127.0.0.1", "9003"),
		imagePortResolver: NewImagePortResolver(&http.Client{}),
	}
	app.imagePortResolver.ports = []string{"1"}

	got := app.resolveImagePortByConfig(context.Background(), "a.jpg")
	if got != "9001" {
		t.Fatalf("port=%q, want %q", got, "9001")
	}
}
