# 任务清单: Android 构建解阻

> 方案类型：修复
> 选定方案：最小增量解阻
> 状态说明：`[ ]` 待执行 / `[√]` 已完成 / `[X]` 执行失败 / `[-]` 已跳过 / `[?]` 待确认

---

## 1. 主题资源修复
- [√] 在 `android-app/app/build.gradle.kts` 中补充 XML 宿主主题所需 Material 依赖，消除 `Theme.Material3.DayNight.NoActionBar` 缺失错误。

## 2. Room / KAPT 兼容修复
- [√] 将 `room-runtime`、`room-ktx`、`room-compiler` 升级到 `2.7.2`，解决 Kotlin metadata 兼容问题。

## 3. 网络与 Compose API 适配
- [√] 修正 `NetworkStack.kt` 中 Retrofit Kotlinx Serialization 转换器导入与过时 HttpUrl API。
- [√] 为 `ChatListFeature.kt`、`ChatRoomFeature.kt`、`SettingsFeature.kt` 加入 Material3 实验 API 标注。
- [√] 移除相关文件中的 `weight` 显式导入，消除 Kotlin 2.x 编译报错。

## 4. 验证
- [√] 执行 `cd android-app && ./gradlew :app:compileDebugKotlin --no-daemon`，结果通过。
- [√] 执行 `cd android-app && ./gradlew testDebugUnitTest --no-daemon`，结果通过。

---

## 执行备注
- 当前 Android 编译链路已从“无法启动构建”推进到“可编译 + 可跑单测”。
- 仍存在 `kapt` 对 Kotlin 2.0+ 的降级警告，但当前不影响编译与测试成功。
