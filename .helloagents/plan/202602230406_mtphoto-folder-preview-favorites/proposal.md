# 变更提案: mtphoto-folder-preview-favorites

## 元信息
```yaml
类型: 新功能
方案类型: implementation
优先级: P1
状态: 草稿
创建: 2026-02-23
```

---

## 1. 需求

### 背景
当前 mtPhoto 能力仅支持“相册维度”浏览（`/api-album` + `/api-album/filesV2/{id}`），无法按 mtPhoto 文件夹树逐级进入浏览，导致用户在按目录管理素材时无法直接定位目标图片。

同时，当前收藏能力仅覆盖 mtPhoto 上游“收藏夹相册（albumId=1）”，不支持“本地定义的文件夹快捷收藏”。用户希望把常用文件夹收藏起来并附加标签、备注，后续可直接跳转定位。

### 目标
- 增加 mtPhoto 文件夹预览能力：支持根目录、子目录逐级进入、查看文件夹内全部图片。
- 增加文件夹级收藏能力：可收藏某个 mtPhoto 文件夹并快速跳转。
- 收藏项支持维护标签与备注，便于二次检索与人工记忆。
- 保持现有 mtPhoto 相册功能可用，避免破坏既有导入与预览链路。

### 约束条件
```yaml
时间约束: 本期以“可用闭环”优先（浏览 + 收藏 + 标签/备注），不扩展复杂检索系统
性能约束: 大文件夹浏览不得一次性阻塞渲染，需保留分页/懒加载体验
兼容性约束: 不改变现有 mtPhoto 相册接口行为；新增接口与状态为向后兼容
业务约束: 收藏元数据以本地数据库为准（不写回 mtPhoto 上游）
安全约束: 新增收藏接口保持 JWT 保护，并对 tags/note 做长度与非法值校验
```

### 验收标准
- [ ] 可通过新入口浏览 mtPhoto 文件夹树（根目录 -> 子目录），并能看到目录内图片列表。
- [ ] 目录图片可复用现有预览能力（缩略图/下载原图/切换浏览）。
- [ ] 支持“收藏当前文件夹”，收藏后可在收藏区一键跳转到该文件夹。
- [ ] 收藏项可编辑标签与备注，刷新页面后数据可保留。
- [ ] `go test ./...`、`cd frontend && npm run build` 通过。

---

## 2. 方案

### 技术方案
#### 2.1 后端：mtPhoto 文件夹接口封装
在 `MtPhotoService` 中新增文件夹链路封装（复用现有登录/续期机制）：
- 根目录：`GET /gateway/folders/root`
- 指定目录内容：`GET /gateway/foldersV2/{folderId}`
- 目录路径信息：`GET /gateway/folderBreadcrumbs/{folderId}`

新增本地 API（命名保持现有风格）：
- `GET /api/getMtPhotoFolderRoot`
- `GET /api/getMtPhotoFolderContent?folderId=...&page=...&pageSize=...`
- `GET /api/getMtPhotoFolderBreadcrumbs?folderId=...`

其中 `getMtPhotoFolderContent` 对上游 `fileList` 做后端分页切片，避免前端一次性渲染大列表。
响应模型保留上游关键字段（`fileName/fileType/size/tokenAt/width/height/duration/status/MD5`），确保前端预览、下载命名与详情展示一致。

#### 2.2 后端：文件夹收藏（含标签/备注）
新增本地收藏表（双方言迁移）：
- `sql/mysql/006_mtphoto_folder_favorite.sql`
- `sql/postgres/006_mtphoto_folder_favorite.sql`

建议结构：
- `folder_id`（唯一）
- `folder_name`
- `folder_path`
- `cover_md5`（可空）
- `tags_json`（JSON 字符串，默认 `[]`）
- `note`（备注）
- `created_at` / `updated_at`

索引建议：
- `UNIQUE KEY uk_mtphoto_folder_id (folder_id)`
- `INDEX idx_mtphoto_folder_updated_at (updated_at DESC)`

新增服务与接口：
- `GET /api/getMtPhotoFolderFavorites`
- `POST /api/upsertMtPhotoFolderFavorite`
- `POST /api/removeMtPhotoFolderFavorite`

`upsert` 语义：同 `folder_id` 重复提交时更新基础信息、标签、备注。

#### 2.3 前端：mtPhoto 弹窗新增“文件夹模式”
在现有 `MtPhotoAlbumModal.vue` 内增加模式切换（相册 / 文件夹）：
- 相册模式：保持现状。
- 文件夹模式：
  - 顶部显示当前路径与返回逻辑。
  - 主区展示子文件夹列表 + 当前目录图片网格。
  - 图片项复用现有 `MediaPreview` 能力。

收藏交互：
- 当前目录支持“收藏/取消收藏”。
- 收藏列表固定展示在文件夹模式侧栏/头部区域，支持一键跳转。
- 收藏编辑面板支持标签 chips 与备注 textarea。

