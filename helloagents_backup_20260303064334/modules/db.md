# 数据库双栈与切换 Runbook（MySQL / PostgreSQL）

> 目标：迁移后同时支持 MySQL + PostgreSQL（双栈），并且仅通过 `DB_URL` 的 scheme 切换数据库类型，不引入额外开关。

## 1. 关键设计与入口（SSOT）

- **数据库选择**：仅通过 `DB_URL` 的 scheme 选择数据库类型（`mysql` / `postgres` / `postgresql`），并兼容 `jdbc:` 前缀。
  - 代码：`internal/config/config.go`（`ParseJDBCURL`）、`internal/database/dialect.go`（`DialectFromScheme`）、`internal/app/app.go`（`openDB`）。
- **迁移机制（版本化 SQL）**：服务启动时执行 `schema_migrations` + `sql/{dialect}/*.sql`。
  - 入口：`internal/app/schema.go`
  - 执行器：`internal/database/migrator.go`
  - 脚本目录：`sql/mysql/`、`sql/postgres/`
- **方言差异集中**：占位符、InsertIgnore/Upsert/Returning 等差异集中在 `internal/database/*`，业务层保持 `?` 占位符写法。

## 2. `DB_URL` 配置示例（仅改 scheme 切换）

### 2.1 MySQL

```bash
export DB_URL='jdbc:mysql://127.0.0.1:3306/hot_img?serverTimezone=Asia/Shanghai&useSSL=false'
export DB_USERNAME='root'
export DB_PASSWORD='******'
```

### 2.2 PostgreSQL

```bash
export DB_URL='postgres://127.0.0.1:5432/hot_img?sslmode=disable&timezone=Asia/Shanghai'
export DB_USERNAME='postgres'
export DB_PASSWORD='******'
```

兼容与兜底行为（由 `internal/app/app.go` 实现）：
- 如果 `DB_URL` 里带了常见 MySQL-only query 参数（如 `useSSL/serverTimezone/characterEncoding/allowPublicKeyRetrieval`），PostgreSQL 连接会自动过滤它们，避免 `unrecognized configuration parameter`。
- 若未显式提供 `sslmode` 且存在 `useSSL=true/false`，会 best-effort 映射为 `sslmode=require/disable`。
- 若未显式提供 `timezone` 且存在 `serverTimezone=Asia/Shanghai`，会 best-effort 映射为 `timezone=Asia/Shanghai`。

## 3. 切换流程（不迁移数据，仅切运行库）

适用于：你准备好了目标数据库（为空库或已有同结构数据），只需要让服务切到另一库跑。

1) 准备目标库（例如 `hot_img`）
- MySQL：创建库并授予账号权限
- PostgreSQL：创建库并授予账号权限

2) 修改环境变量
- 仅修改 `DB_URL` 的 scheme（以及必要时 host/port/dbname），其他变量保持不变（如 `DB_USERNAME/DB_PASSWORD`）。

3) 启动服务并观察迁移
- 启动：`go run ./cmd/liao`
- 预期：启动阶段会自动创建 `schema_migrations` 并执行 `sql/{dialect}/*.sql`

4) 验证（建议抽样）
- 访问关键 API（鉴权、identity、media、douyin tags 等）
- 观察日志中是否有 “数据库迁移失败” 或 SQL 错误

## 4. MySQL → PostgreSQL 数据迁移（双栈切换的常见路径）

### 4.1 推荐顺序

1) 创建一个**空的** PostgreSQL 数据库
2) 先用服务启动一次，让 PostgreSQL 自动跑完迁移建表
3) 再执行数据迁移（导入表数据）
4) 最后启动服务并做验收

理由：先建表可以保证类型/索引/约束与项目预期一致，减少迁移工具“自行推断建表”导致的偏差。

### 4.2 迁移工具选择（示例）

- `pgloader`（常用）：可从 MySQL 直接搬到 PostgreSQL（包含类型映射与批量写入）
- `mysqldump` + 自定义转换：适合数据量小或可控场景

本项目不内置迁移工具脚本；建议在迁移前先做一次全库备份，并在测试环境完整跑通流程。

### 4.3 迁移前必查：大小写不敏感唯一性（标签名）

为对齐 MySQL 常见 collation 行为，PostgreSQL 侧对以下表的 `name` 增加了大小写不敏感唯一索引：
- `douyin_favorite_user_tag`：`UNIQUE (LOWER(name))`
- `douyin_favorite_aweme_tag`：`UNIQUE (LOWER(name))`

迁移前请确保 MySQL 数据里不存在仅大小写不同的重复值（例如 `Food` 与 `food`），否则导入 PostgreSQL 时会触发 `unique_violation(23505)`。

对应迁移文件：`sql/postgres/003_tag_name_ci_unique.sql`

## 5. 校验清单（切库/迁移后）

### 5.1 表结构与迁移状态

- `schema_migrations` 是否存在且有记录：
  - `SELECT * FROM schema_migrations ORDER BY applied_at DESC;`
- 关键表是否存在（抽样）：
  - `identity`, `media_file`, `media_send_log`
  - `douyin_favorite_user`, `douyin_favorite_aweme`
  - `douyin_favorite_user_tag`, `douyin_favorite_aweme_tag`

### 5.2 数据抽样一致性

迁移后建议抽样对比（从业务角度验证）：
- Identity 列表（数量、last_used_at 排序）
- 上传历史分页（排序、update_time 行为）
- 抖音收藏列表与标签操作（新增/重命名/删除/打标签）

### 5.3 回归测试

建议至少跑一次：

```bash
go test ./...
TEST_DB_DIALECT=postgres go test ./...
```

说明：`TEST_DB_DIALECT` 仅影响 sqlmock 单测的方言模式（用于覆盖 PG 语义分支），不需要真实 Postgres 实例。

## 6. 回滚（快速恢复）

双栈的回滚非常直接：

1) 停止服务
2) 将 `DB_URL` scheme 切回原数据库（例如从 `postgres://...` 改回 `mysql://...`）
3) 重新启动服务

建议始终保留原库一段时间，并在切换后做一轮业务验收再决定是否下线旧库。

## 7. 常见问题（FAQ）

### 7.1 启动时报 “migrations directory not found”

迁移默认从相对路径 `sql/{dialect}` 读取。如果你不是从仓库根目录启动二进制，可能找不到脚本目录。

当前处理：
- Docker 镜像已通过 `Dockerfile` 将 `sql/` 复制到镜像内（通常位于 `/app/sql`），容器内启动不受影响。

后续可选增强（未实现）：
- 将迁移脚本 `embed` 到二进制，或增加可配置的 baseDir（如 `MIGRATIONS_DIR`）。

