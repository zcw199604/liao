# 任务清单: chat-uploadmenu-douyin-favorites

> **@status:** completed | 2026-02-07 12:46

目录: `helloagents/plan/202602071149_chat-uploadmenu-douyin-favorites/`

---

## 任务状态符号说明

| 符号 | 状态 | 说明 |
|------|------|------|
| `[ ]` | pending | 待执行 |
| `[√]` | completed | 已完成 |
| `[X]` | failed | 执行失败 |
| `[-]` | skipped | 已跳过 |
| `[?]` | uncertain | 待确认 |

---

## 执行状态
```yaml
总任务: 16
已完成: 16
完成率: 100%
```

---

## 任务列表

### 1. 上传菜单入口改造（chat-ui）

- [√] 1.1 在 `frontend/src/components/chat/UploadMenu.vue` 新增“抖音收藏作者”按钮
  - 说明: 新按钮位于现有操作按钮区，视觉风格与其余按钮一致
  - 验证: 组件渲染时可见按钮文案与图标

- [√] 1.2 在 `UploadMenu.vue` 新增 `openDouyinFavoriteAuthors` 事件声明并绑定点击触发
  - 验证: 单测中点击按钮后 `wrapper.emitted('openDouyinFavoriteAuthors')` 为真

### 2. 聊天页事件链路接入（ChatRoomView）

- [√] 2.1 在 `frontend/src/views/ChatRoomView.vue` 为 `UploadMenu` 增加 `@open-douyin-favorite-authors` 监听
  - 依赖: 1.2
  - 验证: 触发事件可进入处理函数

- [√] 2.2 新增 `handleOpenDouyinFavoriteAuthors`，负责关闭上传菜单并打开抖音弹窗
  - 说明: 保持与现有面板开关逻辑一致，避免叠层冲突
  - 验证: 调用后 `showUploadMenu=false` 且 `douyinStore.showModal=true`

### 3. 抖音弹窗入口上下文能力（store + modal）

- [√] 3.1 在 `frontend/src/stores/douyin.ts` 扩展 `open` 参数，支持可选打开上下文（entry/mode/tab）
  - 说明: 保持对旧签名（仅 prefill）兼容
  - 验证: 旧调用不报错；新调用可读到上下文

- [√] 3.2 在 `douyin.ts` 的 `close` 中清理新增上下文状态
  - 验证: 关闭后再次默认打开不残留上次入口模式

- [√] 3.3 在 `frontend/src/components/media/DouyinDownloadModal.vue` 初始化时消费上下文并设置默认模式
  - 目标: 聊天页入口默认 `activeMode=favorites`、`favoritesTab=users`
  - 验证: 模式按钮高亮与内容区符合预期

- [√] 3.4 校验非聊天入口默认行为不变（仍可从“作品解析”开始）
  - 验证: 既有入口测试通过

### 4. 作者作品浏览与导入体验

- [√] 4.1 验证“收藏作者列表 -> 点击作者 -> 作品列表”链路在聊天入口下可用
  - 验证: 触发 `openFavoriteUserDetail`，并成功拉取 `listDouyinFavoriteUserAwemes`

- [√] 4.2 验证作者作品列表支持继续分页加载（下拉触发）
  - 验证: `hasMore=true` 时触发 append 加载，列表数量递增

- [√] 4.3 验证“作者作品预览 -> 导入本地服务器”流程可用
  - 验证: `importDouyinMedia` 被调用，导入状态更新为 imported/exists

- [√] 4.4 调整必要提示文案，明确“导入本地后可在上传菜单继续发送”
  - 验证: 导入成功提示与操作路径描述一致，避免歧义

### 5. 测试与回归

- [√] 5.1 更新 `frontend/src/__tests__/chat-components.test.ts`
  - 内容: UploadMenu 新按钮事件断言

- [√] 5.2 更新 `frontend/src/__tests__/upload-menu-more.test.ts`
  - 内容: 动作按钮集合断言（新增 openDouyinFavoriteAuthors）

- [√] 5.3 更新 ChatRoomView 相关测试（`chatroom-more.test.ts` / 现有分支测试）
  - 内容: 新事件触发 -> store.open 调用链断言

- [√] 5.4 补充或更新 `frontend/src/__tests__/douyin-download-modal-modes.test.ts`
  - 内容: 新入口上下文触发 favorites/users 默认分支 + 关闭后状态清理

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 1.x-5.x | [√] | 已完成代码实现、单测与构建验证（`npm run test -- src/__tests__/chat-components.test.ts src/__tests__/upload-menu-more.test.ts src/__tests__/chatroom-more.test.ts src/__tests__/douyin-download-modal-modes.test.ts src/__tests__/stores-more.test.ts` + `npm run build`） |
