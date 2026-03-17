# 任务清单: Android WebSocket Client 热点分支补充

目录: `.helloagents/history/2026-03/202603171930_websocket-client-coverage/`

---

## 1. 高收益 JVM 单测补充
- [√] 1.1 新增 `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClientTest.kt`，覆盖连接、发送、入站事件、forceout 与关闭/重连关键分支
- [√] 1.2 视覆盖率结果补充旧连接失效守卫、关闭失败 fallback 与重复连接守卫分支，扩大收益

## 2. 验证与文档
- [√] 2.1 执行 `cd android-app && ./gradlew testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`，记录最新 Android Debug Unit Test 覆盖率（47 files / 292 tests；branch 44.16%；`LiaoWebSocketClient` branch 61.99%）
- [√] 2.2 更新 `android-app/README.md`、`.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与历史索引，并迁移方案包到 `history/`
