package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/zcw199604/scfproxy/pkg/scfproxy"

	"liao/internal/config"
	"liao/internal/upstream/wanfeng"
)

// newWanfengHTTPClient 只装配晚风出口，其他线路继续使用 App.httpClient。
// 返回的清理函数同时关闭 HTTP 空闲连接和活动隧道。
func newWanfengHTTPClient(cfg config.Config) (*http.Client, wanfeng.IPBanEgress, func(), error) {
	client := &http.Client{Timeout: time.Duration(cfg.WanfengTimeoutSeconds) * time.Second}
	if !cfg.WanfengProxyEnabled {
		return client, nil, func() {}, nil
	}
	proxy, err := scfproxy.New(scfproxy.Config{
		PoolFile:      cfg.SCFProxyPoolFile,
		EndpointsFile: cfg.SCFProxyEndpointsFile,
		Token:         cfg.SCFProxyToken,
		Strategy:      scfproxy.StickyByHost,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("初始化晚风 SCF 代理失败: %w", err)
	}
	transport := proxy.Transport()
	client.Transport = transport
	var egress wanfeng.IPBanEgress
	if cfg.SCFProxyPoolFile != "" {
		egress = &wanfengProxyEgress{proxy: proxy}
	}
	slog.Info("晚风 HTTP/WS 已启用 SCF 代理", "pinned", cfg.SCFProxyPoolFile != "")
	return client, egress, func() {
		transport.CloseIdleConnections()
		_ = proxy.Close()
	}, nil
}

// Serialize ban reports and compare the original egress so concurrent/stale
// HTTP and WS failures cannot retire the newly selected healthy standby.
type wanfengProxyEgress struct {
	mu    sync.Mutex
	proxy *scfproxy.Proxy
}

func (e *wanfengProxyEgress) Current() string { return e.proxy.Active().URL }

func (e *wanfengProxyEgress) Switch(expected string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Current() != expected {
		return nil
	}
	old, next, err := e.proxy.Switch("wanfeng: IP_BANNED")
	if err != nil {
		return err
	}
	slog.Warn("晚风封禁 IP，已切换 SCF 出口", "oldIP", old.EgressIP, "newIP", next.EgressIP)
	return nil
}
