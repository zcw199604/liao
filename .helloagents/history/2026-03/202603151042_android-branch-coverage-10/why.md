# 背景

当前 Android `app` 模块已正式接入 `jacocoDebugUnitTestReport`，但 Debug Unit Test 的 branch 覆盖率仅约 2.69%，无法体现近期新增纯单元测试对业务逻辑的实际保护范围。

# 问题

1. JaCoCo 当前会把大量 Kotlin/Compose 生成的 `*Kt$*` 合成内部类计入分支统计，稀释了手写逻辑的单元测试覆盖率。
2. `videoextract / douyin / mtphoto / settings` 等文件中的纯 helper 分支尚未被测试触达。
3. 用户期望将 Android branch 覆盖率推进到 10% 左右，需要一条最小可行、可持续复用的提测路径。

# 目标

1. 在保持统计口径可解释的前提下，优化 Android 单元覆盖率任务的排除范围。
2. 为高收益纯 helper 补充单元测试，优先覆盖 `videoextract / douyin / mtphoto / settings` 的条件分支。
3. 复跑 `jacocoDebugUnitTestReport`，将 branch 覆盖率提升到接近或达到 10%。
