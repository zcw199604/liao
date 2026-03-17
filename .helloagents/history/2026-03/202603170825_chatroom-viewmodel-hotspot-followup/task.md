# 任务清单: Android ChatRoom ViewModel 热点分支继续补充

目录: `.helloagents/history/2026-03/202603170825_chatroom-viewmodel-hotspot-followup/`

---

## 1. 高收益 JVM 单测补充
- [√] 1.1 继续补充 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatroom/ChatRoomViewModelTest.kt`，覆盖 `ChatRoomViewModel` 剩余高收益分支
- [√] 1.2 视覆盖率结果补充分页守卫、入站事件与失败重试的剩余边界分支，扩大收益

## 2. 验证与文档
- [√] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率（44 files / 273 tests；branch 40.81%；`ChatRoomViewModel` branch 76.39%）
- [√] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
