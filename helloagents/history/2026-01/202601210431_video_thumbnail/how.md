# 技术设计: 视频上传缩略图

## 技术方案

### 核心技术
- 后端：Go（`os/exec` 调用 `ffmpeg` 抽取单帧）
- 存储：本地 `./upload` 静态目录（通过 `/upload/*` 提供访问）
- 数据：`media_file` 追加 `thumb_local_path`（保存本地缩略图相对路径）

### 实现要点
- 上传视频落盘成功后，根据 `localPath` 生成缩略图路径：同目录下 `basename + "_thumb.jpg"`。
- 调用 `ffmpeg` 生成缩略图（best-effort）：
  - 超时控制，失败仅记录日志，不中断上传流程
  - 统一输出为 JPG，并做缩放（宽度 320，等比）
- `GET /api/getAllUploadImages` 返回 `MediaFileDTO` 增加 `thumbUrl`：
  - 图片：不返回或为空（前端继续使用 `url`）
  - 视频：返回 `thumbUrl`（若缺失则前端回退）

## API设计

### [GET] /api/getAllUploadImages
- **变更:** `data[]` 的元素增加 `thumbUrl?: string`
- **说明:** `thumbUrl` 为本地静态访问 URL（`http://{host}/upload{thumb_local_path}`）

## 数据模型

```sql
ALTER TABLE media_file
  ADD COLUMN thumb_local_path VARCHAR(500) NULL COMMENT '视频缩略图本地路径（/videos/..._thumb.jpg）';
```

## 安全与性能
- **安全:** `ffmpeg` 调用不拼接 shell 字符串，参数以数组形式传入；缩略图生成失败不影响主流程。
- **性能:** 缩略图只生成 1 帧并缩放；使用超时避免长时间占用资源。

## 测试与部署
- **测试:** `go test ./...`；补齐/更新与 `media_file` 字段变更相关的 handler/service 测试。
- **部署:** 需确保运行环境可访问 `ffmpeg`（可通过 `FFMPEG_PATH` 指定可执行文件路径）。
