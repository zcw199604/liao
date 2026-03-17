# Android Client

## 目的
为现有 Go 服务端补充原生 Android 客户端骨架，承接登录、身份、会话列表、聊天与后续媒体能力扩展。

## 模块概述
- **职责:** 提供 Kotlin + Jetpack Compose 客户端基础设施，复用现有 HTTP / WebSocket 协议，并为媒体、抖音、mtPhoto、视频抽帧等模块预留扩展位。
- **状态:** 🚧开发中
- **最后更新:** 2026-03-15

## 规范

### 需求: Android 客户端工程骨架
**模块:** Android Client
在 `android-app/` 中提供原生 Android 工程基础结构，并以包分层模拟模块化边界。

#### 场景: 新增原生客户端入口
- Android Studio 打开 `android-app/` 后，可以识别 Gradle Kotlin DSL 工程结构
- 仓库已补齐 `gradlew` / `gradlew.bat` / `gradle/wrapper/*`，并固定使用 Gradle 8.9，便于在无系统 Gradle 环境下直接构建
- `app/` 已显式引入 `com.google.android.material:material`，用于提供 XML 宿主主题 `Theme.Material3.DayNight.NoActionBar` 所需资源
- Room 依赖已升级到 `2.7.2`，当前 JDK 17 + Android SDK 环境下可通过 `./gradlew :app:compileDebugKotlin` 与 `./gradlew testDebugUnitTest`
- `app/` 中具备 `core/*` 与 `feature/*` 包目录，便于后续平滑拆分多 module

### 需求: 协议兼容接入
**模块:** Android Client
Android 端需要遵循现有 Go 服务端 `/api` 与 `/ws` 协议，不擅自改动后端接口。

#### 场景: 登录与聊天主链路
- 访问码登录走 `POST /api/auth/login`
- WebSocket 连接走 `/ws?token=`，并在连接后发送 `sign`
- 客户端维护最小可用的 WS `code/act` 协议目录，可结构化识别 `-4 / -3 / 7 / 12 / 13 / 14 / 15 / 16 / 18 / 19 / 30`
- 收到 `code=-3/-4` 且 `forceout=true` 时进入 5 分钟禁止重连状态；普通断线会自动重连并重新发送 `sign`

### 需求: 首期页面骨架
**模块:** Android Client
首期先落地登录、身份、会话列表、聊天与设置页面，复杂媒体与扩展工具保持接口预留。

