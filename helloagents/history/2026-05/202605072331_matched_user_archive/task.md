# 任务清单: 匹配未聊天用户归档

目录: `helloagents/plan/202605072331_matched_user_archive/`

---

## 0. 方案边界确认
- [√] 0.1 确认本次任务仅覆盖 why.md 的范围内切片，范围外内容不进入实现
- [√] 0.2 确认 how.md 的设计边界完整，尤其是模块职责、接口契约、数据边界和依赖边界
- [√] 0.3 大型项目确认最小改动策略: 不做无关重构、目录搬迁、依赖升级或公共API重命名

---

## 1. 后端匹配归档
- [√] 1.1 RED: 在 `internal/app/websocket_manager_test.go` 增加 `code=15` 会保存匹配用户归档的失败测试
- [√] 1.2 GREEN: 在 `internal/app/websocket_manager.go` 接入 `UserArchiveService` 并保存匹配用户快照
- [√] 1.3 VERIFY: 运行 `go test ./internal/app -run TestUpstreamWebSocketClient_OnMessage`

## 2. 前端身份隔离
- [√] 2.1 RED: 在 `frontend/src/__tests__/chat-store-load.test.ts` 增加切换 owner 时清空旧列表的失败测试
- [√] 2.2 GREEN: 在 `frontend/src/stores/chat.ts` 增加列表 owner guard，并在加载列表时应用
- [√] 2.3 VERIFY: 运行 `npm run test -- chat-store-load`

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/user-history.md`
- [√] 4.2 更新 `helloagents/wiki/modules/chat-ui.md`
- [√] 4.3 更新 `helloagents/wiki/modules/identity.md`
- [√] 4.4 更新 `helloagents/CHANGELOG.md`

## 5. 最终验证
- [X] 5.1 VERIFY: 运行 `go test ./...`
> 备注: 全量 Go 测试在当前 Windows 环境失败，失败集中在既有 ffmpeg/ffprobe/sh 不在 PATH、静态目录解析和文件系统错误路径用例；相关最小测试已通过。
- [√] 5.2 VERIFY: 运行 `npm run build`

## 执行总结
- 相关后端测试通过: `go test ./internal/app -run TestUpstreamWebSocketClient_OnMessage`
- 相关前端测试通过: `npm run test -- chat-store-load`
- 相关侧栏测试通过: `npm run test -- chat-sidebar-more`
- 相关 WebSocket 前端测试通过: `npm run test -- useWebSocket`
- 前端构建通过: `npm run build`
