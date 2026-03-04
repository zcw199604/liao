# 任务清单: 修复切换身份后 WS 仍绑定旧用户

目录: `helloagents/plan/202601092143_ws_identity_switch/`

---

## 1. chat-ui（前端 WS 连接管理）
- [√] 1.1 在 `frontend/src/composables/useWebSocket.ts` 中记录当前 WS 绑定身份，并在 `connect()` 中检测身份不一致时重建 WS，验证 why.md#需求-切换身份后正确绑定下游-ws-场景-a--b-后开始匹配
- [√] 1.2 在 `frontend/src/composables/useWebSocket.ts` 中修复 `disconnect(true)` 在无 WS 时遗留手动关闭标记的问题，验证 why.md#需求-切换身份后正确绑定下游-ws
- [√] 1.3 在 `frontend/src/__tests__/useWebSocket.test.ts` 中新增单测：已连接身份 A 时切换到身份 B 再 `connect()` 会创建新连接并发送 B 的 `sign`

## 2. 安全检查
- [√] 2.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`，补充“身份切换会重建下游 WS 并重新 sign”的行为说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次修复

## 4. 测试
- [√] 4.1 执行 `cd frontend && npm test`

## 5. 提交与推送
- [√] 5.1 生成中文 Commit（遵循 Conventional Commits 前缀），并执行 `git push`