#### 场景: 移动端首轮联调
- 用户可以完成 `登录 -> 选择身份 -> 会话列表 -> 聊天页` 主流程跳转
- `AppCoordinatorViewModel` 已统一驱动冷启动恢复、token/currentSession 生命周期、forceout 跳登录与 WS 自动绑定
- 设置页已可修改 Base URL、查看 Token/连接统计、编辑当前身份（含 `updateIdentityId` / `chgname` / `modinfo`）并执行 logout / disconnectAll / clearForceout
- 身份页已支持创建、快速创建、编辑、删除、选择；编辑当前身份会同步当前会话，删除当前身份会清理本地当前会话
- 会话列表已改为以 Room 缓存为真实展示数据源，可被 WS 新消息、`MatchSuccess`、`ConnectNotice`、`MatchCancelled` 等 typed events 实时驱动未读数、排序、提示与收藏状态
- 全局收藏已从占位入口升级为真实页面，支持按身份分组查看、按 ID 取消收藏、切换身份并直接进入会话
- 聊天页已消费结构化 WebSocket 事件，并对齐 `/api/getMessageHistory` 的 `contents_list + firstTid` 历史分页协议，支持“查看历史消息”、首屏/新消息自动贴底、加载旧消息时不误滚到底部，以及 optimistic 发送、echo merge、失败重试、收藏、拉黑、清空重载与 `ShowUserLoginInfo` 在线状态查询入口
- 聊天页已补齐媒体 BottomSheet：支持选择图片 / 视频 / 文件上传、保留“已上传待发送”列表、查询并浏览聊天历史媒体，并在发送时回写 `recordImageSend` 关系记录
- 图片消息会在列表中直接预览；视频 / 文件消息改为可点击打开的卡片，不再只显示原始 `[path]` 文本
- 设置页新增“图片管理”真实入口与图片端口策略表单，可读取 / 修改 `imagePortMode`、`imagePortFixed`、`imagePortRealMinBytes`、`mtPhotoTimelineDeferSubfolderThreshold`
- Android 新增全局媒体库页面，支持浏览全站上传媒体、打开图片/视频/文件、单项删除、批量删除与分页加载更多
- Android 新增 mtPhoto 入口页，支持相册/目录两种基础浏览模式、缩略图预览，以及基于 `mtPhotoTimelineDeferSubfolderThreshold` 的目录时间线延迟加载
- 聊天页媒体 BottomSheet 已接入 mtPhoto 相册入口；预览项可先导入本地，再自动通过 `/api/reuploadHistoryImage` 回灌到当前会话的“已上传待发送”列表
- Android mtPhoto 目录模式已补齐目录收藏主流程：支持查看收藏列表、收藏当前目录、取消收藏、从收藏直接打开目录；图片管理页也可基于本地 `localPath` 查询 mtPhoto 同媒体，并一键跳转到对应目录
- Android 新增抖音下载入口页，支持粘贴分享文本/链接/作品 ID 发起解析，展示作品封面与媒体列表，并提供图片预览、外部打开与系统下载队列入口
- Android 抖音页面已接通 `/api/douyin/import`：设置页入口可将解析结果导入本地媒体库；聊天页媒体 BottomSheet 新增“抖音下载”入口，导入后会通过 `/api/reuploadHistoryImage` 回灌到当前会话待发送列表
- Android 抖音页面现已升级为三模式入口（作品解析 / 用户作品 / 收藏），可收藏作者、收藏作品，并对两类收藏分别创建 / 删除 / 应用标签
- Android 设置页新增“视频抽帧”入口：支持选择本地视频、调用 `/api/uploadVideoExtractInput` 上传、调用 `/api/probeVideo` 探测，并按 Web 默认参数模型提交 `/api/createVideoExtractTask`
- Android 新增“抽帧任务中心”页面与入口：支持查看任务列表、详情、已生成帧结果，并对暂停/运行中的任务执行取消、继续、删除等生命周期操作
- Android 设置页新增主题偏好（跟随系统 / 浅色 / 深色），并通过 DataStore 持久化；应用启动时会按该偏好应用 Material 主题
- Android 现已扩展页面恢复缓存：全局收藏继续由 Room 承接；系统配置、全局媒体库、抽帧任务列表与任务详情改由 DataStore 持久化最近快照；设置页、媒体库、抽帧任务中心在网络失败时会回退本地缓存，聊天页图片端口解析与 mtPhoto 时间线阈值也会复用缓存系统配置
- Android 本轮已完成安全与一致性收口：认证态迁移到加密首选项保存，当前身份恢复不再持久化 cookie；应用禁用 `allowBackup`，未识别 WebSocket 事件日志仅保留元信息；并已通过 `go test ./...`、`cd frontend && npm run build`、Android Kotlin 编译与单测验证
- 聊天页媒体 URL 解析已接通 fixed / probe / real 配置口径：`fixed` 直接使用固定端口，`probe/real` 统一调用 `/api/resolveImagePort`，从而与 Web 端端口策略行为一致

## API接口
### POST /api/auth/login
**描述:** Android 端访问码登录
**输入:** `accessCode`
**输出:** `code/msg/token`

### GET /api/getIdentityList
**描述:** 获取身份列表
**输入:** 无
**输出:** `Identity[]`

### GET /ws?token={jwt}
**描述:** 建立下游 WebSocket 连接
**输入:** Query `token`
**输出:** 建立连接后由客户端立即发送 `sign`

### POST /api/douyin/import
**描述:** 导入抖音解析结果到本地媒体库，供图片管理或聊天页继续复用
**输入:** `userid`、`key`、`index`
**输出:** `state/dedup/localPath/localFilename`

### GET /api/douyin/favoriteUser/list
**描述:** 获取已收藏抖音作者列表
**输入:** 无
**输出:** `{ items: DouyinFavoriteUser[] }`

### GET /api/douyin/favoriteAweme/list
**描述:** 获取已收藏抖音作品列表
**输入:** 无
**输出:** `{ items: DouyinFavoriteAweme[] }`

### POST /api/douyin/favoriteUser/tag/apply / favoriteAweme/tag/apply
**描述:** 为单个作者或作品设置标签（Android MVP 使用 `mode=set`）
**输入:** `secUserIds|awemeIds`、`tagIds`、`mode`
**输出:** `{ success: true }`

### POST /api/getMessageHistory
**描述:** 获取与目标用户的聊天历史，并支持基于 `firstTid` 的向前翻页
**输入:** `myUserID`、`UserToID`、`isFirst`、`firstTid`、`cookieData`
**输出:** `code + contents_list[]`（Android 端会按 Web 口径逆序后转为时间正序时间线）


