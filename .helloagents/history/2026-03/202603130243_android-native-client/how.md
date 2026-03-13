# 原生 Android 客户端技术设计

## 1. 目标与原则

### 1.1 设计目标
- 在不改造现有 Go 服务端协议的前提下，建设原生 Android 客户端
- 在业务能力上对齐网页端
- 用清晰的模块边界支撑后续迭代与测试

### 1.2 设计原则
- **协议兼容优先**：以当前 Go 后端与网页端行为为准，不擅自改协议
- **分层清晰**：UI、领域逻辑、数据访问、基础设施解耦
- **状态可追踪**：聊天、上传、任务、媒体等状态必须可恢复、可观察
- **高频链路优先**：登录、身份、聊天、上传是最高优先级
- **可降级**：抖音、mtPhoto、抽帧等扩展模块支持能力探测与缺省提示

---

## 2. 技术选型

## 2.1 基础技术栈
- 语言：Kotlin
- UI：Jetpack Compose
- 架构：单 Activity + Compose Navigation + MVVM + Repository
- 异步：Kotlin Coroutines + Flow
- 网络：OkHttp + Retrofit
- WebSocket：OkHttp WebSocket
- 本地存储：Room + DataStore
- 图片加载：Coil
- 文件上传/下载：OkHttp + Document Provider + MediaStore
- 长任务：WorkManager（用于上传重试、下载、抽帧任务轮询等后台工作）
- DI：Hilt
- 序列化：kotlinx.serialization 或 Moshi（二选一，优先统一为 kotlinx.serialization）

## 2.2 版本假设
- minSdk：26
- targetSdk：35
- compileSdk：35

⚠️ 不确定因素：如果用户要求兼容更低版本（如 API 24），需要在文件访问、通知权限、后台行为上重新评估。

---

## 3. 工程结构设计

建议采用多模块工程：

```text
android-app/
  app/                         # Application、导航入口、全局组装
  core/common/                 # 通用常量、Result、日志、工具
  core/network/                # Retrofit、OkHttp、拦截器、DTO adapter
  core/database/               # Room、DAO、实体、迁移
  core/datastore/              # Token、用户偏好、全局配置缓存
  core/websocket/              # WebSocket 客户端、状态机、消息分发
  core/media/                  # 上传、下载、文件/Uri、缩略图、缓存
  feature/auth/                # 登录
  feature/identity/            # 身份列表、创建、编辑、选择
  feature/chatlist/            # 历史/收藏列表、全局收藏入口
  feature/chatroom/            # 聊天页、消息列表、输入栏、匹配、表情、上传菜单
  feature/media/               # 所有上传、聊天媒体历史、查重、预览
  feature/mtphoto/             # 相册/目录/导入
  feature/douyin/              # 抖音解析、收藏作者/作品/标签、导入
  feature/videoextract/        # 抽帧创建、任务列表、详情
  feature/settings/            # 系统配置、连接状态、图片服务配置
```

说明：
- 若初期不做 Gradle 多模块，也应至少保持包结构等价，避免后续重构成本
- `app` 层只负责导航与依赖装配，不直接编写业务逻辑

---

## 4. 页面与网页端映射

## 4.1 页面映射

| Web 路由/模块 | Android 页面/容器 | 说明 |
|---|---|---|
| `/login` | LoginScreen | 访问码登录、自动校验 token |
| `/identity` | IdentityScreen | 身份列表、快速创建、创建/删除、图片管理入口 |
| `/list` | ChatListScreen | 历史/收藏切换、搜索/进入会话、全局收藏/设置入口 |
| `/chat/:userId?` | ChatRoomScreen | 消息列表、发送、上传、抽屉、媒体预览、清空/拉黑 |
| AllUploadImageModal | MediaLibraryScreen / BottomSheet | 所有上传媒体管理 |
| MtPhotoAlbumModal | MtPhotoScreen / BottomSheet | 相册或目录浏览 |
| DouyinDownloadModal | DouyinScreen / BottomSheet | 抖音解析、预览、收藏、导入 |
| VideoExtractCreateModal | VideoExtractCreateScreen | 创建抽帧任务 |
| VideoExtractTaskModal | VideoExtractTaskScreen | 任务列表与详情 |
| SettingsDrawer | SettingsScreen / ModalNavigationDrawer | 身份信息、系统设置、图片管理、全局收藏 |

## 4.2 交互迁移原则
- Web 上以 modal / drawer 呈现的模块，在 Android 上优先改为：
  - 独立页面（复杂流程）
  - BottomSheet（短流程）
  - ModalNavigationDrawer（设置、收藏入口）
