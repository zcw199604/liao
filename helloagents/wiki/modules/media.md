# Media（媒体上传与媒体库）

## 目的
提供媒体上传、历史记录、媒体库查询与删除等能力，并保持对上游接口的代理兼容与本地数据可追溯。

## 模块概述
- **职责:** 上传/重传媒体；记录发送日志；分页查询上传/发送历史；全站媒体库分页；删除与批量删除；历史数据修复（repair）
- **状态:** ✅稳定
- **最后更新:** 2026-01-23

## 入口与交互
- **聊天页上传菜单:** “所有上传图片”（浏览后发送）/“mtPhoto 相册”
- **图片管理（Media）:** “所有上传图片”（管理/清理）/“mtPhoto 相册”/“图片查重”
- **身份选择页（Identity）:** 登录后、选择身份前可打开“图片管理”使用上述入口；其中“导入上传/重新上传到上游”等需要身份信息的操作会提示并禁用

## 规范

### 需求: 上传与本地落盘
**模块:** Media
上传接口需将文件落盘到 `./upload` 并写入本地表（详见 `data.md`），同时对上游上传接口保持兼容。

### 需求: 媒体历史与消息关联
**模块:** Media
通过 `recordImageSend` 记录 “fromUserId → toUserId” 的媒体发送关系，用于：
- 查询两人双向历史图片（`getChatImages`）
- 查询用户发送历史（`getUserSentImages`）

### 需求: 媒体端口策略（全局配置）
**模块:** Media
为解决上游媒体服务端口不固定/可达不等于可用的问题，系统提供全局媒体端口策略（DB 配置，所有用户共用）：
- `fixed`：固定媒体端口（默认 `9006`）
- `probe`：可用端口探测（并发 TCP 竞速，仅检测端口是否可连通；用于快速选出“能响应”的端口）
- `real`：真实媒体请求（并发竞速：同时对候选端口请求 `http://{host}:{port}/img/Upload/{path}`，按最小字节阈值校验内容；首个有效响应（HTTP 200/206）即返回并取消其它请求；若全部失败则降级到 `probe` 或最终回退 `fixed`）

前端在解析 WS/历史消息中的 `[path]` 媒体消息时，通过 `/api/resolveImagePort` 解析端口并拼接 URL（图片/视频共用该策略）。

### 需求: 媒体库分页与端口字段
**模块:** Media
“全站图片库”接口返回分页数据，并包含 `port` 字段（用于前端在缺少样本路径时的兜底展示/兼容旧逻辑）。当需要高可靠性时，前端应优先走 `/api/resolveImagePort` 的解析结果。

### 需求: 已上传图片浏览（布局切换/无限滚动）
**模块:** Media
“已上传的图片”（`AllUploadImageModal`）用于从“全站图片库”分页浏览素材并发送，前端要求：
- 支持瀑布流（masonry）与网格（grid）布局切换；为保证按时间排序时的视觉顺序，瀑布流采用“按行从左到右”的布局策略（行优先），并将用户选择持久化到 localStorage（`media_layout_mode`）
- 支持弹窗“全屏/退出全屏”最大化浏览区域，并将全屏偏好持久化到 localStorage（`media_modal_fullscreen`）；快捷键 `F` 切换全屏，`Esc` 优先退出全屏，否则关闭弹窗（预览打开时由 `MediaPreview` 优先处理）
- 支持无限滚动加载更多，复用 `InfiniteMediaGrid` 组件统一滚动/加载/空态/结束态逻辑
- 缩略图使用 `MediaTile` 统一渲染图片/视频（懒加载 + 错误兜底 + 统一的 overlay slot 布局），角落按钮/徽标统一使用 `MediaTileActionButton` / `MediaTileSelectMark` / `MediaTileBadge`；桌面端可按需设置“hover 才显示”，触屏设备默认可见可点；预览背景采用毛玻璃（`backdrop-blur`）提升沉浸感
- 弹窗展示区域针对大屏做放宽：弹窗宽度上限提升（`max-w-[1600px]`），高度提升至 `90vh`（支持 `90dvh`），列表容器内边距下调至 `p-2`，并在网格模式下将间距调整为 `gap-2` 以减少留白

