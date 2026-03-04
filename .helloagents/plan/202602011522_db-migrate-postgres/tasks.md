# 任务清单: db-migrate-postgres

目录: `helloagents/plan/202602011522_db-migrate-postgres/`

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
总任务: 19
已完成: 0
完成率: 0%
```

---

## 任务列表

### 1. 依赖与配置

- [ ] 1.1 更新依赖为 PostgreSQL driver（移除 MySQL driver）
  - 文件: `go.mod`, `go.sum`
  - 验证: `go test ./...`（至少通过编译）

- [ ] 1.2 调整默认环境变量示例为 PostgreSQL（并明确凭证策略）
  - 文件: `internal/config/config.go`
  - 验证: `go test ./internal/config`

### 2. DB_URL 解析与连接层（PostgreSQL）

- [ ] 2.1 将 `ParseJDBCMySQLURL` 重构为可解析 PostgreSQL 的实现（按 PG-only 策略同步更新测试）
  - 文件: `internal/config/config.go`, `internal/config/config_test.go`
  - 验证: 新增 `postgres://` / `jdbc:postgresql://` 用例；`go test ./internal/config`

- [ ] 2.2 改造 `openDB`：使用 PostgreSQL driver 打开连接，补齐 sslmode/时区处理，移除 MySQL DSN 拼接逻辑
  - 文件: `internal/app/app.go`, `internal/app/app_integration_test.go`
  - 验证: `go test ./internal/app -run TestOpenDB`

### 3. Schema 初始化与迁移（ensureSchema）

- [ ] 3.1 将 `ensureSchema` 的 DDL/migrations 改为 PostgreSQL 语法，并保证幂等（IF NOT EXISTS）
  - 文件: `internal/app/schema.go`
  - 验证: 本地连 PG 启动 `go run ./cmd/liao` 可自动建表并启动成功

- [ ] 3.2 明确时间字段与自增主键策略（timestamp/timestamptz、IDENTITY/BIGSERIAL）
  - 依赖: 3.1
  - 验证: 新旧关键接口的时间排序行为一致（补充测试/冒烟）

### 4. SQL 语句迁移（按模块收敛）

- [ ] 4.1 迁移 system_config：`INSERT IGNORE`/`ON DUPLICATE KEY` → `ON CONFLICT`，并替换占位符
  - 文件: `internal/app/system_config.go`, `internal/app/system_config_test.go`, `internal/app/app_integration_test.go`
  - 验证: `go test ./internal/app -run TestSystemConfig`

- [ ] 4.2 迁移 douyin_favorite：MySQL upsert → PostgreSQL `ON CONFLICT ... DO UPDATE`
  - 文件: `internal/app/douyin_favorite.go` 及其相关测试文件
  - 验证: `go test ./internal/app -run DouyinFavorite`

- [ ] 4.3 迁移 douyin_favorite_user_aweme：upsert + 动态 IN 占位符生成（`?` → `$n`）
  - 文件: `internal/app/douyin_favorite_user_aweme.go` 及其相关测试文件
  - 验证: 覆盖 `UpsertUserAwemes`/分页查询相关测试

- [ ] 4.4 迁移 douyin_favorite_tags：`INSERT IGNORE` → `ON CONFLICT DO NOTHING`；移除 `mysql.MySQLError` 依赖；需要自增 ID 时改 `RETURNING`
  - 文件: `internal/app/douyin_favorite_tags.go`, `internal/app/douyin_favorite_tags_test.go`, `internal/app/douyin_favorite_tag_handlers_test.go`, `internal/app/douyin_favorite_tag_handlers_more_test.go`
  - 验证: `go test ./internal/app -run FavoriteTag`

- [ ] 4.5 迁移 video_extract：MySQL upsert → PostgreSQL `ON CONFLICT`
  - 文件: `internal/app/video_extract.go` 及其相关测试文件
  - 验证: `go test ./internal/app -run VideoExtract`

- [ ] 4.6 迁移 media_upload + douyin_media_upload：`LastInsertId` → `RETURNING`；移除 UNION collation workaround；占位符替换
  - 文件: `internal/app/media_upload.go`, `internal/app/douyin_media_upload.go` 及其相关测试文件
  - 验证: `go test ./internal/app -run MediaUpload`

- [ ] 4.7 迁移 favorite：`LastInsertId` → `RETURNING`；占位符替换
  - 文件: `internal/app/favorite.go`, `internal/app/favorite_test.go`
  - 验证: `go test ./internal/app -run Favorite`

- [ ] 4.8 收尾：全局搜索并清理剩余 MySQL 方言与 `?` 占位符（含运行代码与测试）
  - 验证: `rg -n \"go-sql-driver/mysql|MySQLError|INSERT IGNORE|ON DUPLICATE|CONVERT\\(|COLLATE\" internal/` 无匹配（或仅保留文档说明）

### 5. MySQL 依赖清理

- [ ] 5.1 移除 MySQL 专用错误码分支与依赖引用（改由 `IF NOT EXISTS` / `ON CONFLICT` 解决）
  - 文件: `internal/app/schema.go` 等
  - 验证: `go test ./...`

### 6. 文档与知识库同步（SSOT）

- [ ] 6.1 更新运行文档：将 MySQL 描述与示例替换为 PostgreSQL（含 Docker 启动示例）
  - 文件: `README.md`, `CLAUDE.md`, `src/main/resources/application.yml`（仅作为 env 对照表时也需同步）
  - 验证: 文档中的命令可在本地跑通（最少：服务可启动）

- [ ] 6.2 同步更新 helloagents 知识库中的数据库描述（以代码为准）
  - 文件: `helloagents/context.md`, `helloagents/project.md`, `helloagents/modules/*.md`
  - 依赖: 代码迁移完成后执行

### 7. 数据迁移 Runbook（如已有存量 MySQL 数据）

- [ ] 7.1 编写并验证数据迁移与校验步骤（可重复执行）
  - 输出: `docs/db-migrate-mysql-to-postgres.md`（或写入知识库模块文档）
  - 验证: 在测试数据集跑通一次（含行数/抽样校验）

### 8. 端到端验收

- [ ] 8.1 本地冒烟：启动 PostgreSQL → 设置 env → `go run ./cmd/liao` → 覆盖核心 API 流程并记录结果
  - 验证: 启动日志无 SQL 错误；基础功能可用

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
