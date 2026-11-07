package app

import (
	"context"
	"strings"
	"sync"

	"liao/internal/upstream/wanfeng"
)

// 晚风「按身份隔离的 HTTP 客户端」缓存。
//
// 对方的历史会话/收藏/聊天记录/媒体接口都按 Cookie（wf_identity）认定身份：
//   - Cookie 缺失或与服务端身份不一致时，业务接口统一返回 40300「请求错误，请刷新页面」；
//   - 一个进程里可能同时有多个身份，共用一份 Cookie Jar 会互相覆盖（A 看到 B 的会话）。
//
// 因此这里给每个身份克隆一个带独立 Cookie Jar 的客户端，并在创建时用 identityToken
// 换好身份 Cookie。克隆客户端与原客户端共用同一个加密会话（sid/密钥与身份无关），
// 不会产生额外握手。
type wanfengIdentityClient struct {
	api *wanfeng.APIClient
	// token 是该客户端绑定时使用的 identityToken，对方换发新 token 后需要重建。
	token string
}

// wanfengIdentityClients 按 identityID 缓存专用客户端。
type wanfengIdentityClients struct {
	base *wanfeng.APIClient

	mu sync.Mutex
	m  map[string]*wanfengIdentityClient
}

func newWanfengIdentityClients(base *wanfeng.APIClient) *wanfengIdentityClients {
	return &wanfengIdentityClients{base: base, m: make(map[string]*wanfengIdentityClient)}
}

// Get 返回该身份专用客户端（必要时克隆并绑定身份 Cookie）。
//
// 绑定失败不阻塞：返回客户端让调用方自行处理 40300，避免历史接口整块不可用。
func (c *wanfengIdentityClients) Get(ctx context.Context, identityID string, binding *wanfeng.Binding) *wanfeng.APIClient {
	if c == nil || c.base == nil || binding == nil {
		return nil
	}
	identityID = strings.TrimSpace(identityID)
	token := strings.TrimSpace(binding.IdentityToken)
	if identityID == "" {
		return nil
	}

	c.mu.Lock()
	cached := c.m[identityID]
	c.mu.Unlock()
	if cached != nil && cached.token == token {
		return cached.api
	}

	api := c.base.Clone()
	if api == nil {
		return c.base
	}
	if token != "" {
		// 身份 Cookie 换发：对方按 Cookie 认定身份，失败只降级（后续请求可能仍 40300）。
		_ = api.BindIdentityCookie(ctx, token)
	}

	c.mu.Lock()
	if c.m == nil {
		c.m = make(map[string]*wanfengIdentityClient)
	}
	c.m[identityID] = &wanfengIdentityClient{api: api, token: token}
	c.mu.Unlock()
	return api
}

// Drop 丢弃该身份的缓存客户端（身份 Cookie 失效/重新登录后调用，下次访问会重建）。
func (c *wanfengIdentityClients) Drop(identityID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.m, strings.TrimSpace(identityID))
	c.mu.Unlock()
}
