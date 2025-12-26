# Repository Guidelines

## Project Structure & Module Organization
- `src/main/java/com/zcw/`: Spring Boot backend (`config/`, `controller/`, `service/`, `websocket/`, `util/`).
- `src/main/resources/application.yml`: configuration defaults + environment variable overrides.
- `frontend/`: Vue 3 + Vite + TypeScript UI (see `frontend/src/` for `api/`, `stores/`, `composables/`, `components/`, `views/`).
- `sql/`: database schema/bootstrap scripts.
- `upload/`: runtime uploaded media (keep out of git).

## Build, Test, and Development Commands
Backend (Java 17, Spring Boot 3):
- `mvn spring-boot:run` start API + WebSocket on `http://localhost:8080`.
- `mvn test` run JUnit tests.
- `mvn clean package -DskipTests` build the runnable jar.

Frontend (Node 20 in CI, Vite dev on `http://localhost:3000`):
- `cd frontend && npm ci` install dependencies.
- `npm run dev` start dev server (proxies `/api` and `/ws` to `:8080`).
- `npm run build` typecheck + build; outputs to `src/main/resources/static/` (gitignored).
- `npm run preview` preview the production build.

Docker:
- `docker build -t liao:latest .`

## Configuration & Security Tips
- Runtime settings live in `src/main/resources/application.yml` and are overridden via environment variables.
- For local runs, set at least: `DB_URL`, `DB_USERNAME`, `DB_PASSWORD`, `WEBSOCKET_UPSTREAM_URL`, `AUTH_ACCESS_CODE`, `JWT_SECRET`.
- Never commit real credentials or uploaded media; use local env vars or Docker `-e ...` flags.

## Coding Style & Naming Conventions
- Java: 4 spaces, keep packages under `com.zcw.*`; REST endpoints use camelCase paths (project convention).
- Vue/TS: prefer `<script setup>`; components `PascalCase.vue`, composables `useXxx.ts`, Pinia stores in `frontend/src/stores/`.
- Keep changes focused; do not edit generated files under `src/main/resources/static/`.

## Testing Guidelines
- Backend uses `spring-boot-starter-test` (JUnit 5). Put tests in `src/test/java` and name them `*Test.java`.
- Frontend has no dedicated test runner yet; treat `npm run build` as the CI gate.

## Commit & Pull Request Guidelines
- Follow Conventional Commits seen in history: `feat: ...`, `fix: ...`, `refactor: ...`, `chore: ...`.
- PRs should include: summary, how to test, config/env var changes, and screenshots for UI changes.
- Never commit artifacts: `target/`, `node_modules/`, `src/main/resources/static/`, or uploaded files.
