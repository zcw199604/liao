# Liao Android 客户端骨架

## 当前状态

本目录提供 Liao 原生 Android 客户端的首期工程骨架，目标是与现有 Go 服务端保持协议兼容，并为后续功能扩展提供稳定的 Kotlin + Compose 基础设施。

当前已落地：
- 单 `app` module 的 Android 工程骨架，按包结构模拟模块化分层
- Compose 导航入口、Hilt、Retrofit、OkHttp、Room、DataStore、WorkManager、Coil 依赖配置
- 登录 / 身份选择 / 会话列表 / 聊天页 / 设置页最小可运行界面骨架
- Base URL / JWT / WebSocket `sign` / forceout 5 分钟限制的基础实现
- 与现有后端路径对齐的 API Service 拆分（auth / identity / chat / favorite / media / mtphoto / douyin / system / videoExtract）

尚未完全落地：
- 媒体上传/重传/查重的完整交互
- mtPhoto / 抖音 / 视频抽帧的完整页面与业务流程
- UI 自动化测试与真机联调脚本
- Gradle Wrapper（二进制未在当前环境生成）

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
3. 使用本机 Gradle 或由 Android Studio 补全 Wrapper
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

## 已知限制

- 当前仓库环境未安装 JDK / Android SDK，因此本次提交未执行 Android 侧编译与仪器测试
- 复杂业务模块（媒体管理、抖音、mtPhoto、抽帧）仅完成接口与结构预留，未完成完整交互闭环
- 当前骨架先采用“单 module + 包内分层”的方式降低起步成本，后续可平滑拆分为多 Gradle module
