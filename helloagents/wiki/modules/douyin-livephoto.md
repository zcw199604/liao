# Douyin Live Photo（实况照片导出）

## 目的
在站内对“实况照片（Live 图）”提供可下载导出能力，并尽量保证：
- iOS：保存/导入后仍可被系统相册识别为 Live Photo
- MIUI：保存到相册后可被 MIUI 相册识别为“动态照片”

## 入口
- 前端预览组件：`frontend/src/components/media/MediaPreview.vue` 提供“下载实况”按钮
- 后端接口：`GET /api/douyin/livePhoto`

## 输出格式
### 1) `format=zip`（iOS Live Photo）
- 下载文件名：与普通图片下载一致，并追加 `_live`，例如：`标题_01_live.zip`
- ZIP 内包含：
  - `标题_01.jpg`（静态图）
  - `标题_01.mov`（动态视频，QuickTime 容器）
- 通过 `exiftool` 写入相同 `ContentIdentifier`，由 iOS 相册配对识别为同一张 Live Photo

### 2) `format=jpg`（Motion Photo / MIUI 动态照片）
- 下载文件名：与普通图片下载一致，并追加 `_live`，例如：`标题_01_live.jpg`
- 单文件结构（关键点）：
  - JPEG + XMP：包含 `GCamera:MicroVideo*`，其中 `GCamera:MicroVideoOffset` = **尾部 MP4 字节长度**
  - JPEG 尾部追加 MP4（`ftyp...`）
  - 为提升 MIUI 相册识别概率：
    - JPEG 头部段顺序固定为：`APP1(Exif) → APP1(XMP) → APP0(JFIF)`
    - 在 JPEG `EOI(FFD9)` 与 MP4 之间插入固定 **24 字节 gap**

## 文件命名规则
实况导出以静态图的 `imageIndex` 作为“序号基准”：
- 先按普通图片命名规则生成基础名：`buildDouyinOriginalFilename(title, detailID, imageIndex, len(downloads), ".jpg")`
- 去掉 `.jpg` 后追加 `_live` 再拼接目标扩展名（`.jpg/.zip`）

## 实现位置
- 后端：
  - 路由：`internal/app/router.go`（`GET /api/douyin/livePhoto`）
  - 处理逻辑：`internal/app/douyin_livephoto_handlers.go`
    - `handleDouyinLivePhoto`
    - Motion Photo 生成：`buildMotionPhotoJPG` / `buildMotionPhotoExifAPP1Segment` / `buildMotionPhotoXMPAPP1Segment`
- 测试：
  - `internal/app/douyin_motion_photo_test.go`：`TestBuildMotionPhotoJPG_MiuiShape`

## 依赖
- `ffmpeg`：两种格式都依赖（用于转封装/转码）
- `exiftool`：仅 `format=zip` 依赖（写入 iOS Live Photo 配对元数据）

## 快速验证（本地/离线）
对导出的 `*_live.jpg`：
- 搜索 XMP：应包含 `GCamera:MicroVideoOffset="<N>"`
- 计算 `mp4Start = fileSize - N`：应在 `mp4Start+4..+8` 看到 `ftyp`
- `EOI` 后到 `mp4Start` 的 gap 长度应为 24 字节（与后端常量一致）

