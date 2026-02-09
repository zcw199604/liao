# 任务清单: douyin-import-author-link-works

目录: `helloagents/archive/2026-02/202602082347_douyin-import-author-link-works/`

---

## 任务状态符号说明

| 符号 | 状态 | 说明 |
|------|------|------|
| `[ ]` | pending | 待执行 |
| `[√]` | completed | 已完成 |
| `[X]` | failed | 执行失败 |
| `[-]` | skipped | 已跳过 |
| `[?]` | uncertain | 待确认 |

---

## 执行状态
```yaml
总任务: 18
已完成: 18
完成率: 100%
```

---

## 任务列表

### 1. 数据层与导入写库

- [√] 1.1 为 `douyin_media_file` 增加作者快照字段（MySQL + PostgreSQL）
  - 文件: `sql/mysql/005_douyin_media_author_snapshot.sql`, `sql/postgres/005_douyin_media_author_snapshot.sql`
  - 验证: 迁移脚本按 migrator 幂等策略编写（PostgreSQL `IF NOT EXISTS`；MySQL 依赖 duplicate-column 容错）

- [√] 1.2 扩展导入记录结构 `DouyinUploadRecord` 增加作者字段
  - 文件: `internal/app/douyin_media_upload.go`
  - 验证: 编译通过；插入参数与 SQL 列顺序一致

- [√] 1.3 更新 `SaveDouyinUploadRecord` 插入语句与去重分支回填策略
  - 文件: `internal/app/douyin_media_upload.go`
  - 验证: MD5 命中时 `update_time` 更新且作者字段可非空回填

### 2. 抖音解析缓存作者信息贯通

- [√] 2.1 扩展 `douyinCachedDetail` 增加 `AuthorUniqueID/AuthorName`
  - 文件: `internal/app/douyin_downloader.go`
  - 验证: `FetchDetail` 返回结构含作者字段（best-effort）

- [√] 2.2 在作品详情解析中提取作者昵称/抖音号
  - 文件: `internal/app/douyin_downloader.go`
  - 验证: 上游含 author 时可解析；缺失时不报错

- [√] 2.3 在账号作品与收藏用户作品预览缓存中写入作者字段
  - 文件: `internal/app/douyin_handlers.go`, `internal/app/douyin_favorite_user_aweme_handlers.go`
  - 验证: account/favorite 路径导入时缓存作者字段可用

- [√] 2.4 导入接口保存作者字段到 `douyin_media_file`
  - 文件: `internal/app/douyin_handlers.go`
  - 验证: `POST /api/douyin/import` 后数据库记录含作者字段

### 3. 全站媒体库接口返回作者元信息

- [√] 3.1 扩展 `MediaFileDTO` 增加抖音元信息字段（兼容追加）
  - 文件: `internal/app/media_upload.go`
  - 验证: 原有字段保持不变；新增字段 `omitempty`

- [√] 3.2 改造 `GetAllUploadImagesWithDetailsBySource` 查询（local/douyin/all）
  - 文件: `internal/app/media_upload.go`
  - 验证: `source=douyin` 返回作者字段；`source=all` 联合查询字段对齐

- [√] 3.3 确认 `GetAllUploadImagesCountBySource` 行为不变
  - 文件: `internal/app/media_upload.go`
  - 验证: 计数与现网逻辑一致（不受新增字段影响）

### 4. 前端详情展示与作者跳转

- [√] 4.1 扩展前端类型与媒体 store 映射抖音作者元信息
  - 文件: `frontend/src/types/media.ts`, `frontend/src/stores/media.ts`
  - 验证: allUpload 列表项生成 `context.work`（仅抖音来源）

- [√] 4.2 详情面板作者区改为可点击动作（触发“查看作者全部作品”）
  - 文件: `frontend/src/components/media/MediaDetailPanel.vue`
  - 验证: 有 `authorSecUserId` 时显示可点；无值时仅展示文本

- [√] 4.3 预览组件接收作者点击事件并触发抖音弹窗打开
  - 文件: `frontend/src/components/media/MediaPreview.vue`
  - 验证: 点击后关闭详情面板并调用 `douyinStore.open(...)`

- [√] 4.4 扩展 `douyinStore` 打开参数（目标模式 + 自动执行）
  - 文件: `frontend/src/stores/douyin.ts`
  - 验证: 可传入“用户作品模式 + 自动拉取”意图

- [√] 4.5 在 `DouyinDownloadModal` 支持入口参数自动切换到用户作品并拉取
  - 文件: `frontend/src/components/media/DouyinDownloadModal.vue`
  - 验证: 通过 `sec_user_id` 一步进入作者作品列表

### 5. 测试与验收

- [√] 5.1 更新后端单测覆盖新增列与查询字段
  - 文件: `internal/app/douyin_import_test.go`, `internal/app/media_upload_test.go`, `internal/app/media_history_handlers_test.go`（按实际受影响补齐）
  - 验证: `go test ./...`

- [√] 5.2 更新前端单测覆盖 store 映射与作者点击跳转
  - 文件: `frontend/src/__tests__/stores-more.test.ts`, `frontend/src/__tests__/media-preview-more.test.ts`（按实际受影响补齐）
  - 验证: `cd frontend && npm run build`（必要时补跑对应测试）

- [√] 5.3 文档与知识库同步（开发实施完成后）
  - 文件: `helloagents/modules/media.md`, `helloagents/modules/douyin-downloader.md`, `helloagents/CHANGELOG.md`
  - 验证: 文档字段与代码返回一致（以代码为准）

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 1.1~3.3 | [√] | 后端作者快照链路完成：迁移、导入写库、查询 DTO 与 SQL 兼容追加全部落地 |
| 4.1~4.5 | [√] | 前端实现详情作者点击跳转：MediaDetailPanel → MediaPreview → douyinStore.open(account+autoFetch) → DouyinDownloadModal 自动获取作品 |
| 5.1 | [√] | `go test ./internal/app/...`、`go test ./...` 均通过 |
| 5.2 | [√] | `cd frontend && npm test`（72 文件/663 用例）与 `npm run build` 均通过 |
| 5.3 | [√] | 已同步 `helloagents/modules/media.md`、`helloagents/modules/douyin-downloader.md`、`helloagents/CHANGELOG.md` |
