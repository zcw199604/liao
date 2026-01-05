# 任务清单: 前端核心聊天模块测试补全

目录: `helloagents/plan/202601051105_frontend-test-coverage/`

---

## 1. 组件测试（chat/settings/media）
- [√] 1.1 为 `frontend/src/components/chat/ChatInput.vue` 补充交互测试（输入/发送/typing/禁用态）
- [√] 1.2 为 `frontend/src/components/chat/MessageList.vue` 补充测试（loadMore/新消息提示/复制/预览事件）
- [√] 1.3 为 `frontend/src/components/chat/MessageBubble.vue` 补充测试（文本/图片/视频/文件分支）
- [√] 1.4 为 `frontend/src/components/chat/EmojiPanel.vue` / `UploadMenu.vue` 补充 emit 测试
- [√] 1.5 为 `frontend/src/components/chat/MatchButton.vue` 补充测试（短按/长按菜单/取消匹配/外部点击关闭）
- [√] 1.6 为 `frontend/src/components/settings/SettingsDrawer.vue` 补充最小交互测试（打开/关闭/编辑无变更提示）
- [√] 1.7 为 `frontend/src/components/media/MediaPreview.vue` 补充最小交互测试（关闭/切换/重试逻辑）

## 2. composables 单元测试
- [√] 2.1 为 `frontend/src/composables/useChat.ts` 补充测试（匹配/收藏/进入聊天增量策略）
- [√] 2.2 为 `frontend/src/composables/useMessage.ts` 补充测试（文本/图片/视频/输入状态发送）
- [√] 2.3 为 `frontend/src/composables/useWebSocket.ts` 补充关键路径测试（connect/onmessage/send/disconnect）

## 3. Store 单元测试
- [√] 3.1 为 `frontend/src/stores/chat.ts` 补充测试（单一数据源更新/连续匹配状态）
- [√] 3.2 为 `frontend/src/stores/message.ts` 补充测试（addMessage 去重/排序/clear/reset）

## 4. Utils 单元测试
- [√] 4.1 为 `frontend/src/utils/cookie.ts` / `file.ts` / `media.ts` 补充测试（边界条件+核心分支）

## 5. 质量验证
- [√] 5.1 运行 `cd frontend && npm test`
- [√] 5.2 运行 `cd frontend && npm run build`

## 6. 文档与归档
- [√] 6.1 更新 `helloagents/CHANGELOG.md`
- [√] 6.2 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