#### 2.4 数据与状态管理
扩展 `frontend/src/stores/mtphoto.ts`：
- 新增文件夹节点、文件条目、收藏项类型。
- 新增文件夹导航状态（当前 folderId、path、breadcrumb、分页状态）。
- 新增收藏状态（列表、编辑、保存中）。

扩展 `frontend/src/api/mtphoto.ts`：新增文件夹与收藏接口封装。

### 影响范围
```yaml
涉及模块:
  - internal/app/mtphoto_client.go: 新增 gateway folders 接口模型与请求方法
  - internal/app/mtphoto_handlers.go: 新增文件夹浏览/收藏接口 handler
  - internal/app/router.go: 注册新增 mtPhoto 文件夹接口
  - internal/app: 新增 mtphoto_folder_favorite 服务与测试文件
  - sql/mysql + sql/postgres: 新增 006 迁移脚本
  - frontend/src/api/mtphoto.ts: 新增文件夹/收藏 API
  - frontend/src/stores/mtphoto.ts: 新增文件夹浏览与收藏状态
  - frontend/src/components/media/MtPhotoAlbumModal.vue: 新增文件夹模式与收藏 UI
预计变更文件: 14~20
```

### 风险评估
| 风险 | 等级 | 应对 |
|------|------|------|
| 上游目录接口字段不稳定（`cover/s_cover/path` 可能为空） | 中 | 模型字段全部 nullable 处理，前端提供空态兜底 |
| 大目录文件过多导致前端卡顿 | 高 | 后端分页切片 + 前端无限滚动，避免一次渲染全部 |
| 收藏标签数据格式混乱（空白/重复） | 中 | 后端统一 trim + 去重 + 上限控制（如 20 个） |
| 收藏路径过期（上游目录被删除或改名） | 中 | 跳转失败时提示并支持从收藏列表移除/刷新元数据 |
| 新增 DB 表后不同方言行为不一致 | 中 | mysql/postgres 同步迁移 + sqlmock/集成测试覆盖 |
| 上游返回空目录或无权限目录（401/403） | 中 | 明确错误码映射 + 前端空态/错误态 + 续期后单次重试 |

---

## 3. 技术设计

### 架构设计
```mermaid
flowchart TD
    FE[MtPhotoAlbumModal 文件夹模式] --> API1[/api/getMtPhotoFolderRoot]
    FE --> API2[/api/getMtPhotoFolderContent]
    FE --> API3[/api/getMtPhotoFolderBreadcrumbs]
    FE --> API4[/api/getMtPhotoFolderFavorites/upsert/remove]

    API1 --> MTP[MtPhotoService]
    API2 --> MTP
    API3 --> MTP

    MTP --> UPSTREAM[mtPhoto gateway folders API]

    API4 --> DB[(mtphoto_folder_favorite)]
```

### API设计
#### GET /api/getMtPhotoFolderRoot
- **请求**: 无
- **响应**:
```json
{
  "path": "",
  "folderList": [{"id": 518, "name": "photo", "path": "/photo", "cover": "...", "s_cover": "...", "subFolderNum": 16, "subFileNum": 0}],
  "fileList": []
}
```

#### GET /api/getMtPhotoFolderContent
- **请求**: `folderId`(必填), `page`(默认1), `pageSize`(默认60)
- **响应**:
```json
{
  "path": "/photo/我的照片",
  "folderList": [],
  "fileList": [
    {
      "id": 131967,
      "fileName": "faceu_0_20190929232934188.jpg",
      "fileType": "JPEG",
      "size": "1242936",
      "tokenAt": "2024-09-16T18:06:47.000Z",
      "md5": "600e0556a5bd9d03d84ddae23bce66de",
      "width": 1080,
      "height": 1440,
      "duration": null,
      "status": 2
    }
  ],
  "total": 257,
  "page": 1,
  "pageSize": 60,
  "totalPages": 5
}
```

#### GET /api/getMtPhotoFolderFavorites
- **请求**: 无
- **响应**:
```json
{
  "items": [
    {
      "id": 1,
      "folderId": 644,
      "folderName": "我的照片",
      "folderPath": "/photo/我的照片",
      "coverMd5": "e38c3a4e832e7e66538002287d9663b5",
      "tags": ["人像", "常用"],
      "note": "每周更新",
      "updateTime": "2026-02-23T04:06:00+08:00"
    }
  ]
}
```

#### POST /api/upsertMtPhotoFolderFavorite
- **请求**:
```json
{
  "folderId": 644,
  "folderName": "我的照片",
  "folderPath": "/photo/我的照片",
  "coverMd5": "e38c...",
  "tags": ["人像", "常用"],
  "note": "每周更新"
}
```
- **响应**: `{"success": true, "item": {...}}`
- **校验规则**:
  - `folderId > 0`
  - `tags` 规范化后数量上限 20、单标签长度上限 32
  - `note` 长度上限 500

