# 任务清单: Android 会话列表批量删除

目录: `helloagents/master/plan/202607071516_android-chatlist-batch-delete/`

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

## 1. 批量删除协议与缓存
- [√] 1.1 RED: 在 `ChatListRepositoryTest` 中补充批量删除全成功、部分失败和缺失身份测试
  - 执行模式: AFK
  - 涉及文件: `android-app/app/src/test/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListRepositoryTest.kt`
  - 完成标准: 测试期望调用 `/api/batchDeleteUpstreamUsers`，成功项清理本地会话和消息，失败项保留
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListRepositoryTest"`
  - 备注: 已补测试；Gradle 在当前 Java 8 环境被 AGP 8.7.3 的 Java 11+ 要求阻断，未进入测试执行阶段。
- [√] 1.2 GREEN: 实现批量删除网络 DTO/API、DAO 批量删除和 Repository 缓存一致性
  - 执行模式: AFK
  - 涉及文件: `NetworkModels.kt`, `NetworkStack.kt`, `LocalDatabase.kt`, `ChatListFeature.kt`
  - 完成标准: Repository 返回成功/失败计数和失败 ID；仅成功项清理本地缓存
  - 验证方式: 同 1.1

## 2. 选择模式 UI 与状态
- [√] 2.1 RED: 在 `ChatListViewModelTest` 和 `ChatListScreenTest` 中补充选择模式、全选、确认弹窗和部分失败保留选中测试
  - 执行模式: AFK
  - 涉及文件: `ChatListViewModelTest.kt`, `ChatListScreenTest.kt`
  - 完成标准: 测试期望 Android 会话列表可进入选择模式并执行批量删除
  - 验证方式: `cd android-app && ./gradlew testDebugUnitTest --no-daemon --tests "io.github.a7413498.liao.android.feature.chatlist.ChatListViewModelTest"`；`./gradlew :app:compileDebugAndroidTestKotlin --no-daemon`
  - 备注: 已补测试；Gradle 在当前 Java 8 环境被阻断，详见 `qa-review.json`。
- [√] 2.2 GREEN: 实现 ChatList ViewModel 与 Compose 选择模式和批量删除确认
  - 执行模式: AFK
  - 涉及文件: `ChatListFeature.kt`, `AndroidUiTestTags.kt`
  - 完成标准: UI 可进入/退出选择模式，选择会话，全选当前列表，确认批量删除；部分失败保留失败项选中
  - 验证方式: 同 2.1

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）
  - 执行模式: AFK
  - 涉及文件: 本次改动文件
  - 完成标准: 未发现新增敏感信息、危险命令或未确认 EHRB 风险；批量删除有确认弹窗
  - 验证方式: 人工安全核对与 `git diff -U0 -- android-app/app/src/main/kotlin | rg -n "password|secret|token|rm -rf|DROP TABLE"`

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
