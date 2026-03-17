# 任务清单: Android ViewModel branch 覆盖率继续补充

目录: `.helloagents/history/2026-03/202603161541_android-viewmodel-branch-continue/`

---

## 1. 高收益 JVM 单测补充
- [√] 1.1 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/app/AppCoordinator.kt` 补充 `AppCoordinatorViewModel` JVM 测试，覆盖启动路由、会话清理、消息持久化与 Forceout 分支
- [√] 1.2 为 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/mtphoto/MtPhotoFeature.kt` 补充 `MtPhotoViewModel` JVM 测试，覆盖相册/目录模式切换、延迟时间线、目录收藏、刷新、回退与预览导入分支
- [√] 1.3 视覆盖率结果补充现有 ViewModel 边界分支，扩大收益

## 2. 验证与文档
- [√] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率（39 files / 245 tests；branch 35.21%）
- [√] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
