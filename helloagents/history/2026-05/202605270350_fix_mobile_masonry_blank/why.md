# 变更提案: 修复移动端图片瀑布流单侧空白

## 需求背景
当前媒体库和 mtPhoto 复用 `InfiniteMediaGrid` 实现瀑布流。移动端固定为两列时，瀑布流按 `width/height` 预估列高并把下一项放到当前最短列；但全站图片库没有从后端到前端保留媒体宽高，前端只能按 1:1 正方形估算。实际图片加载后如果存在大量竖图、长图或混合比例图片，两列真实高度会明显偏离预估，短列提前结束，用户看到一边是空白、另一边继续显示图片。

现有历史修复机制包括 `/api/repairMediaHistory` 和 `/api/repairVideoPosters`，分别处理历史上传记录 MD5/去重和视频封面，不处理 `media_file`、`douyin_media_file` 的图片宽高。因此本次修复必须新增历史尺寸处理逻辑，而不是只依赖列表接口临时解析。

## 变更内容
1. 为 `media_file`、`douyin_media_file` 新增持久化尺寸字段 `media_width`、`media_height`。
2. 在新上传、本地导入、抖音导入、mtPhoto 导入等写入媒体库的路径中同步写入尺寸元数据。
3. 新增历史数据处理接口 `/api/repairMediaDimensions`，按批次回填历史媒体宽高。
4. 历史处理必须识别文件不存在、路径非法、解析失败、非目标媒体类型等情况，记录统计和 warnings，但不能因单条异常中断整批任务。
5. `GET /api/getAllUploadImages` 从数据库返回 `width`、`height`，前端据此进行瀑布流分列和 `MediaTile` 预占位。
6. 增强 `InfiniteMediaGrid` 的缺尺寸兜底策略，保证少量无法修复的数据不会造成明显单侧空白。

## 影响范围
- **模块:** Media、mtPhoto、前端通用媒体网格
- **文件:**
  - `internal/app/media_upload.go`
  - `internal/app/media_dimensions_repair.go`（新增）
  - `internal/app/media_dimensions_repair_handlers.go`（新增）
  - `internal/app/router.go`
  - `internal/app/mtphoto_handlers.go`
  - `internal/app/douyin_handlers.go`
  - `sql/mysql/001_init.sql`
  - `sql/mysql/008_media_dimensions.sql`（新增）
  - `sql/postgres/001_init.sql`
  - `sql/postgres/008_media_dimensions.sql`（新增）
  - `frontend/src/stores/media.ts`
  - `frontend/src/components/media/AllUploadImageModal.vue`
  - `frontend/src/components/common/InfiniteMediaGrid.vue`
  - 相关测试文件
- **API:**
  - `GET /api/getAllUploadImages` 响应增加可选字段 `width`、`height`
  - 新增 `POST /api/repairMediaDimensions`
- **数据:**
  - `media_file.media_width`
  - `media_file.media_height`
  - `douyin_media_file.media_width`
  - `douyin_media_file.media_height`

## 核心场景

### 需求: 移动端瀑布流两列稳定展示
**模块:** Media
媒体库在移动端 masonry 模式下展示图片列表。

#### 场景: 全站图片库包含不同宽高比图片
用户打开媒体库，列表含横图、竖图、长图混合。
- 两列按数据库中的真实图片比例预占位和分配。
- 滚动加载下一页后不出现长期单侧空白。
- 图片加载完成后布局不发生大幅跳动。

#### 场景: 媒体缺失宽高元数据
部分记录尚未回填尺寸或解析失败。
- 前端使用保守兜底比例，仍能维持两列基本均衡。
- 不阻塞图片展示和分页加载。
- 控制台不产生异常。

### 需求: 历史媒体尺寸批量回填
**模块:** Media
管理员或维护脚本调用历史处理接口，为已有 `media_file`、`douyin_media_file` 记录补齐尺寸。

#### 场景: 历史文件存在且可解析
接口按 `source`、`startAfterId`、`limit` 分批扫描历史记录。
- dry-run 时只统计待更新数量，不写库。
- commit 时写入 `media_width`、`media_height`。
- 返回 `nextAfterId`、`hasMore`，便于多次调用直到处理完成。

#### 场景: 历史文件不存在
数据库存在记录，但 `local_path` 对应文件已经被删除、迁移或损坏。
- 该记录计入 `fileMissing`，不写入宽高。
- warnings 中记录有限数量样例，包括 id、source、local_path。
- 批处理继续处理后续记录，不整体失败。

#### 场景: 历史路径非法或解析失败
记录的 `local_path` 为空、越界、文件格式异常或图片头不可解析。
- 分别计入 `invalidPath`、`decodeFailed` 或 `unsupported`。
- 不写入宽高。
- 返回统计结果供后续人工排查。

### 需求: 新增媒体自动写入尺寸
**模块:** Media
用户上传、抖音导入、mtPhoto 导入媒体时写入媒体库记录。

#### 场景: 新图片入库
图片保存到本地后立即解析宽高。
- `media_file` 或 `douyin_media_file` 写入时包含 `media_width`、`media_height`。
- 后续打开媒体库无需再回填该记录。

### 需求: mtPhoto 与全站图片库行为一致
**模块:** mtPhoto
mtPhoto 上游列表已有 `width/height` 字段，但导入本地媒体库后也应遵循本地媒体尺寸字段。

#### 场景: mtPhoto 文件夹/相册移动端浏览
用户打开 mtPhoto 相册或文件夹。
- 已有宽高继续用于 `MediaTile` 占位和瀑布流分列。
- 通用组件改动不破坏 mtPhoto 视频/图片预览和点击行为。

## 风险评估
- **风险:** 数据库迁移增加字段，需兼容 MySQL 和 PostgreSQL。
- **缓解:** 增加双数据库迁移脚本，并同步更新初始化 schema；字段使用 nullable int，避免历史记录迁移阻塞启动。
- **风险:** 历史文件不存在或路径异常较多。
- **缓解:** 历史处理接口将缺失/异常作为统计状态，不作为整批错误；warnings 限量返回，避免响应过大。
- **风险:** 批量解析历史图片可能产生 I/O 压力。
- **缓解:** 使用 `limit`、`startAfterId` 分批处理，默认 dry-run；单次上限受控，调用方按 `hasMore` 续跑。
- **风险:** 前端 masonry 分列变化可能改变现有图片顺序的视觉排列。
- **缓解:** 保持原始数据顺序和 key 不变，只调整列分配依据；网格模式不受影响。
