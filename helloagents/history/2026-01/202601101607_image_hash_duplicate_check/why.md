# 变更提案: image_hash 媒体查重接口（MD5 + pHash）

## 需求背景
当前前端在上传文件前需要判断“是否已存在相同/相似图片”。本地已有 `image_hash` 表存储了全量图片的路径与哈希信息，但缺少可直接调用的查重接口。

## 变更内容
1. 新增接口 `POST /api/checkDuplicateMedia`：上传文件后返回查重结果（命中类型 + 图片元信息 + 相似度）。
2. 查重策略：
   - **MD5 命中**：按 `md5_hash` 精确匹配，直接返回命中列表。
   - **MD5 未命中**：对“可解码图片”计算 pHash，并按相似度阈值返回相似图片列表（携带 `similarity/distance`）。
   - **视频/不可解码格式**：仅做 MD5 查重（pHash 不可用时返回 `matchType=none` + reason）。

## 影响范围
- **模块:** Backend（Go）/ Frontend（API封装）/ 知识库（API + 数据模型 + 模块规范）
- **文件:**
  - `internal/app/image_hash.go`
  - `internal/app/image_hash_handlers.go`
  - `internal/app/router.go`
  - `internal/app/app.go`
  - `internal/app/schema.go`
  - `frontend/src/api/media.ts`
  - `helloagents/wiki/api.md`
  - `helloagents/wiki/data.md`
  - `helloagents/wiki/arch.md`
  - `helloagents/wiki/modules/media.md`

## 核心场景

### 需求: 上传文件查重
**模块:** Media
上传文件后，返回重复/相似图片的信息，供前端提示或复用本地资源。

#### 场景: MD5 命中
文件 MD5 与 `image_hash.md5_hash` 完全一致。
- 返回 `matchType="md5"` 以及命中图片信息（`filePath/fileName/fileDir/fileSize` 等）
- 无需计算 pHash

#### 场景: MD5 未命中，但相似度命中
文件 MD5 无命中，且文件为可解码图片。
- 计算上传图片 pHash
- 按 `similarityThreshold` 查询相似图片
- 返回 `matchType="phash"`、相似图片信息及 `similarity/distance`

#### 场景: MD5 未命中，且无法计算 pHash
文件为视频或不可解码图片格式。
- 仅返回 `matchType="none"`（items 为空）并携带 reason

## 风险评估
- **风险:** pHash 计算算法与现有入库流程不一致，可能导致相似检索命中率下降。
  - **缓解:** 使用常见 pHash（32x32 灰度 + DCT 低频 8x8 + 中位数阈值）实现，并在文档中明确“仅对可解码图片计算”。
- **风险:** `image_hash` 数据量较大时，相似查询可能带来 DB 压力。
  - **缓解:** 提供 `limit` 参数与阈值换算 `maxDistance`，并为 `md5_hash/phash` 提供索引（新建表时生效）。

