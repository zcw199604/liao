# 变更提案：Android branch 覆盖率提升到 10%

## 背景
- 2026-03-15 已为 Android `app` 模块接入 `jacocoDebugUnitTestReport`。
- 当前 Debug Unit Test 覆盖率为：`line 5.02% / branch 2.69% / method 6.82% / class 5.12%`。
- 现有单测主要集中在协议、DTO、URL 与少量 helper，尚未覆盖多个 repository 的业务分支。

## 问题
1. Android 分支覆盖率过低，无法为后续 Android 端持续演进提供足够回归保障。
2. JaCoCo 当前统计包含大量 Kotlin/Compose 生成的 `*Kt$*` 合成类分支，噪声较高，不利于衡量单元测试对可维护业务逻辑的覆盖质量。
3. 多个 repository（Auth / Identity / ChatList / GlobalFavorites / MediaLibrary / Settings）拥有较多未覆盖分支，但具备通过 mock/fake 做 JVM 单测的条件。

## 目标
- 将 Android `branch coverage` 提升到 **至少 10%**。
- 保持覆盖率统计可复现：继续通过 `./gradlew jacocoDebugUnitTestReport --no-daemon` 生成报告。
- 优先覆盖“高收益且可稳定单测”的业务逻辑，而不是堆叠脆弱 UI 测试。

## 非目标
- 本轮不补 Compose UI / `androidTest` 仪器测试。
- 本轮不追求 Android 全模块高覆盖，仅完成 branch 指标的首轮跨越。

## 策略摘要
- 对覆盖率任务排除明显的 Kotlin/Compose 合成 `*Kt$*` 类，保留真实业务类与 top-level helper。
- 为高分支 repository 补充基于 MockK 的 JVM 单测。
- 视结果补充 settings / media / douyin / mtphoto 中的纯函数分支测试。
