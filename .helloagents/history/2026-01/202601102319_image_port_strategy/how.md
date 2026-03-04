# 技术设计: 全局图片端口策略配置（DB + Settings 面板）

## 技术方案

### 核心技术
- 后端：Go（chi 路由）+ MySQL（新增 `system_config` 表）
- 前端：Vue 3 + Pinia（新增全局系统配置 store）

### 实现要点
- **配置存储（DB，全局共用）**
  - 新增 `system_config`（Key-Value）表，存储系统级配置项。
  - 本次新增配置项：
    - `image_port_mode`: `fixed` / `probe` / `real`
    - `image_port_fixed`: 固定模式图片端口（默认 `9006`）
    - `image_port_real_min_bytes`: 真实图片请求模式最小字节阈值（默认 `2048`）
- **端口解析**
  - 固定模式：直接返回 `image_port_fixed`
  - 可用端口探测模式：复用现有 `detectAvailablePort(imgHost)`（请求 `useripaddressv23.js`）
  - 真实图片请求模式：对候选端口轮询请求 `http://{imgHost}:{port}/img/Upload/{path}`：
    - 读取响应前 `minBytes`（优先 `Range: bytes=0-{minBytes-1}`）
    - 判定条件（任一不满足则继续尝试下一端口）：
      - HTTP 状态为 2xx/206
      - `Content-Type` 不为 `text/html`
      - 响应体前缀（去空白后）不以 `<` 开头（规避 HTML 占位页）
      - 实际读取字节数 ≥ `minBytes`
    - 成功后缓存“当前 imgHost 的图片端口”，避免重复探测
  - **视频端口保持现有固定逻辑**：始终 `8006`（不受图片策略影响）
- **前端接入**
  - 新增系统配置 store：加载/保存系统配置；提供 `resolveImagePort(path)` 方法（按 mode 决定是否调用后端解析 API）。
  - 在 WS 收消息、历史消息解析、缓存图片映射、上传/重传成功拼接 URL 等入口，统一使用该 store 获取图片端口。

## 架构决策 ADR
### ADR-001: 图片端口解析放在后端执行（浏览器只调用解析 API）
**上下文:** 浏览器对跨域端口的 `fetch/HEAD/Range` 探测可能受 CORS 限制；同时需要按字节阈值读取内容进行判定。  
**决策:** 由后端提供 `resolveImagePort` API，负责执行探测与缓存；前端仅根据策略调用该 API 并拼接 URL。  
**理由:** 避免 CORS 限制；统一探测逻辑与缓存；降低前端复杂度。  
**替代方案:** 前端直接对各端口发起请求并判定 → 拒绝原因: CORS/实现复杂度与稳定性风险更高。  
**影响:** 后端会产生少量额外请求；通过超时与缓存控制可接受。

## API设计

### [GET] /api/getSystemConfig
- **响应:**
  - `code=0` 且 `data` 包含：
    - `imagePortMode`
    - `imagePortFixed`
    - `imagePortRealMinBytes`

### [POST] /api/updateSystemConfig
- **请求(JSON):**
  - `imagePortMode`: `fixed` / `probe` / `real`
  - `imagePortFixed`: 端口（可选）
  - `imagePortRealMinBytes`: 最小字节阈值（可选）
- **响应:** `code=0`，返回更新后的配置 `data`

### [POST] /api/resolveImagePort
- **请求(JSON):**
  - `path`: `/img/Upload/` 后的相对路径（例如 `2026/01/a.jpg` 或 `a.png`）
- **响应:** `code=0`，`data.port` 为解析后的图片端口

## 数据模型
新增表 `system_config`：
- `config_key` (PK)
- `config_value`
- `created_at`
- `updated_at`

## 安全与性能
- **安全:** 解析 API 受 JWT 中间件保护；输入 `path` 做最小化清洗（去除 `..`、控制长度），避免构造异常请求路径。
- **性能:** 真实图片请求使用小范围读取 + 超时（如 800ms）+ 结果缓存（按 imgHost 缓存端口），降低重复探测成本。

## 测试与部署
- 后端：使用 `httptest` + `sqlmock` 覆盖配置读写与真实探测判定逻辑。
- 前端：运行 `npm test` 与 `npm run build`，确保类型检查通过。
