# 实施思路

1. 为 `MediaLibraryViewModel` 新增纯 JVM 测试，覆盖分页加载守卫、选择模式切换、批量/单项删除、同媒体查询成功失败与弹层关闭分支。
2. 复用现有 `MainDispatcherRule` 与 mockk，不改动生产代码。
3. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
4. 迁移方案包到 `history/`。
