# 轻量迭代任务 - Android NetworkStack URL 测试补充

- [√] 1. 盘点 Android 网络栈中仍可通过纯单元测试覆盖的 URL 归一化与重写逻辑
- [√] 2. 补充 `NetworkStackUrlTest`，覆盖 API Base URL、WebSocket URL 与动态 Base URL 重写规则
- [√] 3. 收口 `buildWebSocketUrl` 实现，修复 `HttpUrl.Builder.scheme("ws/wss")` 的兼容问题
- [√] 4. 更新 Android 测试覆盖说明与知识库统计
- [√] 5. 执行 Android 单元测试并记录结果
  - 验证: `JAVA_HOME=/mnt/android-scrcpy/.local-jdks/jdk-17 ANDROID_SDK_ROOT=/mnt/android-scrcpy/.android-sdk ./gradlew testDebugUnitTest --no-daemon` 通过（5 files / 30 tests）
