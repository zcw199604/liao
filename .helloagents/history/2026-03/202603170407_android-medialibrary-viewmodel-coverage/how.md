> 说明：该方案为重复草稿，未实际执行；后续已由 `202603171015_android-medialibrary-viewmodel-coverage` 方案替代并完成。

# 实施思路

1. 为 `MediaLibraryViewModel` 新增纯 JVM 测试，覆盖分页加载、缓存 banner、重复请求守卫、选择模式、删除成功/失败与同媒体查询分支。
2. 如覆盖率结果有余量，再视情况补充已存在 ViewModel 测试的剩余边界分支。
3. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引后归档方案包。
