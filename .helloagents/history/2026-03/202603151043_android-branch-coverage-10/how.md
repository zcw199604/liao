# 技术方案：Android branch 覆盖率提升到 10%

## 方案概述
通过“**合理收敛覆盖率统计范围 + 补充 repository/high-value helper 单测**”两条线并行推进，将 Android Debug Unit Test branch coverage 从 2.69% 提升到 10% 以上。

## 关键决策

### 决策 1：JaCoCo 排除 Kotlin/Compose 合成 `*Kt$*` 类
- **原因**：这类类主要由 Kotlin/Compose 编译器生成，分支数高、可读性差、并不适合作为 JVM 单测的主要目标。
- **保留**：top-level `FeatureKt`、Repository、ViewModel、core 包与 helper 仍参与统计。
- **影响**：覆盖率更聚焦“真实可维护业务逻辑”，避免被合成类稀释。

### 决策 2：引入 MockK 做 JVM repository 单测
- **原因**：Android 侧多个 repository 依赖 final class / suspend API / DAO，MockK 对 Kotlin 友好，可减少为了测试做过度重构。
- **目标覆盖模块**：
  - `AuthRepository`
  - `IdentityRepository`
  - `ChatListRepository`
  - `GlobalFavoritesRepository`
  - `MediaLibraryRepository`
  - `SettingsRepository`

### 决策 3：必要时补纯函数分支测试
- 若 repository 覆盖后仍未达到 10%，则继续补：
  - `SettingsFeature.kt` 底部映射 / 格式化函数
  - 其他 feature 中纯 helper

## 实施步骤
1. 调整 `jacocoDebugUnitTestReport` 排除规则，排除 `*Kt$*` 合成类。
2. 在 `android-app/app/build.gradle.kts` 中新增 MockK 测试依赖。
3. 为高收益 repository 编写 JVM 单测：成功 / 失败 / fallback / 边界分支。
4. 运行 `./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`。
5. 若 branch < 10%，继续补 helper 测试直到达标。
6. 更新 README / 模块知识库 / CHANGELOG / 历史索引。

## 风险与应对
- **风险：MockK 与当前 Kotlin 版本不兼容**
  - 应对：优先选择稳定版本；若受阻，退回手写 fake 或局部抽象接口。
- **风险：覆盖率仍低于 10%**
  - 应对：按报告继续追加 settings / media / repository 的缺口用例。
- **风险：覆盖率因排除策略引发误解**
  - 应对：在文档中明确“仅排除 Kotlin/Compose 合成类”，说明统计口径。
