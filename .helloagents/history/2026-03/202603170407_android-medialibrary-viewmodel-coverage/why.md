> 说明：该方案为重复草稿，未实际执行；后续已由 `202603171015_android-medialibrary-viewmodel-coverage` 方案替代并完成。

# 背景

Android Debug Unit Test JaCoCo branch 已提升到 35.21%，但 `MediaLibraryViewModel` 仍是当前 0% branch 覆盖的高收益空白点。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `MediaLibraryViewModel` 的分页、批量选择、删除、同媒体查询与弹窗状态分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
