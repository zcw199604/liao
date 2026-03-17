# 背景

Android Debug Unit Test JaCoCo branch 已提升到 38.64%，但 `ChatRoomViewModel` 仍是当前 0% branch 覆盖的高收益空白点，且已有 `ChatRoomFeatureHelpersTest` / `ChatRoomRepositoryTest` 作为测试基础。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `ChatRoomViewModel` 的绑定初始化、历史分页、媒体面板、入站事件、发送超时与失败重试分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
