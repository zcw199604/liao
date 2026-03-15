# Android / Web 能力对齐实施设计

## 1. 设计概述

本方案以 **Web 当前行为为主基准**，推动 Android 端完成主链路与扩展能力的统一。

整体策略不是简单“逐页补 UI”，而是先建立 Android 侧可承载 Web 行为的基础结构，再将能力逐层接入：

1. **统一状态模型**
   - 认证 / 当前身份 / 会话列表 / 消息 / 收藏 / 媒体 / 系统配置分层明确
2. **统一协议理解**
   - 把 Web 中已实际消费的 HTTP / WS 行为，提炼为 Android 可消费的 typed contracts
3. **统一能力入口**
   - 让 Android 页面结构能容纳 Web 已有的设置、收藏、媒体、扩展模块
4. **统一验收方式**
   - 以跨端能力矩阵和关键用户场景作为验收口径

---

## 2. 对齐原则（ADR）

### ADR-001：以 Web 当前行为作为主要产品基准
- **决策**：本轮默认以 Web 已上线行为为主基准，Android 补齐为主。
- **原因**：Web 已具备最完整的业务闭环，并已与当前 Go 后端长期磨合。
- **影响**：Android 新增或调整的行为，优先参考 Web 当前实现，而非另起一套规则。

### ADR-002：Android 先补状态层和协议层，再补大块 UI
- **决策**：先统一 Android 的 session / conversation / message / websocket event 模型，再挂接 UI。
- **原因**：如果直接按页面补功能，列表实时更新、回显去重、媒体消息、全局收藏会继续重复造状态。
- **影响**：本轮部分 UI 看起来是功能补齐，但核心工作量在状态与协议层。

### ADR-003：扩展能力采用“真实入口 + 可分阶段闭环”策略
- **决策**：用户选择了全量追平，因此本轮要补真实入口；但对高复杂模块（抖音 / 抽帧 / mtPhoto）允许先落主流程，再继续深化交互细节。
- **原因**：避免只做占位入口，同时又避免为了像素级完全追平而卡死整体交付。
- **影响**：验收标准以“可进入、可操作、关键链路闭环”为主，不要求首轮完全复制 Web 的所有微交互。

### ADR-004：跨端差异需要沉淀能力矩阵
- **决策**：在知识库中新增/更新“Android 与 Web 能力矩阵”，作为后续审查依据。
- **原因**：没有矩阵就会不断重复“这个接口有没有接 / 这个入口是不是占位”的问题。
- **影响**：后续所有跨端需求评审，都可以用矩阵做边界检查。

---

## 3. 模块对齐设计

### 3.1 认证与会话生命周期

#### Web 基准行为
- Token 失效时清理本地状态并回登录页
- WS `forceout/reject` 时清 token、断连接、跳转登录页
- 登录成功后进入身份选择页

#### Android 目标设计
- 在 `AuthRepository` / `SettingsViewModel` / `ChatRoomViewModel` 之间建立统一的 **SessionLifecycle** 处理约定：
  - `token invalid` → 清 token + 清 current session + 断 WS + 回登录页
  - `forceout / reject` → 同上，并携带错误原因
  - `logout` → 同上
- `LoginViewModel` 初始化时如果 token 无效，要直接清理失效状态，而不是只返回 false

#### 预期变更点
- `feature/auth/*`
- `feature/settings/*`
- `feature/chatroom/*`
- `core/datastore/*`
- `core/websocket/*`
- `app/LiaoApp.kt`

---

### 3.2 身份管理

#### Web 基准行为
- 支持创建、删除、选择
- 设置页可编辑 `id / name / sex`
- 修改 `name / sex` 时通过 WS `chgname / modinfo` 同步在线状态
- 修改 `id` 时调用 `updateIdentityId` 并重建连接状态

#### Android 目标设计
- 将当前身份页与设置页角色重新划分：
  - 身份页：创建 / 删除 / 选择 / 快速创建
  - 设置页中的身份信息区：编辑 `id / name / sex`
- Android 接入 `updateIdentityId`
- 接入 `chgname / modinfo` 的 WS 同步发送逻辑
- 当前会话被删 / 改 ID 时，保证本地缓存与会话一致更新

#### 预期变更点
- `feature/identity/*`
- `feature/settings/*`
- `core/network/NetworkStack.kt`
- `core/datastore/*`
- `core/websocket/*`

---

