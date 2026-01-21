# 技术设计: 媒体预览暂停遮罩图标隐藏

## 技术方案

### 核心技术

- Vue 3 单文件组件（SFC）`<style scoped>`
- WebKit/Blink `video` 控件伪元素样式覆盖

### 实现要点

- 为 `MediaPreview` 中主预览 `<video>` 增加稳定的专用 class（例如 `media-preview-video`），避免影响其他视频展示。
- 在 `MediaPreview.vue` 的 `<style scoped>` 中加入针对该 class 的 CSS 覆盖：
  - 隐藏 `::-webkit-media-controls-overlay-play-button`（Blink/部分 WebKit）
  - 隐藏 `::-webkit-media-controls-start-playback-button`（iOS Safari）
- 保持 `controls` 不变，确保进度条、音量、全屏等控制能力不受影响。

## 安全与性能

- **安全:** 仅样式变更，无新增数据/权限风险。
- **性能:** 仅增加少量 CSS 选择器，影响可忽略。

## 测试与验证

- 在 Chrome / Edge / Safari（如可用）下打开 `MediaPreview` 预览视频，暂停后确认中央遮罩图标消失。
- 验证控制栏交互：播放/暂停、进度拖动、倍速切换、全屏等仍可用。
- 回归验证：聊天内视频预览（进入 `MediaPreview`）与抽帧相关入口不受影响。

