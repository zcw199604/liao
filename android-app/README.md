# Liao Android 客户端骨架

## 当前状态

本目录提供 Liao 原生 Android 客户端的首期工程骨架，目标是与现有 Go 服务端保持协议兼容，并为后续功能扩展提供稳定的 Kotlin + Compose 基础设施。

当前已落地：
- 单 `app` module 的 Android 工程骨架，按包结构模拟模块化分层
- Compose 导航入口、Hilt、Retrofit、OkHttp、Room、DataStore、WorkManager、Coil 依赖配置
- 登录 / 身份选择 / 会话列表 / 聊天页 / 设置页最小可运行界面骨架
- 登录 / 身份 / 会话列表 / 设置四个默认页面已抽出可测 `ScreenContent` 与统一 test tags，并建立首批 Compose `androidTest` 基线（13 条 UI 用例，当前在无设备场景下已验证编译/装配）
- Base URL / JWT / WebSocket `sign` / forceout 5 分钟限制的基础实现
- 与现有后端路径对齐的 API Service 拆分（auth / identity / chat / favorite / media / mtphoto / douyin / system / videoExtract）

尚未完全落地：
- `connectedDebugAndroidTest` 真机/模拟器执行链路与更完整的 UI 回归脚本
- Room / DataStore / Hilt / WorkManager 等集成测试
- 复杂弱网、断线重连与多设备并发场景的端到端回归

## 默认约定

- 工程位置：`android-app/`
- 包名 / namespace：`io.github.a7413498.liao.android`
- `minSdk = 26`
- `targetSdk = 35`
- 默认联调地址：`http://10.0.2.2:8080/api/`
- 默认按网页端兼容模式复刻 `reportReferrer`、`ShowUserLoginInfo`、`warningreport` 与 `forceout` 处理

## 本地打开方式

1. 使用 Android Studio 打开 `android-app/`
2. 安装 Android SDK 35 与 JDK 17
3. 直接使用仓库内 `./gradlew`，或由 Android Studio 导入后执行 Gradle 任务
4. 连接后端：在设置页或登录页把 API Base URL 指向可从设备访问的地址
   - 模拟器：`http://10.0.2.2:8080/api/`
   - 真机：`http://<你的局域网IP>:8080/api/`
5. 启动应用后，按 `登录 -> 身份 -> 会话列表 -> 聊天页` 的路径联调

## 联调说明

- HTTP 统一前缀：`/api`
- WebSocket 路径：`/ws?token=<JWT>`
- 登录接口：`POST /api/auth/login`，表单字段 `accessCode`
- Token 校验：`GET /api/auth/verify`
- 身份列表：`GET /api/getIdentityList`
- 历史 / 收藏会话：`POST /api/getHistoryUserList`、`POST /api/getFavoriteUserList`
- 聊天历史：`POST /api/getMessageHistory`
- 连接后首条消息：`{ act: "sign", id, name, userSex, userip, useraddree, randomvipcode, ... }`
- forceout：服务端当前以 **5 分钟** 为准，客户端禁重连倒计时也按 5 分钟处理

## 目录结构

```text
android-app/
  app/
    src/main/kotlin/io/github/a7413498/liao/android/
      app/
      core/common/
      core/datastore/
      core/network/
      core/database/
      core/websocket/
      feature/auth/
      feature/identity/
      feature/chatlist/
      feature/chatroom/
      feature/settings/
```

## 测试覆盖情况

