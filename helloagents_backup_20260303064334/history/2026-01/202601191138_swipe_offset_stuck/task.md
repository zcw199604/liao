# 任务清单: 修复列表横滑偏移卡住

目录: `helloagents/history/2026-01/202601191138_swipe_offset_stuck/`

---

## 1. Swipe 交互封装
- [√] 1.1 在 `frontend/src/composables/useInteraction.ts` 扩展 `SwipeActionOptions`，新增 `onSwipeFinish(deltaX, deltaY, isTriggered)`（可选），并在手势结束时无条件调用（先 `onSwipeEnd` 后 `onSwipeFinish`）；验证 why.md#需求-列表横滑后不应残留偏移-场景-阈值内横滑
- [√] 1.2 保持 `onSwipeEnd` 的阈值语义不变（仅超阈值触发方向回调），确保现有手势行为不回归；验证 why.md#需求-列表横滑后不应残留偏移-场景-阈值外横滑
- [√] 1.3 在 `frontend/src/views/ChatRoomView.vue` 为边缘手势补充 `onSwipeFinish` 兜底复位，避免极端情况下残留偏移卡住

## 2. ChatSidebar 复位逻辑
- [√] 2.1 在 `frontend/src/components/chat/ChatSidebar.vue` 将“回弹复位”迁移到 `onSwipeFinish`，确保阈值内横滑也能复位；验证 why.md#需求-列表横滑后不应残留偏移-场景-阈值内横滑
- [√] 2.2 增加复位 timer 清理，避免连续滑动导致 `isAnimating` 状态错乱；验证 why.md#需求-列表横滑后不应残留偏移-场景-阈值外横滑
- [√] 2.3 抽取回弹动画时长常量并与 `transition` 对齐，避免后续修改出现不同步
- [√] 2.4 `onSwipeFinish` 增加上下文菜单场景的 guard：确保收敛到 0 偏移且不触发动画

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/chat-ui.md`，补充“阈值内横滑也会在手势结束复位”的约定与注意事项
- [√] 4.2 更新 `helloagents/CHANGELOG.md`，记录本次交互修复

## 5. 测试
- [√] 5.1 新增/调整 `frontend/src/__tests__/useInteraction.test.ts`：覆盖阈值内/外两条路径，验证 `onSwipeFinish` 必触发、`onSwipeEnd` 仍受 threshold 控制，并校验 `isTriggered` 与调用顺序
- [√] 5.2 补充 `frontend/src/__tests__/chatSidebarSwipe.test.ts`：覆盖 `onSwipeFinish` 触发后的回弹复位与菜单 guard 行为
- [√] 5.3 执行 `cd frontend && npm test`
- [√] 5.4 执行 `cd frontend && npm run build`

## 6. 收尾
- [√] 6.1 一致性审计（代码/文档/知识库同步）
- [√] 6.2 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
