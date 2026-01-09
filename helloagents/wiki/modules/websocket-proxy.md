# WebSocket Proxy（/ws 代理）

## 目的
定义下游（前端）与上游聊天服务之间的 WebSocket 代理行为，包括身份绑定、连接池、广播与 forceout 防重连逻辑。

## 模块概述
- **职责:** `/ws` 下游连接管理；按 `userId` 复用/维护上游连接；上游消息广播；forceout 禁止与自动断开
- **状态:** ✅稳定
- **最后更新:** 2026-01-09

## 规范

### 需求: 下游连接与身份绑定
**模块:** WebSocket Proxy
下游连接在建立后：
- 必须先发送 `act="sign"` 且包含 `id`（userId）
- 后端将 session 与该 userId 绑定，并将 sign 消息转发到上游用于登录
- 未完成绑定前的非 sign 消息会被忽略

### 需求: 一人一条上游连接（连接池）
**模块:** WebSocket Proxy
- 同一 `userId` 的多个下游连接共享一条上游连接
- 最大同时活跃身份数：2（FIFO 淘汰最早创建的 userId）

淘汰行为：
- 向被淘汰身份的下游广播：`{"code":-6,"content":"由于新身份连接，您已被自动断开","evicted":true}`
- 1 秒后关闭该身份的上游连接与全部下游连接

### 需求: 延迟关闭上游连接
**模块:** WebSocket Proxy
当某个 `userId` 的最后一个下游断开后，不立即关闭上游连接；延迟 80 秒关闭。延迟期间如有新下游加入则取消关闭任务。

### 需求: Forceout（防重连/封禁）
**模块:** WebSocket Proxy
当上游消息满足 `code=-3 && forceout=true`：
- 将 `userId` 加入禁止列表 5 分钟
- 广播原始 forceout 消息到该 `userId` 全部下游
- 关闭该 `userId` 的上游连接，并在 1 秒后关闭全部下游连接

被禁止 `userId` 再次注册（sign）时：
- 先向下游发送拒绝消息（`code=-4`，包含剩余秒数）并立刻关闭连接

### 需求: 上游消息缓存增强
**模块:** WebSocket Proxy
上游消息在转发给下游前，会尝试解析并执行缓存增强：
- `code=15`：缓存匹配用户信息（UserInfo）
- `code=7`：缓存最后一条消息（LastMessage），并进行会话 key 归一化补写以提升历史/收藏列表命中率

## API接口
### [WS] /ws?token=...
**描述:** 下游 WebSocket 入口（鉴权见 `modules/auth.md`）

## 依赖
- `internal/app/websocket_proxy.go`
- `internal/app/websocket_manager.go`
- `internal/app/forceout.go`
- `internal/app/user_info_cache.go`
- `internal/app/user_info_cache_redis.go`

## 变更历史
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并实现 WS 代理/连接池/forceout

