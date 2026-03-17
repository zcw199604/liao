# 背景

上一轮已建立 Android 默认页面的 Compose `androidTest` 基线，但同时让 Debug Unit Test JaCoCo 分支覆盖率从 31.50% 回落到 29.69%。

# 目标

- 继续补齐高收益 JVM 单测，优先选择无需设备、无需 Hilt/Room/网络真依赖的分支热点，并把默认页面与应用级协调器 ViewModel 纳入可测范围。
- 让 Android Debug Unit Test branch 覆盖率回到 **30%+**。
- 保持现有 `androidTest` 编译/装配链路不回退。