- 当前已具备 **44 个 Android 本地单元测试文件 / 269 条用例**，并配套 `MainDispatcherRule.kt` 统一管理 `Dispatchers.Main`；当前 JVM 回归已覆盖协议、Repository、helper 与关键 ViewModel，可通过 `cd android-app && ./gradlew testDebugUnitTest --no-daemon` 执行。
- 当前已具备 **4 个 Compose `androidTest` 页面测试文件 / 13 条仪器用例**，覆盖登录 / 身份 / 会话列表 / 设置四个默认页面；另有 1 个 `ComposeUiTestSupport.kt` 提供统一 `LiaoTheme` 测试包装。
- `ProtocolAlignmentTest.kt` / `NetworkModelAlignmentTest.kt` / `WebSocketProtocolCatalogTest.kt` / `CommonAndCacheSnapshotTest.kt` / `NetworkStackUrlTest.kt` / `MediaLibraryHelpersTest.kt`：覆盖协议目录、DTO/时间线映射、缓存快照、URL 重写、媒体 MIME/展示文案等纯 Kotlin 基线逻辑。
- `NetworkModelBranchTest.kt`：补齐 `ChatMessageDto/HistoryMessageDto` 时间线映射、收藏 JSON 解析、`stringOrNull` / `inferPrivateMessageIsSelf` / `inferMessageType` 的边界与容错分支。
- `LoginViewModelTest.kt` / `IdentityViewModelTest.kt` / `ChatListViewModelTest.kt` / `SettingsViewModelTest.kt` / `AppCoordinatorViewModelTest.kt` / `MtPhotoViewModelTest.kt` / `MediaLibraryViewModelTest.kt` / `DouyinViewModelTest.kt` / `VideoExtractTaskCenterViewModelTest.kt` / `ChatRoomViewModelTest.kt`：补齐默认页面、应用级启动协调器、mtPhoto、全局媒体库、抖音解析、抽帧任务中心与聊天页 ViewModel 的初始化、刷新、切页、选择、删除、同媒体查询、消息消费、保存、forceout、目录收藏、时间线延迟加载、标签管理、历史分页、媒体面板、发送超时与任务动作等 JVM 分支，并配套 `MainDispatcherRule.kt` 提供 `Dispatchers.Main` 测试调度器。
- `AuthRepositoryTest.kt` / `IdentityRepositoryTest.kt` / `ChatListRepositoryTest.kt` / `GlobalFavoritesRepositoryTest.kt` / `MediaLibraryRepositoryTest.kt` / `SettingsRepositoryTest.kt` / `MtPhotoSameMediaRepositoryTest.kt` / `AuthFeatureHelpersTest.kt` / `IdentityFeatureHelpersTest.kt` / `ChatListFeatureHelpersTest.kt` / `SettingsFeatureHelpersTest.kt`：覆盖登录/恢复会话、身份 CRUD、列表合并、全局收藏、媒体库、mtPhoto 同媒体查询、系统配置 fallback，以及默认页面 helper / 文案 / 按钮启用条件分支。
- `VideoExtractCreateFeatureHelpersTest.kt` / `VideoExtractTaskCenterFeatureHelpersTest.kt` / `VideoExtractTaskCenterRepositoryTest.kt` / `VideoExtractTaskCenterViewModelTest.kt`：覆盖视频抽帧创建校验、payload 组装、任务中心文案/路径/JSON 映射，以及任务列表/详情缓存 fallback、删除更新与 ViewModel 刷新/分页/详情/继续/终止/删除动作分支。
- `ChatRoomFeatureHelpersTest.kt` / `ChatRoomRepositoryTest.kt` / `ChatRoomViewModelTest.kt`：覆盖聊天时间线合并、回显匹配、显示文案、连接建立、收藏切换、历史回退、媒体 URL 解析，以及 ViewModel 绑定初始化、历史分页、媒体面板、在线状态、发送超时、失败重试与清空重载等高价值分支。
- `DouyinFeatureHelpersTest.kt` / `DouyinRepositoryTest.kt` / `DouyinRepositoryOperationsTest.kt` / `DouyinRepositoryMappingsTest.kt` / `DouyinRepositoryBranchTest.kt` / `DouyinViewModelTest.kt`：覆盖抖音媒体类型归一化、导入文案、JSON 字段解析、详情/账号/收藏分页，以及 ViewModel 模式切换、导入、收藏、标签编辑/管理与消息消费等成功/失败/边界分支。
- `MtPhotoFeatureHelpersTest.kt` / `MtPhotoFolderFavoritesRepositoryTest.kt` / `MtPhotoRepositoryTest.kt` / `MtPhotoRepositoryMappingsTest.kt` / `MtPhotoRepositoryBranchTest.kt`：覆盖 mtPhoto 可见项计数、错误文案、目录名/封面 MD5 归一化，以及目录收藏/相册目录仓储的成功、失败、fallback、默认值、映射与缩略图 URL 分支。
- Android UI 测试当前直接面向 `LoginScreenContent`、`IdentityScreenContent`、`ChatListScreenContent`、`SettingsScreenContent`，避免 Hilt / Room / DataStore / WebSocket 实依赖，先建立稳定的页面内容 smoke 基线。
- 现已正式接入覆盖率命令：`cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，会输出 XML/HTML 报告到 `app/build/reports/jacoco/jacocoDebugUnitTestReport/`，且 `executionData` 仅统计 Debug Unit Test 明确产物。
- 2026-03-17 最新本地单测覆盖率（Debug Unit Test，fresh JaCoCo）：`line 68.37%`、`branch 40.27%`、`method 70.94%`、`class 77.54%`。
- 2026-03-16 已验证：`./gradlew :app:compileDebugAndroidTestKotlin --no-daemon` ✅、`./gradlew :app:assembleDebugAndroidTest --no-daemon` ✅；当前环境**无模拟器/真机**，因此尚未执行 `connectedDebugAndroidTest`。
- 当前**未覆盖/未闭环**：Hilt 注入链路、Room/DataStore 持久化集成、真实网络/WebSocket 生命周期、系统权限与多设备并发/弱网场景，以及基于设备的真实 `androidTest` 执行结果。
- 因此现阶段 Android 测试覆盖属于“**JVM 回归已提升到 line 68%+ / branch 40%+，同时已建立四个默认页面的 Compose `androidTest` 骨架，但设备执行与系统级集成回归仍待补齐**”的状态。

## 已知限制

- 当前虽已具备 `src/androidTest/` 首批 Compose UI 基线，但缺少可用模拟器/真机，`connectedDebugAndroidTest` 与实际交互回归仍待执行
- Room/DataStore/WebSocket 等集成链路尚未建立 fake/server 驱动的系统级测试矩阵
- 当前工程先采用“单 module + 包内分层”的方式降低起步成本，后续可按能力域继续拆分多 Gradle module
