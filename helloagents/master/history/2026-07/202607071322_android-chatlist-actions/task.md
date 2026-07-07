# 任务清单: Android 会话项动作对齐

目录: `helloagents/master/history/2026-07/202607071322_android-chatlist-actions/`

---

## 0. 方案边界确认
- [√] 0.1 确认本次任务仅覆盖会话项清未读和删除，范围外能力不进入实现
  - 执行模式: AFK
  - 涉及文件: `why.md`, `how.md`, `task.md`
  - 完成标准: 范围内/范围外边界一致
  - 验证方式: 只读核对方案包三件套
- [√] 0.2 确认不新增依赖、不变更 Room schema、不改动 Vue 端
  - 执行模式: AFK
  - 涉及文件: `how.md`
  - 完成标准: 设计边界明确
  - 验证方式: 只读核对 how.md

## 1. RED 测试
- [√] 1.1 为 Repository 删除会话成功/失败路径添加失败测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListRepositoryTest.kt`
  - 完成标准: 测试断言成功后清理会话和消息，失败时不清理
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListRepositoryTest"`
  - 备注: 测试已补充；RED/单测执行被当前 Java 8 环境阻断，未取得运行结果。
- [√] 1.2 为 ViewModel 清未读/删除确认状态添加失败测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListViewModelTest.kt`
  - 完成标准: 测试断言清未读代理调用、删除成功/失败提示
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListViewModelTest"`
  - 备注: 测试已补充；当前环境缺少 JDK 17，未能执行 Android Gradle 单测。

## 2. GREEN 实现
- [√] 2.1 增加 Android `/deleteUpstreamUser` 网络接口和 DAO 单会话删除方法
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/network/NetworkStack.kt`, `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/core/database/LocalDatabase.kt`
  - 完成标准: Repository 可调用后端删除并清理本地缓存
  - 验证方式: 编译和 Repository 测试
- [√] 2.2 实现 ChatListRepository / ChatListViewModel 删除和清未读流程
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`
  - 完成标准: 成功删除后关闭确认弹窗并显示成功提示，失败时保留会话并显示错误
  - 验证方式: ViewModel 测试
- [√] 2.3 在 Compose 会话列表中增加清未读和删除入口
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`, `android-app/app/src/androidTest/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListScreenTest.kt`
  - 完成标准: 每个会话项能触发清未读和删除确认
  - 验证方式: Compose UI 测试编译或可用环境执行

## 3. 文档同步
- [√] 3.1 更新 Android 客户端知识库和 CHANGELOG
  - 执行模式: AFK
  - 涉及文件: `helloagents/master/wiki/modules/android-client.md`, `helloagents/master/CHANGELOG.md`
  - 完成标准: 文档记录清未读和删除会话能力
  - 验证方式: 只读核对知识库

## 4. 验证
- [√] 4.1 执行格式和可用测试验证
  - 执行模式: AFK
  - 涉及文件: 本次改动文件
  - 完成标准: `git diff --check` 通过；如 Gradle 环境不可用则记录具体原因
  - 验证方式: `git diff --check` 和 Android 相关测试命令
  - 备注: `git diff --check` 通过；Android 单测命令失败于 Java 8 环境，AGP 8.7.3 至少需要 Java 11，项目配置需要 JDK 17。
- [√] 4.2 记录 QA 证据并迁移方案包
  - 执行模式: AFK
  - 涉及文件: `qa-review.json`, `history/index.md`
  - 完成标准: QA 证据可复核，方案包进入 history
  - 验证方式: 只读核对迁移后的方案包
