# 实施思路

1. 在现有 `ChatRoomViewModelTest` 基础上继续补充剩余高收益分支，不改动生产代码。
2. 复用 `MainDispatcherRule`、mockk、`MutableStateFlow` / `MutableSharedFlow`；必要时通过 mock `LiaoLogger` 避开 Android Log 依赖。
3. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
4. 迁移方案包到 `history/`。
