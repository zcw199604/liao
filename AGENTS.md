# Repository Guidelines

## Project Structure & Module Organization
- `cmd/liao/` + `internal/`: Go backend（入口与业务实现，HTTP + WebSocket + 存储/缓存/上传）。
- `src/main/resources/application.yml`: 历史配置文件（仅用于记录环境变量名/默认值；Go 服务不读取该文件）。
- `src/main/resources/static/`: 前端生产构建产物输出目录（gitignored，运行时由 Go 服务托管）。
- `frontend/`: Vue 3 + Vite + TypeScript UI (see `frontend/src/` for `api/`, `stores/`, `composables/`, `components/`, `views/`).
- `sql/`: database schema/bootstrap scripts.
- `upload/`: runtime uploaded media (keep out of git).

## Build, Test, and Development Commands
Backend (Go 1.25.x):
- `go run ./cmd/liao` start API + WebSocket on `http://localhost:8080`.
- `go test ./...` run Go tests.
- `go build ./cmd/liao` build the runnable binary.

Frontend (Node 20 in CI, Vite dev on `http://localhost:3000`):
- `cd frontend && npm ci` install dependencies.
- `npm run dev` start dev server (proxies `/api` and `/ws` to `:8080`).
- `npm run build` typecheck + build; outputs to `src/main/resources/static/` (gitignored).
- `npm run preview` preview the production build.

Docker:
- `docker build -t liao:latest .`

## Configuration & Security Tips
- Runtime settings are provided via environment variables（Go 服务不读取 `src/main/resources/application.yml`，该文件仅作为 env 默认值对照表保留）。
- For local runs, set at least: `DB_URL`, `DB_USERNAME`, `DB_PASSWORD`, `WEBSOCKET_UPSTREAM_URL`, `AUTH_ACCESS_CODE`, `JWT_SECRET`.
- Never commit real credentials or uploaded media; use local env vars or Docker `-e ...` flags.

## Coding Style & Naming Conventions
- Go: run `gofmt`; keep code under `cmd/` + `internal/`; REST endpoints use camelCase paths (project convention).
- Vue/TS: prefer `<script setup>`; components `PascalCase.vue`, composables `useXxx.ts`, Pinia stores in `frontend/src/stores/`.
- Keep changes focused; do not edit generated files under `src/main/resources/static/`.

## Testing Guidelines
- Backend uses Go testing. Put tests next to code as `*_test.go`, and run via `go test ./...`.
- Frontend has no dedicated test runner yet; treat `npm run build` as the CI gate.

## Commit & Pull Request Guidelines
- Follow Conventional Commits seen in history: `feat: ...`, `fix: ...`, `refactor: ...`, `chore: ...`.
- PRs should include: summary, how to test, config/env var changes, and screenshots for UI changes.
- Never commit artifacts: `target/`, `node_modules/`, `src/main/resources/static/`, or uploaded files.
