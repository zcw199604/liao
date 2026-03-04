# 技术设计: Go 文件功能测试补齐

## 技术方案

### 核心技术
- Go `testing` 标准库
- `net/http/httptest`：模拟 HTTP handler 与上游服务
- `github.com/DATA-DOG/go-sqlmock`：模拟 `database/sql`，避免依赖真实 MySQL

### 实现要点
- 为“端口探测”提供可替换的注入点（默认仍使用现有探测逻辑），在测试中替换为固定值，避免慢测试与外网依赖。
- 通过测试工具函数构造 `multipart/form-data` 请求，生成 `*multipart.FileHeader`，用于复用 `CalculateMD5/SaveFile/CalculatePHash` 等逻辑。
- 文件系统相关测试全部使用 `t.TempDir()` 作为 `upload` 根目录，避免污染工作区。
- handler 测试直接调用具体 handler（不走 JWT 中间件），验证响应 JSON/文本与副作用（落盘、缓存写入、DB 调用）。

## 架构决策 ADR

### ADR-001: 使用 sqlmock 替代真实 MySQL 进行测试
**上下文:** 文件功能涉及 `media_file/image_hash/media_upload_history` 等表查询/写入。真实 MySQL 依赖会降低测试稳定性与可移植性。
**决策:** 在测试中使用 `go-sqlmock` 模拟 `*sql.DB` 行为，针对关键 SQL 设定期望与返回数据。
**理由:** 不依赖外部服务、执行速度快、可精确验证 SQL 调用路径。
**替代方案:** 使用 Docker MySQL/本地 MySQL → 拒绝原因: 增加环境成本与不稳定因素。
**影响:** 需要在 `go.mod` 中新增 test 依赖；测试需维护 SQL 期望（用正则匹配降低耦合）。

## API设计
无（仅新增测试验证现有接口行为）。

## 数据模型
无（仅使用 mock 数据行模拟现有表结构）。

## 安全与性能
- **安全:** 测试中不引入真实密钥/令牌，不访问未授权生产服务。
- **性能:** 避免测试中触发真实 `detectAvailablePort` 的多端口探测；使用注入点替换，确保 `go test` 快速稳定。

## 测试与部署
- **测试:** `go test ./...`
- **局部测试:** `go test ./internal/app -run TestName`
- **部署:** 无（测试与少量可测试性改动不影响部署）

