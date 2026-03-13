# 任务清单: mtphoto-folder-preview-favorites

目录: `helloagents/plan/202602230406_mtphoto-folder-preview-favorites/`

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
总任务: 32
已完成: 31
完成率: 96.9%
```

---

## 任务列表

### 1. 后端：mtPhoto 文件夹接口封装

- [√] 1.1 在 `internal/app/mtphoto_client.go` 新增文件夹模型（root/folder/folder file）与字段映射
  - 验证: 目录、文件字段可从上游 JSON 正确反序列化

- [√] 1.2 在 `internal/app/mtphoto_client.go` 新增 `GetFolderRoot`（调用 `/gateway/folders/root`）
  - 验证: `go test ./internal/app -run MtPhotoService.*FolderRoot`

- [√] 1.3 在 `internal/app/mtphoto_client.go` 新增 `GetFolderContent`（调用 `/gateway/foldersV2/{id}`）
  - 验证: `go test ./internal/app -run MtPhotoService.*FolderContent`

- [√] 1.4 在 `internal/app/mtphoto_client.go` 新增 `GetFolderBreadcrumbs`（调用 `/gateway/folderBreadcrumbs/{id}`）
  - 验证: `go test ./internal/app -run MtPhotoService.*FolderBreadcrumbs`

- [√] 1.5 在 `internal/app/mtphoto_client.go` 为 folder file 增加分页切片工具（page/pageSize/totalPages）
  - 验证: 超大 fileList 场景分页返回稳定

### 2. 后端：HTTP handler 与路由

- [√] 2.1 在 `internal/app/mtphoto_handlers.go` 新增 `handleGetMtPhotoFolderRoot`
  - 验证: 参数为空时也可正常返回 root 结构

- [√] 2.2 在 `internal/app/mtphoto_handlers.go` 新增 `handleGetMtPhotoFolderContent`（含参数校验与分页参数默认值）
  - 验证: folderId 非法返回 400，合法返回分页结构

- [√] 2.3 在 `internal/app/mtphoto_handlers.go` 新增 `handleGetMtPhotoFolderBreadcrumbs`
  - 验证: 返回 `path + folderList + fileList` 数据

- [√] 2.4 在 `internal/app/router.go` 注册新接口（3 个 GET）
  - 验证: 路由可访问，且仍受 JWT 中间件保护

- [√] 2.5 在文件夹 handler 中统一空目录/无权限/目录不存在错误映射
  - 验证: 上游 401/403/404 时前端可获得可区分错误信息

### 3. 后端：文件夹收藏数据层

- [√] 3.1 新增迁移 `sql/mysql/006_mtphoto_folder_favorite.sql`
  - 验证: MySQL 下迁移成功，可重复执行

- [√] 3.2 新增迁移 `sql/postgres/006_mtphoto_folder_favorite.sql`
  - 验证: PostgreSQL 下迁移成功，可重复执行

- [√] 3.3 新增 `internal/app/mtphoto_folder_favorite.go`（service + model）
  - 验证: 支持 list/upsert/remove，tags JSON 读写正确

- [√] 3.4 在 `internal/app/app.go` 注入 `mtphotoFolderFavorite` 服务实例
  - 验证: App 初始化通过

### 4. 后端：文件夹收藏接口

- [√] 4.1 新增 `internal/app/mtphoto_folder_favorite_handlers.go`，实现 `get/upsert/remove` handler
  - 验证: 请求体非法返回 400，服务错误返回 500

- [√] 4.2 在 `internal/app/router.go` 注册收藏接口（`GET /api/getMtPhotoFolderFavorites`、`POST /api/upsertMtPhotoFolderFavorite`、`POST /api/removeMtPhotoFolderFavorite`）
  - 验证: 接口可被 JWT 保护下正常访问

- [√] 4.3 对 tags 做统一规范化（trim、去重、空值过滤、数量上限）
  - 验证: 异常 tags 输入下返回结果稳定且可预期

- [√] 4.4 对备注 note 与标签长度做服务端校验（note<=500，tag<=32）
  - 验证: 超长输入返回 400 且错误文案明确

### 5. 前端：API 与 store 扩展

- [√] 5.1 在 `frontend/src/api/mtphoto.ts` 新增文件夹浏览接口封装
  - 验证: `frontend/src/__tests__/api-wrappers.test.ts` 覆盖新增 wrapper

- [√] 5.2 在 `frontend/src/api/mtphoto.ts` 新增文件夹收藏接口封装
  - 验证: wrapper 参数透传与路径正确

- [√] 5.3 扩展 `frontend/src/stores/mtphoto.ts`：新增文件夹模式状态（root/current/breadcrumb/files/pagination）
  - 验证: 可完成 root -> folder -> subfolder 导航

- [√] 5.4 扩展 `frontend/src/stores/mtphoto.ts`：新增收藏列表状态与 CRUD 动作
  - 验证: 收藏新增/编辑/删除后状态一致

- [√] 5.5 扩展 `frontend/src/stores/mtphoto.ts`：支持从收藏项直接跳转目录
  - 验证: 快捷跳转可定位到目标目录并加载图片

### 6. 前端：UI 交互实现

- [√] 6.1 修改 `frontend/src/components/media/MtPhotoAlbumModal.vue`，增加“相册/文件夹”模式切换
  - 验证: 现有相册模式功能不回归

- [√] 6.2 在文件夹模式实现目录列表展示与进入/返回交互
  - 验证: 路径切换准确，空目录有空态提示

- [√] 6.3 在文件夹模式复用 `InfiniteMediaGrid` 展示目录图片并接入预览
  - 验证: 图片可预览/左右切换/下载原图

- [√] 6.4 新增收藏入口与编辑面板（标签 chips + 备注）
  - 验证: 保存后 UI 即时更新

- [√] 6.5 收藏区展示标签与备注摘要，并支持一键跳转
  - 验证: 刷新后收藏数据可恢复

- [√] 6.6 收藏目录跳转失败时提供失效提示与移除入口（目录删除/无权限）
  - 验证: 失效收藏不会阻断其他目录浏览

### 7. 测试与回归

- [√] 7.1 新增后端 mtPhoto 文件夹 client 单测（`internal/app/mtphoto_client*_test.go`）
  - 验证: 401 重试、字段解析、分页切片分支均覆盖

- [√] 7.2 新增后端文件夹 handler 单测（`internal/app/mtphoto_handlers*_test.go`）
  - 验证: 参数校验、成功返回、异常返回覆盖

- [√] 7.3 新增后端文件夹收藏 service/handler 单测
  - 验证: upsert/remove/list 与 tags 规范化覆盖

- [-] 7.4 新增前端 store 与组件关键用例（`frontend/src/__tests__/stores-more.test.ts`、`settings-media-components.test.ts` 等）
  - 验证: 模式切换、目录导航、收藏跳转、标签备注编辑覆盖

- [√] 7.5 执行 `go test ./...` 与 `cd frontend && npm run build`
  - 验证: 两端门禁均通过

- [√] 7.6 新增异常回归：空目录、目录不存在、上游 401/403、收藏失效跳转
  - 验证: 错误分支具备稳定回归测试

### 8. 文档与知识库同步

- [√] 8.1 更新 `helloagents/modules/mtphoto.md`，补充文件夹浏览与收藏规则
  - 依赖: 1~6

- [√] 8.2 更新 `helloagents/modules/api.md`，补充新增 mtPhoto 文件夹/收藏接口
  - 依赖: 2、4

- [√] 8.3 更新 `helloagents/CHANGELOG.md` 记录该功能
  - 依赖: 7

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
