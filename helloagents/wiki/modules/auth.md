# Auth（认证与鉴权）

## 目的
定义访问码登录、JWT 生成与校验、HTTP 与 WebSocket 的鉴权拦截规则，确保与历史 Spring Boot 行为一致。

## 模块概述
- **职责:** 访问码登录换取 JWT；`/api/**` Bearer Token 校验；`/ws` 握手 token 校验
- **状态:** ✅稳定
- **最后更新:** 2026-01-09

## 规范

### 需求: 访问码登录换取 JWT
**模块:** Auth
`POST /api/auth/login` 使用 `accessCode`（form）换取 JWT。

行为约束：
- `accessCode` 为空或错误：返回 HTTP 400，`code=-1`
- 登录成功：返回 HTTP 200，`code=0` 且包含 `token`

### 需求: HTTP 鉴权拦截（/api/**）
**模块:** Auth
除以下接口外，所有 `/api/**` 均要求 `Authorization: Bearer <token>`：
- `/api/auth/login`
- `/api/auth/verify`

错误响应（强一致性要求）：
- 缺失 Token：HTTP 401，`{"code":401,"msg":"未登录或Token缺失"}`
- Token 无效/过期：HTTP 401，`{"code":401,"msg":"Token无效或已过期"}`

### 需求: WebSocket 握手鉴权（/ws）
**模块:** Auth
`GET /ws?token=...` 在握手前校验 query 参数 `token`，无效则拒绝连接（HTTP 401）。

### 需求: JWT 规则
**模块:** Auth
- 算法：HS256
- 过期：`TOKEN_EXPIRE_HOURS`（默认 24 小时）
- claims：`sub="user"` + `iat/exp`

## API接口
### [POST] /api/auth/login
**描述:** 访问码登录，返回 JWT Token

### [GET] /api/auth/verify
**描述:** 校验 Bearer Token 是否有效（返回 `valid`）

### [WS] /ws?token=...
**描述:** WebSocket 握手鉴权入口（token 来自 JWT）

## 依赖
- `internal/app/auth_handlers.go`
- `internal/app/jwt.go`
- `internal/app/middleware.go`
- `internal/app/websocket_proxy.go`
- `internal/config/config.go`

## 变更历史
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并对齐鉴权规则

