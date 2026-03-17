# 背景

Android Debug Unit Test JaCoCo branch 已回升到 33.91%，但 `AppCoordinatorViewModel` 与 `MtPhotoViewModel` 仍是当前 JVM 分支覆盖率的主要空白热点。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `AppCoordinatorViewModel` 与 `MtPhotoViewModel` 的初始化、事件消费、守卫条件与错误回退分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
