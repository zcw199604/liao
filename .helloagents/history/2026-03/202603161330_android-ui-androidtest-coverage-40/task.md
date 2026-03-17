# 任务清单: Android UI / androidTest 覆盖补充

目录: `.helloagents/history/2026-03/202603161330_android-ui-androidtest-coverage-40/`

---

## 1. androidTest 基础设施与默认页面可测性
- [√] 1.1 在 `android-app/app/build.gradle.kts` 与 `android-app/app/src/androidTest/` 下补齐 `androidTest` 基础依赖、目录结构、Compose test rule / helper，并验证 why.md#场景-androidtest-骨架可编译并具备设备执行入口
- [√] 1.2 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/auth/` 与 `.../identity/` 中为默认页面补充统一测试 tag、必要的 `ScreenContent` 纯 UI 入口或测试友好状态结构，并验证 why.md#场景-默认页面具备稳定的-ui-可测性，依赖任务1.1
- [√] 1.3 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/` 与 `.../settings/` 中为默认页面补充统一测试 tag、必要的 `ScreenContent` 纯 UI 入口或测试友好状态结构，并验证 why.md#场景-默认页面具备稳定的-ui-可测性，依赖任务1.2

## 2. 首批 UI / androidTest 用例补齐
- [√] 2.1 在 `android-app/app/src/androidTest/kotlin/io/github/a7413498/liao/android/feature/auth/` 与 `.../identity/` 中新增默认页面 Compose UI 仪器测试，覆盖主路径 smoke 与关键空态/交互分支，并验证 why.md#场景-androidtest-骨架可编译并具备设备执行入口，依赖任务1.3
- [√] 2.2 在 `android-app/app/src/androidTest/kotlin/io/github/a7413498/liao/android/feature/chatlist/` 与 `.../settings/` 中新增默认页面 Compose UI 仪器测试，覆盖 tab / 导航入口 / 空态错误态等分支，并验证 why.md#场景-androidtest-骨架可编译并具备设备执行入口，依赖任务2.1
- [√] 2.3 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/{auth,identity,chatlist,settings}/` 中补充新抽取 UI helper / state 映射 / tag 相关 JVM 单测，确保 why.md#场景-可量化覆盖率保持在-40-以上，依赖任务2.2

## 3. 验证、知识库与归档
- [√] 3.1 执行 `cd android-app && ./gradlew :app:compileDebugAndroidTestKotlin :app:assembleDebugAndroidTest testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录在无设备场景下的可验证结果，并验证 why.md#场景-androidtest-骨架可编译并具备设备执行入口 与 why.md#场景-可量化覆盖率保持在-40-以上，依赖任务2.3
- [√] 3.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，说明 UI / androidTest 基线、无设备执行约束与最新覆盖率，并验证 why.md#场景-可量化覆盖率保持在-40-以上，依赖任务3.1
