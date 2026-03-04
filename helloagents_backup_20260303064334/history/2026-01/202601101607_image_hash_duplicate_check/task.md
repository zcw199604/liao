# 任务清单: image_hash 媒体查重接口（MD5 + pHash）

目录: `helloagents/history/2026-01/202601101607_image_hash_duplicate_check/`

---

## 1. Backend（Go）
- [√] 1.1 在 `internal/app/image_hash.go` 中实现 `image_hash` 查询与 pHash 计算，覆盖 why.md#需求-上传文件查重-场景-md5-命中 与 why.md#需求-上传文件查重-场景-md5-未命中但相似度命中
- [√] 1.2 在 `internal/app/image_hash_handlers.go` 中实现 `POST /api/checkDuplicateMedia` 处理逻辑与输入校验，覆盖 why.md#需求-上传文件查重-场景-md5-未命中且无法计算-phash
- [√] 1.3 在 `internal/app/router.go` 注册路由，并在 `internal/app/app.go` 组装服务依赖
- [√] 1.4 在 `internal/app/schema.go` 增加 `image_hash` 表的 `CREATE TABLE IF NOT EXISTS` 兜底（不改写既有表结构）

## 2. Frontend（Vue）
- [√] 2.1 在 `frontend/src/api/media.ts` 增加 `checkDuplicateMedia` API 封装

## 3. 安全检查
- [√] 3.1 执行安全检查（输入校验、避免落盘/写库、返回结构不含敏感信息）

## 4. 文档更新（知识库 SSOT）
- [√] 4.1 更新 `helloagents/wiki/api.md`：补充 `/api/checkDuplicateMedia` 接口说明
- [√] 4.2 更新 `helloagents/wiki/data.md`：补充 `image_hash` 数据表说明
- [√] 4.3 更新 `helloagents/wiki/arch.md` 与 `helloagents/wiki/modules/media.md`：补充媒体查重能力与依赖
- [√] 4.4 更新 `helloagents/CHANGELOG.md` 与 `helloagents/history/index.md`

## 5. 质量验证
- [√] 5.1 执行 `go test ./...`（编译验证）
- [√] 5.2 执行 `cd frontend && npm run build`（前端构建验证）
