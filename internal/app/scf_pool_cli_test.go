package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"liao/internal/config"
)

func prepareManagedSCFCLI(t *testing.T, script string) (config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "scfproxy")
	argsFile := filepath.Join(dir, "args.txt")
	pool := filepath.Join(dir, "pool.json")
	packagePath := filepath.Join(dir, "scf-server.zip")
	// This subprocess simulates the CLI, never the cloud SDK. Its result is a
	// real pool file consumed by the normal scfproxy client constructor.
	if err := os.WriteFile(cli, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packagePath, []byte("test package"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"WANFENG_ENABLED": "true", "WANFENG_PROXY_ENABLED": "true", "SCFPROXY_MANAGED": "true",
		"SCFPROXY_CLI_PATH": cli, "SCFPROXY_PACKAGE_PATH": packagePath,
		"SCFPROXY_POOL_FILE": pool, "SCFPROXY_ENDPOINTS_FILE": "", "SCFPROXY_TOKEN": "test-tunnel-token",
		"SCFPROXY_REGIONS": "ap-shanghai,ap-beijing,ap-chengdu", "SCFPROXY_POOL_SIZE": "2",
		"TENCENTCLOUD_SECRET_ID": "test-secret-id", "TENCENTCLOUD_SECRET_KEY": "test-secret-key",
		"SCF_TEST_ARGS": argsFile, "SCF_TEST_POOL_TEMPLATE": writeWanfengTestPool(t, 2),
	} {
		t.Setenv(name, value)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg, argsFile
}

const managedTestCLI = "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$SCF_TEST_ARGS\"\nprintf 'call\\n' >> \"$SCF_TEST_ARGS.calls\"\nif [ ! -f \"$SCFPROXY_POOL_FILE\" ]; then cp \"$SCF_TEST_POOL_TEMPLATE\" \"$SCFPROXY_POOL_FILE\"; fi\n"

func TestManagedSCFPoolBootstrapsUsingCLI(t *testing.T) {
	cfg, argsFile := prepareManagedSCFCLI(t, managedTestCLI)
	_, egress, cleanup, err := newWanfengHTTPClient(cfg)
	if err != nil {
		t.Fatalf("Liao did not initialize an empty pool using the CLI: %v", err)
	}
	defer cleanup()
	if egress == nil || egress.Current() != "ws://127.0.0.1:19001" {
		t.Fatal("initialized pool not connected to proxy")
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "pool\nreplenish\n--pool\n" + cfg.SCFProxyPoolFile + "\n--regions\nap-shanghai,ap-beijing,ap-chengdu\n--size\n2\n--prefix\nscfproxy\n--package\n" + cfg.SCFProxyPackagePath + "\n--yes\n"
	if string(args) != want {
		t.Fatalf("CLI arguments mismatch: %q", args)
	}
	if strings.Contains(string(args), "test-secret") || strings.Contains(string(args), "test-tunnel-token") {
		t.Fatal("credentials passed as command arguments")
	}
}

func waitSCFCLICalls(t *testing.T, path string, count int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(path + ".calls")
		if strings.Count(string(data), "call\n") >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("CLI was not called %d times", count)
}

func TestManagedSCFPoolSwitchTriggersBackgroundCLI(t *testing.T) {
	cfg, argsFile := prepareManagedSCFCLI(t, managedTestCLI)
	_, egress, cleanup, err := newWanfengHTTPClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := egress.Switch(egress.Current()); err != nil {
		t.Fatal(err)
	}
	waitSCFCLICalls(t, argsFile, 2)
	if egress.Current() != "ws://127.0.0.1:19002" {
		t.Fatal("maintenance replaced the new active egress")
	}
}

func TestManagedSCFPoolRetriesAndCancelsRunningCLI(t *testing.T) {
	script := "#!/bin/sh\nmode=normal\nif [ -f \"$SCF_TEST_ARGS.slow\" ]; then mode=slow; elif [ -f \"$SCF_TEST_ARGS.fail\" ]; then mode=fail; fi\n" + strings.TrimPrefix(managedTestCLI, "#!/bin/sh\n") + "case $mode in slow) exec sleep 30;; fail) exit 1;; esac\n"
	cfg, argsFile := prepareManagedSCFCLI(t, script)
	cfg.SCFProxyReconcileIntervalSeconds = 1
	m, err := startSCFPoolCLI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := os.WriteFile(argsFile+".fail", nil, 0600); err != nil {
		t.Fatal(err)
	}
	m.request()
	waitSCFCLICalls(t, argsFile, 2)
	// The periodic reconciliation retries a failed operation without another ban.
	waitSCFCLICalls(t, argsFile, 3)
	if err := os.WriteFile(argsFile+".slow", nil, 0600); err != nil {
		t.Fatal(err)
	}
	m.request()
	waitSCFCLICalls(t, argsFile, 4)
	started := time.Now()
	m.Close()
	if time.Since(started) > 3*time.Second {
		t.Fatal("shutdown did not cancel the running CLI")
	}
}

func TestManagedSCFPoolCommandTimeout(t *testing.T) {
	cfg, _ := prepareManagedSCFCLI(t, "#!/bin/sh\nexec sleep 30\n")
	cfg.SCFProxyCommandTimeoutSeconds = 1
	started := time.Now()
	m, err := startSCFPoolCLI(cfg)
	if err == nil || m != nil || !strings.Contains(err.Error(), "pool replenish") || time.Since(started) > 5*time.Second {
		t.Fatalf("CLI timeout was not enforced: %v", err)
	}
}

func TestManagedSCFPoolRequiresCredentialsAndRedactsCLIOutput(t *testing.T) {
	cfg, argsFile := prepareManagedSCFCLI(t, "#!/bin/sh\nprintf '%s %s %s %s' \"$TENCENTCLOUD_SECRET_ID\" \"$TENCENTCLOUD_SECRET_KEY\" \"$SCFPROXY_TOKEN\" \"$TENCENTCLOUD_SESSION_TOKEN\"\nexit 1\n")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "")
	if m, err := startSCFPoolCLI(cfg); err == nil || m != nil {
		t.Fatal("missing credential was accepted")
	}
	if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
		t.Fatal("CLI ran before credential validation")
	}
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "test-secret-key")
	t.Setenv("TENCENTCLOUD_SESSION_TOKEN", "test-session-token")
	m, err := startSCFPoolCLI(cfg)
	if err == nil || m != nil {
		t.Fatal("CLI failure did not stop startup")
	}
	for _, secret := range []string{"test-secret-id", "test-secret-key", "test-tunnel-token", "test-session-token"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatal("CLI error leaked credentials")
		}
	}
}
