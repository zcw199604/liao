# 背景

当前 Android Debug Unit Test branch 已提升到 42.28%，但 `LiaoWebSocketClient` 仍处于 0% 覆盖状态，是当前最大的剩余 JVM 热点之一。

# 目标

- 在不改动生产代码的前提下，为 `LiaoWebSocketClient` 补充纯 JVM 单测。
- 优先覆盖连接建立、重复连接守卫、旧连接替换、发送指令、入站事件分发、forceout 与关闭/重连调度关键分支。
- 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，继续提升 Android branch 覆盖率。