## 数据模型
### CurrentIdentitySession
| 字段 | 类型 | 说明 |
|------|------|------|
| id | String | 当前身份 ID |
| name | String | 当前身份名称 |
| sex | String | 当前身份性别 |
| cookie | String | 为兼容上游列表与历史接口生成的本地 cookie |
| ip | String | 当前客户端随机 IP |
| area | String | 当前客户端地区占位，默认“未知” |

## 依赖
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/network/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/auth/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/identity/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatroom/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/settings/`
- `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/favorites/`

## 测试覆盖现状
- 当前 Android 侧已有 **47 个本地测试源文件（共 292 条用例）**：在原有协议/基础映射/Repository 回归之外，新增 `LoginViewModelTest.kt`、`IdentityViewModelTest.kt`、`ChatListViewModelTest.kt`、`SettingsViewModelTest.kt`、`AppCoordinatorViewModelTest.kt`、`MtPhotoViewModelTest.kt`、`MediaLibraryViewModelTest.kt`、`DouyinViewModelTest.kt`、`VideoExtractCreateViewModelTest.kt`、`VideoExtractTaskCenterViewModelTest.kt`、`ChatRoomViewModelTest.kt`，并配套 `MainDispatcherRule.kt`，继续扩大默认页面、应用级启动协调器、mtPhoto/媒体库浏览流程、抖音解析、视频抽帧创建页、抽帧任务中心、聊天页状态机、同媒体查询、DTO/时间线映射、JSON 容错与消息类型判定的 JVM 分支覆盖。
- 当前 Android 侧已新增 **`src/androidTest/` Compose UI 基线**：`LoginScreenTest.kt`、`IdentityScreenTest.kt`、`ChatListScreenTest.kt`、`SettingsScreenTest.kt` 共 13 条用例，统一通过 `ComposeUiTestSupport.kt` 包装 `LiaoTheme`，直接面向 `*ScreenContent(...)` 验证四个默认页面的空态、tab、按钮启用与关键回调。
- 已覆盖：协议辅助函数（MD5 自发消息判定、`forceout` 5 分钟禁连）、WS `code/act` 目录解析、`privateMessage` act 组装、协议用户显示名兜底、私聊 envelope → 时间线转换、消息类型推断、文件名提取、消息预览文案、`IdentityDto/ChatUserDto/ChatMessageDto/HistoryMessageDto` 映射、历史消息 / 全局收藏 JSON 解析、匹配候选与重连退避上限、`generateCookie` / `generateRandomIp` / `normalizeTextForMatch` 与缓存快照序列化稳定性、API Base URL / WebSocket URL / 动态 Base URL 重写规则、媒体 MIME/标签与上传路径归一化，以及 Auth / Identity / ChatList / ChatRoom / GlobalFavorites / MediaLibrary / Settings / Douyin / mtPhoto / VideoExtract Repository、`AppCoordinatorViewModel` 启动路由/forceout/消息落库、`MtPhotoViewModel` 相册/目录模式切换/刷新/目录收藏/时间线延迟加载、`MediaLibraryViewModel` 分页/选择/删除/同媒体查询、`DouyinViewModel` 模式切换/详情账号解析/导入收藏/标签管理、`VideoExtractTaskCenterViewModel` 列表刷新分页/详情切换/帧分页/继续终止删除动作，以及 `LiaoWebSocketClient` 连接建立、重复连接守卫、旧连接替换、sign/私聊/提示/改名/改资料指令发送、入站消息/bytes 事件分发、forceout、关闭失败 fallback 与重连调度，`VideoExtractCreateRepository` 上传元数据解析、multipart 构造、probe/createTask payload 组装与错误兜底，`VideoExtractCreateViewModel` 上传/探测/创建守卫，以及 `ChatRoomViewModel` 会话绑定、历史分页并发守卫、媒体面板缓存守卫、在线状态、通知/强制下线、非当前会话消息忽略、空 peer guard、发送超时、失败重试 fallback clientId 与清空重载分支，并覆盖默认页面 helper 与 Compose 内容层交互分支。
- 未闭环：Hilt / `MainActivity` 启动注入链路、Room/DataStore 持久化集成、真实网络与 WebSocket 生命周期、多设备并发 / 弱网场景，以及需要设备执行的 `connectedDebugAndroidTest`。
- 当前已正式接入 `./gradlew jacocoDebugUnitTestReport --no-daemon`，XML/HTML 报告输出到 `android-app/app/build/reports/jacoco/jacocoDebugUnitTestReport/`，且 `executionData` 仅统计 Debug Unit Test 明确产物。
- 2026-03-17 最新本地单测覆盖率（Debug Unit Test，fresh JaCoCo）：`line 75.19%`、`branch 44.16%`、`method 76.59%`、`class 82.11%`。
- 其中 `LiaoWebSocketClient` 当前 JaCoCo 分支覆盖率已提升至 `61.99%`（`106/171`），`VideoExtractCreateRepository` 维持 `70.24%`（`59/84`），`VideoExtractCreateViewModel` 维持 `100%`（`10/10`）。
- 当前判断：Android 已建立“协议对齐 + Repository / helper JVM 回归 + 默认页面 Compose `androidTest` 骨架”的新基线；其中可量化的 Debug Unit Test 已提升到 `line 75%+ / branch 44%+`，但设备执行与系统级集成测试仍需后续补齐。

## 编译验证
- 2026-03-13：`cd android-app && ./gradlew :app:compileDebugKotlin --no-daemon` ✅
- 2026-03-13：`cd android-app && ./gradlew testDebugUnitTest --no-daemon` ✅
- 2026-03-14：`cd android-app && ./gradlew :app:compileDebugKotlin --no-daemon` ✅
- 2026-03-14：`cd android-app && ./gradlew testDebugUnitTest --no-daemon` ✅
- 2026-03-15：`cd android-app && ./gradlew testDebugUnitTest --no-daemon` ✅（6 files / 35 tests）
- 2026-03-15：`cd android-app && ./gradlew jacocoDebugUnitTestReport --no-daemon` ✅（line 5.02% / branch 2.69%）
- 2026-03-15：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（13 files / 93 tests；line 17.50% / branch 10.15%）
- 2026-03-15：`cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（22 files / 134 tests；line 32.46% / branch 21.63%）
- 2026-03-15：`cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（28 files / 184 tests；line 44.06% / branch 31.50% / method 43.61% / class 38.25%）
- 2026-03-16：`cd android-app && ./gradlew :app:compileDebugAndroidTestKotlin --no-daemon` ✅
- 2026-03-16：`cd android-app && ./gradlew :app:assembleDebugAndroidTest --no-daemon` ✅
- 2026-03-16：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（31 files / 192 tests；line 42.84% / branch 29.69% / method 43.91% / class 39.65%；无设备，未执行 `connectedDebugAndroidTest`）
- 2026-03-16：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（33 files / 210 tests；line 43.60% / branch 31.29% / method 44.86% / class 41.40%；通过补齐 `MtPhotoSameMediaRepositoryTest` / `NetworkModelBranchTest` 等 JVM 用例回到 `branch >= 30%`）
- 2026-03-16：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（38 files / 229 tests；line 48.28% / branch 32.47% / method 50.86% / class 50.88%；继续补齐 `LoginViewModelTest` / `IdentityViewModelTest` / `ChatListViewModelTest` / `SettingsViewModelTest` 后进一步抬升分支覆盖率）
- 2026-03-16：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（39 files / 234 tests；line 50.77% / branch 33.91% / method 53.42% / class 55.79%；继续补齐 `AppCoordinatorViewModelTest` 与 `SettingsViewModelTest` 管理动作分支后进一步扩大应用级 JVM 回归范围）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（39 files / 245 tests；line 55.20% / branch 35.21% / method 57.46% / class 59.65%；继续补齐 `MtPhotoViewModelTest` 的刷新/回退/收藏/时间线守卫分支后，进一步抬升 mtPhoto 与整体 ViewModel 覆盖率）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（40 files / 248 tests；line 57.05% / branch 35.76% / method 59.48% / class 61.75%；新增 `MediaLibraryViewModelTest` 后，进一步补齐全局媒体库分页/选择/删除/同媒体查询分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（41 files / 254 tests；line 61.44% / branch 37.35% / method 64.23% / class 67.02%；新增 `DouyinViewModelTest` 后，进一步补齐抖音模式切换、解析、导入、收藏与标签管理分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（43 files / 261 tests；line 64.27% / branch 38.64% / method 66.90% / class 70.53%；新增 `VideoExtractTaskCenterViewModelTest` 后，进一步补齐抽帧任务中心列表/详情/动作分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（44 files / 269 tests；line 68.37% / branch 40.27% / method 70.94% / class 77.54%；新增 `ChatRoomViewModelTest` 后，进一步补齐聊天页绑定、历史分页、媒体面板与发送超时/重试分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（44 files / 273 tests；line 68.63% / branch 40.81% / method 71.24% / class 78.25%；继续补齐 `ChatRoomViewModelTest` 的历史分页并发守卫、媒体面板缓存守卫、入站事件忽略/通知与失败重试 fallback clientId 分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（46 files / 286 tests；line 71.51% / branch 42.28% / method 74.21% / class 81.05%；新增 `VideoExtractCreateRepositoryTest` / `VideoExtractCreateViewModelTest` 后，进一步补齐抽帧创建页上传、探测、创建与 payload/错误分支）
- 2026-03-17：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` ✅（47 files / 292 tests；line 75.19% / branch 44.16% / method 76.59% / class 82.11%；新增 `LiaoWebSocketClientTest` 后，进一步补齐 WebSocket 连接、入站事件、forceout 与关闭/重连关键分支）

