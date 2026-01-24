package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"liao/internal/app"
	"liao/internal/config"
)

var (
	loadConfigFn  = config.Load
	newAppFn      = app.New
	appHandlerFn  = (*app.App).Handler
	appShutdownFn = (*app.App).Shutdown

	notifySignalsFn  = signal.Notify
	listenAndServeFn = (*http.Server).ListenAndServe
	shutdownServerFn = (*http.Server).Shutdown
)

func buildLogger(w io.Writer) *slog.Logger {
	logLevel := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	handlerOptions := &slog.HandlerOptions{Level: logLevel}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "text") {
		handler = slog.NewTextHandler(w, handlerOptions)
	} else {
		handler = slog.NewJSONHandler(w, handlerOptions)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func run() error {
	logger := buildLogger(os.Stdout)

	cfg, err := loadConfigFn()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		return err
	}

	application, err := newAppFn(cfg)
	if err != nil {
		logger.Error("初始化应用失败", "error", err)
		return err
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           appHandlerFn(application),
		ReadHeaderTimeout: 10 * time.Second,
	}

	shutdownSignals := make(chan os.Signal, 1)
	notifySignalsFn(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownSignals

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		appShutdownFn(application, ctx)

		if err := shutdownServerFn(server, ctx); err != nil {
			logger.Error("HTTP 服务停止失败", "error", err)
		}
	}()

	logger.Info("HTTP 服务启动", "addr", server.Addr)

	if err := listenAndServeFn(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP 服务异常退出", "error", err)
		return err
	}

	return nil
}
