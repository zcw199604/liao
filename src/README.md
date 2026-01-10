# Java 版本（已弃用，仅供参考）

本目录 `src/` 包含历史遗留的 Java(Spring Boot) 后端代码（主要位于 `src/main/java/` 与 `src/test/java/`），**已不再维护/不可用**，仅保留用于参考对照。

## 当前可用后端（Go 版本）
- 入口: `cmd/liao/main.go`
- 业务实现: `internal/`
- 运行方式请参考仓库根目录 `README.md`

## 说明
- Java 启动类位于 `src/main/java/com/zcw/Main.java`（仅供参考）
- `src/main/resources/static/` 仍会被 Go 后端作为默认静态资源目录使用（前端构建产物可能输出到此目录）
