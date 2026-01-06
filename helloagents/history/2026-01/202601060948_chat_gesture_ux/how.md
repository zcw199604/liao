# 技术设计: 聊天手势与弹层交互增强

## 技术方案

### 核心技术
- Vue 3 / TypeScript
- `@vueuse/core`：用于实现手势与点击外关闭（减少手写事件的边界问题）

### 实现要点

#### 1) `ChatSidebar`：左右滑切换“消息/收藏”
- 在 `ChatSidebar` 的列表可交互区域上使用 `useSwipe` 监听横向滑动，在 `onSwipeEnd` 中判定方向：
  - `left`：`history -> favorite`
  - `right`：`favorite -> history`
- 增加阈值（如 50~80px）并要求“水平位移 > 垂直位移”，降低误触
- 手势触发时先 `closeContextMenu()`，避免菜单悬挂

#### 2) 长按更多菜单：点击外隐藏 + 长按释放不立即关闭
- 当前实现存在“长按后松手触发 click 冒泡导致立即关闭”的风险
- 调整 `handleClick` 接口为 `handleClick(user, event)`：
  - 若本次点击由长按触发（`isLongPressHandled`），则在 `handleClick` 内 `event.stopPropagation()` 并直接返回，避免触发列表容器/`document` 的关闭逻辑
- 菜单点击外关闭使用 `onClickOutside(contextMenuRef, closeContextMenu)`（或保留现有 `document` 监听但需规避长按释放的 click 冒泡）

#### 3) `PullToRefresh`：规避横滑误判
- 在 `touchstart` 记录 `startX/startY`
- 在 `touchmove` 中增加“方向优先判定”：
  - 当 `abs(deltaX) > abs(deltaY)` 时，直接取消下拉刷新处理（不 `preventDefault`，允许横向滑动组件接管）
  - 仅当 `deltaY > 0` 且 `abs(deltaY) >= abs(deltaX)` 才进入下拉刷新逻辑

#### 4) `ChatRoomView`：屏幕左边缘右滑返回列表
- 在聊天页容器上监听手势（`useSwipe` 或轻量 touch 逻辑）：
  - 起手点满足 `startX <= edgeThreshold`（如 24px）
  - `deltaX` 达到阈值（如 80px）且 `abs(deltaY)` 不大（如 ≤ 30px）
  - 满足条件触发 `handleBack()`（复用现有退出会话逻辑并跳转 `/list`）
- 当侧边栏抽屉打开（`showSidebar=true`）时禁用该边缘返回手势，避免与抽屉交互冲突

#### 5) 侧边栏抽屉：面板内左滑关闭
- 在抽屉容器上监听左滑关闭：
  - 为避免与抽屉内“左右滑切换 Tab”冲突，限制为“抽屉右侧边缘起手”（例如 `startX >= drawerWidth - 32px`）
  - 满足 `deltaX <= -80px` 且 `abs(deltaY)` 较小则关闭（`showSidebar=false`）

## 安全与性能
- 事件监听在组件 `onMounted/onUnmounted` 生命周期内注册/注销，避免泄漏
- 对 `preventDefault` 仅在确认为纵向下拉刷新意图时调用，避免影响浏览器滚动与可访问性

## 测试与部署
- 前端单测：`cd frontend && npm test`
- 构建验证：`cd frontend && npm run build`
- 手工验证清单：
  - 列表页左右滑切换 Tab（滚动时不误触）
  - 聊天页左边缘右滑返回列表
  - 抽屉内右边缘左滑关闭
  - 长按菜单：长按松手后菜单保持；点击空白处关闭；点击菜单按钮可正常执行

