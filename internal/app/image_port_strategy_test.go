package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

func TestGetImageServerHostOnly(t *testing.T) {
	var a *App
	if got := a.getImageServerHostOnly(); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}

	a = &App{}
	if got := a.getImageServerHostOnly(); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}

	s := NewImageServerService("127.0.0.1", "9003")
	s.host = "   "
	a = &App{imageServer: s}
	if got := a.getImageServerHostOnly(); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}

	s2 := NewImageServerService("127.0.0.1", "9003")
	a = &App{imageServer: s2}
	if got := a.getImageServerHostOnly(); got != "127.0.0.1" {
		t.Fatalf("got=%q, want %q", got, "127.0.0.1")
	}
}

func TestResolveImagePortByConfig_FixedWhenNoImageHost(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	svc := &SystemConfigService{
		db:     wrapMySQLDB(db),
		loaded: true,
		cached: SystemConfig{
			ImagePortMode:         ImagePortModeProbe,
			ImagePortFixed:        "9006",
			ImagePortRealMinBytes: 2048,
		},
	}
	img := NewImageServerService("127.0.0.1", "9003")
	img.host = " "

	app := &App{
		systemConfig: svc,
		imageServer:  img,
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != "9006" {
		t.Fatalf("port=%q, want %q", got, "9006")
	}
}

func TestResolveImagePortByConfig_ProbeUsesDetectAvailablePort(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9002" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeProbe,
				ImagePortFixed:        "9006",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer: NewImageServerService("127.0.0.1", "9003"),
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != "9002" {
		t.Fatalf("port=%q, want %q", got, "9002")
	}
}

func TestResolveImagePortByConfig_DefaultFixedWhenConfigFixedEmpty(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeFixed,
				ImagePortFixed:        "",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer: &ImageServerService{host: " ", port: "9003"},
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != defaultSystemConfig.ImagePortFixed {
		t.Fatalf("port=%q, want %q", got, defaultSystemConfig.ImagePortFixed)
	}
}

func TestResolveImagePortByConfig_Real_ReturnsResolvedPort(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	minBytes := int64(8)
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBuffer(bytes.Repeat([]byte("x"), int(minBytes)))),
				Request:    r,
			}
			res.Header.Set("Content-Type", "image/jpeg")
			return res, nil
		}),
	}
	resolver := NewImagePortResolver(client)
	resolver.ports = []string{"9002"}

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeReal,
				ImagePortFixed:        "9006",
				ImagePortRealMinBytes: minBytes,
			},
		},
		imageServer:       NewImageServerService("127.0.0.1", "9003"),
		imagePortResolver: resolver,
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != "9002" {
		t.Fatalf("port=%q, want %q", got, "9002")
	}
}

func TestResolveImagePortByConfig_Real_UsesCachedWhenRealFails(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	resolver := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString("nope")),
			Request:    r,
		}
		return res, nil
	})})
	resolver.cache["127.0.0.1"] = "9005"

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeReal,
				ImagePortFixed:        "9006",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer:       NewImageServerService("127.0.0.1", "9003"),
		imagePortResolver: resolver,
	}

	if got := app.resolveImagePortByConfig(context.Background(), ""); got != "9005" {
		t.Fatalf("port=%q, want %q", got, "9005")
	}
}

func TestResolveImagePortByConfig_Real_FallsBackToFixedWhenDetectEmpty(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return " " }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortModeReal,
				ImagePortFixed:        "9006",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer: NewImageServerService("127.0.0.1", "9003"),
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != "9006" {
		t.Fatalf("port=%q, want %q", got, "9006")
	}
}

func TestResolveImagePortByConfig_DefaultModeReturnsFixed(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
			loaded: true,
			cached: SystemConfig{
				ImagePortMode:         ImagePortMode("weird"),
				ImagePortFixed:        "9003",
				ImagePortRealMinBytes: 2048,
			},
		},
		imageServer: NewImageServerService("127.0.0.1", "9003"),
	}

	if got := app.resolveImagePortByConfig(context.Background(), "a.jpg"); got != "9003" {
		t.Fatalf("port=%q, want %q", got, "9003")
	}
}

func TestResolveImagePortByConfig_RealFallsBackToProbe(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9001" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	app := &App{
		systemConfig: &SystemConfigService{
			db:     wrapMySQLDB(db),
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
