# Chat UI（聊天界面交互）

## 目的
提供聊天相关页面的交互规范（手势/弹层关闭），确保移动端单手操作体验一致且避免误触。

## 模块概述
- **职责:** 会话列表/聊天页/侧边栏抽屉的手势交互与弹层关闭行为
- **状态:** ✅稳定
- **最后更新:** 2026-01-19

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
- 无论是否达到切换阈值，松手后列表都必须回弹复位到 `translateX(0)`，避免残留偏移卡住

实现约定:
- `useSwipeAction.onSwipeEnd`：仅在超阈值时触发（用于 Tab 切换等业务动作）
- `useSwipeAction.onSwipeFinish`：手势结束必触发（用于位移复位/收尾清理）

### 手势: 聊天页左边缘右滑返回列表
**模块:** Chat UI
在 `ChatRoomView` 从屏幕左边缘起手向右滑，返回 `/list`（复用返回按钮逻辑）。

交互约束:
- 侧边栏抽屉打开时禁用该手势，避免与抽屉交互冲突
- 无论是否触发返回，松手后都必须回弹复位到 `translateX(0)`，避免残留偏移

### 手势: 侧边栏抽屉左滑关闭
**模块:** Chat UI
在 `ChatRoomView` 抽屉面板内向左滑关闭抽屉。

冲突规避:
- 关闭手势以“抽屉右侧边缘起手”为优先触发区域，降低与列表左右滑切换的冲突
- 无论是否触发关闭，松手后都必须收敛到 `translateX(0)`，避免抽屉残留偏移

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

会话列表顺序（置顶）：
- 无论是收到对方消息，还是自己发送消息的回显（`isSelf=true`），只要能定位到会话用户，都应将该会话置顶到“消息(历史)”列表
- 若该会话也在“收藏”列表中，同样置顶到“收藏”列表（仅对已收藏的会话生效，避免把非收藏用户插入收藏列表）

lastMsg 预览（与后端缓存增强对齐）：
- 自己发送消息（`isSelf=true`）：`lastMsg` 增加前缀 `我: ` 以便在列表中区分方向

会话状态清理：
- 离开聊天页时应调用 `chatStore.exitChat()`，避免 `currentChatUser` 在列表页残留导致误判“正在聊天中”

### 消息方向: 自己发送消息回显（isSelf 推断）
**模块:** Chat UI
上游私信消息（`code=7`）的左右方向判定需与 `randomdeskry.js` 保持一致：通过 `fromuser.id` 与 `md5(user_id)` 的一致性来判定“是否为自己发送”。

前端实现（`frontend/src/composables/useWebSocket.ts` → `inferWsPrivateMessageIsSelf`）：
- `isSelf = (lowercase(fromuser.id) === md5Hex(currentUserId))`
- 暂不增加昵称/peerId 等兜底推断，避免与上游脚本判定不一致

### 发送体验: 乐观发送（Optimistic UI）
**模块:** Chat UI
发送消息时不等待服务端回显，前端先插入一条“本地临时消息”以实现秒级反馈；当收到 WebSocket 回显后再合并更新，避免重复渲染。

核心字段（`frontend/src/types/message.ts`）：
- `clientId`: 前端生成的本地消息标识
- `sendStatus`: `sending | sent | failed`
- `sendError`: 失败原因（用于兜底提示）
- `optimistic`: 是否为乐观消息（仅前端字段）

行为约束：
- 发送（`frontend/src/composables/useMessage.ts`）：`sendText/sendImage/sendVideo` 会先插入 `sendStatus=sending` 的本地消息，并启动超时计时器（默认 15s）
- 失败：
  - WebSocket 未连接（`useWebSocket.send()` 返回 `false`）：立即标记为 `failed`
  - 超时未回显：标记为 `failed` 并填充 `sendError=发送超时`
- 回显合并（`frontend/src/stores/message.ts` → `confirmOutgoingEcho`）：收到 `isSelf=true` 的回显消息时，按“内容/媒体路径 + 时间窗口(30s)”匹配最近的 `sending/failed` 本地消息并更新为 `sent`，避免列表出现两条相同消息
- UI 展示（`frontend/src/components/chat/MessageList.vue`）：自己发送的消息在 `sending/failed` 时显示状态文案；失败时提供“重试”按钮（调用 `useMessage.retryMessage`）

### 加载体验: 骨架屏（Skeleton Loading）
**模块:** Chat UI
为减少历史消息/用户列表加载时的布局跳动，统一使用灰色脉冲骨架块进行占位。

