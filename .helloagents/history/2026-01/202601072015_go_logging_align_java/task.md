# 任务清单: Go 日志对齐 Java 版（容器可见）

目录: `helloagents/plan/202601072015_go_logging_align_java/`

---

## 1. User History 日志对齐
- [√] 1.1 为 `/api/getHistoryUserList` 增加分段耗时日志（`upstreamMs/enrichUserInfoMs/lastMsgMs/totalMs/size/cacheEnabled`）
- [√] 1.2 为 `/api/getFavoriteUserList` 增加分段耗时日志（`upstreamMs/enrichUserInfoMs/lastMsgMs/totalMs/size/cacheEnabled`）
- [√] 1.3 为 User History 代理接口补齐关键请求/上游返回日志（不输出 Cookie 明文，仅记录是否存在与长度）
- [√] 1.4 为上传媒体与收藏相关接口补齐关键日志（上传参数/失败原因/耗时等）

## 2. 鉴权告警日志
- [√] 2.1 JWT 鉴权缺失/校验失败时输出 warn 日志（对齐 Java JwtInterceptor 行为）

## 3. 日志可配置
- [√] 3.1 支持通过 `LOG_LEVEL`（debug/info/warn/error）控制 slog 日志级别
- [√] 3.2 支持通过 `LOG_FORMAT=text` 切换为文本日志输出（默认 JSON）

## 4. 知识库同步
- [√] 4.1 更新 `helloagents/wiki/modules/user-history.md` 依赖指向 Go 实现
- [√] 4.2 更新 `helloagents/CHANGELOG.md` 记录本次日志对齐变更

## 5. 验证
- [X] 5.1 运行 `go test ./...` 验证编译与测试通过
  > 备注: 当前环境未安装/未配置 Go CLI（`go` 命令不可用），需在具备 Go 1.22+ 的环境中复验。

