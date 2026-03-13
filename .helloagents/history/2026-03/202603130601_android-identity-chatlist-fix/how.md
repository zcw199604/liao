# 技术设计：Android 身份与会话列表修复

## 1. 设计概述

本次修复保持“单文件内最小闭环”策略，不额外新增 Android 页面或 Repository 文件，而是在现有 `IdentityFeature.kt` 与 `ChatListFeature.kt` 内完成 Repository、ViewModel 与 Compose UI 的补齐。

## 2. 技术方案

### 2.1 IdentityFeature
- Repository：
  - 复用 `IdentityApiService.updateIdentity` 与 `IdentityApiService.deleteIdentity`。
  - 编辑当前已选择身份时，同步回写 `AppPreferencesStore.currentSession` 中的 `name/sex`。
  - 删除当前已选择身份时，清理 `currentSession`，避免本地会话残留。
- ViewModel：
  - 新增消息消费与刷新辅助逻辑，避免“成功提示刚写入又被刷新覆盖”。
  - 统一处理创建 / 编辑提交、删除确认与列表重载。
- UI：
  - 增加编辑、删除、选择三个操作入口。
  - 保留删除确认弹窗，并在编辑状态下复用上方表单。

### 2.2 ChatListFeature
- ViewModel：
  - 区分 `infoMessage` 与 `errorMessage`，信息提示用于占位入口反馈，错误信息用于显式错误态渲染。
- UI：
  - 增加“全局收藏入口（占位）”按钮，点击后通过 Snackbar 提示后续将接入独立页面。
  - 补齐显式错误态卡片与显式空态卡片，均包含刷新与入口按钮。
  - 保留顶部“全局收藏”文字入口，避免隐藏主入口。

## 3. 风险与规避

- 风险：删除当前已选择身份后，本地会话仍残留。
  - 规避：Repository 在删除成功后检查当前会话 ID，命中则清理。
- 风险：编辑身份后，当前会话昵称与性别仍显示旧值。
  - 规避：Repository 在编辑成功后同步刷新 `AppPreferencesStore`。
- 风险：Android 项目当前未提供 Gradle Wrapper，难以在仓库内直接做标准编译验证。
  - 规避：本次先完成静态代码检查，并在知识库中记录验证限制，后续由具备 Android 构建环境的同学补跑编译。

## 4. 验证计划

- 静态检查目标文件中的状态流、删除确认、空态 / 错误态与占位入口是否齐全。
- 尝试执行 Android 编译命令；若环境缺少 Gradle Wrapper 或 `gradle` 命令，则记录为环境限制。
