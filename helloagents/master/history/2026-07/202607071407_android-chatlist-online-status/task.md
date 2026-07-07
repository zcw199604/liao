# 任务清单: Android 会话列表查询在线状态

目录: `helloagents/master/plan/202607071407_android-chatlist-online-status/`

---

## 0. 方案边界确认
- [√] 0.1 确认本次任务仅覆盖 why.md 的范围内切片，范围外内容不进入实现
  - 执行模式: AFK
  - 涉及文件: `why.md`, `how.md`, `task.md`
  - 完成标准: 范围内/范围外边界一致且无互相矛盾描述
  - 验证方式: 只读核对方案包三件套
- [√] 0.2 确认 how.md 的设计边界完整，尤其是模块职责、接口契约、数据边界和依赖边界
  - 执行模式: AFK
  - 涉及文件: `how.md`
  - 完成标准: 每个受影响模块均有明确边界说明
  - 验证方式: 只读核对 how.md 设计边界
- [√] 0.3 大型项目确认最小改动策略: 不做无关重构、目录搬迁、依赖升级或公共API重命名
  - 执行模式: AFK
  - 涉及文件: `how.md`, `task.md`
  - 完成标准: 非必要扩大项已列入范围外或后续计划
  - 验证方式: 只读核对任务清单和设计边界

## 1. WebSocket 在线状态协议
- [√] 1.1 RED: 在 `LiaoWebSocketClientTest` 中补充 `code=30` 解析 `IF_Online` 和 `TimeAll` 的失败测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClientTest.kt`
  - 完成标准: 测试期望 `OnlineStatus.isOnline` 和 `lastTime`，在生产实现前失败
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClientTest"`
  - 备注: 已补测试；RED 命令在当前环境被 Java 8 阻断，未进入测试执行阶段，详见 `qa-review.json`。
- [√] 1.2 GREEN: 扩展 `LiaoWsEvent.OnlineStatus` 并解析在线状态字段
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/websocket/LiaoWebSocketClient.kt`
  - 完成标准: `IF_Online` 为 `"1"` 得到 `true`，`"0"` 得到 `false`，缺失/异常得到 `null`，`TimeAll` 作为最后时间
  - 验证方式: 同 1.1

## 2. 会话列表查询动作
- [√] 2.1 RED: 在 `ChatListRepositoryTest` 中补充请求在线状态成功/失败测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListRepositoryTest.kt`
  - 完成标准: 测试期望 Repository 读取当前身份并调用 `sendShowUserLoginInfo`
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListRepositoryTest"`
  - 备注: 已补测试；Gradle 命令受 Java 8 环境阻断，详见 `qa-review.json`。
- [√] 2.2 RED: 在 `ChatListViewModelTest` 与 `ChatListScreenTest` 中补充按钮、查询中状态和弹窗测试
  - 执行模式: AFK
  - 涉及文件: `ChatListViewModelTest.kt`, `ChatListScreenTest.kt`
  - 完成标准: 测试期望点击“查在线”触发请求，收到 `OnlineStatus` 后展示在线状态
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListViewModelTest"`；`./gradlew :app:compileDebugAndroidTestKotlin --no-daemon`
  - 备注: 已补测试；Gradle 命令受 Java 8 环境阻断，详见 `qa-review.json`。
- [√] 2.3 GREEN: 实现 Repository、ViewModel、Compose UI 查询在线状态闭环
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`, `AndroidUiTestTags.kt`
  - 完成标准: 会话项展示“查在线”按钮；查询成功等待回包；回包展示在线状态弹窗；失败展示错误提示
  - 验证方式: 同 2.1、2.2

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）
  - 执行模式: AFK
  - 涉及文件: 本次改动文件
  - 完成标准: 未发现新增敏感信息、危险命令或未确认 EHRB 风险
  - 验证方式: 人工安全核对与 `rg -n "password|secret|token|rm -rf|DROP TABLE" android-app/app/src/main/kotlin`

## 4. 文档更新
- [√] 4.1 更新 Android Client 知识库和 CHANGELOG
  - 执行模式: AFK
  - 涉及文件: `helloagents/master/wiki/modules/android-client.md`, `helloagents/master/CHANGELOG.md`
  - 完成标准: 知识库与代码事实一致，CHANGELOG 记录本切片
  - 验证方式: 只读核对文档

## 5. 测试
- [√] 5A.1 RED 证据记录: 记录本轮新增测试在生产实现前的失败情况，或记录环境阻断
  - 执行模式: AFK
  - 涉及文件: `task.md`, `qa-review.json`
  - 完成标准: RED 失败原因或 Java/Gradle 环境阻断原因可复核
  - 验证方式: 查看测试命令输出与 QA 证据
- [√] 5A.2 GREEN/VERIFY: 运行可用验证命令并记录结果
  - 执行模式: AFK
  - 涉及文件: 本次改动文件
  - 完成标准: 可运行验证通过；不可运行项记录具体环境原因
  - 验证方式: `git diff --check` 与 Android 相关测试命令
  - 备注: `git diff --check` 通过；Android Gradle 命令因当前 JVM 为 Java 8、AGP 8.7.3 要求 Java 11+ 而未执行到测试阶段。