- 聊天页保留高频入口：
  - 文本输入
  - 表情面板
  - 上传菜单
  - 历史媒体入口
  - 收藏与拉黑操作

---

## 5. 网络层设计

## 5.1 Retrofit 服务拆分
按网页端 API 模块映射 Android Service：
- `AuthApiService`
- `IdentityApiService`
- `ChatApiService`
- `FavoriteApiService`
- `MediaApiService`
- `MtPhotoApiService`
- `DouyinApiService`
- `SystemApiService`
- `VideoExtractApiService`

补充说明：
- `ChatApiService` 需覆盖 `getHistoryUserList`、`getFavoriteUserList`、`getMessageHistory`、`reportReferrer`、`toggleFavorite`、`cancelFavorite` 等接口
- `SystemApiService` 除系统配置外，还需覆盖 `resolveImagePort`、连接统计、断开全部连接、forceout 清理等接口
- 下载代理类接口（如 `/api/douyin/download`、`/api/douyin/cover`、`/api/downloadMtPhotoOriginal`）需保留 `ResponseBody` 直读能力

## 5.2 请求形态支持
由于后端接口风格不统一，网络层必须同时支持：
- `@FormUrlEncoded`
- `@Multipart`
- `@GET` + Query
- `@POST` + JSON Body
- 原始 `ResponseBody`（用于图片/视频/下载代理）

## 5.3 网络基础设施
- `AuthInterceptor`：自动附加 `Authorization: Bearer <token>`
- `TokenAuthenticator`：401 时统一处理 token 失效，必要时跳回登录页
- `BaseUrlProvider`：允许动态切换服务端地址（开发/测试/生产）
- `CapabilityProbeRepository`：在应用启动后探测抖音、mtPhoto、抽帧可用性，用于 UI 降级

## 5.4 响应模型策略
后端响应不统一，因此：
- 不使用一个全局 `BaseResponse<T>` 硬套全部接口
- 每个接口按实际返回结构定义 DTO
- Repository 层统一转换为内部 `AppResult<T>`

建议基础结果模型：
```text
AppResult<T>
  - Success(data)
  - BizError(code, message)
  - NetworkError(exception)
  - Unauthorized
```

---

## 6. WebSocket 设计

## 6.1 连接模型
- 使用 OkHttp WebSocket 连接 `/ws?token=...`
- 连接建立成功后，立即发送 `sign` 登录消息
- `sign` 不能只发送 `id`，至少要兼容网页端当前发送字段：
  - `act=sign`
  - `id` / `name` / `userSex`
  - `address_show=false`
  - `randomhealthmode=0`
  - `randomvipsex=0`
  - `randomvipaddress=0`
  - `userip` / `useraddree`
  - `randomvipcode=""`
- 每次身份切换时，若当前连接身份不同，则重建连接

## 6.2 组件职责
- `WebSocketClient`：底层连接与回调
- `ChatSocketManager`：连接状态机、重连策略、sign 管理、forceout 管理
- `SocketMessageDispatcher`：解析消息并分发到聊天、列表、用户状态模块

## 6.3 状态机建议
- Idle
- Connecting
- Connected(userId)
- Reconnecting(attempt)
- Forceout(until)
- Closed

## 6.4 行为要求
- token 缺失时禁止连接
- identity 未选择时禁止连接
- 收到 `forceout` 后进入禁止重连状态并提示用户
- `forceout` 的禁止重连时长按后端代码实现为 **5 分钟**，Android 端倒计时与提示文案必须以此为准
- 收到上游关闭时按策略重连，但需遵守 forceout 限制
- WebSocket 消息与历史分页消息需要统一归并与去重

## 6.5 消息方向与去重规则
- 私聊消息（`code=7`）不能简单以 `fromUserId == identity.id` 判断是否为自己发送
- 需对齐网页端规则：`fromuser.id === md5(identity.id)` 时判定为自己发送的回显消息
- 本地乐观发送消息、WebSocket 回显消息、HTTP 历史消息三者必须在消息仓库统一去重
- 去重优先级建议：`tid` > 媒体语义 key（方向 + path + 时间桶）> 兜底内容 key

## 6.6 上游 WebSocket 协议兼容表

