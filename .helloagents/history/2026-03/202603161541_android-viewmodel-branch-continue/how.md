# 实施思路

1. 为 `AppCoordinatorViewModel` 新增纯 JVM 测试，覆盖启动路由切换、WebSocket 事件持久化、Forceout 清理与消息消费分支。
2. 为 `MtPhotoViewModel` 新增纯 JVM 测试，覆盖相册翻页、目录延迟时间线、目录收藏、刷新、回退、预览导入与错误回退分支。
3. 如有必要，补充已有 ViewModel 测试的剩余边界分支。
4. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引后归档方案包。
