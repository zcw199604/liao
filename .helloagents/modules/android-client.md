# Android Client

## 目的
为现有 Go 服务端补充原生 Android 客户端骨架，承接登录、身份、会话列表、聊天与后续媒体能力扩展。

## 模块概述
- **职责:** 提供 Kotlin + Jetpack Compose 客户端基础设施，复用现有 HTTP / WebSocket 协议，并为媒体、抖音、mtPhoto、视频抽帧等模块预留扩展位。
- **状态:** 🚧开发中
- **最后更新:** 2026-03-13

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
- 设置页可以修改 Base URL，并查看 Token 与当前身份快照
- 身份页已支持创建、快速创建、编辑、删除、选择；编辑当前身份会同步当前会话，删除当前身份会清理本地当前会话
- 会话列表已补齐显式空态、显式错误态与“全局收藏入口（占位）”按钮，后续可直接接入独立收藏页
- 聊天页已消费结构化 WebSocket 事件；头部 Info 按钮会触发 `ShowUserLoginInfo` 查询，消息提示会在 Snackbar 展示后消费
- 聊天页顶部 Info 按钮可主动重新发送 `ShowUserLoginInfo`，用于刷新对端资料与对齐 Web 端最小主动动作

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

## 变更历史
- [202603131135_android-build-unblock](../history/2026-03/202603131135_android-build-unblock/) - 修复主题资源、Room/KAPT 兼容与 Kotlin/Compose API 适配，恢复 Android 编译与单测
- [202603131116_android-gradle-wrapper-upgrade](../history/2026-03/202603131116_android-gradle-wrapper-upgrade/) - 为 `android-app/` 补齐 Gradle Wrapper 8.9，并完成首次本地构建入口验证
- [202603130550_android-client-acceptance-fixes](../history/2026-03/202603130550_android-client-acceptance-fixes/) - 汇总第二轮 Android 验收修复，回写首轮 task/知识库，并经 Claude 复核确认无 P0/P1 阻断项
- [202603130243_android-native-client](../history/2026-03/202603130243_android-native-client/) - 首轮落地 Android 原生客户端工程骨架、协议基线与主流程页面结构
- [202603130550_android-client-acceptance-fixes](../history/2026-03/202603130550_android-client-acceptance-fixes/) - 第二轮验收修复：补齐身份页编辑/删除、会话列表显式状态、WS 最小协议目录、forceout(-3/-4)、真实自动重连与聊天页 Info 查询入口
- [202603130601_android-identity-chatlist-fix](../history/2026-03/202603130601_android-identity-chatlist-fix/) - 补齐 Android 身份页编辑/删除闭环，以及会话列表空态/错误态和全局收藏入口占位
