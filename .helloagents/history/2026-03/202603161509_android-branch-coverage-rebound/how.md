# 实施思路

1. 优先补 `MtPhotoSameMediaRepository` 纯 JVM 仓储测试，覆盖空路径、异常响应、error 字段、混合 items 与默认值回退。
2. 补 `NetworkModels` / `AppSupport` 纯函数边界测试，扩大 DTO/时间线/JSON 容错分支覆盖。
3. 补 `Login/Identity/ChatList/Settings/AppCoordinator` ViewModel JVM 单测与 `MainDispatcherRule`，扩大初始化、刷新、切页、保存、forceout 与消息落库分支。
4. 视缺口补 `SettingsRepository` 的错误/成功边界测试，覆盖 fallback 分支。
5. 复跑 `testDebugUnitTest jacocoDebugUnitTestReport`，确认 branch >= 30%。
6. 同步文档并归档方案包。
