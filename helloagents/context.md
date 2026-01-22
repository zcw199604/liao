# 项目上下文

## 代码结构
- 后端入口: `cmd/liao/`
- 后端实现: `internal/`
- 前端工程: `frontend/`（Vue 3 + Vite + TypeScript）
- SQL/初始化脚本: `sql/`
- 前端构建产物输出（gitignored）: `src/main/resources/static/`
- 运行时上传目录（不要提交）: `upload/`

## 常用命令
- 后端启动: `go run ./cmd/liao`
- 后端测试: `go test ./...`
- 后端构建: `go build ./cmd/liao`
- 前端开发: `cd frontend && npm run dev`
- 前端单测: `cd frontend && npm test`
- 前端构建: `cd frontend && npm run build`

## 配置约定
- 运行时配置通过环境变量提供（Go 服务不读取 `src/main/resources/application.yml`，该文件仅作为 env 默认值对照表保留）
- 本地运行至少需要（按实际功能启用情况）:
  - `DB_URL`, `DB_USERNAME`, `DB_PASSWORD`
  - `WEBSOCKET_UPSTREAM_URL`
  - `AUTH_ACCESS_CODE`, `JWT_SECRET`

