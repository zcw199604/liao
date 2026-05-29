# Video Extract

## 目的
提供视频抽帧任务能力，支持上传视频或 mtPhoto 视频作为来源，并分页预览输出帧。

## 模块概述
- **职责:** 源视频上传/清理、ffprobe 探测、ffmpeg 抽帧、任务队列、暂停/续跑/删除、帧索引分页。
- **状态:** 稳定
- **最后更新:** 2026-05-16

## 规范

### 需求: 任务参数持久化
**模块:** Video Extract
抽帧参数、源信息、进度、状态和错误日志保存在 `video_extract_task`。

#### 场景: 创建任务
- 先探测源视频元数据。
- 按 `mode=keyframe/fps/all` 生成 ffmpeg 参数。
- 输出目录位于 `./upload/extract/{taskId}`。

### 需求: 运行时可恢复
**模块:** Video Extract
任务支持暂停、因上限停止后续跑，并通过 `cursor_out_time_sec` 和已生成帧数继续。

#### 场景: 续跑
- 后续帧序号必须跨续跑单调递增。

### 需求: 源文件安全
**模块:** Video Extract
清理临时输入和删除任务时必须限定在允许目录内，避免误删非工作区文件。

### 需求: 从媒体预览创建抽帧任务
**模块:** Video Extract
`MediaPreview` 中“视频工具”菜单的“创建抽帧任务”入口会调用 `videoExtractStore.openCreateFromMedia`，将可定位的 upload 视频或 mtPhoto 视频转为抽帧创建来源，然后关闭预览并打开 `VideoExtractCreateModal`。

#### 场景: 入口可用性
- 仅当视频能够解析为 upload `localPath` 或 mtPhoto `md5` 时允许进入抽帧创建。
- upload 来源只接受 `/videos/` 或 `/tmp/video_extract_inputs/` 下的可定位本地路径；空 URL、普通远程 URL 和不可解析来源不会打开创建弹窗。
- mtPhoto 来源要求 `md5` 且 URL 来自 `/lsp/` 或 `/api/` 路径。
- 抽帧创建弹窗和抽帧任务中心中的源视频预览会传入 `showExtractTask=false`，避免在任务流程内重复显示“创建抽帧任务”入口。

#### 场景: 创建门禁
- 创建任务前必须完成 ffprobe 探测。
- `probe` 缺失、`probeLoading` 未完成或 `probeError` 存在时，前端按钮禁用，store 层也会阻断提交。

#### 场景: 续跑与分页
- 继续抽帧必须至少填写新的 `endSec` 或 `maxFrames`。
- `endSec` 必须大于当前 `cursorOutTimeSec` 或任务 `startSec`。
- `maxFrames` 必须为正整数且大于已输出帧数。
- 帧分页增量合并时按有效 `seq` 去重，保留已加载帧，避免重复渲染。

## API接口
- `POST /api/uploadVideoExtractInput`
- `POST /api/cleanupVideoExtractInput`
- `GET /api/probeVideo`
- `POST /api/createVideoExtractTask`
- `GET /api/getVideoExtractTaskList`
- `GET /api/getVideoExtractTaskDetail`
- `POST /api/cancelVideoExtractTask`
- `POST /api/continueVideoExtractTask`
- `POST /api/deleteVideoExtractTask`

## 数据模型
- `video_extract_task`
- `video_extract_frame`

## 依赖
- `internal/app/video_extract.go`
- `internal/app/video_extract_handlers.go`
- `frontend/src/api/videoExtract.ts`
- `frontend/src/stores/videoExtract.ts`