### 需求: 删除兼容性（localPath 归一化）
**模块:** Media
删除接口需兼容多种 `localPath` 形式（含/不含前导 `/`、带 `/upload` 前缀、完整 URL、含 `%2F` 编码等），并在错误场景返回与现状兼容的 HTTP 状态与 body。

### 需求: 历史媒体数据修复（repairMediaHistory）
**模块:** Media
提供历史表 `media_upload_history` 的修复与去重能力（默认 dry-run，需显式开启写入/删除）。

### 需求: 媒体查重（image_hash）
**模块:** Media
提供“上传文件后查重”的能力：
- MD5 精确命中即返回重复文件信息
- 无 MD5 命中时，按 pHash 相似度阈值查询相似图片并返回相似度（视频/不可解码文件仅做 MD5 查重）

### 需求: 媒体预览（画廊模式）
**模块:** Media
`MediaPreview` 提供全屏沉浸式预览（图片/视频/文件），支持“受控 visible + 内部索引”的画廊模式（`mediaList`）：
- **入口:** 聊天消息/历史预览、全站图片库、mtPhoto 相册、抽帧任务中心等场景复用该组件进行预览。
- **画廊与同步:** 支持左右按钮/左右键/滑动切换；内部切换当前项会触发 `media-change`。父组件如提供“上传/重传/导入”等动作，应监听该事件同步目标，避免“切换后仍对首张执行操作”。
- **快捷键:** `Esc` 关闭预览；`←/→` 切换上一张/下一张（仅 `mediaList` > 1 生效）。
- **图片预览能力:** 点击放大/还原（默认放大倍数 3x）；放大后支持拖动平移；未放大时支持水平滑动切换（Swipe）。
- **视频预览能力:** 主视频预览使用 Plyr 美化控制栏 UI（暗色风格可主题化），并为移动端添加 `playsinline/webkit-playsinline` 避免 iOS 系统全屏接管；支持单击画面切换播放/暂停并浮现“倒退/播放暂停/快进”三按钮（快进/快退步长 1 秒，1 秒自动隐藏）；支持双击/双击触控进入/退出全屏（全屏右侧提供“抓帧/抽帧”快捷按钮，倍速按钮在全屏时移至左上避免重叠）；支持手势左右滑动快进/倒退（按 1 秒步进，约每 40~80px 触发 1 秒，降低误触）、上下滑动调音量（iOS/Safari 移动端可能限制网页调音量，会降级提示使用实体按键）。倍速/慢放（`playbackRate`）默认档位 0.1/0.25/0.5/1/1.5/2/5，持久化到 localStorage（`media_preview_playback_rate`），且倍速按钮支持长按临时 2x 播放，松开恢复原倍速。
- **非全屏布局:** 非全屏下由 `.plyr` 容器控制 `max-width/max-height:95%`，`video` 元素保持 `width/height:100%` 且 `object-fit: contain`，避免出现视频偏左或右侧留黑。
- **真全屏布局:** 真全屏（Fullscreen API / Plyr fullscreen）目标元素为 `.media-preview-video-wrapper`；需在 `:fullscreen/:-webkit-full-screen` 下强制 flex 居中并覆盖 `.plyr/video` 的 `max-*` 限制、圆角与阴影，避免全屏后出现偏移与不均匀黑边。
- **暂停抓帧:** 点击“抓帧”会先暂停视频，再基于 Canvas 抓取当前帧生成 PNG，并同时执行“直接下载 + 上传到图片库”；未选择身份则降级为仅下载；跨域视频可能因 CORS 限制无法抓帧，会提示并引导使用“抽帧任务/先上传到本地库”等替代路径。
- **详情面板:** 仅当媒体对象携带任一元信息字段（如 `md5/fileSize/pHash/similarity` 等）时显示入口；如列表阶段无法拿到真实文件名（仅有 `md5`），可传入 `resolveOriginalFilename` 回调，在用户打开面板前按需解析并补齐 `originalFilename`（仅展示 basename，避免泄露目录结构）。
- **下载策略:** 下载 `/api/*` 资源时使用带 Authorization 的 blob 下载，并解析 `Content-Disposition` 的 `filename*`（RFC 5987）与 `filename`；当 `filename` 为 URL 编码时会在保存前解码，避免中文文件名被编码。
- **大列表优化:** 当 `mediaList` 过大（>200）时，底部缩略图栏自动切换为虚拟滚动（`vue-virtual-scroller` 的 `RecycleScroller`）。
- **加载重试:** 图片/视频在短时间内可能 404（刚上传到上游）；预览对加载失败做轻量重试（最多 2 次，通过追加 cache buster 重新加载）。

