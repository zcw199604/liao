# 任务清单: Android 会话列表全局收藏切换

目录: `helloagents/master/history/2026-07/202607071351_android-chatlist-global-favorite/`

---

## 0. 方案边界确认
- [√] 0.1 确认本次只覆盖会话列表全局收藏切换，不实现在线状态查询
  - 执行模式: AFK
  - 涉及文件: `why.md`, `how.md`, `task.md`
  - 完成标准: 范围内/范围外描述一致
  - 验证方式: 只读核对方案包三件套
- [√] 0.2 确认复用现有 `FavoriteApiService`，不新增依赖或 Room schema
  - 执行模式: AFK
  - 涉及文件: `how.md`
  - 完成标准: 技术边界清晰
  - 验证方式: 只读核对 how.md

## 1. RED 测试
- [√] 1.1 为 Repository 添加全局收藏加载、添加、移除和失败测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListRepositoryTest.kt`
  - 完成标准: 测试覆盖当前身份筛选、add/remove API 参数和失败不更新
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListRepositoryTest"`
  - 备注: 测试已补充；当前 Java 8 环境无法执行 Android Gradle 单测。
- [√] 1.2 为 ViewModel 添加全局收藏状态加载和切换测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListViewModelTest.kt`
  - 完成标准: 测试断言成功更新集合和提示，失败保留原状态
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListViewModelTest"`
  - 备注: 测试已补充；当前 Java 8 环境无法执行 Android Gradle 单测。

## 2. GREEN 实现
- [√] 2.1 实现 `ChatListRepository` 全局收藏加载和切换方法
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`
  - 完成标准: Repository 正确调用 `/favorite/listAll`、`/favorite/add`、`/favorite/remove`
  - 验证方式: Repository 测试和静态审查
- [√] 2.2 实现 `ChatListViewModel` 全局收藏状态和切换动作
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`
  - 完成标准: 初始加载当前身份收藏 ID 集合；切换成功后局部更新状态
  - 验证方式: ViewModel 测试
- [√] 2.3 在会话项 UI 增加全局收藏按钮
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt`, `android-app/app/src/androidTest/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListScreenTest.kt`, `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/app/testing/AndroidUiTestTags.kt`
  - 完成标准: 按钮文案随收藏状态切换，点击回调传入目标会话和当前状态
  - 验证方式: Compose UI 测试或静态编译审查

## 3. 文档同步
- [√] 3.1 更新 Android 客户端知识库和 CHANGELOG
  - 执行模式: AFK
  - 涉及文件: `helloagents/master/wiki/modules/android-client.md`, `helloagents/master/CHANGELOG.md`
  - 完成标准: 文档记录会话项全局收藏切换能力
  - 验证方式: 只读核对知识库

## 4. 验证与归档
- [√] 4.1 执行格式检查和可用测试
  - 执行模式: AFK
  - 涉及文件: 本次改动文件
  - 完成标准: `git diff --check` 通过；Android 单测不可执行时记录环境原因
  - 验证方式: `git diff --check` 和 Android 测试命令
  - 备注: `git diff --check` 通过；Android 单测失败于 Java 8 环境，AGP 8.7.3 至少需要 Java 11，项目需要 JDK 17。
- [√] 4.2 写入 QA 证据并迁移方案包
  - 执行模式: AFK
  - 涉及文件: `qa-review.json`, `helloagents/master/history/index.md`
  - 完成标准: QA 证据可复核，方案包迁移至 history
  - 验证方式: 只读核对迁移结果
