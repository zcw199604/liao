# 实施思路

1. 新增 `LiaoWebSocketClientTest.kt`，使用 mockk 捕获 `OkHttpClient.newWebSocket(...)` 的 `Request` 与 `WebSocketListener`。
2. 通过模拟 `onOpen/onMessage/onClosing/onClosed/onFailure`，覆盖发送 sign、指令下发、入站协议分发、forceout、旧连接失效守卫与关闭回退逻辑。
3. 使用 `mockkObject(LiaoLogger)` 避免 JVM 单测中触发 Android Log。
4. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
5. 完成后迁移方案包到 `history/`。