### 需求: 抽帧任务临时视频（落盘隔离 + 退出清理）
**模块:** Media / Video Extract
抽帧任务中心“上传视频”入口使用 `/api/uploadVideoExtractInput` 将文件落盘到系统临时目录（默认 `os.TempDir()/video_extract_inputs`；不在 `./upload` 下）（不写入 `media_file`），并在退出任务中心时调用 `/api/cleanupVideoExtractInput` 删除该临时视频，避免污染媒体库目录与堆积占用空间。

### 需求: 测试覆盖（Go）
**模块:** Media
为文件功能补齐可重复执行的测试用例，覆盖：
- `FileStorageService`（落盘/读取/删除/MD5/复用查询）
- `ImageCacheService`（写入/读取/过期/重建/清空）
- `ImageHashService`（pHash 计算、阈值换算）
- `MediaUploadService`（localPath 归一化、本地 URL 转换、删除行为）
- handler：`/api/uploadMedia`、`/api/checkDuplicateMedia`

## API接口
### [POST] /api/uploadMedia
**描述:** 上传媒体（代理上游 + 本地落盘/记录）

### [POST] /api/checkDuplicateMedia
**描述:** 上传文件后按 `image_hash` 进行查重/相似检索（返回 similarity）

### [GET] /api/getAllUploadImages
**描述:** 全站媒体库分页（返回 `data/total/page/pageSize/totalPages/port`）

### [POST] /api/deleteMedia
**描述:** 删除单个媒体（DB 记录 + 可能的物理文件删除）

### [POST] /api/batchDeleteMedia
**描述:** 批量删除媒体（最多 50 个）

### [POST] /api/repairMediaHistory
**描述:** 修复/去重历史媒体数据（默认 dry-run）

## 数据模型
详见 `helloagents/wiki/data.md`：
- `media_file`
- `media_send_log`
- `media_upload_history`（历史遗留）
- `image_hash`（图片哈希索引）
- `system_config`（系统全局配置）

## 依赖
- 后端：
  - `internal/app/media_upload.go`
  - `internal/app/media_history_handlers.go`
  - `internal/app/media_repair.go`
  - `internal/app/media_repair_handlers.go`
  - `internal/app/file_storage.go`
  - `internal/app/image_cache.go`
  - `internal/app/image_server.go`
  - `internal/app/image_hash.go`
  - `internal/app/image_hash_handlers.go`
  - `internal/app/system_config.go`
  - `internal/app/system_config_handlers.go`
  - `internal/app/image_port_strategy.go`
  - `internal/app/image_port_resolver.go`
  - `internal/app/port_detect.go`
  - `internal/app/schema.go`
- 前端：
  - `frontend/src/components/media/AllUploadImageModal.vue`
  - `frontend/src/components/media/MediaPreview.vue`
  - `frontend/src/components/media/VideoExtractTaskModal.vue`
  - `frontend/src/components/common/InfiniteMediaGrid.vue`
  - `frontend/src/components/common/MediaTile.vue`
  - `frontend/src/components/common/MediaTileActionButton.vue`
  - `frontend/src/components/common/MediaTileSelectMark.vue`
  - `frontend/src/components/common/MediaTileBadge.vue`

