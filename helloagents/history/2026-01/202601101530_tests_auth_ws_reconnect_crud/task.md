# 任务清单: 测试覆盖补齐（认证/WebSocket/重连/CRUD）

目录: `helloagents/history/2026-01/202601101530_tests_auth_ws_reconnect_crud/`

---

## 1. 后端 P0（认证 + WebSocket 核心）
- [√] 1.1 新增 `internal/app/auth_handlers_test.go`，覆盖登录/验签关键分支
- [√] 1.2 新增 `internal/app/websocket_manager_test.go`，覆盖 sign 刷新、延迟关闭、淘汰、缓存写入与 forceout 标记
- [√] 1.3 新增 `internal/app/websocket_proxy_test.go`，覆盖 `/ws` 握手鉴权与 sign/消息代理到上游的关键路径
- [√] 1.4 调整 `internal/app/websocket_proxy.go`：sign 绑定后仅转发同 `userId` 消息，并在重复 sign（切换身份）时自动解绑旧身份
- [√] 1.5 调整 `internal/app/websocket_manager.go`：将关键 delay 抽为可覆盖变量，提升测试可控性
- [√] 1.6 调整 `internal/app/jwt.go`：密钥缺失时拒绝签发/校验 Token，确保错误分支可回归
- [√] 1.7 补充 `internal/app/websocket_manager_test.go`：广播到下游时自动清理僵尸连接

## 2. 前端 P1（useWebSocket 断线重连）
- [√] 2.1 更新 `frontend/src/__tests__/useWebSocket.test.ts`：补充断线自动重连与手动断开不重连用例
- [√] 2.2 更新 `frontend/src/__tests__/useWebSocket.test.ts`：补充 forceout 后清理 token 且不再重连用例

## 3. 后端 P2（user_history + favorite CRUD）
- [√] 3.1 新增 `internal/app/favorite_test.go`：覆盖收藏服务核心分支
- [√] 3.2 新增 `internal/app/favorite_handlers_test.go`：覆盖收藏接口成功/失败/查询分支
- [√] 3.3 新增 `internal/app/user_history_handlers_test.go`：覆盖历史/收藏列表代理的上游失败与缓存增强路径

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9：无生产环境操作、无敏感信息硬编码、无破坏性命令）

## 5. 文档更新
- [√] 5.1 更新 `helloagents/CHANGELOG.md`
- [√] 5.2 更新 `helloagents/history/index.md`

## 6. 测试
- [√] 6.1 运行 Go 测试：`go test ./...`
- [√] 6.2 运行前端测试：`npm test`
- [√] 6.3 运行前端构建：`npm run build`
