# 技术设计: 图片弹窗全屏模式（全站图片库/mtPhoto 相册）

## 技术方案

### 核心技术
- Vue 3（`<script setup>`）
- TailwindCSS（条件 class 切换）
- localStorage（偏好持久化）

### 实现要点
- 在 `AllUploadImageModal` / `MtPhotoAlbumModal` 增加 `isFullscreen` 状态（初始化读取 localStorage）
- 通过 `:class` 切换弹窗容器尺寸：
  - 普通模式：沿用当前 `w-[95%] max-w-[1600px] h-[90vh] h-[90dvh] rounded-2xl shadow-2xl`
  - 全屏模式：`w-screen max-w-none h-screen h-[100dvh] rounded-none shadow-none`
- 头部新增按钮：图标 `fa-expand` / `fa-compress`，并提供 `title` 提示
- 键盘快捷键：
  - `window.addEventListener('keydown', ...)` 仅在弹窗打开时注册，关闭时卸载
  - `F` 切换全屏；`Esc`：全屏优先退出，否则关闭弹窗
- localStorage key 建议统一：`media_modal_fullscreen = '1' | '0'`（与 `media_layout_mode` 同风格）；两处弹窗共用，保持一致体验
- （可选）抽取 `useModalFullscreen(key)` composable 复用逻辑，减少两处重复代码

## 架构决策 ADR

### ADR-001: 全屏实现采用“应用内最大化”而非 Fullscreen API
**上下文:** 需求是“最大化显示内容”以便浏览大量图片（不强制隐藏浏览器 UI）。  
**决策:** 优先实现“应用内最大化”（弹窗占满视口）而非 `requestFullscreen()`。  
**理由:** 无权限弹窗/失败路径；移动端兼容更稳；实现简单且满足核心诉求。  
**替代方案:** Fullscreen API（`requestFullscreen()`）→ 拒绝原因：平台差异、失败处理复杂、用户手势限制。  
**影响:** 仍处于浏览器窗口内；若后续需要“沉浸式真全屏”，可在现有最大化基础上增量加入第二层开关。

## 安全与性能
- **安全:** 不涉及用户数据与权限；仅存储 UI 偏好到 localStorage
- **性能:** 事件监听按需注册/清理避免泄露；全屏仅为样式变更，不新增网络请求

## 测试与部署
- **构建验证:** `npm -C frontend run build`
- **手工验收:** 两处弹窗全屏切换、滚动加载、布局切换与关闭行为正常；与 `MediaPreview` 交互不冲突
- **可选单测:** 使用 vitest 断言全屏切换时容器 class 与 localStorage 写入