## 测试
- 运行：`go test ./...`
- 相关测试文件：
  - `internal/app/file_storage_test.go`
  - `internal/app/image_cache_test.go`
  - `internal/app/image_hash_test.go`
  - `internal/app/media_history_handlers_test.go`
  - `internal/app/media_repair_handlers_test.go`
  - `internal/app/media_upload_test.go`
  - `internal/app/media_handlers_test.go`
  - `internal/app/static_file_server_test.go`
  - `internal/app/test_helpers_test.go`
  - `internal/app/user_history_media_handlers_test.go`

## 变更历史
- [202601201117_video_pause_capture_frame](../../history/2026-01/202601201117_video_pause_capture_frame/) - 视频预览支持倍速/慢放与暂停抓帧（下载 + 上传到图片库）
- [202601210322_media_preview_plyr](../../history/2026-01/202601210322_media_preview_plyr/) - 媒体预览主视频播放器升级为 Plyr（控制栏美化，功能保持一致）
- [202601210422_media_preview_video_click_hold_x2](../../history/2026-01/202601210422_media_preview_video_click_hold_x2/) - MediaPreview 视频交互增强（单击浮现三按钮/滑动快进&音量/长按临时 2x/抓帧抽帧按钮美化）
- [202601210515_media_preview_video_gesture_tune](../../history/2026-01/202601210515_media_preview_video_gesture_tune/) - MediaPreview 视频交互微调（滑动减敏/浮层±1秒/双击全屏/全屏右侧抓帧抽帧）
- [202601210551_media_preview_video_gesture_step_seek_fullscreen_ui](../../history/2026-01/202601210551_media_preview_video_gesture_step_seek_fullscreen_ui/) - MediaPreview 视频交互再微调（左右滑动 1 秒步进/方向锁定更保守/全屏倍速布局避让）
- [202601210823_media_preview_video_fullscreen_layout_fix](../../history/2026-01/202601210823_media_preview_video_fullscreen_layout_fix/) - 修复 MediaPreview 视频真全屏布局偏移（居中/黑边）
- [202601191522_media_gallery_expand](../../history/2026-01/202601191522_media_gallery_expand/) - 放宽“全站图片库/mtPhoto 相册”弹窗与图片列表展示区域，减少留白
- [202601181549_mtphoto_preview_gallery](../../history/2026-01/202601181549_mtphoto_preview_gallery/) - 媒体预览画廊切换时对外同步当前媒体（用于 mtPhoto/全站图片库等场景）
- [202601190109_mtphoto_preview_detail](../../history/2026-01/202601190109_mtphoto_preview_detail/) - 媒体预览支持按元信息显示“查看详情”入口并展示详情面板
- [202601190702_mtphoto_preview_real_filename](../../history/2026-01/202601190702_mtphoto_preview_real_filename/) - mtPhoto 预览“查看详情”按需解析并展示真实文件名
- [202601190728_mtphoto_download_filename_cn](../../history/2026-01/202601190728_mtphoto_download_filename_cn/) - 修复预览下载中文文件名被编码的问题（下载时解码）
- [202601072058_fix_delete_media_403](../../history/2026-01/202601072058_fix_delete_media_403/) - 修复删除接口对多种 localPath 形式的兼容性（待复验）
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并实现媒体上传/记录/媒体库
- [202601101607_image_hash_duplicate_check](../../history/2026-01/202601101607_image_hash_duplicate_check/) - 新增媒体查重接口（MD5 + pHash 相似度）
- [202601102011_go_file_tests](../../history/2026-01/202601102011_go_file_tests/) - Go 文件功能测试补齐（FileStorage/MediaUpload/ImageHash/handlers）
- [202601102319_image_port_strategy](../../history/2026-01/202601102319_image_port_strategy/) - 新增全局图片端口策略配置（fixed/probe/real）与前端 Settings 切换
- [202601110511_image_port_race](../../history/2026-01/202601110511_image_port_race/) - 图片端口解析优化：real 并发竞速与全失败降级兜底（回退 probe/fixed）