### 3.3 会话列表（历史 / 收藏 / 全局收藏）

#### Web 基准行为
- HTTP 拉取初始列表
- WS 新消息到达时实时更新：
  - `lastMsg`
  - `lastTime`
  - `unreadCount`
  - 历史/收藏列表置顶
- 支持房间内收藏切换
- 提供全局收藏真实页面

#### Android 目标设计
- 从“页面内临时状态”升级为“可被 WS 驱动的会话状态层”
- 会话列表状态要支持：
  - 初始加载
  - WS 增量更新
  - 未读清零
  - 收藏切换
  - 全局收藏数据聚合
- 接通 `FavoriteApiService`，落地全局收藏真实页面
- 会话项点击进入房间时同步清零未读

#### 预期变更点
- `feature/chatlist/*`
- 新增 `feature/favorites/*`
- `core/database/*`
- `core/network/*`
- `core/websocket/*`

---

### 3.4 聊天房间与消息模型

#### Web 基准行为
- 历史消息支持去重、排序、增量更新、清空重载
- 发送消息支持 optimistic、超时失败、重试、echo merge
- 聊天头部支持收藏、拉黑、清空记录
- 支持聊天历史媒体入口

#### Android 目标设计
- 扩展 `ChatTimelineMessage` 为富消息模型，至少支持：
  - `text`
  - `image`
  - `video`
  - `file`
  - 基础 segments / url/path 元信息
- 为 Android 引入最小 message store / reducer：
  - optimistic 本地消息
  - 服务端回显合并
  - 失败状态与重试
  - 历史翻页与消息排序
- 将当前 `ChatRoomFeature` 从“纯文本聊天页”扩成“房间能力容器”
- 增加头部操作：收藏、拉黑、清空重载、在线状态查询、聊天历史媒体

#### 预期变更点
- `feature/chatroom/*`
- `core/common/*`
- `core/database/*`
- `core/network/NetworkModels.kt`
- `core/websocket/*`

---

### 3.5 WebSocket 事件分层

#### Web 基准行为
Web 实际区分并消费的事件包含：
- `-4 reject`
- `-3 forceout`
- `7 private message`
- `12 connect notice`
- `13/14 typing`
- `15 match success`
- `16 match cancel`
- `18 randomOut`
- `19 warningreport`
- `30 ShowUserLoginInfo result`

#### Android 目标设计
- 将现有 `LiaoWsEvent` 扩为 typed events：
  - `Forceout`
  - `Reject`
  - `ConnectNotice`
  - `Typing`
  - `MatchSuccess`
  - `MatchCancel`
  - `OnlineStatus`
  - `WarningReportAck / Notice`
  - `ChatMessage`
  - `Unknown`
- 不再只把大多数事件降级为 `Notice`
- 由不同 feature 消费不同事件：
  - chatlist：列表更新 / 匹配状态
  - chatroom：消息 / typing / 在线状态 / 拉黑反馈
  - app/session：forceout / reject / logout

#### 预期变更点
- `core/websocket/LiaoWebSocketClient.kt`
- `feature/chatlist/*`
- `feature/chatroom/*`
- 可能新增 app 级 event coordinator

---

### 3.6 媒体能力对齐

#### Web 基准行为
- 上传菜单支持图片 / 视频 / 文件来源
- 已上传媒体列表可复用发送
- 支持聊天历史图片 / 视频预览
- 支持全局图片管理与查重

#### Android 目标设计
- 为房间页增加 Android 版上传菜单 / BottomSheet
- 接入 `MediaApiService`：
  - 上传
  - 查询已上传媒体
  - 聊天历史媒体
  - 记录发送关系
  - 批量删除
- 房间中支持图片 / 视频 / 文件消息发送与渲染
- 新增媒体中心页面作为“图片管理”承接页

#### 预期变更点
- 新增 `feature/media/*`
- `feature/chatroom/*`
- `core/network/*`
- `core/database/*`

---

### 3.7 mtPhoto / 抖音 / 视频抽帧

#### Web 基准行为
- mtPhoto：相册 / 目录浏览、导入上传、收藏 / 同媒体查询
- 抖音：解析、下载、导入上传、收藏作者/作品/标签
- 视频抽帧：任务创建、任务中心、详情、取消、继续、删除

#### Android 目标设计
- 不再只保留 service interface，而是补齐真实入口页：
  - `feature/mtphoto/*`
  - `feature/douyin/*`
  - `feature/videoextract/*`
