package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"liao/internal/config"
)

// scfPoolCLI owns one serialized worker. CLI pool transactions preserve creation
// intents on failure, so subsequent reconciliations can safely resume them.
type scfPoolCLI struct {
	cfg        config.Config
	executable string
	ctx        context.Context
	cancel     context.CancelFunc
	requested  chan struct{}
	done       chan struct{}
}

func startSCFPoolCLI(cfg config.Config) (*scfPoolCLI, error) {
	if !cfg.WanfengProxyEnabled || cfg.SCFProxyPoolFile == "" || cfg.SCFProxyEndpointsFile != "" || cfg.SCFProxyPoolSize < 2 || cfg.SCFProxyCommandTimeoutSeconds <= 0 || cfg.SCFProxyReconcileIntervalSeconds <= 0 {
		return nil, fmt.Errorf("SCF 托管配置无效：需要固定池、至少2个节点及正数超时/维护间隔")
	}
	for _, name := range []string{"TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY"} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return nil, fmt.Errorf("SCF 托管模式缺少 %s", name)
		}
	}
	if cfg.SCFProxyToken == "" {
		return nil, fmt.Errorf("SCF 托管模式缺少 SCFPROXY_TOKEN")
	}
	executable, err := exec.LookPath(cfg.SCFProxyCLIPath)
	if err != nil {
		return nil, fmt.Errorf("SCFPROXY_CLI_PATH 不可执行: %w", err)
	}
	info, err := os.Stat(cfg.SCFProxyPackagePath)
	if err != nil {
		return nil, fmt.Errorf("SCFPROXY_PACKAGE_PATH 不可读: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("SCFPROXY_PACKAGE_PATH 必须是部署包文件")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.SCFProxyPoolFile), 0700); err != nil {
		return nil, fmt.Errorf("创建 SCF 代理池目录失败: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &scfPoolCLI{cfg: cfg, executable: executable, ctx: ctx, cancel: cancel, requested: make(chan struct{}, 1), done: make(chan struct{})}
	// Replenish initializes an empty pool and repairs partially completed work;
	// it retains the active egress when an existing pool is already populated.
	if err := m.reconcile(); err != nil {
		cancel()
		return nil, err
	}
	go m.run()
	return m, nil
}

func (m *scfPoolCLI) reconcile() error {
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(m.cfg.SCFProxyCommandTimeoutSeconds)*time.Second)
	defer cancel()
	args := []string{"pool", "replenish", "--pool", m.cfg.SCFProxyPoolFile, "--regions", m.cfg.SCFProxyRegions, "--size", strconv.Itoa(m.cfg.SCFProxyPoolSize), "--prefix", m.cfg.SCFProxyPoolPrefix, "--package", m.cfg.SCFProxyPackagePath, "--yes"}
	cmd := exec.CommandContext(ctx, m.executable, args...)
	cmd.Env = append(os.Environ(), "SCFPROXY_TOKEN="+m.cfg.SCFProxyToken)
	cmd.WaitDelay = 5 * time.Second
	output, err := cmd.CombinedOutput()
	text := string(output)
	for _, secret := range []string{m.cfg.SCFProxyToken, os.Getenv("TENCENTCLOUD_SECRET_ID"), os.Getenv("TENCENTCLOUD_SECRET_KEY"), os.Getenv("TENCENTCLOUD_SESSION_TOKEN")} {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[REDACTED]")
		}
	}
	if err != nil {
		return fmt.Errorf("scfproxy pool replenish 失败: %w；%s", err, strings.TrimSpace(text))
	}
	slog.Info("SCF 代理池维护完成", "output", strings.TrimSpace(text))
	return nil
}

func (m *scfPoolCLI) request() {
	select {
	case <-m.ctx.Done():
		return
	default:
	}
	select {
	case m.requested <- struct{}{}:
	default:
	}
}

func (m *scfPoolCLI) run() {
	defer close(m.done)
	ticker := time.NewTicker(time.Duration(m.cfg.SCFProxyReconcileIntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
		case <-m.requested:
		}
		if m.ctx.Err() != nil {
			return
		}
		if err := m.reconcile(); err != nil && m.ctx.Err() == nil {
			slog.Error("SCF 代理池自动补位失败，将定时重试", "error", err)
		}
	}
}

func (m *scfPoolCLI) Close() { m.cancel(); <-m.done }
