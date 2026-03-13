# 任务清单: Android 客户端验收问题修复

目录: `.helloagents/plan/202603130550_android-client-acceptance-fixes/`

---

## 1. 身份模块修复
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/identity/IdentityFeature.kt` 中补齐身份编辑与删除闭环，验证 why.md#需求-修正身份模块完成度-场景-用户维护身份池（已由 `202603130601_android-identity-chatlist-fix` 落地）

## 2. 会话列表修复
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt` 中补齐显式空态、错误态和全局收藏入口，验证 why.md#需求-修正会话列表完成度-场景-会话列表无数据或加载失败（已由 `202603130601_android-identity-chatlist-fix` 落地）

## 3. WebSocket 与聊天链路修复
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClient.kt` 中建立最小协议目录并补齐 `code=-3/-4` forceout 与真实自动重连，验证 why.md#需求-修正-websocket-协议与重连实现-场景-聊天连接异常（已由 `202603130550_android-ws-protocol-reconnect` 落地）
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatroom/ChatRoomFeature.kt` 中消费结构化 WS 消息结果并保持现有最小聊天链路可用，依赖任务3.1（已补齐顶部 Info 按钮触发 `ShowUserLoginInfo` 与消息消费）

## 4. 文档与验收同步
- [√] 更新 `.helloagents/history/2026-03/202603130243_android-native-client/task.md` 与相关知识库，回写本轮修复结果
- [√] 执行验证并迁移当前方案包至历史记录


---

## 执行备注
- 本轮验收修复拆分为两个子方案包执行：`202603130601_android-identity-chatlist-fix` 与 `202603130550_android-ws-protocol-reconnect`。
- 已补齐聊天页顶部 Info 按钮主动请求 `ShowUserLoginInfo`，并增加 Snackbar 消息消费，避免旧提示残留。
- 已执行 `go test ./...` 通过；Android 侧因当前环境缺少 Java / Android SDK / Gradle Wrapper，未执行编译与单元测试。
- 已复用 Claude 会话 `5928806c-8b29-4c2e-8312-3ceb526e046a` 做只读复核，结论为“有条件通过”，且无 P0/P1 剩余问题。
