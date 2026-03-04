# 任务清单: db-dualstack-mysql-postgres

目录: `helloagents/archive/2026-02/202602011533_db-dualstack-mysql-postgres/`

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
总任务: 22
已完成: 21
完成率: 95%
```

---

## 任务列表

### 1. 数据库抽象层（internal/database）

- [√] 1.1 新增 Dialect 抽象：Name/DriverName/Rebind/ExpandIn/IsDuplicateKey/Upsert 生成能力
  - 文件: `internal/database/dialect.go`, `internal/database/dialect_mysql.go`, `internal/database/dialect_postgres.go`
  - 验证: `go test ./...`（至少编译通过）

- [√] 1.2 实现 `Rebind`（PG 将 `?` → `$1..$n`），并补齐边界测试（字符串字面量中的 ? 不替换）
  - 文件: `internal/database/rebind.go`, `internal/database/rebind_test.go`
  - 依赖: 1.1
  - 验证: `go test ./internal/database -run Rebind`

- [√] 1.3 实现 `ExpandIn`：将 `IN (?)` + slice 参数展开为多个 `?`，再交由 Rebind
  - 文件: `internal/database/in.go`, `internal/database/in_test.go`
  - 依赖: 1.2
  - 验证: `go test ./internal/database -run ExpandIn`

- [√] 1.4 封装 DB 执行入口：统一在 Query/Exec 前执行 Rebind（避免业务层散落调用）
  - 文件: `internal/database/db.go`
  - 依赖: 1.1
  - 验证: 业务层替换后 `go test ./...` 通过

### 2. 配置解析与连接（选择 MySQL/PG）

- [√] 2.1 设计并实现 DB 选择策略：仅以 `DB_URL` scheme 决定数据库类型（mysql/postgres/postgresql；支持 `jdbc:` 前缀）
  - 文件: `internal/config/config.go`, `internal/config/config_test.go`
  - 验证: `go test ./internal/config`

- [√] 2.2 改造 `openDB` 支持双驱动：MySQL 使用现有 DSN 拼接；PostgreSQL 使用 pgx/std 并处理 sslmode/时区
  - 文件: `internal/app/app.go`, `internal/app/app_integration_test.go`
  - 依赖: 2.1
  - 验证: `go test ./internal/app -run TestOpenDB`

### 3. 迁移框架（双套 schema 脚本 + 版本表）

- [√] 3.1 引入迁移机制：创建 `schema_migrations` 表，并按版本顺序执行 `sql/{dialect}/*.sql`
  - 文件: `internal/database/migrator.go`, `internal/database/migrator_test.go`
  - 验证: sqlmock 单测 + 本地双库冒烟

- [√] 3.2 抽取现有 schema：将 `internal/app/schema.go` 里的建表/迁移改为脚本形式，分别落到 `sql/mysql/` 与 `sql/postgres/`
  - 文件: `sql/mysql/001_init.sql`, `sql/postgres/001_init.sql`（必要时拆分更多版本）
  - 依赖: 3.1
  - 验证: 空库启动能自动建表；两库表结构对齐（抽样比对）

- [√] 3.3 `ensureSchema` 改造为调用 Migrator（按 Dialect 路由），并移除 MySQL 错误码分支依赖
  - 文件: `internal/app/schema.go`, `Dockerfile`
  - 依赖: 3.2
  - 验证: `go test ./internal/app` + 双库启动

### 4. SQL 方言差异集中封装（Upsert/InsertIgnore/Returning）

- [√] 4.1 实现 `InsertReturningID` / `ExecInsertIgnore` / `ExecUpsert`（业务层禁用直接 LastInsertId/手写方言）
  - 文件: `internal/database/crud.go`, `internal/database/crud_test.go`
  - 依赖: 1.1, 2.2
  - 验证: 单测覆盖 MySQL/PG 两种生成语句

### 5. 逐模块迁移（把 MySQL 特性从业务层移走）

- [√] 5.1 system_config：`INSERT IGNORE`/`ON DUPLICATE` 改为调用封装；SQL 统一 `?` + Rebind
  - 文件: `internal/app/system_config.go`, `internal/app/system_config_test.go`, `internal/app/app_integration_test.go`
  - 验证: `go test ./internal/app -run SystemConfig`

- [√] 5.2 douyin_favorite / douyin_favorite_user_aweme：Upsert + 动态 IN 全量走 Dialect 工具
  - 文件: `internal/app/douyin_favorite.go`, `internal/app/douyin_favorite_user_aweme.go` 及其测试
  - 验证: `go test ./internal/app -run DouyinFavorite`

- [√] 5.3 douyin_favorite_tags：`INSERT IGNORE` + 自增 id 获取（MySQL=LastInsertId，PG=RETURNING）
  - 文件: `internal/app/douyin_favorite_tags.go` 及其测试/handler 测试
  - 验证: `go test ./internal/app -run FavoriteTag`

- [√] 5.4 media_upload：移除 MySQL collation workaround（`CONVERT ... COLLATE`），为 MySQL/PG 各提供兼容查询（通过 Dialect 路由）
  - 文件: `internal/app/media_upload.go` 及其测试
  - 验证: `GetAllUploadImagesWithDetailsBySource(source=all)` 两库结果一致（抽样对比）

- [√] 5.5 favorite + 其他插入：统一改为 `InsertReturningID`，避免直接 `LastInsertId`
  - 文件: `internal/app/favorite.go`, `internal/app/douyin_media_upload.go`, `internal/app/media_upload.go` 等
  - 验证: `rg -n \"LastInsertId\\(\" internal/` 仅允许出现在封装层

- [√] 5.6 video_extract：Upsert 语句走封装；占位符统一
  - 文件: `internal/app/video_extract.go` 及其测试
  - 验证: `go test ./internal/app -run VideoExtract`

### 6. 一致性规则落地（切换不受影响）

- [√] 6.1 明确并实现“大小写策略”：对用户可输入的标签名（tag.name）按“大小写不敏感”处理，避免 MySQL/PG 行为差异
  - 文件: `sql/postgres/003_tag_name_ci_unique.sql`, `sql/mysql/003_tag_name_ci_unique.sql`
  - 说明: PostgreSQL 通过 `UNIQUE INDEX (LOWER(name))` 对齐 MySQL 常见 collation 行为
  - 验证: 新增迁移脚本 + 业务侧 duplicate key -> `标签已存在` 行为保持一致

- [√] 6.2 统一连接参数的“兼容与兜底”：PostgreSQL 连接自动过滤 MySQL-only 参数，并将 `serverTimezone` 映射为 `timezone`（同时兼容 `useSSL`→`sslmode`）
  - 文件: `internal/app/app.go`, `internal/app/app_integration_test.go`
  - 验证: `go test ./internal/app -run TestOpenDB`

### 7. 测试与 CI（双库闸门）

- [√] 7.1 调整 sqlmock 断言：同一套 sqlmock 测试可在 MySQL/PG 方言模式下复用（不被 `$n` 占位符影响）
  - 文件: `internal/app/test_helpers_test.go`（`placeholderNormalizingMatcher` + `TEST_DB_DIALECT`）
  - 验证: `go test ./...` + `TEST_DB_DIALECT=postgres go test ./...`

- [ ] 7.2 新增集成测试入口（可选 tag）：在 MySQL 与 PG 两个容器下分别跑同一套冒烟用例
  - 输出: `tests/integration/`（或 `internal/app/*_integration_test.go`）
  - 验证: 本地可执行两次：`TEST_DB=mysql ...` 与 `TEST_DB=postgres ...`

### 8. 文档与运维切换 Runbook

- [√] 8.1 更新文档：README/CLAUDE/知识库补充双栈配置、切换方式、注意事项（时区/错误码等）
  - 文件: `README.md`, `CLAUDE.md`, `helloagents/context.md`, `helloagents/modules/db.md`
  - 验证: 文档中的命令与 env 示例可跑通

- [√] 8.2 提供 MySQL→PostgreSQL 数据迁移与回滚/切换 runbook（含校验清单）
  - 输出: `helloagents/modules/db.md`
  - 验证: 在测试数据集跑通一次迁移与校验流程

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 3.x | [√] | 已落地 `schema_migrations` + `sql/mysql|postgres` 版本化迁移脚本，并将 `ensureSchema` 改为调用 `internal/database/migrator.go` |
| 4.1 | [√] | 已实现通用 `InsertReturningID`/`ExecInsertIgnore`/`ExecUpsert`，并在多个业务点位收敛 Upsert（减少业务层方言分支） |
| 7.1 | [√] | sqlmock 使用 `placeholderNormalizingMatcher` 归一化占位符；通过 `TEST_DB_DIALECT=postgres` 覆盖 PG 语义分支（无需真实 Postgres 实例） |
| 8.1 | [√] | 已更新 `README.md`/`CLAUDE.md`/`helloagents/context.md`，并新增 `helloagents/modules/db.md` 作为 DB 双栈与切换 Runbook（SSOT） |
| bun | [√] | 用户选择“沿用（方案2）”：回到 `database/sql` + `internal/database`（保留参数绑定，避免 `bun` 内联参数导致 `sqlmock.WithArgs` 失效）；`go test ./...` 已通过 |
