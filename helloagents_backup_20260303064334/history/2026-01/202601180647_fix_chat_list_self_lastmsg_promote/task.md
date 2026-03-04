# 任务清单: 会话列表在自己发消息时置顶 + lastMsg 预览对齐（轻量迭代）

目录: `helloagents/history/2026-01/202601180647_fix_chat_list_self_lastmsg_promote/`

---

## 1. 会话列表行为
- [√] 1.1 自己发送消息且处于 `/chat` 当前会话时：同步更新 `lastMsg/lastTime`，并将该会话置顶到“消息(历史)”列表
- [√] 1.2 若该会话也在“收藏”列表：同样置顶到“收藏”列表
- [√] 1.3 `lastMsg` 预览对齐后端缓存格式：自己发送加前缀 `我: `

## 2. 测试与验证
- [√] 2.1 补充 `useWebSocket` 单测覆盖：self 回显置顶两列表 + `我: ` 前缀
- [√] 2.2 运行 `cd frontend && npm test`

## 3. 文档同步
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`（会话列表置顶与 lastMsg 预览规则）
- [√] 3.2 更新 `helloagents/CHANGELOG.md`
