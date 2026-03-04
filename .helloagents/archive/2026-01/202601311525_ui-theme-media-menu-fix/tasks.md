# 任务清单: ui-theme-media-menu-fix

> **@status:** completed | 2026-01-31 15:44

目录: `helloagents/archive/2026-01/202601311525_ui-theme-media-menu-fix/`

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
总任务: 9
已完成: 9
完成率: 100%
```

---

## 任务列表

### 1. 复现与定位（frontend）

- [√] 1.1 复现浅色主题下“图片管理”入口卡片文字对比度问题
  - 文件: `frontend/src/components/settings/SettingsDrawer.vue`
  - 验证: 浅色主题打开“图片管理”，记录看不清的具体文案/截图与对应 DOM 类名
  > 备注: 已在 Android Via + Chrome 真机确认修复有效

- [√] 1.2 复现移动端“菜单标题竖排”（逐字换行）问题并定位具体元素
  - 文件: `frontend/src/components/media/AllUploadImageModal.vue`
  - 验证: Android Via（或最小宽度 360px）打开弹窗，确认是标题/计数/筛选按钮中的哪一块触发逐字换行，记录截图
  > 备注: 已在 Android Via + Chrome 真机确认修复有效

### 2. 修复：浅色主题图片管理入口可读性

- [√] 2.1 为媒体入口卡片增加深色基底，稳定透明渐变在浅色主题下的对比度
  - 文件: `frontend/src/components/settings/SettingsDrawer.vue`
  - 实施: 在卡片容器增加 `bg-slate-900`/`bg-gray-900`；必要时用 `dark:` 保留深色主题的透明渐变质感
  - 验证: 浅色/深色主题下标题与副标题均清晰；hover 状态无明显反差问题

- [√] 2.2 调整副标题/提示文案颜色，避免在深色底上使用过浅灰导致发虚
  - 文件: `frontend/src/components/settings/SettingsDrawer.vue`
  - 实施: `text-gray-400/500` → `text-white/70`（或 `text-white/60`），与深色底保持一致
  - 验证: 浅色主题下可读性提升；深色主题下不显“脏/灰”

### 3. 修复：移动端菜单标题竖排（逐字换行）

- [√] 3.1 给标题/计数/筛选按钮增加禁止换行与禁止压缩的约束
  - 文件: `frontend/src/components/media/AllUploadImageModal.vue`
  - 实施: 为 `<h3>`、计数 `<span>`、以及“全部/本地/抖音”按钮添加 `whitespace-nowrap shrink-0`（必要时补 `min-w-0`）
  - 验证: Android Via 下不再出现每个字一行

- [√] 3.2 将弹窗头部改为响应式分行布局，避免小屏挤压
  - 文件: `frontend/src/components/media/AllUploadImageModal.vue`
  - 实施: 头部从单行 `flex` 调整为 `flex-col sm:flex-row`；筛选区使用 `overflow-x-auto no-scrollbar`（或 `flex-wrap`）保证可操作性
  - 验证: 小屏可用（不挡点击）；桌面端布局无回归

### 4. 回归验证与构建检查

- [√] 4.1 前端构建与基础测试
  - 命令: `cd frontend && npm run build`（可选再跑 `npm test`）
  - 验证: 构建通过，无 TS 错误；样式未被 Tailwind purge

- [√] 4.2 真机回归（至少 2 个浏览器）
  - 环境: Android Via + Android Chrome
  - 验证: 两个问题均修复；不出现新的溢出/遮挡/点击区域异常
  > 备注: 用户反馈“都 ok”

### 5. 文档同步（可选，随 ~exec 完成）

- [√] 5.1 补充前端主题/色板使用约定，减少未来回归
  - 目标: 在 `helloagents/modules/` 新增 `frontend-theme.md`（或更新既有文档，如存在）
  - 内容: 在主题敏感区域优先使用 `text-fg-*`/`bg-surface-*`，避免固定 `text-gray-*`；移动端头部避免同一行塞入过多控件
  - 验证: 文档与实际代码一致

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 1.1 | completed | 已在 Android Via + Chrome 真机确认 |
| 1.2 | completed | 已在 Android Via + Chrome 真机确认 |
| 2.1 | completed | 卡片补 `bg-slate-900` 作为透明渐变基底，浅色主题下对比度更稳定 |
| 2.2 | completed | 卡片副标题统一改为 `text-white/70`，避免浅灰在浅色背景上发虚 |
| 3.1 | completed | 标题/计数/筛选按钮补 `whitespace-nowrap` 防止逐字换行 |
| 3.2 | completed | 头部改为移动端分行（`flex-col sm:flex-row`），缓解小屏挤压 |
| 4.1 | completed | `npm run build` + `npm test` 通过 |
| 4.2 | completed | Android Via + Chrome 真机验证通过（用户反馈） |
| 5.1 | completed | 新增前端主题/排版约定文档，降低未来回归概率 |
