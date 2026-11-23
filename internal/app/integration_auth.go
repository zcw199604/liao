package app

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

// integrationAuth 校验只读集成接口的 Bearer Token 与可选 IP allowlist。
// Token 为空时整组路由不会注册，这里仍返回 404 作为兜底。
func (a *App) integrationAuth(next http.Handler) http.Handler {
	expected := []byte(strings.TrimSpace(a.cfg.IntegrationToken))
	allow := a.cfg.IntegrationAllowedIPs

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(expected) == 0 {
			http.NotFound(w, r)
			return
		}
		if len(allow) > 0 && !integrationIPAllowed(a.integrationClientIP(r), allow) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "来源 IP 不允许"})
			return
		}

		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "缺少集成 Token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(token), expected) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "集成 Token 无效"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// integrationClientIP 默认使用 RemoteAddr；只有直接连接来自 INTEGRATION_TRUSTED_PROXIES
// 时才读取转发头，并从右向左跳过可信代理，避免客户端伪造 X-Real-IP/X-Forwarded-For。
func (a *App) integrationClientIP(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	remote := parseIP(host)
	if remote == nil {
		return nil
	}
	trusted := a.cfg.IntegrationTrustedProxies
	if len(trusted) == 0 || !integrationIPAllowed(remote, trusted) {
		return remote
	}
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		for _, part := range reversed(strings.Split(forwarded, ",")) {
			ip := parseIP(part)
			if ip == nil || integrationIPAllowed(ip, trusted) {
				continue
			}
			return ip
		}
		return remote
	}
	if ip := parseIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip
	}
	return remote
}

func reversed(values []string) []string {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
	return values
}

func firstHeaderIP(raw string) net.IP {
	for _, part := range strings.Split(raw, ",") {
		if ip := parseIP(part); ip != nil {
			return ip
		}
	}
	return nil
}

func parseIP(raw string) net.IP {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}

func integrationIPAllowed(ip net.IP, rules []string) bool {
	if ip == nil {
		return false
	}
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(rule); err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}
		if candidate := parseIP(rule); candidate != nil && candidate.Equal(ip) {
			return true
		}
	}
	return false
}
