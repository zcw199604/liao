# 实施思路

1. 为 `ChatRoomViewModel` 新增纯 JVM 测试，覆盖会话绑定、历史分页、媒体面板、WebSocket 入站事件、文本发送与失败重试等高收益分支。
2. 复用现有 `MainDispatcherRule`、`MutableStateFlow` / `MutableSharedFlow` 与 mockk，不改动生产代码。
3. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
4. 迁移方案包到 `history/`。
