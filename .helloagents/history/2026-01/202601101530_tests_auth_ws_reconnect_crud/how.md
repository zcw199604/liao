# 技术设计: 测试覆盖补齐（认证/WebSocket/重连/CRUD）

## 技术方案

### 核心技术
- Go: `testing` + `httptest` + `sqlmock` + `gorilla/websocket`
- 前端: `vitest`（jsdom）+ fake timers

### 实现要点
- 后端认证测试：直接调用 handler，覆盖登录与 verify 的关键分支。
- WebSocket Manager 测试：
  - 上游使用 `httptest` + WebSocket Upgrade 模拟，验证 `sign` 队列刷新与连接管理行为。
  - 将 `close/evict/forceout` delay 抽为包级变量以便测试缩短并稳定断言。
  - 使用 `spy` cache 验证 `code=15/7` 的缓存写入与兼容分支。
  - 使用关闭的下游连接模拟僵尸会话，验证广播时自动清理。
- WebSocket Proxy（`/ws`）测试：使用 `httptest` 启动下游 WS 服务，并连接到上游 WS stub，端到端验证握手 token 校验与 sign/消息代理链路。
- `/ws` 绑定一致性：sign 后仅允许转发与已绑定 `userId` 一致的消息；重复 sign 时先解绑旧身份，避免 session 泄漏与跨身份注入。
- `favorite` 测试：用 `sqlmock` 覆盖 service 与 handler 的 CRUD 分支与错误返回。
- `user_history` 测试：为 `http.Client.Transport` 注入自定义 RoundTripper，避免真实上游依赖并覆盖缓存增强/上游失败路径。
- JWT：在密钥缺失时拒绝签发与校验，保持与配置校验一致，避免误配导致测试/运行行为不确定。
 - 前端 `useWebSocket`：使用 fake timers + FakeWebSocket 模拟断线/forceout，验证重连策略与 token 清理行为。

## 安全与性能
- **安全:** 测试不访问生产资源；上游调用全用 stub；不引入任何密钥明文。
- **性能:** 测试通过缩短 delay 与清理连接，避免长时间等待与 goroutine 泄漏。

## 测试与验证
- 运行 Go 测试：`go test ./...`
- 运行前端测试：`npm test`
- 运行前端构建门禁：`npm run build`
