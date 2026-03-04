# 技术设计: image_hash 媒体查重接口（MD5 + pHash）

## 技术方案

### 核心技术
- 后端：Go 1.22（`image.Decode` + 标准库图片解码器）
- 数据库：MySQL（`BIT_COUNT` + `^` 计算汉明距离）

### 实现要点
- **MD5**：复用 `FileStorageService.CalculateMD5`（流式读取）。
- **pHash（仅图片）**：
  1. 解码图片 → 灰度缩放至 `32x32`（Lanczos3，尽量对齐 PIL `ANTIALIAS`）
  2. 计算 2D DCT（对齐 `scipy.fftpack.dct` 默认：type=2, norm=None），取低频 `8x8`
  3. 使用 `numpy.median(dctlowfreq)`（包含 DC，且偶数长度取中间两值均值）作为阈值生成 64 位 hash
- **相似度**：
  - 距离：`distance = BIT_COUNT(phash ^ inputPhash)`（0~64）
  - 相似度：`similarity = (64 - distance) / 64`
  - 阈值：默认 `distanceThreshold=10`；若提供 `similarityThreshold` 则换算为 `distanceThreshold=floor((1-sim)*64)`
- **结果排序**：按 `distance` 升序（更相似优先），再按 `id` 倒序。

## API设计

### [POST] /api/checkDuplicateMedia
- **请求（multipart/form-data）**
  - `file`：上传文件（必填）
  - `similarityThreshold`：最小相似度阈值（`0-1` 或 `0-100`），提供则优先使用
  - `distanceThreshold`（或 `threshold`）：最大汉明距离阈值（0-64），默认 `10`
  - `limit`：返回条数上限，默认 `20`，最大 `200`

- **响应（HTTP 200）**
  - `data.matchType`：`md5 | phash | none`
  - `data.thresholdType`：`similarity | distance`
  - `data.similarityThreshold` / `data.distanceThreshold`：实际使用的阈值（便于前端展示）
  - `data.items[]`：命中列表（包含 `filePath/fileName/fileDir/fileSize/md5Hash/pHash/createdAt` 以及 `distance/similarity`）
  - pHash 不可计算时返回 `matchType=none` 且 `reason` 说明原因

## 数据模型

```sql
CREATE TABLE image_hash (
  id INT AUTO_INCREMENT PRIMARY KEY,
  file_path VARCHAR(1000) NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  file_dir VARCHAR(500) NULL,
  md5_hash VARCHAR(32) NOT NULL,
  phash BIGINT NOT NULL,
  file_size BIGINT NULL,
  created_at DATETIME NOT NULL
);
```

## 安全与性能
- **安全**
  - 参数校验：`similarityThreshold` 必须可解析为数字；`distanceThreshold` 必须为 0~64；`limit` 限制范围（1~200）
  - 不落盘：查重接口仅读取上传文件内容，不写入本地文件系统与数据库
- **性能**
  - 优先 MD5 精确命中，避免 pHash 计算与相似查询
  - 相似查询使用 DB 侧 `BIT_COUNT` 过滤并限制返回条数
  - 新建表时为 `md5_hash/phash` 建索引（仅对首次创建生效）

## 测试与部署
- 运行 `go test ./...` 验证后端编译通过
- 运行 `cd frontend && npm run build` 验证前端构建通过
