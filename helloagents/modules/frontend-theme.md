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

