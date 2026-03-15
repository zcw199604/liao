# 技术方案

## 1. 覆盖率口径收口
- 在 Android JaCoCo 任务中新增 `**/*Kt$*` 排除规则，仅排除 Kotlin 顶层函数生成的合成内部类（常见于 Compose/lambda 包装），保留手写顶层 `*FeatureKt` / Repository / ViewModel / core 类。
- 不排除业务主类与仓储类，避免通过缩小统计范围“虚高”覆盖率。

## 2. 高收益 helper 提测
- `feature/videoextract/VideoExtractCreateFeature.kt`
  - `validationError`
  - `toCreatePayloadOrError`
  - `estimateFrames`
  - `formatDuration`
  - `toReadableSize`
- `feature/videoextract/VideoExtractTaskCenterFeature.kt`
  - `toTaskItem`
  - `toFrameItem`
  - `displayTitle / displaySubtitle / statusText / modeText / progressText / limitText`
  - `buildUploadPreviewUrl`
  - `formatDuration`
- `feature/douyin/DouyinFeature.kt`
  - `normalizeDouyinMediaType`
  - `resolveFavoriteMediaType`
  - `resolveDouyinMediaTypeLabel`
  - `resolveDouyinImportActionText`
  - `defaultDouyinFileName`
  - `resolveTagNames / longList`
- `feature/mtphoto/MtPhotoFeature.kt`
  - `currentVisibleItemCount`
  - `errorMessage`
  - `toFolderName`
  - `firstCoverMd5`
- `feature/settings/SettingsFeature.kt`
  - `toDisplayText`
  - `normalizeImagePortMode`
  - `toUiMode / toUiRealMinBytes / toUiMtPhotoThreshold`
  - `toDisplayLabel`
  - `defaultSystemConfig`

## 3. 实施策略
- 仅把上述 helper 从 `private` 调整为 `internal`，不改变业务行为。
- 新增对应测试文件，集中覆盖边界输入、兜底分支与格式化逻辑。
- 每次增补后跑 `./gradlew :app:jacocoDebugUnitTestReport --no-daemon` 观察 branch 变化，直到达到目标区间。

# 风险与规避
- 风险：helper 可见性提升后 API 面增大。
  - 规避：仅改为 `internal`，并明确只服务测试；不外泄到 module 外。
- 风险：覆盖率口径调整被误解为“排除真实业务代码”。
  - 规避：仅排除 `*Kt$*` 生成类，并在文档中明确这是 Kotlin/Compose 合成类，不是手写业务类。
