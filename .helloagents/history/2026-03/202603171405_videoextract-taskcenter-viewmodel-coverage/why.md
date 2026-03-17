# 背景

Android Debug Unit Test JaCoCo branch 已提升到 37.35%，但 `VideoExtractTaskCenterViewModel` 仍是当前 0% branch 覆盖的高收益空白点，且已有 `VideoExtractTaskCenterRepositoryTest` / `VideoExtractTaskCenterFeatureHelpersTest` 作为测试基础。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `VideoExtractTaskCenterViewModel` 的任务列表刷新/分页、详情切换、帧分页、继续/终止/删除动作与消息消费分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
