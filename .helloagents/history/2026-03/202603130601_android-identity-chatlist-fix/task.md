# Android 身份与会话列表修复任务清单

> 方案类型：修复
> 选定方案：最小闭环补齐
> 状态说明：`[ ]` 待执行 / `[√]` 已完成 / `[X]` 执行失败 / `[-]` 已跳过 / `[?]` 待确认

---

## 1. 身份模块修复
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/identity/IdentityFeature.kt` 中补齐编辑 / 删除 Repository 调用，并复用现有 API。
- [√] 在同文件 ViewModel 中补齐编辑态、删除确认、消息消费与刷新闭环。
- [√] 在同文件 Compose UI 中补齐编辑、删除、选择入口及最小空态提示。

## 2. 会话列表修复
- [√] 在 `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/chatlist/ChatListFeature.kt` 中区分信息提示与错误状态。
- [√] 在同文件 Compose UI 中补齐显式空态、显式错误态与“全局收藏入口（占位）”按钮。

## 3. 验证与知识库
- [√] 更新 `.helloagents/modules/android-client.md`、`.helloagents/CHANGELOG.md` 与 `.helloagents/history/index.md`，同步本次 Android 修复信息。
- [√] 尝试执行 Android 编译验证，并记录“缺少 Gradle Wrapper / 本机无 gradle 命令”的环境限制。
