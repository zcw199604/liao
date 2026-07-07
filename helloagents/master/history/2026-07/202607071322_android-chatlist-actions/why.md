# 变更提案: Android 会话项动作对齐

## 需求背景
Vue 会话侧边栏已经支持对单个会话执行清未读和删除用户等上下文动作。Android 会话列表目前只能打开会话、刷新、跨身份接入和归档搜索，缺少对已有会话的直接管理能力，导致移动端和 Web 端操作闭环不一致。

## 变更内容
1. Android 会话列表为单个会话提供动作入口，支持清除未读。
2. Android 会话列表支持删除会话用户，调用后端 `/api/deleteUpstreamUser`，成功后清理本地会话与消息缓存。
3. 为新增行为补充 Repository、ViewModel 和 Compose 页面测试。

## 范围边界
- **范围内:** Android 会话列表的清未读、删除用户动作；`ChatApiService` 增加 `/deleteUpstreamUser` 契约；Room DAO 增加单会话删除。
- **范围外:** 全局收藏切换、在线状态查询、批量选择删除、聊天页头部收藏切换、真实设备 UI 回归。
- **拆分说明:** 全量 Android/Vue 对齐范围较大，本方案包只覆盖会话项上下文动作中的低风险可验证切片。

## 影响范围
- **模块:** Android Client、Chat API
- **文件:** `ChatListFeature.kt`, `NetworkStack.kt`, `LocalDatabase.kt`, Android chatlist 相关测试
- **API:** `POST /api/deleteUpstreamUser`
- **数据:** `conversation_cache` 和 `message_cache` 的单用户删除，无 schema 版本变更

## 核心场景

### 需求: 清除会话未读
**模块:** Android Client
Android 用户应能在会话列表中对某个会话执行清未读操作，与 Vue 端 `handleClearUnread` 行为保持一致。

#### 场景: 会话存在未读数
用户在 Android 会话列表对未读会话点击清未读。
- 本地 `conversation_cache.unreadCount` 更新为 0。
- 不触发后端请求。
- UI 通过 Room 观察流刷新为无未读标记。

### 需求: 删除会话用户
**模块:** Android Client
Android 用户应能在会话列表中删除某个会话用户，与 Vue 端调用 `deleteUser(myUserId, targetUserId)` 后移除本地会话保持一致。

#### 场景: 后端删除成功
用户确认删除某个会话。
- Android 调用 `POST /api/deleteUpstreamUser`，参数包含当前身份 ID 和目标用户 ID。
- 删除成功后移除本地 `conversation_cache` 对应会话。
- 同时清理 `message_cache` 中该会话历史，避免后续临时接入时显示旧缓存。
- UI 显示删除成功提示。

#### 场景: 后端删除失败
用户确认删除某个会话，但后端返回失败或网络异常。
- 本地会话和消息缓存不被删除。
- UI 显示删除失败提示。

## 风险评估
- **风险:** 删除会话涉及远端和本地缓存一致性，若失败时仍删除本地缓存会造成移动端状态与服务端不一致。
- **缓解:** Repository 先确认后端成功，再删除本地缓存；测试覆盖成功和失败路径。
