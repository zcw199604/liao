# 任务清单: 修复聊天媒体消息重复显示

目录: `helloagents/history/2026-01/202601171004_fix_chat_media_dedup/`

---

## 1. 去重策略（前端消息存储）
- [√] 1.1 在 `frontend/src/stores/message.ts` 中为媒体消息增加语义去重 Key（remotePath + isSelf + 时间窗口），避免 WS/历史合并时重复显示
- [√] 1.2 在 `frontend/src/stores/message.ts` 中将增量历史合并的“临时消息清理”由 content 对比改为 remotePath 对比

## 2. 测试
- [√] 2.1 在 `frontend/src/__tests__/stores.test.ts` 中补充单元测试：同一 remotePath、不同 tid、时间接近时仅保留一条

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`：补充媒体消息去重规则说明

## 4. 变更记录
- [√] 4.1 更新 `helloagents/CHANGELOG.md`：记录修复项

## 5. 方案包归档
- [√] 5.1 迁移方案包到 `helloagents/history/2026-01/202601171004_fix_chat_media_dedup/` 并更新 `helloagents/history/index.md`
