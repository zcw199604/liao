package app

import (
	"context"
	"net/http"
	"strings"
)

// ProviderHeaderName 是前端声明「当前上游线路」的请求头。
//
// 取值与 /api/provider/list 下发的 id 一致（chat2019 / wanfeng）。
// 之所以必须让前端把当前线路带上来：identity_provider_binding 只增不删（没有解绑），
// 若历史/设置类接口只按「绑定关系」分流，用户从晚风切回 chat2019 之后，
// 这些接口仍会被晚风接管——刷新页面时列表会被晚风侧数据整体替换，
// 于是 chat2019 上的会话（包括刚匹配到的人）全部消失。
const ProviderHeaderName = "X-Liao-Provider"

// providerCtxKey 是当前线路在 context 中的键（空结构体不占内存，也避免跨包撞键）。
type providerCtxKey struct{}

// normalizeProviderID 把任意输入归一成已知线路 id；未知或空返回 ""。
func normalizeProviderID(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ProviderWanfeng:
		return ProviderWanfeng
	case ProviderChat2019:
		return ProviderChat2019
	}
	return ""
}

// providerOfRequest 从请求头（或 query 兜底，便于 curl 调试）解析前端声明的线路。
func providerOfRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if provider := normalizeProviderID(r.Header.Get(ProviderHeaderName)); provider != "" {
		return provider
	}
	return normalizeProviderID(r.URL.Query().Get("provider"))
}

// providerContextMiddleware 把前端声明的当前线路放进 request context，
// 供历史会话/收藏/聊天记录/匹配偏好/位置设置等接口按线路分流。
//
// 前端未声明线路（老客户端）时不写 context，下游沿用「按身份绑定分流」的旧行为，
// 保证向后兼容。
func (a *App) providerContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := providerOfRequest(r)
		if provider == "" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), providerCtxKey{}, provider)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestProviderContext 返回「带上当前线路」的 context。
//
// 中间件已经做过一次注入；这里再做一次兜底，保证直接调用分流函数的路径
// （单元测试、内部调用、将来新增的 handler）也能拿到同样的判断。
func requestProviderContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	ctx := r.Context()
	if providerFromContext(ctx) != "" {
		return ctx
	}
	if provider := providerOfRequest(r); provider != "" {
		return context.WithValue(ctx, providerCtxKey{}, provider)
	}
	return ctx
}

// providerFromContext 读取当前线路；"" 表示前端未声明线路。
func providerFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(providerCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// currentProviderIsChat2019 判断当前请求是否明确走 chat2019 线路。
//
// 返回 true 时，晚风专属的分流/下发逻辑必须全部让路（不能被身份绑定劫持）。
func currentProviderIsChat2019(ctx context.Context) bool {
	return providerFromContext(ctx) == ProviderChat2019
}
