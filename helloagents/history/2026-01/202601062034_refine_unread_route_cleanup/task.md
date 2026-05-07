# 任务清单: 未读判定改用路由实例 + 卸载清理简化

目录: `helloagents/plan/202601062034_refine_unread_route_cleanup/`

---

## 1. 必须修改（修复bug）
- [√] 1.1 `frontend/src/composables/useWebSocket.ts`：未读判定改用 `router.currentRoute`（避免仅依赖 `window.location`）
- [√] 1.2 `frontend/src/views/ChatRoomView.vue`：简化 `onUnmounted` 清理逻辑，仅保留必要的事件解绑与 `chatStore.exitChat()`

## 2. 建议优化（提升健壮性）
- [√] 2.1 `frontend/src/__tests__/useWebSocket.test.ts`：用真实路由切换（或 mock pathname）驱动未读判定测试

## 3. 回归测试
- [√] 3.1 运行 `cd frontend && npm test`
