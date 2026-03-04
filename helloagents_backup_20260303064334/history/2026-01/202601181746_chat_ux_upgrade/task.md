# 任务清单: 聊天体验优化（乐观发送 / 骨架屏 / 虚拟滚动 / 媒体组件化）

目录: `helloagents/plan/202601181746_chat_ux_upgrade/`

---

## 1. 乐观发送（Optimistic UI）
- [√] 1.1 在 `frontend/src/types/message.ts` 扩展 `ChatMessage`（`clientId/sendStatus/...`），验证 why.md#需求-乐观发送-optimistic-send-文本发送即时渲染--回显确认
- [√] 1.2 在 `frontend/src/stores/message.ts` 增加“本地消息更新/状态流转/超时失败”能力，验证 why.md#需求-乐观发送-optimistic-send-发送失败超时后重试，依赖任务1.1
- [√] 1.3 在 `frontend/src/composables/useMessage.ts` 实现发送时插入 `sending` 消息 + 超时计时器 + 重试入口，验证 why.md#需求-乐观发送-optimistic-send-文本发送即时渲染--回显确认，依赖任务1.2
- [√] 1.4 在 `frontend/src/composables/useWebSocket.ts` 实现回显确认与合并（更新本地消息为 `sent`，避免重复追加），验证 why.md#需求-乐观发送-optimistic-send-文本发送即时渲染--回显确认，依赖任务1.2
- [√] 1.5 在 `frontend/src/components/chat/MessageList.vue`（及/或 `frontend/src/components/chat/MessageBubble.vue`）展示 `sending/failed` 状态与“重试”按钮，验证 why.md#需求-乐观发送-optimistic-send-发送失败超时后重试，依赖任务1.3

## 2. 骨架屏加载（Skeleton Loading）
- [√] 2.1 新增 `frontend/src/components/common/Skeleton.vue`（通用灰色脉冲块），验证 why.md#需求-骨架屏加载-skeleton-loading-首次加载聊天历史
- [√] 2.2 在 `frontend/src/components/chat/MessageList.vue` 首次加载历史时渲染“消息气泡骨架”，验证 why.md#需求-骨架屏加载-skeleton-loading-首次加载聊天历史，依赖任务2.1
- [√] 2.3 在 `frontend/src/components/chat/ChatSidebar.vue` 增加首次加载骨架条目（头像+昵称），验证 why.md#需求-骨架屏加载-skeleton-loading-加载侧边栏收藏列表，依赖任务2.1
- [√] 2.4 在 `frontend/src/components/settings/GlobalFavorites.vue` 使用骨架条目替代 Spinner（如适用），验证 why.md#需求-骨架屏加载-skeleton-loading-加载侧边栏收藏列表，依赖任务2.1

## 3. 消息列表虚拟滚动（Virtual Scrolling）
- [√] 3.1 在 `frontend/package.json` 引入 `vue-virtual-scroller` 并完成样式引入（如需要），验证 why.md#需求-消息列表虚拟滚动-virtual-scrolling-超长对话滚动保持流畅
- [√] 3.2 重构 `frontend/src/components/chat/MessageList.vue` 使用 `DynamicScroller` 渲染消息，保持“回到底部/新消息/加载更多”交互一致，验证 why.md#需求-消息列表虚拟滚动-virtual-scrolling-超长对话滚动保持流畅，依赖任务3.1

## 4. 媒体组件化（ChatMedia）
- [√] 4.1 新增 `frontend/src/components/chat/ChatMedia.vue`（加载占位/懒加载/错误兜底/比例占位），验证 why.md#需求-媒体组件化-chatmedia-图片加载占位--懒加载--错误兜底
- [√] 4.2 在 `frontend/src/components/chat/MessageList.vue`（以及 `frontend/src/components/chat/MessageBubble.vue` 如需保持一致）替换图片/视频渲染为 `ChatMedia`，验证 why.md#需求-媒体组件化-chatmedia-图片加载占位--懒加载--错误兜底，依赖任务4.1

## 5. 测试
- [√] 5.1 在 `frontend/src/__tests__/composables.test.ts` 补充“乐观发送插入/超时失败/重试”单测，验证 why.md#需求-乐观发送-optimistic-send-发送失败超时后重试
- [√] 5.2 在 `frontend/src/__tests__/useWebSocket.test.ts` 补充“回显合并更新本地消息”单测，验证 why.md#需求-乐观发送-optimistic-send-文本发送即时渲染--回显确认

## 6. 安全检查
- [√] 6.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 7. 文档更新
- [√] 7.1 更新 `helloagents/wiki/modules/chat-ui.md`（描述消息状态、骨架屏与虚拟滚动、ChatMedia 组件），并在 `helloagents/CHANGELOG.md` 记录变更
