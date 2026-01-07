package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"liao/internal/app"
	"liao/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		logger.Error("初始化应用失败", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           application.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownSignals

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		application.Shutdown(ctx)

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP 服务停止失败", "error", err)
		}
	}()

	logger.Info("HTTP 服务启动", "addr", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP 服务异常退出", "error", err)
		os.Exit(1)
	}
}