| 类型 | code / act | 当前网页端行为 | Android 兼容要求 |
|---|---|---|---|
| 被服务端强退 | `code=-3` + `forceout=true` | 停止重连、清 token、跳登录页 | 进入 Forceout 状态，停止自动重连并提示用户 |
| 被后端拒绝连接 | `code=-4` + `forceout=true` | 提示剩余秒数并阻止连接 | 与网页端保持一致，禁止短时间重试 |
| 连接成功提示 | `code=12` | Toast 提示，不入聊天记录 | 仅提示，不写入消息流 |
| 静默系统消息 | `code=13/14/16/19` | 静默忽略 | 不弹 Toast、不入聊天记录 |
| 忽略消息 | `code=18` | 直接忽略 | 不展示 |
| 匹配成功 | `code=15` | 更新匹配对象、历史列表、进入聊天或连续匹配流 | 更新会话列表与匹配状态 |
| 在线状态结果 | `code=30` | 通过事件派发到 UI | 进入用户在线状态查询结果流 |
| 私聊消息 | `code=7` | 解析消息片段、判断 self、更新列表/未读/当前会话 | 作为聊天主消息类型，必须完整兼容 |
| 输入状态 | `act=inputStatusOn_* / inputStatusOff_*` | 显示/隐藏“正在输入” | 建立 typing 状态流 |

说明：如后续发现网页端新增 code/act，Android 方案需继续补充该协议表。

## 6.7 主动动作兼容清单
- `sign`：建立 WS 后立即发送
- `ShowUserLoginInfo`：查询对方在线状态，当前网页端附带固定 `randomvipcode`
- `warningreport`：聊天页拉黑/举报动作，通过 WS 主动发送
- `inputStatusOn_* / inputStatusOff_*`：输入状态同步
- `reportReferrer`：虽然是 HTTP 而非 WS，但属于聊天链路初始化动作；当前网页端在 WS 建立后立即上报 `referrerUrl/currUrl/userid/cookieData/referer/userAgent`，Android 端应复刻为兼容模式

## 6.8 消息模型
- 建立 `SocketEnvelope`、`ChatMessageModel`、`UserPresenceModel`、`TypingStatusModel` 等内部模型
- Repository / UseCase 层负责把 WebSocket 原始消息和 HTTP 历史消息转换为统一消息流

---

## 7. 本地存储设计

## 7.1 DataStore
存储轻量全局状态：
- token
- 服务端 baseUrl
- 最近使用身份 ID
- 主题/偏好项
- 能力探测结果缓存

## 7.2 Room
建议建立以下表：
- `identity_local_cache`
- `chat_session_cache`
- `chat_message_cache`
- `favorite_user_cache`
- `media_upload_cache`
- `douyin_recent_record`
- `douyin_favorite_user_cache`
- `douyin_favorite_aweme_cache`
- `douyin_tag_cache`
- `video_extract_task_cache`
- `system_config_cache`

说明：
- Room 是 Android 客户端缓存，不替代服务端数据库
- 只缓存高频查询与 UI 恢复所需数据
- 消息以“会话 + 时间/TID + 本地发送状态”组织，支持去重与失败重试

---

## 8. 领域模块设计

## 8.1 Auth
职责：
- 登录
- token 校验
- token 失效退出
- App 冷启动恢复登录态

## 8.2 Identity
职责：
- 身份列表加载
- 创建/快速创建/更新/删除/选择
- 最近身份恢复

## 8.3 ChatList
职责：
- 历史列表与收藏列表
- 用户信息增强展示
- 最后一条消息预览
- 进入聊天页

## 8.4 ChatRoom
职责：
- WebSocket 实时消息
- 历史消息分页
- 文本/表情/媒体发送
- `reportReferrer` 初始化上报
- `ShowUserLoginInfo` 在线状态查询
- `warningreport` 拉黑/举报动作
- 收藏切换、拉黑、清空本地记录
- 侧边栏/底部菜单状态管理

## 8.5 Media
职责：
- 上传图片/视频
- 查看所有上传媒体
- 删除/批量删除
- 查看聊天历史媒体
- 重传与查重

## 8.6 MtPhoto
职责：
- 相册/目录浏览
- 缩略图展示
- 原图/原视频访问
- 导入到上传链路

## 8.7 Douyin
职责：
- 链接/口令解析
- 下载代理预览
- 导入上传
- 收藏作者/作品/标签管理

## 8.8 VideoExtract
职责：
- 上传输入视频或选择 mtPhoto 视频
- 创建任务
- 查看任务列表/详情
- 取消、继续、删除

## 8.9 Settings
职责：
- 图片服务地址与系统配置
- 连接统计
- 清理 forceout
- 高级调试入口

---

## 9. UI 状态管理

每个 Feature 采用：
- `UiState`：页面状态
- `UiEvent`：用户动作
- `UiEffect`：一次性副作用（Toast、导航、打开文件选择器）
- `ViewModel`：协调 UseCase / Repository

