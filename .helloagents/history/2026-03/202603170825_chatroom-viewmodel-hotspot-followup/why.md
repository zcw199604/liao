# 背景

Android Debug Unit Test JaCoCo branch 已提升到 40.27%，`ChatRoomViewModel` 仍有一批高收益剩余分支，尤其集中在历史分页守卫、入站事件分发、媒体面板和失败重试路径。

# 目标

- 继续补充无需设备、无需生产代码改动的 JVM ViewModel 单测。
- 优先覆盖 `ChatRoomViewModel` 剩余的分页守卫、通知/强制下线事件、非当前会话消息忽略、空 peer 兜底与重试成功路径。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，在保持现有测试链路稳定的前提下继续提升 Android branch 覆盖率。
