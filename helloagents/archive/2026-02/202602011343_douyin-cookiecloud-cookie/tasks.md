# 任务清单: douyin-cookiecloud-cookie

> **@status:** completed | 2026-02-01 14:16

目录: `helloagents/plan/202602011343_douyin-cookiecloud-cookie/`

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
总任务: 9
已完成: 9
完成率: 100%
```

---

## 任务列表

### 1. 配置（env）与文档

- [√] 1.1 在 `internal/config/config.go` 中新增 CookieCloud 抖音 Cookie 缓存 TTL 配置项（单位：小时），默认 72h（3 天）
  - 约定环境变量: `COOKIECLOUD_COOKIE_EXPIRE_HOURS`（兼容别名 `COOKIE_CLOUD_COOKIE_EXPIRE_HOURS`）
  - 验证: `go test ./...`（覆盖 config 校验与默认值）

- [√] 1.2 更新 `src/main/resources/application.yml`（仅作为 env 对照表）补充 CookieCloud 相关 env 与 TTL env
  - 依赖: 1.1

### 2. 抖音 CookieCloud Provider（缓存 + 懒加载）

- [√] 2.1 新增 `internal/app/douyin_cookiecloud_provider.go`（或同等命名）实现：
  - 从 CookieCloud 拉取并本地解密（复用 `internal/cookiecloud`）
  - 缓存策略：`CACHE_TYPE=redis` 写入 Redis（带 TTL）；否则进程内缓存（带 TTL）
  - 并发控制：同一进程内避免并发重复拉取（double-check + mutex）
  - 验证: 单测覆盖（见 2.2）

- [√] 2.2 为 provider 补齐单测：
  - 内存模式：首次触发拉取；TTL 内命中缓存不重复拉取
  - Redis 模式：写入 Redis + TTL；重建 provider 仍能从 Redis 读到缓存
  - 建议使用 `miniredis`（仓库已引入依赖）

### 3. 上游请求注入（douyin-downloader）

- [√] 3.1 在 `internal/app/douyin_downloader.go` 中接入 provider：当请求未显式传 `cookie` 时，通过 provider 获取 cookie 并注入到上游请求 payload 的 `cookie` 字段
  - 验证: handler 单测或对 `TikTokDownloaderClient.postJSON` 的入参断言（确保 `cookie` 字段非空）

- [√] 3.2 在 `internal/app/app.go` 中组装并注入 provider（从 `config.Config` 读取 CookieCloud 配置；若 `CACHE_TYPE=redis` 复用现有 Redis client）
  - 验证: `go test ./...` + 基础集成测试（启动 App 并请求任一抖音 API）

### 4. 知识库同步与验收

- [√] 4.1 更新模块文档 `helloagents/modules/douyin-downloader.md`：新增“CookieCloud 自动 cookie”说明 + 安全注意事项 + env 列表
  - 依赖: 1.1, 2.1, 3.1

- [√] 4.2 更新外部参考 `helloagents/modules/external/cookiecloud.md`：补充本项目使用方式与 TTL 环境变量
  - 依赖: 1.1, 2.1

- [√] 4.3 全量验证：`go test ./...` 通过；并记录手工验证步骤（可选）
  - 依赖: 2.2, 3.2

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
