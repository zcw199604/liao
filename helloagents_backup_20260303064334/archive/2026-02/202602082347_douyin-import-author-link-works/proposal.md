# 变更提案: douyin-import-author-link-works

## 元信息
```yaml
类型: 新功能
方案类型: implementation
优先级: P1
状态: 已完成
创建: 2026-02-08
```

---

## 1. 需求

### 背景
当前“抖音导入到本地”链路能完成图片/视频落盘与媒体库展示，但本地媒体记录缺少稳定的作者快照字段。
导致在“所有上传图片 → 详情”中，虽然部分场景可通过上下文看到作品信息，但无法保证每条导入记录都能展示作者信息，也无法一键跳转到该作者的全部作品列表。

### 目标
- 抖音导入媒体在本地记录中保留作者信息（至少包含 `sec_user_id`、作者昵称、作者抖音号）。
- 全站媒体库接口返回抖音作者元信息，前端详情面板可稳定展示。
- 在图片/视频详情中点击作者后，直接打开“抖音下载”并进入该作者作品列表（用户作品页）。

### 约束条件
```yaml
时间约束: 无硬截止，按一个迭代完成
性能约束: /api/getAllUploadImages 查询不能出现明显退化（保持分页 + 索引可用）
兼容性约束: 保持现有接口字段兼容，仅追加可选字段
业务约束: 老数据允许字段为空，需保证前端降级显示与可点击策略
```

### 验收标准
- [√] `POST /api/douyin/import` 导入后的 `douyin_media_file` 能写入作者快照（新数据）。
- [√] `GET /api/getAllUploadImages?source=douyin` 返回项中包含抖音作者相关字段。
- [√] 全站媒体详情可展示作者昵称/抖音号/sec_user_id（有值即展示）。
- [√] 在详情中点击作者可直接打开抖音弹窗并加载该作者作品（失败时保留输入并给出错误提示）。
- [√] `go test ./...` 与前端相关单测/构建验证通过。

---

## 2. 方案

### 技术方案
采用“后端补齐作者元数据 + 前端事件直达作者作品”的组合方案：

1) **后端导入链路补齐作者快照**
- 扩展 `douyinCachedDetail`，增加 `AuthorUniqueID`、`AuthorName`。
- 在两个入口补齐缓存作者信息：
  - 用户作品流（`/api/douyin/account`、收藏用户作品预览）从现有字段透传。
  - 作品解析流（`/api/douyin/detail`）从上游 `author` 字段提取（best-effort）。
- `handleDouyinImport` 保存记录时将作者字段写入 `douyin_media_file`。

2) **数据模型与查询扩展（兼容追加）**
- 新增数据库迁移：MySQL/PostgreSQL 均为 `douyin_media_file` 增加
  - `author_unique_id`
  - `author_name`
- 扩展 `MediaFileDTO` 与 `GetAllUploadImagesWithDetailsBySource` 查询，使抖音来源项可返回：
  - `douyinSecUserId`
  - `douyinDetailId`
  - `douyinAuthorUniqueId`
  - `douyinAuthorName`
  - `source`（可选，便于前端识别来源）

3) **前端详情展示与“一键打开作者作品”**
- `mediaStore` 将后端抖音元信息映射到 `UploadedMedia.context.work`。
- `MediaDetailPanel` 将作者区域改为可点击动作（仅当 `authorSecUserId` 有值）。
- `MediaPreview` 接收作者点击事件并调用 `douyinStore.open(...)`。
- 扩展 `douyinStore` 打开参数，支持“指定初始模式=用户作品 + 自动执行拉取”，实现一键直达作者全部作品。

### 影响范围
```yaml
涉及模块:
  - internal/app/douyin_downloader.go: 作品详情作者信息提取
  - internal/app/douyin_handlers.go: 缓存结构扩展、导入保存参数扩展
  - internal/app/douyin_favorite_user_aweme_handlers.go: 收藏作品预览缓存补齐作者字段
  - internal/app/douyin_media_upload.go: 导入记录结构与写库字段扩展
  - internal/app/media_upload.go: 全站媒体分页 DTO 与 SQL 查询扩展
  - sql/mysql + sql/postgres: 新增 005 迁移
  - frontend/src/stores/media.ts: 抖音元信息映射到 context.work
  - frontend/src/components/media/MediaDetailPanel.vue: 作者可点击跳转
  - frontend/src/components/media/MediaPreview.vue: 处理详情面板事件并打开抖音作者作品
  - frontend/src/stores/douyin.ts + DouyinDownloadModal.vue: 支持“指定模式 + 自动拉取”
预计变更文件: 12~18
```

