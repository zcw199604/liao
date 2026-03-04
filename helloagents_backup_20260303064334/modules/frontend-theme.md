# Frontend Theme & Typography

本模块记录前端主题（light/dark/auto）与排版相关的约定，用于避免浅色主题对比度不足、以及移动端小屏挤压导致的“逐字换行/竖排”问题。

## 1. 主题 Token 使用原则

- 优先使用语义化颜色 token（`text-fg*` / `bg-surface*` / `bg-canvas` / `border-line*`），让组件在浅色/深色主题下自动获得正确对比度。
- 避免在普通页面/抽屉/弹窗中直接使用 `text-gray-*` 或固定 `bg-[#...]` 作为正文文本颜色；除非该区域明确是“固定深色卡片/固定浅色卡片”风格。

## 2. 固定深色卡片的特殊规则

当组件内存在“固定深色卡片”（即卡片希望在 light/dark 下都保持深色视觉）时：

- 卡片容器应提供一个不透明的深色基底（例如 `bg-slate-900`），再叠加透明渐变（例如 `bg-gradient-to-br from-*-900/40 to-*-900/40`）。
  - 目的：避免透明渐变在浅色背景上被“抬亮”，从而导致 `text-white`/浅灰字对比度不足。
- 卡片内部文案可使用 `text-white`、`text-white/70` 等“固定白字体系”，不要使用 `text-fg*`（因为浅色主题下 `--fg` 为深色，放在深色卡片上会不可读）。

典型场景：`frontend/src/components/settings/SettingsDrawer.vue` 的“图片管理”入口卡片。

## 3. 移动端“逐字换行/竖排”排查与修复

现象：在 Android 浏览器（如 Via）中，标题/按钮文字变成每个字一行。

常见根因：
- Flex 容器同一行内容过多导致挤压，文字元素被压缩到接近 1 个字符宽度（中文会在字符间断行，形成“竖排”观感）。

推荐修复策略（从小到大）：
1) 对关键标题/按钮增加：
   - `whitespace-nowrap`（禁止逐字换行）
   - `shrink-0`（禁止被挤压到极窄）
2) 仍拥挤时，将头部改为响应式分行：
   - 外层：`flex flex-col sm:flex-row ...`
   - 筛选区：必要时使用 `flex-wrap` 或 `overflow-x-auto no-scrollbar`

典型场景：`frontend/src/components/media/AllUploadImageModal.vue` 的头部区域。



## 4. 统一 UI 复用类（2026-02 样式收敛）

为减少组件内重复 Tailwind 组合、提升视觉一致性，新增以下全局复用类（定义于 `frontend/src/index.css` 的 `@layer components`）：

- `ui-card`：统一毛玻璃卡片容器（用于登录卡片、确认弹窗）
- `ui-input`：统一普通输入框
- `ui-input-shell`：统一聊天输入框外壳（含 `focus-within` 反馈）
- `ui-btn-primary`：统一高强调主按钮（如“登录”）
- `ui-btn-secondary`：统一中性按钮（如“取消”）
- `ui-icon-btn`：统一图标按钮底座
- `ui-icon-btn-ghost`：弱化背景的图标按钮变体
- `ui-fab-primary`：统一圆形主操作按钮（如“发送”）

落地原则：
- 组件优先复用以上类，不在 SFC 内重复拼接同构样式。
- 若确有业务差异，仅在组件内追加最小差异类（例如 `hover:text-yellow-500`）。
- 需要全局视觉调整时，优先改 `index.css` 的复用类，避免批量改动多个组件模板。


## 5. 第二轮精修：列表与媒体预览复用类

为进一步统一“聊天侧栏 + 媒体预览”视觉语言，新增并落地以下复用类：

- `ui-list-item`：会话列表项统一卡片样式（边框/阴影/hover/active）
- `ui-empty-state`：空状态统一布局与文字层级
- `ui-glass-topbar`：侧栏顶部玻璃化栏位
- `ui-overlay-icon-btn`：媒体预览顶部/侧边圆形工具按钮
- `ui-overlay-pill-btn`：媒体预览倍速等胶囊按钮
- `ui-overlay-menu`：媒体预览下拉菜单容器

落地文件：
- `frontend/src/components/chat/ChatSidebar.vue`
- `frontend/src/components/media/MediaPreview.vue`

兼容约束（测试稳定性）：
- 对测试中依赖的模板选择器类（如 `div.flex.items-center.p-4`、`div.cursor-pointer`）保留显式类名；
- 复用类用于收敛样式组合，不改变现有事件名、文案和交互流程。


## 6. 审查后回归修正（Claude Review Follow-up）

针对外部审查提出的视觉一致性与副作用风险，追加以下约束：

- 小尺寸菜单（如侧栏顶部菜单、会话右键菜单）优先使用 `ui-card-sm`，避免在窄容器使用 `rounded-2xl` 导致圆角比例失衡。
- 媒体预览顶部工具区使用浅色半透明变体：
  - `ui-overlay-icon-btn-light`
  - `ui-overlay-pill-btn-light`
  - `ui-overlay-menu-light`
  以保持原有“白色半透明”视觉语言；全屏覆盖控件继续使用深色变体。
- 全局 `body` 默认不启用颜色过渡动画，避免主题切换初始阶段可能出现闪烁。
- 复用类设计避免与模板类重复堆叠：布局/结构类留在模板，复用类侧重视觉 token 收敛。

## 7. 媒体工具弹窗主题适配基线（2026-02）

本轮修复将以下组件从“硬编码暗色”迁移到语义 token 体系：

- `frontend/src/components/media/DuplicateCheckModal.vue`
- `frontend/src/components/media/DouyinDownloadModal.vue`
- `frontend/src/components/media/MediaDetailPanel.vue`

约束：
- 普通容器/表单/列表项必须使用 `bg-surface*`、`text-fg*`、`border-line*`，避免 `bg-[#18181b]` / `text-gray-*` / `border-white/10`。
- 可保留固定深色的场景仅限“语义性遮罩/媒体覆盖层”（如 `bg-black/60` 的全屏遮罩、加载蒙层）。
- 交互按钮遵循：中性按钮 `bg-surface-3 + text-fg`，品牌/危险按钮可继续使用 `bg-emerald-*`/`bg-red-* + text-white`。

