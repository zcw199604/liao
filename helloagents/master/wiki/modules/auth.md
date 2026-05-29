# Auth

## 目的
提供访问码登录、JWT 签发与 HTTP/WebSocket 鉴权。

## 模块概述
- **职责:** 登录、Token 校验、HTTP Bearer 拦截、WS 握手 token 校验。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: 访问码换取 JWT
**模块:** Auth  
客户端通过 `/api/auth/login` 提交 `accessCode`，后端与 `AUTH_ACCESS_CODE` 比对，成功后返回 JWT。

#### 场景: 登录成功
- 返回 `code=0`、`msg=登录成功` 和 `token`。

#### 场景: 登录失败
- 空访问码或错误访问码返回 HTTP 400 与 `code=-1`。

### 需求: API 鉴权
**模块:** Auth  
除中间件明确放行接口外，所有 `/api/**` 请求必须携带 `Authorization: Bearer <token>`。

#### 场景: Token 缺失或无效
- 返回 HTTP 401。
- 前端 Axios 拦截器清除本地 token 并跳转登录页。

## API接口
- `POST /api/auth/login`
- `GET /api/auth/verify`
- `GET /ws?token=...`

## 数据模型
无数据库表；JWT 使用运行时 `JWT_SECRET` 和 `TOKEN_EXPIRE_HOURS`。

## 依赖
- `internal/app/jwt.go`
- `internal/app/middleware.go`
- `internal/app/auth_handlers.go`
- `frontend/src/api/auth.ts`