示例：
- `ChatRoomUiState`
  - currentUser
  - wsState
  - messages
  - loadingHistory
  - inputText
  - uploadQueue
  - activePanels (emoji/upload/sidebar/mediaPreview)

复杂界面必须避免直接在 Compose 中堆叠业务逻辑。

---

## 10. 媒体与文件设计

## 10.1 选择与上传
- 使用系统文件选择器选择图片/视频
- 统一封装 `Uri -> UploadPayload`
- 上传前读取 MIME、大小、文件名，必要时做查重预检

## 10.2 缩略图与预览
- 图片：Coil 直接加载
- 视频：优先使用服务端生成的 poster，降级显示视频首帧或图标
- 大图/长图：使用可缩放预览组件

## 10.3 下载与缓存
- 对抖音下载、mtPhoto 原图、上传文件访问建立统一下载器
- 大文件下载建议接入 WorkManager + 通知

## 10.4 权限与系统集成
- Android 13+ 媒体权限按图片/视频分类处理
- 保存文件到本地时优先走 MediaStore
- 对 App 内临时文件建立清理策略

---

## 11. 错误处理与可观测性

## 11.1 错误分类
- 网络错误
- 服务端业务错误
- token/鉴权错误
- 外部能力不可用
- 文件系统错误
- WebSocket 连接错误

## 11.2 展示策略
- 关键主流程错误：页面级提示 + 明确重试操作
- 非关键扩展模块错误：局部提示 + 模块降级，不影响聊天主流程
- forceout：强提示并禁止短时间自动重连

## 11.3 调试与日志
- debug 包记录网络摘要与 WebSocket 事件
- release 包避免输出敏感日志
- 为上传、WebSocket、任务轮询建立统一日志标签

---

## 12. 测试策略

### 12.1 单元测试
- DTO 到领域模型转换
- 消息去重逻辑
- WebSocket 状态机
- Repository 错误映射

### 12.2 UI 测试
- 登录流程
- 身份切换
- 聊天发送与消息展示
- 媒体管理入口

### 12.3 集成测试
- 与本地 Go 服务联调
- 上传/下载/预览联调
- forceout、断线重连、身份切换联调

---

## 13. 实施顺序建议

### Phase 1：工程基础与核心链路
- Android 工程初始化
- 网络层、DataStore、Room、Hilt、导航
- 协议对齐：补齐 WebSocket code 表、主动动作、消息方向判定的实现清单
- 登录、身份选择、聊天列表
- WebSocket 建立与聊天页基础

### Phase 2：媒体与设置
- 上传、历史媒体、媒体管理
- 系统设置、连接状态、全局收藏

### Phase 3：扩展业务模块
- mtPhoto
- 抖音模块
- 视频抽帧

### Phase 4：质量完善
- 性能优化
- 断线恢复
- UI 细节统一
- 测试补齐

说明：实施节奏允许分阶段，但最终目标不变：功能覆盖网页端。

---

## 14. ADR

### ADR-20260313-01：Android UI 采用 Jetpack Compose
- **状态**：✅ 已采纳
- **原因**：页面复杂、状态多、适合长期维护
- **备选**：XML + Fragment
- **结论**：采用 Compose

### ADR-20260313-02：采用单 Activity + Navigation + Feature 分层
- **状态**：✅ 已采纳
- **原因**：便于统一导航与状态恢复，减少 Fragment 栈复杂度
- **备选**：多 Activity、Fragment 混合架构
- **结论**：采用单 Activity

### ADR-20260313-03：网络层按接口分 DTO，不强行统一响应壳
- **状态**：✅ 已采纳
- **原因**：后端返回结构不统一，强行统一会增加解析复杂度和错误风险
- **备选**：单一 BaseResponse
- **结论**：按接口建模，在 Repository 层统一结果

### ADR-20260313-04：WebSocket 与 HTTP 历史消息统一在客户端消息仓库归并
- **状态**：✅ 已采纳
- **原因**：保证聊天页渲染、去重、重试逻辑集中管理
- **备选**：UI 层直接拼装
- **结论**：统一在数据/领域层处理

---

## 15. 安全与性能

### 安全
- 不在日志中输出 accessCode、token、Cookie 完整值
- Token 仅存储在加固后的本地持久层中
- 文件上传前检查大小与类型
- 下载文件名、Uri、临时文件路径统一清洗，避免路径注入

### 性能
- 列表使用 Paging 或惰性加载
- 图片列表与聊天消息按需分页
- 视频缩略图优先使用服务端 poster
- WebSocket 消息解析放在后台线程
- 大文件上传/下载与任务轮询避免阻塞主线程
