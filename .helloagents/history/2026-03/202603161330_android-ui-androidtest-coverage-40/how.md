# 技术设计: Android UI / androidTest 覆盖补充

## 技术方案
### 核心技术
- Kotlin / Jetpack Compose / AndroidJUnitRunner / Compose UI Test / JUnit4 / MockK / kotlinx-coroutines-test / JaCoCo

### 方案选择
- 已选方案: **双轨推进**
- 选择原因:
  1. 当前仓库尚无 `src/androidTest/`，需要先补齐首轮仪器测试骨架与 UI 可测性基础设施。
  2. 当前无可用设备环境，若只做 `androidTest` 将无法本地形成完整验证闭环；因此需要同时补充可本地执行的 UI 相关 JVM 测试，以保持 `40%+` 的可量化覆盖率。
  3. 默认页面（登录 / 身份 / 会话列表 / 设置）业务耦合相对较低，适合作为首批 UI/仪器测试切入点。

## 实现要点
### 1. androidTest 基础设施补齐
- 在 `android-app/app/build.gradle.kts` 中补齐 `androidTest` 所需依赖（如 `androidx.test.ext:junit`、`espresso-core`、必要时的 `testRules`），并明确 `debugImplementation("ui-test-manifest")` 与 `androidTestImplementation("ui-test-junit4")` 的组合。
- 新增 `android-app/app/src/androidTest/kotlin/...` 目录，提供:
  - Compose UI Test 通用基类 / Rule 封装
  - 默认页面测试 helper
  - 无设备环境下的执行说明
- 验证目标优先级:
  1. `:app:compileDebugAndroidTestKotlin`
  2. `:app:assembleDebugAndroidTest`
  3. 有设备时再执行 `:app:connectedDebugAndroidTest`

### 2. 默认页面 UI 可测性改造
- 对下列页面增加统一 `Semantics tag` 与可复用 Content 层：
  - `LoginScreen`
  - `IdentityScreen`
  - `ChatListScreen`
  - `SettingsScreen`
- 改造原则:
  - 尽量保持现有 ViewModel 与业务流程不变
  - 优先抽取 `*ScreenContent(state, callbacks...)` 形式的纯 UI 组合函数
  - 仅为关键输入框、按钮、tab、空态/错误态卡片、导航入口增加 tag
  - tag 集中在测试常量文件中管理，避免散落硬编码

### 3. UI / androidTest 用例策略
- 首批 `androidTest` 用例聚焦默认页面的“主路径 smoke + 关键状态分支”:
  - 登录页: 输入框展示、登录按钮启停、loading 状态
  - 身份页: 空态、快速创建/手动创建入口、列表渲染与选择回调
  - 会话列表: 历史/收藏 tab 切换、空态/错误态、设置/全局收藏入口
  - 设置页: 返回、主题切换、连接地址保存入口、媒体库 / mtPhoto / 抖音 / 视频抽帧导航入口
- 在当前无设备环境下，先保证这些测试源码可编译、结构可执行；设备执行结果延后到后续回合或 CI。

### 4. 双轨覆盖率保持策略
- 对新增的 UI tag、状态映射、Content helper、页面展示辅助逻辑补充 `src/test` 下的 JVM 单测。
- 若页面拆分出纯 Kotlin / Compose 无状态 helper，则优先建立对应单测，确保现有 Debug Unit Test fresh 覆盖率保持在 `40%+`。
- 文档中明确说明：
  - Debug Unit Test 覆盖率 = 当前 JaCoCo 主要统计口径
  - androidTest = 设备/CI 执行口径，本轮先建立基线与入口

## 安全与性能
- **安全:** 仅增加测试基础设施、UI tag、测试 helper 与少量 Content 层拆分，不触碰生产密钥、账号或真实外部服务。
- **性能:** 限制首轮仪器测试页面范围，避免在无设备场景下引入过重的端到端基建；优先使用 screen-level Compose UI Test，而非全导航全链路大而全方案。

## 测试与部署
- **本轮本地验证:**
  - `cd android-app && ./gradlew :app:compileDebugAndroidTestKotlin --no-daemon`
  - `cd android-app && ./gradlew :app:assembleDebugAndroidTest --no-daemon`
  - `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`
- **有设备/CI 后的追加验证:**
  - `cd android-app && ./gradlew :app:connectedDebugAndroidTest --no-daemon`
- **部署:** 无部署动作，仅提交代码、测试与知识库更新。
