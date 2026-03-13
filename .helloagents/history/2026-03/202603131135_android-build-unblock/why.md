# Android 构建解阻提案

## 1. 背景

在补齐 `android-app/` 的 Gradle Wrapper 之后，Android 本地构建已能启动，但继续执行 `:app:compileDebugKotlin` 时先后暴露出主题资源缺失、Room/KAPT 与 Kotlin 2.x metadata 兼容，以及若干 Kotlin/Compose API 适配问题，导致客户端无法完成基础编译。

## 2. 目标

- 让 `android-app` 在现有 JDK 17 + Android SDK 环境下至少通过 `:app:compileDebugKotlin`。
- 继续执行 `testDebugUnitTest`，验证当前最小单元测试链路可运行。
- 所有修复保持最小增量，不改变既有业务设计，只解决构建与编译阻塞。

## 3. 范围

### 3.1 范围内
- 修复 XML 宿主主题依赖缺失。
- 升级 Room 版本，解决 Kotlin 2.x metadata 与 kapt/Room 编译器兼容问题。
- 修正 Retrofit Kotlinx Serialization 转换器导入与过时 HttpUrl API。
- 修正 Compose 文件中的实验 API 标注与 `weight` 导入兼容问题。

### 3.2 范围外
- 切换 KAPT 到 KSP。
- 大规模重构 Android 架构或替换现有依赖体系。
- UI 行为调整与业务能力扩展。
