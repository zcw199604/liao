package app

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
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

type urlLogInfo struct {
	RawLen    int
	Hash      string
	Scheme    string
	Host      string
	Path      string
	QueryKeys int
}

func buildURLLogInfo(raw string) urlLogInfo {
	raw = strings.TrimSpace(raw)
	info := urlLogInfo{RawLen: len(raw)}
	if raw == "" {
		return info
	}

	sum := sha256.Sum256([]byte(raw))
	// Keep it short for logs; enough to correlate requests without leaking full URL params.
	info.Hash = hex.EncodeToString(sum[:8])

	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return info
	}
	info.Scheme = strings.TrimSpace(u.Scheme)
	info.Host = strings.TrimSpace(u.Host)
	info.Path = strings.TrimSpace(u.Path)
	if strings.TrimSpace(u.RawQuery) != "" {
		if q, err := url.ParseQuery(u.RawQuery); err == nil {
			info.QueryKeys = len(q)
		}
	}
	return info
}

func truncateForLog(raw string, maxLen int) string {
	raw = strings.TrimSpace(raw)
	if maxLen <= 0 || len(raw) <= maxLen {
		return raw
	}
	return raw[:maxLen] + "..."
}

func douyinRefererForDetail(detailID string, mediaType string) string {
	detailID = strings.TrimSpace(detailID)
	if detailID == "" {
		return douyinDefaultReferer
	}

	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "video":
		return "https://www.douyin.com/video/" + detailID
	case "image":
		// Image posts typically use /note/<id> in Douyin web.
		return "https://www.douyin.com/note/" + detailID
	default:
		return douyinDefaultReferer
	}
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
		strings.HasSuffix(h, ".iesdouyin.com") ||
		// Video/image CDN domains may also require cookies (best-effort).
		h == "douyinvod.com" ||
		strings.HasSuffix(h, ".douyinvod.com") ||
		h == "douyinpic.com" ||
		strings.HasSuffix(h, ".douyinpic.com")
}
