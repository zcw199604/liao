# 任务清单: 修复 WS 回显消息方向判定（自己消息显示在右侧）

目录: `helloagents/plan/202601101526_fix_ws_self_echo_alignment/`

---

## 1. 前端：私信消息方向判定
- [√] 1.1 修复 `frontend/src/composables/useWebSocket.ts` 中 `code=7` 的 `isSelf` 推断：兼容上游 `fromuser.id/touser.id` 存在别名或不回传本地 userId 的情况
- [√] 1.2 修复 `frontend/src/composables/useWebSocket.ts` 中 `shouldDisplayById` 的会话归属判定：仅当消息涉及当前会话 `peerId` 时才视为“当前聊天中”

## 2. 测试
- [√] 2.1 新增单测覆盖：from/to id 正常、别名 id、昵称兜底、已知用户兜底

## 3. 文档与变更记录
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`，补充“WS 私信回显方向判定”的规则说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次修复

## 4. 验证
- [√] 4.1 执行 `cd frontend && npm test`
- [√] 4.2 执行 `cd frontend && npm run build`

## 5. 方案包迁移
- [√] 5.1 迁移方案包至 `helloagents/history/2026-01/202601101526_fix_ws_self_echo_alignment/` 并更新 `helloagents/history/index.md`
