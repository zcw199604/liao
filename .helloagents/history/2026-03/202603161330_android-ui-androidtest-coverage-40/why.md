# 变更提案: Android UI / androidTest 覆盖补充

## 需求背景
- 2026-03-15 Android `app` 的 Debug Unit Test fresh JaCoCo 已达到 `line 44.06% / branch 31.50% / method 43.61% / class 38.25%`，说明 Repository / helper 的 JVM 回归基线已较稳定。
- 用户希望继续补齐 **Compose UI 测试** 与 **`androidTest` 仪器测试**，并允许为测试补充 `Semantics tag`、Fake / 测试入口、测试基类等最小基础设施。
- 当前工程已配置 `testInstrumentationRunner` 与 `androidx.compose.ui:ui-test-junit4`，但仓库中尚无 `app/src/androidTest/` 目录，也没有默认页面的 UI 可测性约定，因此需要先完成测试骨架与页面可测性建设。
- ⚠️ 不确定因素: 用户设定的“覆盖率到 40%”在当前 Debug Unit Test 口径下已超额达成；基于当前信息，本轮将其解释为：**在继续补 UI / androidTest 覆盖的同时，保持 Android 整体可量化覆盖率不低于 40%，并建立可持续扩展的 androidTest 基线**。

## 变更内容
1. 为 Android 客户端补齐 `androidTest` 测试目录、基础依赖、测试基类与默认页面的首批 Compose UI 仪器测试骨架。
2. 为默认页面（按“默认”假设为 **登录 / 身份 / 会话列表 / 设置**）增加最小 `Semantics tag`、可复用的纯 UI Content 层或测试友好的状态入口，降低页面 UI 回归成本。
3. 在无模拟器/真机场景下，同时补充可本地执行的 UI 相关 JVM 测试 / 状态映射测试，确保现有可量化覆盖率保持在 `40%+`。
4. 补充文档、执行说明与历史索引，为后续接入设备或 CI 后直接运行 `connectedDebugAndroidTest` 做准备。

## 影响范围
- **模块:** Android client / testing / auth / identity / chatlist / settings / app navigation
- **文件:**
  - `android-app/app/build.gradle.kts`
  - `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/app/**`
  - `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/{auth,identity,chatlist,settings}/**`
  - `android-app/app/src/test/**`
  - `android-app/app/src/androidTest/**`
- **API:** 无外部接口协议变更
- **数据:** 无持久化 schema 变更

## 核心场景
### 需求: Android UI / androidTest 覆盖继续补齐
**模块:** android-client
在保持当前 Debug Unit Test 可量化基线的前提下，补齐默认页面的 UI 可测性与 `androidTest` 仪器测试骨架。

#### 场景: 默认页面具备稳定的 UI 可测性
登录、身份、会话列表、设置页增加统一的测试 tag 与可复用 Content 层，使交互与状态断言可在 Compose UI Test 中稳定定位。
- 预期结果: 默认页面不再依赖纯文本查找与脆弱节点结构，后续 UI 回归可持续扩展。

#### 场景: androidTest 骨架可编译并具备设备执行入口
在仓库中新增 `app/src/androidTest/` 测试目录、Compose rule / 测试 helper 与首批页面用例；当前无设备时至少完成 `compileDebugAndroidTestKotlin` / `assembleDebugAndroidTest` 级别验证。
- 预期结果: 一旦后续补齐模拟器、真机或 CI 设备，即可直接执行 `connectedDebugAndroidTest`。

#### 场景: 可量化覆盖率保持在 40% 以上
针对为 UI 测试抽取出的状态映射、tag 常量、Content helper 等补充 JVM 测试，保证现有 Debug Unit Test 覆盖率不因架构调整而回退。
- 预期结果: 当前可复现的覆盖率结果保持 `40%+`，并为后续将 UI 覆盖纳入统计打基础。

## 风险评估
- **风险:** 当前没有可用模拟器/真机，无法本地完整执行 `connectedDebugAndroidTest`。
- **缓解:** 本轮先以 `compileDebugAndroidTestKotlin`、`assembleDebugAndroidTest`、测试源码自检与文档说明完成第一阶段闭环，并为后续设备环境保留直接运行入口。
- **风险:** 现有页面直接依赖具体 ViewModel，导致 Compose UI Test 难以注入假数据。
- **缓解:** 优先抽取轻量的 `*ScreenContent` 或测试友好的 UI 状态入口，仅做最小结构调整，不改变业务逻辑。
- **风险:** 大量新增 tag / Content 层可能引发 UI 代码噪音。
- **缓解:** 统一集中管理测试 tag 常量，限制在默认页面主路径节点上使用，避免全量泛滥。
- **风险:** 用户口中的“40%”与当前 JaCoCo 口径可能不完全一致。
- **缓解:** 在文档与执行结果中明确区分“当前 Debug Unit Test fresh 覆盖率”和“androidTest 设备执行状态”，避免误读。