## 变更历史
- [202603171930_websocket-client-coverage](../history/2026-03/202603171930_websocket-client-coverage/) - 继续补齐 LiaoWebSocketClient 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 44.16%，类分支覆盖率提升至 61.99%
- [202603171745_videoextract-create-coverage](../history/2026-03/202603171745_videoextract-create-coverage/) - 继续补齐 VideoExtractCreate Repository / ViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 42.28%
- [202603171405_videoextract-taskcenter-viewmodel-coverage](../history/2026-03/202603171405_videoextract-taskcenter-viewmodel-coverage/) - 继续补齐 VideoExtractTaskCenterViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 38.64%
- [202603171140_douyin-viewmodel-coverage](../history/2026-03/202603171140_douyin-viewmodel-coverage/) - 继续补齐 DouyinViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 37.35%
- [202603171015_android-medialibrary-viewmodel-coverage](../history/2026-03/202603171015_android-medialibrary-viewmodel-coverage/) - 继续补齐 MediaLibraryViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 35.76%
- [202603170825_chatroom-viewmodel-hotspot-followup](../history/2026-03/202603170825_chatroom-viewmodel-hotspot-followup/) - 继续补齐 ChatRoomViewModel 剩余高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 40.81%，类分支覆盖率提升至 76.39%
- [202603170713_chatroom-viewmodel-coverage](../history/2026-03/202603170713_chatroom-viewmodel-coverage/) - 继续补齐 ChatRoomViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 40.27%
- [202603161541_android-viewmodel-branch-continue](../history/2026-03/202603161541_android-viewmodel-branch-continue/) - 继续补齐 AppCoordinator / mtPhoto ViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 35.21%
- [202603161509_android-branch-coverage-rebound](../history/2026-03/202603161509_android-branch-coverage-rebound/) - 补齐 mtPhoto 同媒体、网络模型、默认页面与应用级协调器 ViewModel 高收益 JVM 单测，Debug Unit Test branch 覆盖率提升至 33.91%
- [202603161330_android-ui-androidtest-coverage-40](../history/2026-03/202603161330_android-ui-androidtest-coverage-40/) - 建立 Android 默认页面 Compose androidTest 基线，并补充 ScreenContent/testTag 与 helper JVM 单测
- [202603151234_android-branch-coverage-30](../history/2026-03/202603151234_android-branch-coverage-30/) - 继续补齐 chatroom / douyin / mtphoto 高收益单测，Debug Unit Test branch 覆盖率提升至 31.50%
- [202603131135_android-build-unblock](../history/2026-03/202603131135_android-build-unblock/) - 修复主题资源、Room/KAPT 兼容与 Kotlin/Compose API 适配，恢复 Android 编译与单测
- [202603131116_android-gradle-wrapper-upgrade](../history/2026-03/202603131116_android-gradle-wrapper-upgrade/) - 为 `android-app/` 补齐 Gradle Wrapper 8.9，并完成首次本地构建入口验证
- [202603130550_android-client-acceptance-fixes](../history/2026-03/202603130550_android-client-acceptance-fixes/) - 汇总第二轮 Android 验收修复，回写首轮 task/知识库，并经 Claude 复核确认无 P0/P1 阻断项
- [202603130243_android-native-client](../history/2026-03/202603130243_android-native-client/) - 首轮落地 Android 原生客户端工程骨架、协议基线与主流程页面结构
- [202603130550_android-client-acceptance-fixes](../history/2026-03/202603130550_android-client-acceptance-fixes/) - 第二轮验收修复：补齐身份页编辑/删除、会话列表显式状态、WS 最小协议目录、forceout(-3/-4)、真实自动重连与聊天页 Info 查询入口
- [202603130601_android-identity-chatlist-fix](../history/2026-03/202603130601_android-identity-chatlist-fix/) - 补齐 Android 身份页编辑/删除闭环，以及会话列表空态/错误态和全局收藏入口占位
