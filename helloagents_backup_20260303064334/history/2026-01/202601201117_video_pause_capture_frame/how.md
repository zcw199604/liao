# 技术设计: 视频播放暂停抓帧与倍速/慢放

## 技术方案

### 核心技术
- Vue 3 组件内管理 `HTMLVideoElement`
- `video.playbackRate` 实现倍速/慢放
- Canvas 2D：`drawImage(video)` + `canvas.toBlob()` 抓取当前帧
- 复用现有上传链路：`frontend/src/composables/useUpload.ts` 的 `uploadFile`

### 实现要点
- 在 `MediaPreview` 的视频元素上增加 `ref`，统一读取视频状态（暂停/当前时间/分辨率）并设置 `playbackRate`
- UI 增加倍速选择器（0.1/0.25/0.5/1/1.5/2/5），默认 1；切换媒体时保持默认或沿用上次选择（可使用 localStorage 记忆，作为可选增强）
- “抓帧”流程（满足“暂停后抓取这一帧”的目标）：
  1. 用户点击“抓帧”
  2. 若视频正在播放，先 `pause()`，确保画面稳定
  3. 校验 `videoWidth/videoHeight` 与 `readyState`，不足则提示“未加载到可抓帧状态”
  4. 使用 canvas 生成 PNG Blob（单帧）
  5. 同时执行：
     - 直接下载：复用现有 `blob` 下载逻辑
     - 上传到图片库：将 Blob 包装为 `File`，调用 `uploadFile(file, userId, userName)`；若未选择身份则仅下载并提示
- 跨域/CORS 限制：canvas 抓帧可能抛出 `SecurityError`，需捕获并提示，避免静默失败

## 安全与性能
- **安全:** 不在日志/文件名中写入本地路径/Token；仅在用户触发抓帧时生成并上传；未登录/未选择身份时不执行上传
- **性能:** 仅单帧截图；不保留大对象引用；下载使用 `URL.revokeObjectURL` 及时释放

## 测试与部署
- **测试:** 前端构建（`cd frontend && npm run build`）；手动验证倍速/慢放生效、暂停抓帧下载与上传可用、跨域失败提示清晰
- **部署:** 无新增依赖与配置，随前端发布

