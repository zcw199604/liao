package app

import (
	"log/slog"
	"net/http"
	"strings"
)

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		if reqHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *App) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		switch r.URL.Path {
		// 说明：/api/getMtPhotoThumb 由 <img> 直接请求，浏览器无法附带 Authorization 头；
		// 因此该接口需要放行，具体安全约束由 handler 内部的 size 白名单等策略兜底。
		//
		// 说明：/api/douyin/download 也会被 <img>/<video> 直接请求用于预览；抖音 CDN 对跨站媒体子资源有校验，
		// 必须经由本服务代请求（Referer/User-Agent 等）才能稳定预览，因此同样需要放行。
		// 安全性依赖：key 为随机值且有过期时间，且只能通过已鉴权的 detail 接口生成。
		case "/api/auth/login", "/api/auth/verify", "/api/getMtPhotoThumb", "/api/douyin/download":
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			slog.Warn("请求缺少Token", "method", r.Method, "path", r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"code": 401,
				"msg":  "未登录或Token缺失",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if !a.jwt.ValidateToken(tokenString) {
			slog.Warn("Token验证失败", "method", r.Method, "path", r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"code": 401,
				"msg":  "Token无效或已过期",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
