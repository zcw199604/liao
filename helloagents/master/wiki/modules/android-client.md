# Android Client

## 目的
提供 Liao 的 Android 客户端，实现与 Web 客户端对齐的登录、身份、聊天、媒体、抖音、mtPhoto、视频抽帧和设置能力。

## 模块概述
- **职责:** Kotlin/Compose UI、API/WS 网络栈、本地缓存、Room/DataStore、功能页面与单元/UI 测试。
- **状态:** 开发中
- **最后更新:** 2026-05-07

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
