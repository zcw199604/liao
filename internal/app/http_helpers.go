package app

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// requestHostHeader 返回与 Spring @RequestHeader("Host") 等价的 Host 值。
// 注意：Go 标准库会将 Host 放在 r.Host 字段中，而不是 r.Header["Host"]。
func requestHostHeader(r *http.Request) string {
	if r == nil {
		return ""
	}

	// 兼容常见反向代理/Ingress：优先使用外部 Host。
	if v := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); v != "" {
		parts := strings.Split(v, ",")
		if len(parts) > 0 {
			if first := strings.TrimSpace(parts[0]); first != "" {
				return first
			}
		}
		return v
	}

	return strings.TrimSpace(r.Host)
}