#### POST /api/removeMtPhotoFolderFavorite
- **请求**: `{"folderId": 644}`
- **响应**: `{"success": true}`

### 数据模型
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| folder_id | bigint | mtPhoto 文件夹 ID（唯一） |
| folder_name | varchar(255) | 文件夹名称 |
| folder_path | varchar(1024) | 文件夹路径（用于展示和兜底跳转） |
| cover_md5 | varchar(32) | 收藏卡片封面 md5（可空） |
| tags_json | text | 标签数组 JSON（如 `["常用","人像"]`） |
| note | text | 备注 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

约束/索引：
- `UNIQUE(folder_id)`
- `INDEX(updated_at DESC)`

---

## 4. 核心场景

### 场景: 浏览 mtPhoto 文件夹树
**模块**: `frontend/src/components/media/MtPhotoAlbumModal.vue` + `internal/app/mtphoto_client.go`
**条件**: mtPhoto 配置可用，用户已登录本系统
**行为**: 打开 mtPhoto 弹窗 -> 切换到文件夹模式 -> 进入根目录/子目录逐级浏览
**结果**: 可看到子文件夹与目录图片，并可继续深入或返回上级

### 场景: 收藏目录并快速跳转
**模块**: `frontend/src/stores/mtphoto.ts` + `internal/app/mtphoto_folder_favorite*.go`
**条件**: 当前已进入某个具体目录
**行为**: 点击收藏 -> 填写标签与备注 -> 保存 -> 在收藏列表点击该项
**结果**: 目录被持久化保存，后续可直接跳回该目录

### 场景: 编辑收藏标签与备注
**模块**: `frontend/src/components/media/MtPhotoAlbumModal.vue` + 收藏 API
**条件**: 已存在收藏目录
**行为**: 打开编辑面板 -> 修改 tags/note -> 提交 upsert
**结果**: 收藏元信息更新，刷新后仍保留

---

## 5. 技术决策

### mtphoto-folder-preview-favorites#D001: 收藏元数据存储位置
**日期**: 2026-02-23
**状态**: ✅采纳
**背景**: 需为目录收藏保存标签与备注，且要可跨会话持久化。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 独立数据库表（本地） | 持久化稳定，可查询排序，便于后续扩展 | 需要新增迁移和接口 |
| B: system_config 大 JSON | 开发快、表结构零新增 | 并发更新冲突大，查询维护成本高 |
| C: 前端 localStorage | 零后端成本 | 无法跨设备/会话共享，数据不可控 |
**决策**: 选择方案 A
**理由**: 该功能属于长期可运营能力，需稳定存储与可维护查询。
**影响**: `sql/*` 迁移、后端新增收藏服务、前端新增收藏 API。

### mtphoto-folder-preview-favorites#D002: 文件夹内容分页策略
**日期**: 2026-02-23
**状态**: ✅采纳
**背景**: 上游 `foldersV2/{id}` 可能返回超大 `fileList`，前端直接全量渲染风险高。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 后端分页切片（推荐） | 与现有相册分页一致，前端平滑接入 | 后端多一层分页逻辑 |
| B: 前端全量拉取后本地切片 | 实现快 | 首次 payload 大，移动端易卡顿 |
**决策**: 选择方案 A
**理由**: 与既有 mtPhoto 相册体验一致，性能风险更低。
**影响**: `getMtPhotoFolderContent` 响应新增 `total/page/totalPages` 字段。

### mtphoto-folder-preview-favorites#D003: 收藏标签模型
**日期**: 2026-02-23
**状态**: ✅采纳
**背景**: 标签需求是“给收藏目录打标”，但当前未要求全局标签管理。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 每条收藏内嵌 tags_json（推荐） | 实现简单，满足当前需求 | 跨收藏标签统计较弱 |
| B: 独立标签表 + 映射表 | 扩展性高 | 复杂度高，超出当前需求 |
**决策**: 选择方案 A
**理由**: 以最小可用闭环优先，后续如需全局标签管理再演进。
**影响**: 后端需做标签规范化（trim/去重/上限）。

### mtphoto-folder-preview-favorites#D004: 收藏作用域
**日期**: 2026-02-23
**状态**: ✅采纳
**背景**: 当前系统中的 mtPhoto/抖音素材能力以“全局共享库”思路为主（非强隔离多租户），需决定文件夹收藏是否按用户隔离。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 全局收藏（推荐） | 与现有素材库一致，结构简单，便于共享常用目录 | 不能做“个人私有收藏”隔离 |
| B: 按 user_id 隔离 | 个性化更强 | 需引入 user_id 口径，增加迁移与查询复杂度 |
**决策**: 选择方案 A
**理由**: 与现有系统形态一致，满足当前“快速跳转”目标，后续如需私有化可再演进。
**影响**: 收藏表唯一键使用 `folder_id`，不引入 `user_id` 维度。
