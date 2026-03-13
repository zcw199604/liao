# Android 构建解阻技术设计

## 1. 方案概述

本次以“最小构建解阻”为原则，优先修复直接导致 Android 编译失败的问题，并在每次修改后立即复跑 Gradle 构建检查，直到 `:app:compileDebugKotlin` 与 `testDebugUnitTest` 通过。

## 2. 实施要点

### 2.1 主题资源
- 在 `app/build.gradle.kts` 中加入 `com.google.android.material:material:1.12.0`，为 `Theme.Material3.DayNight.NoActionBar` 提供 XML 侧资源宿主。

### 2.2 Room / KAPT 兼容
- 将 Room 依赖从 `2.6.1` 升级到 `2.7.2`：
  - `room-runtime`
  - `room-ktx`
  - `room-compiler`
- 目标是消除旧版 Room 编译器对 Kotlin metadata 2.1.0 的读取限制。

### 2.3 Retrofit / Compose API 适配
- 修正 `asConverterFactory` 的导入路径到 JakeWharton 提供的 Retrofit Kotlinx Serialization 扩展包。
- 将过时的 `HttpUrl.get(...)` 切换为 `toHttpUrl()`。
- 为 `ChatListFeature.kt`、`ChatRoomFeature.kt`、`SettingsFeature.kt` 加入 `ExperimentalMaterial3Api` opt-in。
- 移除导致 Kotlin 2.x 编译报错的 `weight` 显式导入，保留作用域内调用写法。

## 3. 验证策略
- `cd android-app && ./gradlew :app:compileDebugKotlin --no-daemon`
- `cd android-app && ./gradlew testDebugUnitTest --no-daemon`

## 4. 风险与说明
- 当前仍保留 `Kapt currently doesn't support language version 2.0+` 警告，但不再阻断编译与单测通过。
- 若后续继续升级 Kotlin / AGP，建议评估把 Room 与 Hilt 链路逐步迁移到 KSP。
