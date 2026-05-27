# 任务清单: 修复移动端图片瀑布流单侧空白

目录: `helloagents/plan/202605270350_fix_mobile_masonry_blank/`

---

## 1. 数据库尺寸字段
- [√] 1.1 在 `sql/mysql/008_media_dimensions.sql` 中新增 `media_file`、`douyin_media_file` 的 `media_width`、`media_height` 字段迁移，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析
- [√] 1.2 在 `sql/postgres/008_media_dimensions.sql` 中新增对应 PostgreSQL 迁移，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析
- [√] 1.3 更新 `sql/mysql/001_init.sql` 和 `sql/postgres/001_init.sql` 的初始化表结构，验证新库初始化包含尺寸字段，依赖任务1.1、1.2

## 2. 后端尺寸模型与新媒体写入
- [√] 2.1 在 `internal/app/media_upload.go` 中为 `UploadRecord`、`DouyinUploadRecord`、`MediaFileDTO` 增加宽高字段，验证 why.md#需求-新增媒体自动写入尺寸-场景-新图片入库
- [√] 2.2 在 `internal/app/media_upload.go` 中实现本地图片尺寸解析辅助函数，使用安全路径解析和 `image.DecodeConfig`，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史路径非法或解析失败
- [√] 2.3 在 `SaveUploadRecord`、`SaveDouyinUploadRecord` 中写入 `media_width`、`media_height`，验证 why.md#需求-新增媒体自动写入尺寸-场景-新图片入库，依赖任务2.1
- [√] 2.4 在上传、抖音导入、mtPhoto 导入路径中解析本地文件尺寸并传入保存记录，验证 why.md#需求-新增媒体自动写入尺寸-场景-新图片入库，依赖任务2.2、2.3
- [√] 2.5 在 `GetAllUploadImagesWithDetailsBySource` 查询中读取并返回数据库尺寸字段，验证 why.md#需求-移动端瀑布流两列稳定展示-场景-全站图片库包含不同宽高比图片

## 3. 历史尺寸处理接口
- [√] 3.1 新增 `internal/app/media_dimensions_repair.go`，实现 `RepairMediaDimensionsRequest`、`RepairMediaDimensionsResult` 和批处理服务，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析
- [√] 3.2 在历史处理服务中实现 `commit=false` dry-run 逻辑，只统计 `needUpdate` 不写库，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析，依赖任务3.1
- [√] 3.3 在历史处理服务中实现 `commit=true` 回填逻辑，按 `source`、`startAfterId`、`limit` 扫描并更新尺寸，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析，依赖任务3.1
- [√] 3.4 在历史处理服务中处理文件不存在，计入 `fileMissing` 并继续后续记录，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件不存在，依赖任务3.1
- [√] 3.5 在历史处理服务中处理路径非法、解析失败、非图片或不支持格式，分别计入 `invalidPath`、`decodeFailed`、`unsupported`，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史路径非法或解析失败，依赖任务3.1
- [√] 3.6 新增 `internal/app/media_dimensions_repair_handlers.go` 并在 `internal/app/router.go` 注册 `POST /api/repairMediaDimensions`，验证 why.md#需求-历史媒体尺寸批量回填-场景-历史文件存在且可解析，依赖任务3.1

## 4. 前端数据映射与展示占位
- [√] 4.1 在 `frontend/src/stores/media.ts` 中保留后端返回的 `width`、`height` 字段，验证 why.md#需求-移动端瀑布流两列稳定展示-场景-全站图片库包含不同宽高比图片
- [√] 4.2 在 `frontend/src/components/media/AllUploadImageModal.vue` 中为 masonry 模式的 `MediaTile` 传入 `aspect-ratio`，并保持 grid 模式不变，验证 why.md#需求-移动端瀑布流两列稳定展示-场景-全站图片库包含不同宽高比图片，依赖任务4.1
- [√] 4.3 检查 `frontend/src/components/media/MtPhotoAlbumModal.vue` 的调用兼容性，确保通用组件改动不破坏 mtPhoto，验证 why.md#需求-mtphoto-与全站图片库行为一致-场景-mtphoto-文件夹相册移动端浏览

## 5. 通用瀑布流兜底
- [√] 5.1 在 `frontend/src/components/common/InfiniteMediaGrid.vue` 中抽出高度估算函数，优先使用 `width/height`，缺失时按媒体类型使用保守默认比例，验证 why.md#需求-移动端瀑布流两列稳定展示-场景-媒体缺失宽高元数据
- [√] 5.2 在 `frontend/src/components/common/InfiniteMediaGrid.vue` 中评估移动端缺尺寸数据的降级策略，必要时对连续缺尺寸项目采用 round-robin 分配，验证 why.md#需求-移动端瀑布流两列稳定展示-场景-媒体缺失宽高元数据，依赖任务5.1

## 6. 安全检查
- [√] 6.1 执行安全检查（按G9: 本地路径解析、文件读取边界、历史文件不存在处理、无敏感信息输出、无破坏性操作）

## 7. 测试
- [√] 7.1 增加或更新 Go 测试，覆盖 `getAllUploadImages` 返回宽高、上传写入宽高、数据库字段兼容，验证点: DTO 字段和接口兼容性
- [√] 7.2 增加 Go 测试覆盖 `repairMediaDimensions` dry-run、commit、force、文件缺失、路径非法、解析失败、unsupported、hasMore 游标，验证点: 统计字段和跳过逻辑
- [√] 7.3 增加或更新前端测试，覆盖 `InfiniteMediaGrid` 移动端两列分配、有宽高估算、缺宽高兜底，验证点: 列数据分布和 item key 保持稳定
- [√] 7.4 执行 `go test ./...`，验证后端回归
- [√] 7.5 执行 `cd frontend && npm run build`，验证前端类型检查和生产构建

## 8. 文档更新
- [√] 8.1 更新 `helloagents/wiki/modules/media.md`，记录媒体库瀑布流依赖持久化宽高、历史尺寸处理接口和文件不存在处理策略
- [√] 8.2 如实施中调整 mtPhoto 行为，更新 `helloagents/wiki/modules/mtphoto.md`
- [√] 8.3 更新 `helloagents/wiki/api.md`，记录 `POST /api/repairMediaDimensions`
- [√] 8.4 更新 `helloagents/wiki/data.md`，记录 `media_width`、`media_height`
- [√] 8.5 更新 `helloagents/CHANGELOG.md`，记录移动端瀑布流修复和历史尺寸回填能力
