# 任务清单: Android branch 覆盖率回补

目录: `.helloagents/history/2026-03/202603161509_android-branch-coverage-rebound/`

---

## 1. 高收益 JVM 单测补充
- [√] 1.1 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/media/MtPhotoSameMediaFeature.kt` 补充纯 JVM 仓储测试，覆盖空路径、错误响应、非法 item 过滤与默认值回退
- [√] 1.2 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/network/NetworkModels.kt` 与 `core/common/AppSupport.kt` 补充边界分支单测，覆盖 DTO → 时间线映射、JSON 解析容错与消息类型判定
- [√] 1.3 补 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/{app,feature/{auth,identity,chatlist,settings}}/` 的 ViewModel JVM 分支测试，并按需扩展 `SettingsRepositoryTest`，确保 branch 可继续提升

## 2. 验证与文档
- [√] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率（39 files / 234 tests；branch 33.91%）
- [√] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
