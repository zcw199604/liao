# 任务清单: Android branch 覆盖率提升到 30%

目录: `.helloagents/plan/202603151234_android-branch-coverage-30/`

---

## 1. chatroom helper / repository 单测
- [√] 1.1 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatroom/` 中补充 `ChatRoomFeature` 纯 helper 单测，并对必要 helper 做最小可测性调整，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-聊天核心分支具备-jvm-回归
- [√] 1.2 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatroom/` 中补充 `ChatRoomRepository` 单测，覆盖连接、历史、收藏、媒体 URL、重传与发送分支，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-聊天核心分支具备-jvm-回归，依赖任务1.1

## 2. douyin / mtphoto 剩余热点单测
- [√] 2.1 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/douyin/` 中扩展 `DouyinRepository` 剩余高分支映射/操作用例，并对必要 helper 做最小可测性调整，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-抖音mtphoto-仓储映射分支具备-jvm-回归
- [√] 2.2 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/mtphoto/` 中扩展 `MtPhotoRepository` 剩余高分支映射/仓储用例，并对必要 helper 做最小可测性调整，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-抖音mtphoto-仓储映射分支具备-jvm-回归，依赖任务2.1

## 3. 覆盖率验证与知识库同步
- [√] 3.1 执行 `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，确认 fresh JaCoCo 结果，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-覆盖率结果可复现且达到-30，依赖任务1.2
- [√] 3.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，记录新的测试规模与覆盖率结果，验证 why.md#需求-android-branch-覆盖率继续提升到-30-场景-覆盖率结果可复现且达到-30，依赖任务3.1


## 执行结果
- 2026-03-15：执行 `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` 成功。
- 单测规模：28 个测试文件 / 184 条用例。
- 覆盖率结果：`line 44.06%`、`branch 31.50%`、`method 43.61%`、`class 38.25%`。