### 风险评估
| 风险 | 等级 | 应对 |
|------|------|------|
| 老数据无作者字段导致展示不完整 | 中 | 前端按“有值展示”降级，保留 sec_user_id 优先跳转能力 |
| SQL 联合查询改动引入兼容问题 | 中 | 保持原字段不变，仅追加字段；补齐 query 单测 |
| 弹窗自动拉取触发风控/缺 Cookie | 低 | 失败时保留输入并提示，用户可手动重试 |
| 去重复用路径下作者信息未更新 | 中 | 在 MD5 命中分支执行“非空作者字段回填更新”策略 |

---

## 3. 技术设计（可选）

### 架构设计
```mermaid
flowchart TD
    A[Douyin 解析/账号列表] --> B[douyinCachedDetail(含作者信息)]
    B --> C[/api/douyin/import]
    C --> D[douyin_media_file(作者快照)]
    D --> E[/api/getAllUploadImages]
    E --> F[mediaStore -> UploadedMedia.context.work]
    F --> G[MediaDetailPanel 作者点击]
    G --> H[douyinStore.open(用户作品+自动拉取)]
    H --> I[DouyinDownloadModal 用户作品列表]
```

### API设计
#### GET `/api/getAllUploadImages`
- **变更类型**: 向后兼容追加响应字段（仅抖音来源有值）
- **新增响应字段（item）**:
  - `source?: "local" | "douyin"`
  - `douyinSecUserId?: string`
  - `douyinDetailId?: string`
  - `douyinAuthorUniqueId?: string`
  - `douyinAuthorName?: string`

> 现有字段（`url/type/posterUrl/localFilename/...`）保持不变。

### 数据模型
| 字段 | 类型 | 说明 |
|------|------|------|
| author_unique_id | varchar(64) / varchar(64) | 抖音号（unique_id）快照 |
| author_name | varchar(128) / varchar(128) | 作者昵称快照 |

---

## 4. 核心场景

### 场景: 抖音导入后在媒体详情看到作者信息
**模块**: Media + Douyin Downloader
**条件**: 用户从抖音作品导入图片/视频到本地
**行为**: 导入接口保存作者快照 → 全站媒体接口返回作者字段 → 详情面板展示
**结果**: 用户可在详情中看到作者昵称/抖音号/sec_user_id（按字段可用性展示）

### 场景: 详情点击作者直接打开作者全部作品
**模块**: MediaPreview + DouyinDownloadModal
**条件**: 当前媒体 `context.work.authorSecUserId` 非空
**行为**: 点击作者 → 打开抖音弹窗（用户作品模式）→ 自动发起该 sec_user_id 的作品拉取
**结果**: 用户无需手动粘贴链接即可看到该作者作品列表

---

## 5. 技术决策

### douyin-import-author-link-works#D001: 作者信息以“导入时快照”写入 douyin_media_file
**日期**: 2026-02-08
**状态**: ✅采纳
**背景**: 详情页需要稳定显示作者信息，不能依赖运行时临时缓存。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 写入 `douyin_media_file` 作者字段 | 查询稳定、可审计、对详情页最直接 | 需要数据库迁移 |
| B: 每次详情动态回源/关联其它表 | 少改表结构 | 实时依赖高、老数据不稳定 |
**决策**: 选择方案 A
**理由**: 导入记录天然是“快照点”，最符合“导入作品报告作者信息”的业务语义。
**影响**: `sql/*` 迁移、导入保存逻辑、媒体库查询 DTO

### douyin-import-author-link-works#D002: 作者跳转采用“全局抖音弹窗直达用户作品模式”
**日期**: 2026-02-08
**状态**: ✅采纳
**背景**: 需求强调“点击即可打开作者全部作品”。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 打开现有 DouyinDownloadModal 并自动拉取 | 复用现有能力，改动收敛 | 需扩展 modal 打开参数 |
| B: 新建独立作者作品弹窗 | 体验可深度定制 | 重复建设，维护成本高 |
**决策**: 选择方案 A
**理由**: 已有用户作品拉取与分页能力，扩展入口参数即可满足需求。
**影响**: `douyinStore`、`DouyinDownloadModal`、`MediaPreview/MediaDetailPanel` 交互
