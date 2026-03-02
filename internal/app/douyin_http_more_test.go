package app

import "testing"

func TestDouyinHTTPHelpers_UncoveredBranches(t *testing.T) {
	if got := douyinRefererForDetail("123", "unknown-type"); got != douyinDefaultReferer {
		t.Fatalf("referer=%q, want %q", got, douyinDefaultReferer)
	}

	if isDouyinHost("  ") {
		t.Fatalf("empty host should be false")
	}

	// net.SplitHostPort("douyin.com:80:90") fails, then fallback split should keep "douyin.com".
	if !isDouyinHost("douyin.com:80:90") {
		t.Fatalf("fallback host split should still identify douyin domain")
	}
}

