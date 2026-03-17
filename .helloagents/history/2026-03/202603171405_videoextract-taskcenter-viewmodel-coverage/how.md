# 实施思路

1. 为 `VideoExtractTaskCenterViewModel` 新增纯 JVM 测试，覆盖初始化刷新、分页加载、详情切换、帧分页与任务动作分支。
2. 复用现有 `MainDispatcherRule` 与 mockk，不改动生产代码。
3. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
4. 迁移方案包到 `history/`。
