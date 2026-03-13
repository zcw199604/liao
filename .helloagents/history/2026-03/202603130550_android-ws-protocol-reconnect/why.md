# 变更提案: Android WebSocket 协议与重连修复

## 需求背景
Android 聊天链路当前仅以原始字符串处理 WebSocket 消息，`code=-3` forceout 采用字符串包含判断，`code=-4` 被后端拒绝时没有统一处理，断线后也只是把状态切到 `Reconnecting`，并未发起真实重连。这与后端 `/ws` 行为和首轮文档宣称的能力存在偏差。

## 变更内容
1. 为 Android WebSocket 建立最小可用的 `code/act` 协议目录与结构化入站解析。
2. 统一处理 `code=-3` 与 `code=-4` 且 `forceout=true` 的禁止重连行为。
3. 在不推翻现有骨架的前提下补齐真实自动重连，并让聊天页消费结构化事件而不是直接硬解析原始 JSON。

## 影响范围
- **模块:** Android Client
- **文件:** `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClient.kt`、`android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatroom/ChatRoomFeature.kt`、`android-app/app/src/test/kotlin/io/github/a7413498/liao/android/ProtocolAlignmentTest.kt`
- **API:** 复用现有 `/ws?token=`、`sign`、`ShowUserLoginInfo`、`warningreport`、私信动作协议
- **数据:** 无新增服务端数据结构，仅增强客户端本地状态机与解析模型

## 核心场景

### 需求: 最小协议目录与结构化解析
**模块:** Android Client
为 Android WebSocket 增加最小可用的协议目录，至少能识别当前聊天链路依赖的 `code` / `act`，并输出结构化事件。

#### 场景: 聊天页消费 WebSocket 消息
- 聊天页不再依赖裸字符串 contains / 手写 JSON 取字段判断核心协议
- 私信消息可以转换为结构化时间线消息
- 已知系统消息保留 `code` / `act` / `content` 元信息，便于后续扩展

### 需求: forceout 与自动重连修复
**模块:** Android Client
让客户端对齐后端 forceout 行为，并在非 forceout 的异常断线时自动重连。

#### 场景: 聊天连接异常
- 收到 `code=-3` 或 `code=-4` 且 `forceout=true` 时进入 5 分钟禁止重连状态
- 非手动断开且不在 forceout 窗口内时，客户端会实际重新发起 `/ws?token=` 连接
- 重连成功后会重新发送 `sign`，聊天页状态流随之更新

## 风险评估
- **风险:** 当前仓库环境可能缺少 Android SDK / Gradle Wrapper，无法完整跑通 Android 编译链
- **缓解:** 采用最小增量修改，优先复用现有 DTO / 状态流骨架，并补充纯单元可覆盖的协议解析断言
