# 任务清单: 聊天手势与弹层交互增强

目录: `helloagents/plan/202601060948_chat_gesture_ux/`

---

## 1. ChatSidebar（列表页/抽屉）交互增强
- [√] 1.1 在 `frontend/src/components/chat/ChatSidebar.vue` 增加左右滑动切换 `history/favorite`，验证 why.md#需求-消息列表滑动切换
- [√] 1.2 在 `frontend/src/components/chat/ChatSidebar.vue` 修复长按更多菜单：长按松手不应被 click 冒泡立即关闭；点击/触摸其它位置隐藏，验证 why.md#需求-长按更多菜单点击外关闭

## 2. PullToRefresh 冲突规避
- [√] 2.1 在 `frontend/src/components/common/PullToRefresh.vue` 增加横滑判定，避免横向滑动误触发下拉刷新，验证 why.md#风险评估

## 3. ChatRoomView 手势增强
- [√] 3.1 在 `frontend/src/views/ChatRoomView.vue` 实现“屏幕左边缘右滑返回列表（/list）”，复用 `handleBack()`，验证 why.md#需求-聊天页边缘右滑返回列表
- [√] 3.2 在 `frontend/src/views/ChatRoomView.vue` 实现“侧边栏抽屉内左滑关闭”（右边缘起手），验证 why.md#需求-侧边栏抽屉滑动关闭

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9: 事件监听清理、无敏感信息泄露、无破坏性操作）

## 5. 知识库同步
- [√] 5.1 新增或更新 `helloagents/wiki/modules/chat-ui.md` 记录手势规范与弹层关闭规则
- [√] 5.2 更新 `helloagents/CHANGELOG.md` 记录本次交互增强

## 6. 测试
- [√] 6.1 运行 `cd frontend && npm test`
- [√] 6.2 运行 `cd frontend && npm run build`