应用点：
- 消息列表：首次加载历史记录且暂无消息时，渲染若干“气泡骨架”（`frontend/src/components/chat/MessageList.vue`）
- 侧边栏/收藏列表：加载中渲染“头像 + 昵称”骨架条目（`frontend/src/components/chat/ChatSidebar.vue`、`frontend/src/components/settings/GlobalFavorites.vue`）

组件：
- 通用骨架组件：`frontend/src/components/common/Skeleton.vue`

### 性能: 消息列表虚拟滚动（Virtual Scrolling）
**模块:** Chat UI
为避免长对话渲染过多 DOM 导致滚动卡顿，消息列表改为仅渲染视口可见项。

实现：
- 依赖：`vue-virtual-scroller`
- 组件：`frontend/src/components/chat/MessageList.vue` 使用 `DynamicScroller/DynamicScrollerItem` 渲染（保持“查看历史消息/回到底部/新消息提示”等交互一致）
- 样式：`frontend/src/main.ts` 引入 `vue-virtual-scroller` CSS

### 稳定性: 媒体加载/失败导致的贴底滚动抖动治理
**模块:** Chat UI
聊天消息中存在图片/视频时，媒体加载/失败会触发消息项高度变化；若同时存在“贴底滚动”与平滑滚动动画，容易出现“抖动/回弹”的视觉问题。

行为约束（前端实现）：
- **系统触发贴底（新消息/布局变化/媒体 load/error）：** 使用 `behavior: 'auto'`（避免与布局变化产生 scroll fighting）
- **用户触发贴底（点击“回到底部/新消息”按钮）：** 可使用 `behavior: 'smooth'`（仅交互路径允许平滑）
- **合并滚动请求：** 同一帧内多次贴底请求需合并为一次（`requestAnimationFrame` + `nextTick`），避免短时间重复滚动
- **仅在用户位于底部时自动贴底：** 通过 `MessageList.getIsAtBottom()` 判定；用户阅读历史消息时不自动打断
- **媒体占位：** `ChatMedia` 支持 `aspectRatio` 预占位减少 CLS；未知尺寸时使用默认占位比例（image=4/3, video=16/9），并在 `load/error/loadeddata` 触发 `layout` 事件供列表触发一次贴底
- **滚动锚定：** 消息列表滚动容器显式启用 `overflow-anchor: auto`，减少内容尺寸变化导致的可视区域跳动

### 移动端键盘: 避免遮挡最新消息
**模块:** Chat UI
移动端（尤其 iOS Safari）弹出软键盘时，`100vh` 容器高度可能不会收缩，导致聊天页底部区域被键盘覆盖；同时由于滚动容器高度变小但滚动位置未同步更新，最新消息会“掉到视口外”。

治理策略（前端实现）：
- **动态视口高度：** `frontend/src/main.ts` 通过 `visualViewport.height`（无则回退 `innerHeight`）写入 CSS 变量 `--app-height`；`frontend/src/index.css` 让 `.page-container` 使用 `height: var(--app-height, 100dvh)`
- **视口变化保持贴底：** `frontend/src/components/chat/MessageList.vue` 使用 `ResizeObserver` 监听滚动容器尺寸变化；仅当用户原本位于底部时触发一次 `scrollToBottom()`，避免打断用户阅读历史消息

### 媒体消息: 端口策略（全局配置）
**模块:** Chat UI
聊天消息中的媒体以 `[path]` 形式出现（`useMessage.sendImage/sendVideo` 发送），前端在展示时需将其拼接为 `http://{imgServer}:{port}/img/Upload/{path}`。

为避免端口写死导致“媒体打不开”，前端需按全局系统配置解析端口（图片/视频共用）：
- 配置来源：`/api/getSystemConfig`（Settings 面板可保存到 DB）
- 解析接口：`/api/resolveImagePort`（后端按 `fixed/probe/real` 返回端口）
- 接入点：
  - WS 收消息：`frontend/src/composables/useWebSocket.ts`
  - 历史聊天记录解析：`frontend/src/stores/message.ts`（`loadHistory`）

约束：
- 图片/视频/文件统一使用策略解析端口（`/api/resolveImagePort`），再拼接访问地址。

### 媒体消息: 图文混排（文字 + `[path]`）
**模块:** Chat UI
消息内容允许在同一条消息中混排文字与媒体占位符（例如 `喜欢吗[2026/01/18/Random/xxx.jpg]`），前端需要在渲染层同时展示文字与媒体。

