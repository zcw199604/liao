# WebSocket Proxy

## 目的
代理客户端与上游匿名聊天 WebSocket，实现连接复用、转发、缓存增强和防重连。

## 模块概述
- **职责:** `/ws` 鉴权、下游 session 注册、上游连接池、消息双向转发、forceout 处理。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: 下游连接必须先 sign
**模块:** WebSocket Proxy  
客户端连接 `/ws?token=...` 后，第一条有效业务消息应包含 `act=sign` 和 `id=userId`。

#### 场景: sign 成功
- 后端注册 session 与 userId。
- 创建或复用该 userId 的上游连接。
- 将 sign 原文转发到上游。

### 需求: 上游连接池
**模块:** WebSocket Proxy  
同一 userId 的多个下游连接共享一条上游连接。

#### 场景: 超过活跃身份上限
- 当前代码限制最多 2 个活跃身份。
- 超出时淘汰最早创建的身份，并发送 `code=-6` 通知。

### 需求: Forceout 防重连
**模块:** WebSocket Proxy  
上游返回 `code=-3` 且 `forceout=true` 时，该 userId 5 分钟内禁止重新 sign。

#### 场景: 禁止期间重新连接
- 后端发送 `code=-4` 拒绝消息，并关闭下游连接。

### 需求: 上游连接不主动 ping
**模块:** WebSocket Proxy  
后端上游 WebSocket 客户端不主动发送 ping frame，避免触发部分上游节点 reset。

#### 场景: 上游空闲保持
- 连接建立后仅依赖业务消息和读循环感知断开。
- 保留 pong 处理用于对端主动心跳时刷新读超时。

## API接口
- `GET /ws?token=<jwt>`
- `GET /api/getConnectionStats`
- `POST /api/disconnectAllConnections`
- `GET /api/getForceoutUserCount`
- `POST /api/clearForceoutUsers`

## 数据模型
无持久表；运行时状态在 `UpstreamWebSocketManager` 和 `ForceoutManager` 中维护。

## 依赖
- `internal/app/websocket_proxy.go`
- `internal/app/websocket_manager.go`
- `internal/app/forceout.go`
- `frontend/src/composables/useWebSocket.ts`

## 运维诊断
- `scripts/probe_upstream_ws.go` 可手动探测上游 WebSocket 稳定性，默认通过 rand-server 发现上游并被动保持连接 5 分钟。
- 默认用法：`go run scripts/probe_upstream_ws.go -timeout=5m`。
- 如需验证上游是否接受客户端 ping：`go run scripts/probe_upstream_ws.go -timeout=5m -ping-interval=25s`。
