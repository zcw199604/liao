# 任务清单: Android WebSocket 协议与重连修复

目录: `.helloagents/plan/202603130550_android-ws-protocol-reconnect/`

---

## 1. WebSocket 协议与状态机
- [√] 1.1 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClient.kt` 中建立最小可用的 `code/act` 协议目录与结构化入站解析，验证 why.md#需求-最小协议目录与结构化解析-场景-聊天页消费-websocket-消息
- [√] 1.2 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClient.kt` 中补齐 `code=-3/-4` forceout 与真实自动重连，验证 why.md#需求-forceout-与自动重连修复-场景-聊天连接异常，依赖任务1.1

## 2. 聊天页消费链路
- [√] 2.1 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatroom/ChatRoomFeature.kt` 中消费结构化 WS 事件并保持最小文本聊天链路可用，验证 why.md#需求-最小协议目录与结构化解析-场景-聊天页消费-websocket-消息，依赖任务1.1

## 3. 安全检查
- [√] 3.1 执行安全检查（输入验证、forceout 禁重连、避免过期 socket 回调污染状态）

## 4. 文档更新
- [√] 4.1 更新 `.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引/方案记录

## 5. 测试
- [√] 5.1 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/ProtocolAlignmentTest.kt` 中补充协议解析断言

---

## 执行备注
- 已完成最小 WS `code/act` 协议目录、结构化解析、`code=-3/-4` forceout 与真实自动重连。
- 已补充 `ProtocolAlignmentTest` 的协议目录 / 解析断言。
- 当前环境缺少 Java / Android SDK / Gradle Wrapper，未执行 Android 侧编译与单元测试，只完成静态检查与文件级校对。
