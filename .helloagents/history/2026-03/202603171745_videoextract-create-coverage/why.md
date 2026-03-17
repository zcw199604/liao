# 背景

当前 Android Debug Unit Test branch 已提升到 40.81%，但 `VideoExtractCreateRepository` 与 `VideoExtractCreateViewModel` 仍处于 0% 覆盖状态，是下一批高收益 JVM 热点。

# 目标

- 在不改动生产代码的前提下，为 `VideoExtractCreateRepository` 与 `VideoExtractCreateViewModel` 补充纯 JVM 单测。
- 优先覆盖视频上传元数据解析、探测/创建任务成功失败分支，以及创建页 ViewModel 的上传、探测、创建与消息消费守卫。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，继续提升 Android branch 覆盖率。
