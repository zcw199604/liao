# Changelog

本文件记录项目所有重要变更。
格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/),
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
- 前端：接入 Vitest + jsdom，并为核心模块补充单元测试（utils/time/string、useToast、request、auth store）。
- 前端：补充 Vue 组件级测试（Dialog/Toast/Loading/UserList/ChatSidebar）。
- 前端：补充视图级页面测试（LoginPage/IdentityPicker/ChatListView/ChatRoomView）。
- 前端：补充核心聊天业务测试（useChat/useMessage/useWebSocket、chat/message store）。
- 前端：补充核心聊天与设置/媒体组件测试（ChatInput/MessageList/MessageBubble/EmojiPanel/UploadMenu/MatchButton/SettingsDrawer/MediaPreview）。
- 前端：补充 utils 测试（cookie/file/media）。
- 后端：为 `/api/getHistoryUserList` 与 `/api/getFavoriteUserList` 增加分段耗时日志（上游/补充用户信息/最后消息/总耗时）。

### 修复
- 后端：`/api/getFavoriteUserList` 对齐 `/api/getHistoryUserList` 的缓存增强逻辑，补全用户信息并补齐 `lastMsg/lastTime`。
- 前端：修复聊天列表（侧边栏）最后一条消息时间未使用统一格式化（`formatTime`）的问题。
- 前端：修复 ChatRoomView 预览事件监听清理的生命周期注册，避免 Vue 警告。
- 修复 `/api/getHistoryUserList` 在上游用户ID字段为 `UserID/userid` 时，未能填充 `lastMsg` / `lastTime` 的问题。
- 修复上游返回的消息 `id/toid` 与 `myUserID` 不一致时，最后消息缓存会话Key无法命中导致 `lastMsg` / `lastTime` 缺失的问题。