解析规则（前端实现）：
- 扫描消息内容中的所有 `[...]` 片段，按顺序拆分为“文本段 + 媒体段”进行渲染
- 表情兼容：若 `[...]` 完整命中 `emojiMap`（如 `[doge]`），按文本处理，不识别为媒体
- 媒体识别保护：仅当 `[...]` 内内容满足“无 `://` 且扩展名可识别/合理”时才视为媒体路径
- URL 拼接：对媒体段的 `path` 仍按“端口策略”规则生成可访问 URL

会话列表预览（lastMsg）规则：
- 文本部分按 30 字截断
- 若包含媒体段，则在末尾追加标签（优先级：`[图片]` > `[视频]` > `[文件]`），例如 `喜欢吗 [图片]`

接入点：
- WS 收消息：`frontend/src/composables/useWebSocket.ts`
- 历史聊天记录：`frontend/src/stores/message.ts`（`loadHistory`）
- 历史预览弹窗：`frontend/src/components/chat/ChatHistoryPreview.vue`
- 复用工具：`frontend/src/utils/messageSegments.ts`

### 媒体消息: 去重策略（WS 推送 vs 历史拉取）
**模块:** Chat UI
由于上游 `Tid` 可能缺失或在 WS/历史接口中不一致，前端需在消息存储侧做一次“语义去重”，避免同一媒体消息短时间内重复渲染（表现为“同一张图出现两条”）。

语义去重规则（前端实现）：
- key: `remotePath + isSelf + 时间窗口(5s)`（remotePath 从 `[path]` 或 `/img/Upload/{path}` URL 中提取）
- 适用: 图片/视频/文件（不包含表情文本，如 `[doge]`）

## 相关文件
- `frontend/src/components/chat/ChatSidebar.vue`
- `frontend/src/components/chat/MessageList.vue`
- `frontend/src/components/chat/ChatMedia.vue`
- `frontend/src/components/common/Skeleton.vue`
- `frontend/src/composables/useMessage.ts`
- `frontend/src/composables/useWebSocket.ts`
- `frontend/src/views/ChatRoomView.vue`
- `frontend/src/components/common/PullToRefresh.vue`
- `frontend/src/stores/message.ts`
- `frontend/src/types/message.ts`
- `frontend/src/stores/systemConfig.ts`
- `frontend/src/components/settings/SystemSettings.vue`

## 变更历史
- [202601060948_chat_gesture_ux](../../history/2026-01/202601060948_chat_gesture_ux/) - 聊天手势与弹层交互增强
- [202601062010_fix_unread_badge_list](../../history/2026-01/202601062010_fix_unread_badge_list/) - 修复列表页未读气泡误判不显示（路由判定 + 会话状态清理双保险）
- [202601062034_refine_unread_route_cleanup](../../history/2026-01/202601062034_refine_unread_route_cleanup/) - 未读判定改用路由实例，并简化聊天页卸载清理逻辑
- [202601092143_ws_identity_switch](../../history/2026-01/202601092143_ws_identity_switch/) - 修复切换身份后 WS 仍绑定旧用户导致匹配无响应/仍收旧消息
- [202601101526_fix_ws_self_echo_alignment](../../history/2026-01/202601101526_fix_ws_self_echo_alignment/) - 修复 WS 私信回显自己消息方向判定（避免自己消息显示在左侧）
- [202601102319_image_port_strategy](../../history/2026-01/202601102319_image_port_strategy/) - 聊天/历史消息的图片端口改为配置驱动解析，并在 Settings 提供切换
- [202601171004_fix_chat_media_dedup](../../history/2026-01/202601171004_fix_chat_media_dedup/) - 修复聊天记录媒体消息偶发重复显示（WS/历史合并语义去重）
- [202601181746_chat_ux_upgrade](../../history/2026-01/202601181746_chat_ux_upgrade/) - 聊天体验升级：乐观发送/骨架屏/虚拟滚动/ChatMedia
- [202601191052_fix_chat_mobile_keyboard_cover](../../history/2026-01/202601191052_fix_chat_mobile_keyboard_cover/) - 修复移动端键盘弹出导致最新消息被遮挡（动态视口高度 + ResizeObserver 贴底）
- [202601190956_fix_chat_scroll_jitter](../../history/2026-01/202601190956_fix_chat_scroll_jitter/) - 修复聊天页媒体加载/失败导致的贴底滚动抖动（auto/smooth 分离 + 合并滚动 + 媒体 layout 事件）
## 匹配行为

- 聊天侧边栏的匹配按钮（`MatchButton`）点击后调用 `startContinuousMatch(1)`，匹配成功不会自动进入聊天；需在 `MatchOverlay` 中手动点击“进入聊天”。
