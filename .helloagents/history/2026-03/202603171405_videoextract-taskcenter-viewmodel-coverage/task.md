# 任务清单: Android VideoExtractTaskCenter ViewModel branch 覆盖率继续补充

目录: `.helloagents/history/2026-03/202603171405_videoextract-taskcenter-viewmodel-coverage/`

---

## 1. 高收益 JVM 单测补充
- [√] 1.1 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/videoextract/VideoExtractTaskCenterFeature.kt` 补充 `VideoExtractTaskCenterViewModel` JVM 测试，覆盖任务列表刷新/分页、详情切换、帧分页与动作分支
- [√] 1.2 视覆盖率结果补充当前热点 ViewModel 的剩余边界分支，扩大收益

## 2. 验证与文档
- [√] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率（43 files / 261 tests；branch 38.64%）
- [√] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
