# 任务清单: 修复列表页未读气泡不显示（双保险）

目录: `helloagents/plan/202601062010_fix_unread_badge_list/`

---

## 1. 修复未读计数更新逻辑（方案1）
- [√] 1.1 在 `frontend/src/composables/useWebSocket.ts` 中，只有当用户确实处于 `/chat` 路由时，才把当前会话收到的新消息视为“已读”（`unreadCount=0`）；否则应累加未读数

## 2. 修复会话状态残留（方案2）
- [√] 2.1 在 `frontend/src/views/ChatRoomView.vue` 中，离开聊天页时确保调用 `chatStore.exitChat()`，避免 `currentChatUser` 在列表页残留

## 3. 测试与回归用例
- [√] 3.1 补充/更新 `frontend/src/__tests__/useWebSocket.test.ts`：列表路由下即使 `currentChatUser` 存在，也应累加未读；聊天路由下应清零未读
- [√] 3.2 补充/更新 `frontend/src/__tests__/views.test.ts`：离开聊天页（unmount）后应清空 `currentChatUser`

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9：无敏感信息泄露、无破坏性操作、事件监听清理）

## 5. 知识库同步
- [√] 5.1 更新 `helloagents/wiki/modules/chat-ui.md`：记录“未读计数=路由 + 当前会话”判定规则
- [√] 5.2 更新 `helloagents/CHANGELOG.md`：记录本次修复

## 6. 测试
- [√] 6.1 运行 `cd frontend && npm test`
