package config

import "testing"

func TestWanfengProxyEnvironmentConfiguration(t *testing.T) {
	t.Setenv("WANFENG_PROXY_ENABLED", "true")
	t.Setenv("SCFPROXY_POOL_FILE", "/config/pool.json")
	t.Setenv("SCFPROXY_ENDPOINTS_FILE", "")
	t.Setenv("SCFPROXY_TOKEN", "test-only-token")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.WanfengProxyEnabled || cfg.SCFProxyPoolFile != "/config/pool.json" || cfg.SCFProxyToken != "test-only-token" {
		t.Fatal("SCF proxy environment configuration not loaded")
	}
	t.Setenv("WANFENG_PROXY_ENABLED", "")
	t.Setenv("SCFPROXY_POOL_FILE", "")
	t.Setenv("SCFPROXY_ENDPOINTS_FILE", "/config/endpoints.txt")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WanfengProxyEnabled || cfg.SCFProxyEndpointsFile != "/config/endpoints.txt" {
		t.Fatal("proxy switch or endpoints file not loaded")
	}
}
