package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"liao/internal/app"
	"liao/internal/config"
)

func TestBuildLogger_CoversBranches(t *testing.T) {
	cases := []struct {
		level  string
		format string
	}{
		{"debug", "text"},
		{"info", "json"},
		{"warn", "text"},
		{"warning", "json"},
		{"error", "text"},
		{"", ""},
	}

	for _, c := range cases {
		t.Setenv("LOG_LEVEL", c.level)
		t.Setenv("LOG_FORMAT", c.format)
		if got := buildLogger(io.Discard); got == nil {
			t.Fatalf("expected logger")
		}
	}
}

func TestRun_ConfigLoadError(t *testing.T) {
	oldLoad := loadConfigFn
	t.Cleanup(func() { loadConfigFn = oldLoad })

	loadConfigFn = func() (config.Config, error) {
		return config.Config{}, errors.New("load fail")
	}

	if err := run(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_AppInitError(t *testing.T) {
	oldLoad := loadConfigFn
	oldNew := newAppFn
	t.Cleanup(func() {
		loadConfigFn = oldLoad
		newAppFn = oldNew
	})

	loadConfigFn = func() (config.Config, error) {
		return config.Config{ServerPort: 8080}, nil
	}
	newAppFn = func(cfg config.Config) (*app.App, error) {
		return nil, errors.New("init fail")
	}

	if err := run(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_ListenAndServeError(t *testing.T) {
	oldLoad := loadConfigFn
	oldNew := newAppFn
	oldNotify := notifySignalsFn
	oldListen := listenAndServeFn
	t.Cleanup(func() {
		loadConfigFn = oldLoad
		newAppFn = oldNew
		notifySignalsFn = oldNotify
		listenAndServeFn = oldListen
	})

	loadConfigFn = func() (config.Config, error) {
		return config.Config{ServerPort: 8080}, nil
	}
	newAppFn = func(cfg config.Config) (*app.App, error) {
		return &app.App{}, nil
	}
	notifySignalsFn = func(c chan<- os.Signal, sig ...os.Signal) {}
	listenAndServeFn = func(s *http.Server) error { return errors.New("boom") }

	if err := run(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_SignalTriggersShutdown(t *testing.T) {
	oldLoad := loadConfigFn
	oldNew := newAppFn
	oldHandler := appHandlerFn
	oldAppShutdown := appShutdownFn
	oldNotify := notifySignalsFn
	oldListen := listenAndServeFn
	oldShutdown := shutdownServerFn
	t.Cleanup(func() {
		loadConfigFn = oldLoad
		newAppFn = oldNew
		appHandlerFn = oldHandler
		appShutdownFn = oldAppShutdown
		notifySignalsFn = oldNotify
		listenAndServeFn = oldListen
		shutdownServerFn = oldShutdown
	})

	var shutdownSignals chan<- os.Signal
	notifySignalsFn = func(c chan<- os.Signal, sig ...os.Signal) { shutdownSignals = c }

	appShutdownCalled := make(chan struct{})
	appShutdownFn = func(a *app.App, ctx context.Context) {
		close(appShutdownCalled)
	}
	appHandlerFn = func(a *app.App) http.Handler { return http.NewServeMux() }
	newAppFn = func(cfg config.Config) (*app.App, error) {
		return &app.App{}, nil
	}
	loadConfigFn = func() (config.Config, error) {
		return config.Config{ServerPort: 8080}, nil
	}

	shutdownServerFn = func(s *http.Server, ctx context.Context) error { return nil }
	listenAndServeFn = func(s *http.Server) error {
		if shutdownSignals != nil {
			shutdownSignals <- os.Interrupt
		}
		return http.ErrServerClosed
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	select {
	case <-appShutdownCalled:
	case <-time.After(1 * time.Second):
		t.Fatalf("expected shutdown called")
	}
}

func TestRun_ShutdownServerErrorBranch(t *testing.T) {
	oldLoad := loadConfigFn
	oldNew := newAppFn
	oldHandler := appHandlerFn
	oldAppShutdown := appShutdownFn
	oldNotify := notifySignalsFn
	oldListen := listenAndServeFn
	oldShutdown := shutdownServerFn
	t.Cleanup(func() {
		loadConfigFn = oldLoad
		newAppFn = oldNew
		appHandlerFn = oldHandler
		appShutdownFn = oldAppShutdown
		notifySignalsFn = oldNotify
		listenAndServeFn = oldListen
		shutdownServerFn = oldShutdown
	})

	var shutdownSignals chan<- os.Signal
	notifySignalsFn = func(c chan<- os.Signal, sig ...os.Signal) { shutdownSignals = c }

	appShutdownCalled := make(chan struct{})
	appShutdownFn = func(a *app.App, ctx context.Context) {
		close(appShutdownCalled)
	}
	appHandlerFn = func(a *app.App) http.Handler { return http.NewServeMux() }
	newAppFn = func(cfg config.Config) (*app.App, error) {
		return &app.App{}, nil
	}
	loadConfigFn = func() (config.Config, error) {
		return config.Config{ServerPort: 8080}, nil
	}

	shutdownServerFn = func(s *http.Server, ctx context.Context) error { return errors.New("shutdown fail") }
	listenAndServeFn = func(s *http.Server) error {
		if shutdownSignals != nil {
			shutdownSignals <- os.Interrupt
		}
		return http.ErrServerClosed
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	select {
	case <-appShutdownCalled:
	case <-time.After(1 * time.Second):
		t.Fatalf("expected shutdown called")
	}
}
