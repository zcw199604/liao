# 技术设计: 媒体预览视频播放器升级为 Plyr

## 技术方案

### 核心技术

- Plyr（前端视频播放器 UI）
- Vue 3 `<script setup>` 生命周期与 `watch`（确保创建/销毁时序正确）
- Tailwind + scoped `:deep()`（对 Plyr 组件进行暗色主题与圆角/阴影适配）

### 实现要点

1. **依赖引入**
   - `frontend/package.json` 增加 `plyr`
   - 通过 `frontend/package-lock.json` 固化版本

2. **组件集成（仅主预览视频）**
   - `MediaPreview` 的主 `<video>` 保留 `ref="videoRef"`，在 `visible=true && currentMedia.type==='video'` 时 `new Plyr(video, options)`。
   - 在以下场景销毁实例：媒体切换（url/type 变化）、关闭预览（`visible=false`）、组件卸载（`onUnmounted`）。
   - 初始化失败时保留原生 `controls` 作为降级。

3. **功能一致性保证**
   - 倍速仍由现有 `playbackRate` 状态驱动，继续写入 `localStorage: media_preview_playback_rate`，并同步到底层 video（`video.playbackRate/defaultPlaybackRate`）。
   - 抓帧逻辑仍基于 `videoRef` 的 `HTMLVideoElement` 执行 `canvas.drawImage`。

4. **样式与交互**
   - 引入 `plyr/dist/plyr.css`（全局样式）。
   - 通过 scoped `:deep()` 设置 Plyr 暗色主题变量（例如 `--plyr-color-main`）并适配圆角/阴影。
   - 隐藏 Plyr 的中央大按钮（如 `:deep(.plyr__control--overlaid)`），避免遮挡暂停画面。
   - 为 iOS 增加 `playsinline`/`webkit-playsinline`，避免系统全屏接管导致自定义 UI 丢失。

## 安全与性能

- **安全:** 仅前端依赖与 UI 变更，不涉及鉴权/敏感数据处理变更。
- **性能:** Plyr 主要为 UI 包装，预览场景频率低；通过按需初始化与销毁避免事件泄露。

## 测试与验证

- `MediaPreview`：图片预览（放大/拖动/滑动切换）、视频预览（播放/暂停/拖动/倍速/抓帧/抽帧入口）、详情面板、下载行为、快捷键。
- 前端构建：`cd frontend && npm run build`。

