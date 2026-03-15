# 任务清单: Android branch 覆盖率提升到 20%

目录: `.helloagents/plan/202603151201_android-branch-coverage-20/`

---

## 1. videoextract helper / repository 单测
- [√] 1.1 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/videoextract/` 中补充 `VideoExtractCreateFeature` helper 单测，验证 why.md#需求-android-branch-覆盖率继续提升-场景-helper-分支回归稳定
- [√] 1.2 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/videoextract/` 中补充 `VideoExtractTaskCenterFeature` helper 单测，验证 why.md#需求-android-branch-覆盖率继续提升-场景-helper-分支回归稳定，依赖任务1.1
- [√] 1.3 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/videoextract/` 中补充 `VideoExtractTaskCenterRepository` 单测，并对必要 helper 做最小可测性调整，验证 why.md#需求-android-branch-覆盖率继续提升-场景-repository-远端缓存fallback-分支回归稳定，依赖任务1.2

## 2. douyin / mtphoto helper / repository 单测
- [√] 2.1 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/douyin/` 中补充 `DouyinFeature` helper 单测，验证 why.md#需求-android-branch-覆盖率继续提升-场景-helper-分支回归稳定
- [√] 2.2 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/mtphoto/` 中补充 `MtPhotoFeature` helper 单测，验证 why.md#需求-android-branch-覆盖率继续提升-场景-helper-分支回归稳定，依赖任务2.1
- [√] 2.3 在 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/mtphoto/` 中补充 `MtPhotoFolderFavoritesRepository` 单测，并对必要 helper 做最小可测性调整，验证 why.md#需求-android-branch-覆盖率继续提升-场景-repository-远端缓存fallback-分支回归稳定，依赖任务2.2

## 3. 覆盖率验证与知识库同步
- [√] 3.1 执行 `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，确认 fresh JaCoCo 结果，验证 why.md#需求-android-branch-覆盖率继续提升-场景-覆盖率指标可复现，依赖任务1.3
- [√] 3.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，记录新的测试规模与覆盖率结果，验证 why.md#需求-android-branch-覆盖率继续提升-场景-覆盖率指标可复现，依赖任务3.1


## 执行结果
- 2026-03-15：执行 `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon` 成功。
- 单测规模：22 个测试文件 / 134 条用例。
- 覆盖率结果：`line 32.46%`、`branch 21.63%`、`method 35.00%`、`class 35.09%`。