- 先完成“能进入 → 能请求 → 能完成主流程”的闭环，再逐步深化复杂交互
- 与媒体中心建立复用关系，避免导入上传各写一套

#### 预期变更点
- 新增对应 feature 模块
- `core/network/*`
- `feature/settings/*` / `feature/media/*`

---

### 3.8 设置与系统能力

#### Web 基准行为
- 身份信息
- 系统设置
- 连接统计 / 断开全部连接 / forceout 清理
- 图片端口策略
- 主题偏好
- 图片管理 / 全局收藏入口

#### Android 目标设计
- 将当前简单设置页升级为分组设置页：
  - 身份信息
  - 系统设置
  - 图片管理
  - 全局收藏
  - 连接与调试
- 系统设置接入 `SystemApiService`
- 允许查看连接统计、清理 forceout、调整图片端口策略

#### 预期变更点
- `feature/settings/*`
- 新增部分 system/media sub screens
- `core/network/*`

---

## 4. 数据与状态设计

### 4.1 Android 侧目标状态分层
- `AuthState`：token、登录态、失效原因
- `SessionState`：当前身份、当前 cookie、当前 WS 绑定
- `ConversationState`：历史列表、收藏列表、全局收藏、未读、排序
- `MessageState`：按 peerId 分组的消息、optimistic 状态、分页状态
- `MediaState`：上传队列、已上传媒体、聊天历史媒体、预览态
- `SystemState`：图片端口策略、连接统计、forceout 用户数量等

### 4.2 Room / DataStore 扩展方向
- 保留现有 `Identity / Conversation / Message` 表
- 追加：
  - `FavoriteEntity`
  - `MediaEntity`
  - `VideoExtractTaskEntity`
  - 必要的系统配置缓存结构
- DataStore 继续承接 token、baseUrl、currentSession 及必要轻量配置

---

## 5. 风险与规避

### 风险 1：一次性对齐范围过大
- **等级**：高
- **规避**：尽管方案目标是全量追平，但在任务编排上仍按“认证/身份 → 列表/房间/WS → 收藏/媒体 → 扩展模块 → 设置/系统”逐层推进。

### 风险 2：Android 状态层继续页面化，导致返工
- **等级**：高
- **规避**：优先处理 conversation/message/ws event 的统一状态层，而不是继续在单个 ViewModel 中堆逻辑。

### 风险 3：媒体与扩展能力链路长、联调复杂
- **等级**：高
- **规避**：统一通过媒体中心承接上传、预览、导入，避免 mtPhoto / 抖音 / 抽帧各自实现上传闭环。

### 风险 4：forceout / token / reconnect 状态机不一致
- **等级**：高
- **规避**：先定义统一的 session lifecycle，再修改登录、设置、房间与 WS 客户端。

### 风险 5：回归面广，容易破坏 Web 或后端现有行为
- **等级**：中
- **规避**：Web 端尽量少改行为，只抽公共规则或补文档；所有对齐以不破坏现有 Go API 为前提。

---

## 6. 验证策略

### 6.1 构建验证
- `go test ./...`
- `cd frontend && npm run build`
- `cd android-app && ./gradlew :app:compileDebugKotlin --no-daemon`
- `cd android-app && ./gradlew testDebugUnitTest --no-daemon`

### 6.2 核心场景验证
1. 登录成功 / token 失效 / logout / forceout
2. 身份创建 / 编辑 / 改 ID / 删除 / 选择
3. 收到消息后列表实时更新 / 未读变更 / 置顶
4. 文本发送 optimistic / echo merge / retry
5. 图片 / 视频 / 文件消息收发
6. 收藏切换 / 全局收藏 / 聊天历史媒体
7. mtPhoto / 抖音 / 抽帧主流程可用
8. 系统设置与连接管理可用

### 6.3 文档验证
- `.helloagents/modules/android-client.md`
- `.helloagents/modules/api.md`
- `.helloagents/modules/arch.md`
- `.helloagents/modules/overview.md`
- `.helloagents/CHANGELOG.md`

---

## 7. 交付说明

本方案包虽然选择“全量能力一次追平”，但实施时仍采用**单包内分阶段推进**：
- 先打通状态与协议基线
- 再补核心聊天与收藏闭环
- 最后补扩展媒体与系统能力

这样既满足用户的“整体对齐”目标，也能控制开发与回归风险。
