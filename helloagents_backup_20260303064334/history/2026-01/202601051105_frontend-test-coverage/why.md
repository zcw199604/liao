# 变更提案: 前端核心聊天模块测试补全

## 需求背景
当前 `frontend/src/__tests__` 的覆盖重点集中在基础工具、认证流程与页面跳转逻辑；核心聊天业务（消息收发、WebSocket 状态同步）与多数关键 UI 组件缺乏测试保护，导致回归风险较高，且后续重构缺少安全网。

## 变更内容
1. 补充聊天相关核心组件的 SFC 渲染/交互测试，覆盖关键交互与事件。
2. 为核心业务 Composables 编写单元测试，覆盖消息发送、匹配流程与关键状态同步。
3. 补充 chat/message Store 的独立测试，覆盖去重/排序/状态变更等核心逻辑。
4. 补全 cookie/file/media 等工具类测试，覆盖常用边界条件。

## 影响范围
- **模块:** 前端（Vue 3 / Pinia / Composables / Utils）
- **文件:** `frontend/src/__tests__/**`、部分组件/视图文件（如为可测性做小幅调整）
- **API:** 无（仅 mock 调用，不改变接口）
- **数据:** 无

## 核心场景

### 需求: 核心聊天组件具备最小交互保障
**模块:** `frontend/src/components/chat/*`
覆盖输入/发送/表情/上传菜单/消息列表/消息气泡/匹配按钮等关键交互，确保组件 emit 与 UI 状态符合预期。

#### 场景: ChatInput 基础输入与发送
- 输入触发 `update:modelValue`，并触发 typing start/end 节流逻辑
- Enter / Ctrl+Enter 触发发送（受 disabled 与空内容约束）

#### 场景: MessageList 新消息提示与复制/预览
- 非底部时新增消息显示“新消息”按钮；点击回到底部
- 文本双击复制提示；图片点击触发预览事件

### 需求: 核心业务 Composables 可回归
**模块:** `frontend/src/composables/*`

#### 场景: useChat 匹配/收藏/进入聊天
- WebSocket 未连接时匹配失败提示；连接后发送 random/randomOut
- 收藏/取消收藏更新单一数据源与列表 ID，并提示结果
- 进入聊天时按缓存情况选择增量/全量拉取历史

#### 场景: useMessage 文本/图片/视频/输入状态发送
- 构造正确的 `act` 与 `msg`；图片/视频发送同时记录发送关系

#### 场景: useWebSocket 关键消息处理
- 连接成功后设置 ws 状态并发送 sign 消息
- 处理正在输入与匹配成功消息，更新 store 并触发事件

### 需求: Store 逻辑可独立验证
**模块:** `frontend/src/stores/chat.ts`、`frontend/src/stores/message.ts`
- chat：单一数据源 upsert/update/remove 与连续匹配状态
- message：addMessage 去重/排序稳定性、clear/reset 行为

## 风险评估
- **风险:** WebSocket 与浏览器 API（clipboard/location）依赖导致测试不稳定；模块级单例状态导致串扰。
- **缓解:** 使用 Fake WebSocket 与 `vi.useFakeTimers()`；在每个用例中重置全局与 Pinia 状态；尽量用局部 mock 限定影响范围。

