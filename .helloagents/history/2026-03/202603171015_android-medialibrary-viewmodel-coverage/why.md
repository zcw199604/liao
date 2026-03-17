# 背景

Android Debug Unit Test JaCoCo branch 已提升到 35.21%，但 `MediaLibraryViewModel` 仍是当前 JVM 覆盖率中的显著空白点，且已有 `MediaLibraryRepositoryTest` / `MediaLibraryHelpersTest` 作为测试基础。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `MediaLibraryViewModel` 的分页、选择、删除、同媒体查询与消息消费分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
