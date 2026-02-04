package app

import (
	"net"
	"net/http"
	"strings"
)

// Douyin endpoints can be sensitive to header fingerprinting (e.g. User-Agent/Referer),
// and some play URLs may require cookies.
const (
	douyinDefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	douyinDefaultReferer   = "https://www.douyin.com/"
	douyinDefaultOrigin    = "https://www.douyin.com"
)

func effectiveDouyinUserAgent(r *http.Request) string {
	if r == nil {
		return douyinDefaultUserAgent
	}
	if ua := strings.TrimSpace(r.Header.Get("User-Agent")); ua != "" {
		return ua
	}
	return douyinDefaultUserAgent
}

func isDouyinHost(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return false
	}

	// Strip port if present.
	if strings.Contains(h, ":") {
		if hostOnly, _, err := net.SplitHostPort(h); err == nil {
			h = hostOnly
		} else {
			// Best-effort: handle non-bracketed IPv6 or other odd forms.
			if parts := strings.Split(h, ":"); len(parts) > 0 {
				h = parts[0]
			}
		}
	}

	h = strings.ToLower(strings.TrimSpace(h))
	return h == "douyin.com" ||
		strings.HasSuffix(h, ".douyin.com") ||
		h == "iesdouyin.com" ||
		strings.HasSuffix(h, ".iesdouyin.com")
}
