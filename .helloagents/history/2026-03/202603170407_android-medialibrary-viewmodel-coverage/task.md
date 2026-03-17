# 任务清单: Android MediaLibrary ViewModel branch 覆盖率继续补充

目录: `.helloagents/history/2026-03/202603170407_android-medialibrary-viewmodel-coverage/`

---

> 说明：该方案为重复草稿，未执行，已由 `202603171015_android-medialibrary-viewmodel-coverage` 替代。

## 1. 高收益 JVM 单测补充
- [-] 1.1 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/media/MediaLibraryFeature.kt` 补充 `MediaLibraryViewModel` JVM 测试，覆盖分页加载、选择模式、删除、同媒体查询与消息消费分支
- [-] 1.2 视覆盖率结果补充当前热点 ViewModel 的剩余边界分支，扩大收益

## 2. 验证与文档
- [-] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率
- [-] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
