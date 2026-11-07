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

func TestManagedSCFProxyEnvironmentConfiguration(t *testing.T) {
	for name, value := range map[string]string{
		"WANFENG_ENABLED": "true", "WANFENG_PROXY_ENABLED": "true", "SCFPROXY_MANAGED": "true",
		"SCFPROXY_POOL_FILE": "/config/pool.json", "SCFPROXY_ENDPOINTS_FILE": "",
		"SCFPROXY_CLI_PATH": "/usr/local/bin/scfproxy", "SCFPROXY_PACKAGE_PATH": "/app/scf-server.zip",
		"SCFPROXY_REGIONS": "ap-shanghai,ap-beijing,ap-chengdu", "SCFPROXY_POOL_SIZE": "3",
		"SCFPROXY_POOL_PREFIX": "scfproxy-liao", "SCFPROXY_COMMAND_TIMEOUT_SECONDS": "300", "SCFPROXY_RECONCILE_INTERVAL_SECONDS": "30",
	} {
		t.Setenv(name, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SCFProxyManaged || cfg.SCFProxyCLIPath != "/usr/local/bin/scfproxy" || cfg.SCFProxyPackagePath != "/app/scf-server.zip" || cfg.SCFProxyRegions != "ap-shanghai,ap-beijing,ap-chengdu" || cfg.SCFProxyPoolSize != 3 || cfg.SCFProxyPoolPrefix != "scfproxy-liao" || cfg.SCFProxyCommandTimeoutSeconds != 300 || cfg.SCFProxyReconcileIntervalSeconds != 30 {
		t.Fatal("managed SCF configuration not loaded")
	}
	for name, value := range map[string]string{"WANFENG_ENABLED": "false", "WANFENG_PROXY_ENABLED": "false", "SCFPROXY_POOL_FILE": "", "SCFPROXY_ENDPOINTS_FILE": "/config/endpoints.txt", "SCFPROXY_POOL_SIZE": "1", "SCFPROXY_COMMAND_TIMEOUT_SECONDS": "0", "SCFPROXY_RECONCILE_INTERVAL_SECONDS": "-1"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, value)
			if _, err := Load(); err == nil {
				t.Fatalf("accepted invalid managed setting: %s", name)
			}
		})
	}
}
