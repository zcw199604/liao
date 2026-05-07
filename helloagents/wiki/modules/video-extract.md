# Video Extract

## 目的
提供视频抽帧任务能力，支持上传视频或 mtPhoto 视频作为来源，并分页预览输出帧。

## 模块概述
- **职责:** 源视频上传/清理、ffprobe 探测、ffmpeg 抽帧、任务队列、暂停/续跑/删除、帧索引分页。
- **状态:** 稳定
- **最后更新:** 2026-05-07

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
