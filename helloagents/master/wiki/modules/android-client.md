# Android Client

## 目的
提供 Liao 的 Android 客户端，实现与 Web 客户端对齐的登录、身份、聊天、媒体、抖音、mtPhoto、视频抽帧和设置能力。

## 模块概述
- **职责:** Kotlin/Compose UI、API/WS 网络栈、本地缓存、Room/DataStore、功能页面与单元/UI 测试。
- **状态:** 开发中
- **最后更新:** 2026-07-07

## 规范

### 需求: 协议对齐
**模块:** Android Client
Android 客户端应复用后端 `/api` 和 `/ws` 协议，字段语义与 Web 客户端保持一致。

#### 场景: 默认开发地址
- `DEFAULT_API_BASE_URL` 指向 Android 模拟器访问宿主机的 `http://10.0.2.2:8080/api/`。

### 需求: 分层结构
**模块:** Android Client
代码按 `app`、`core`、`feature` 分层；网络模型在 `core/network`，WebSocket 在 `core/websocket`，各业务在 `feature/*`。

#### 场景: 新增功能
- 优先补齐 repository/viewmodel/helper 测试。
- UI 测试使用稳定 test tags。

### 需求: 身份切换聊天缓存隔离
**模块:** Android Client
Android 选择身份和全局收藏页的“切换身份并聊天”应与 Web 端保持一致：先完成身份切换，再进入新身份会话，避免沿用旧身份的本地聊天状态。

#### 场景: 从 A 身份切换到 B 身份
- 选择新身份成功后，清理旧身份的本地会话缓存和消息缓存。
- 应用级会话协调器监听到非空身份 ID 变化时，也会清理旧身份聊天缓存，覆盖后续新增的身份切换入口。
- 会话列表由新身份上下文重新加载，避免展示上一身份的历史/收藏会话。

#### 场景: 从 A 身份点击 B 身份收藏对象
- 选择收藏所属身份成功后，清理旧身份的本地会话缓存和消息缓存。
- 进入聊天前写入目标收藏对象的会话占位，使用收藏对象名称作为聊天标题，名称缺失时按 `用户{targetUserId前4位}` 兜底。
- 聊天历史由聊天页在新身份会话上下文中重新加载，避免同一目标用户在不同身份下串消息。

### 需求: 全局归档搜索
**模块:** Android Client
Android 会话列表应提供与 Web 端一致的全局归档搜索入口，可按用户 ID 或归档名称搜索所有身份下的本地归档会话。

#### 场景: 搜索归档用户并进入聊天
- 调用 `GET /api/chat/archiveSearch?q=&limit=` 获取归档搜索结果。
- 结果展示目标用户名称、目标用户 ID、所属身份 ID、来源标签和最近消息。
- 选中结果后在当前身份上下文中写入临时会话并进入聊天页，聊天历史由聊天页按当前身份重新加载。

### 需求: 跨身份联系人接入
**模块:** Android Client
Android 会话列表应提供与 Web 端一致的“从其他身份接入”入口，可从其它身份的历史、收藏和本地归档候选中临时接入当前身份聊天。

#### 场景: 从其它身份候选进入聊天
- 来源身份列表排除当前身份；本地身份缓存为空时回源加载身份列表。
- 调用 `GET /api/chat/contactCandidates` 获取来源身份下的候选联系人，默认包含上游历史、上游收藏和本地归档。
- 选中候选后在当前身份上下文中写入临时会话并进入聊天页，聊天历史由聊天页按当前身份重新加载。

### 需求: 会话项动作
**模块:** Android Client
Android 会话列表应逐步补齐 Web 端会话侧边栏的单项管理动作。

#### 场景: 清除会话未读
- 会话项提供“清未读”入口，仅在 `unreadCount > 0` 时启用。
- 操作只更新本地 `conversation_cache.unreadCount = 0`，由 Room 观察流刷新 UI。

#### 场景: 切换全局收藏
- 会话项提供“全局收藏 / 取消全局收藏”入口，状态来自当前身份下的 `/api/favorite/listAll` 结果。
- 加入全局收藏调用 `POST /api/favorite/add`，提交 `identityId`、`targetUserId` 和 `targetUserName`。
- 取消全局收藏调用 `POST /api/favorite/remove`，提交 `identityId` 和 `targetUserId`。
- 操作成功后局部更新会话列表按钮状态，并显示“已加入全局收藏”或“已取消全局收藏”。

#### 场景: 查询在线状态
- 会话项提供“查在线”入口，点击后通过 WebSocket 发送 `ShowUserLoginInfo`，`id` 为当前身份 ID，`msg` 为目标用户 ID。
- WebSocket `code=30` 回包会解析 `data.IF_Online` 和 `data.TimeAll`；`"1"` 展示在线，`"0"` 展示离线，其它或缺失展示状态未知。
- 查询中禁用当前会话项的“查在线”按钮；WebSocket 不可用或当前身份缺失时不展示状态弹窗，并向用户显示失败原因。
- 在线状态仅保存在会话列表内存 UI 状态中，不写入 Room 或 DataStore。

#### 场景: 删除会话用户
- 会话项提供“删除”入口，需二次确认。
- 确认后调用 `POST /api/deleteUpstreamUser`，表单字段为 `myUserId` 和 `userToId`。
- 远端返回成功后删除本地 `conversation_cache` 目标会话，并清理 `message_cache` 对应聊天记录。
- 远端失败或当前身份缺失时不清理本地缓存，并向用户显示失败原因。

#### 场景: 批量删除会话用户
- 会话列表提供“批量管理”入口，进入选择模式后可点击会话项选择/取消选择，也可全选当前可见列表。
- 确认批量删除后调用 `POST /api/batchDeleteUpstreamUsers`，请求 JSON 字段为 `myUserId` 和 `userToIds`。
- 远端返回成功后仅删除成功项的本地 `conversation_cache` 记录，并逐个清理 `message_cache` 对应聊天记录。
- 若远端返回 `failedItems`，成功项仍清理，失败项保持选中并停留在选择模式，向用户显示成功/失败数量。
- 未选择会话或当前身份缺失时不调用远端接口，并向用户显示失败原因。

## API接口
通过 Retrofit/OkHttp 调用后端 `/api`，通过 WebSocket client 调用 `/ws`。

## 数据模型
- Room 本地数据库：`android-app/app/src/main/kotlin/.../core/database/LocalDatabase.kt`
- DataStore 偏好与缓存快照：`android-app/app/src/main/kotlin/.../core/datastore/*`

## 依赖
- `android-app/app/build.gradle.kts`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/network/*`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/*`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/*`
