# Chat UI（聊天界面交互）

## 目的
提供聊天相关页面的交互规范（手势/弹层关闭），确保移动端单手操作体验一致且避免误触。

## 模块概述
- **职责:** 会话列表/聊天页/侧边栏抽屉的手势交互与弹层关闭行为
- **状态:** ✅稳定
- **最后更新:** 2026-01-09

## 规范

### WebSocket: 身份切换时重建下游连接
**模块:** Chat UI
WebSocket 连接在浏览器侧为全局单例；Go 后端会将下游连接与首次 `act=sign` 的 `userId` 绑定，并按该 `userId` 广播上游消息。

因此当身份从 A 切换到 B 时，如果复用旧 WS 连接，会出现：
- 仍能收到发给身份 A 的消息
- 身份 B 发起匹配/聊天请求可能无响应（因为 B 没有下游订阅）

行为约束（前端实现）：
- 同一身份重复进入页面：复用现有连接，不重复建立
- 检测到身份不一致：先断开旧连接，再建立新连接并重新发送 `sign`

### 手势: 会话列表左右滑切换（消息/收藏）
**模块:** Chat UI
在 `ChatSidebar` 的列表区域左右滑动切换 Tab：
- 左滑: `history -> favorite`
- 右滑: `favorite -> history`

交互约束:
- 不影响纵向滚动
- 当长按更多菜单打开时，左右滑将先关闭菜单，不执行切换

### 手势: 聊天页左边缘右滑返回列表
**模块:** Chat UI
在 `ChatRoomView` 从屏幕左边缘起手向右滑，返回 `/list`（复用返回按钮逻辑）。

交互约束:
- 侧边栏抽屉打开时禁用该手势，避免与抽屉交互冲突

### 手势: 侧边栏抽屉左滑关闭
**模块:** Chat UI
在 `ChatRoomView` 抽屉面板内向左滑关闭抽屉。

冲突规避:
- 关闭手势以“抽屉右侧边缘起手”为优先触发区域，降低与列表左右滑切换的冲突

### 弹层: 长按更多菜单点击外关闭
**模块:** Chat UI
在 `ChatSidebar` 长按用户条目触发更多菜单：
- 长按松手后菜单保持打开（不应被 click 冒泡立即关闭）
- 点击/触摸其它位置应关闭菜单
- 点击菜单内部按钮不触发外部关闭（按钮逻辑可主动关闭）

### 组件冲突: 下拉刷新 vs 横向滑动
**模块:** Chat UI
`PullToRefresh` 需在检测到横向滑动优先时退出下拉刷新逻辑，避免误触发与阻塞横向手势。

### 未读: 列表页未读气泡判定（双保险）
**模块:** Chat UI
会话列表的未读气泡由 `user.unreadCount > 0` 决定显示。

消息接收时的未读判定（WebSocket `code=7`）：
- **仅当处于 `/chat` 路由** 且消息属于当前会话时，才将其视为“已读”，更新 `lastMsg/lastTime` 并将 `unreadCount` 置 0
- 其他情况下（例如在 `/list`），收到对方消息应更新 `lastMsg/lastTime` 并对该会话 `unreadCount + 1`

会话状态清理：
- 离开聊天页时应调用 `chatStore.exitChat()`，避免 `currentChatUser` 在列表页残留导致误判“正在聊天中”

## 相关文件
- `frontend/src/components/chat/ChatSidebar.vue`
- `frontend/src/composables/useWebSocket.ts`
- `frontend/src/views/ChatRoomView.vue`
- `frontend/src/components/common/PullToRefresh.vue`

## 变更历史
- [202601060948_chat_gesture_ux](../../history/2026-01/202601060948_chat_gesture_ux/) - 聊天手势与弹层交互增强
- [202601062010_fix_unread_badge_list](../../history/2026-01/202601062010_fix_unread_badge_list/) - 修复列表页未读气泡误判不显示（路由判定 + 会话状态清理双保险）
- [202601062034_refine_unread_route_cleanup](../../history/2026-01/202601062034_refine_unread_route_cleanup/) - 未读判定改用路由实例，并简化聊天页卸载清理逻辑
- [202601092143_ws_identity_switch](../../history/2026-01/202601092143_ws_identity_switch/) - 修复切换身份后 WS 仍绑定旧用户导致匹配无响应/仍收旧消息
