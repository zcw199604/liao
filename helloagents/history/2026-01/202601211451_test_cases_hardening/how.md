# 怎么做

## 1. 前端（Vue + Vitest）

1) 增加 Vitest 全局 setup：
- 为 `jsdom` 补齐 `window.matchMedia` 等浏览器 API，避免第三方依赖（如播放器组件）在测试环境初始化时报错。
- 视情况补齐 `ResizeObserver`/`IntersectionObserver`/`scrollTo` 等常用 API 的最小实现，避免未来新增组件引入同类问题。

2) 稳定现有用例：
- 修复由环境缺失引起的视图/路由相关测试失败。
- 对 WebSocket 相关用例进行必要的 mock/隔离，避免真实网络请求造成噪音或不稳定。

3) 增补关键边界用例（以不引入脆弱性为原则）：
- 覆盖 `useWebSocket` 对不同 `code` 的分支处理（拒绝/静默/提示/匹配/在线状态等）。

## 2. 后端（Go）

1) 新增测试覆盖：
- JWT：空 secret、过期 token、算法不匹配等。
- JWT 中间件：缺少 Bearer、白名单路径、有效 token 放行。
- Identity：参数校验与 DB 交互边界（使用 `sqlmock`）。
- Forceout：过期清理、计数与清空逻辑。
- System handlers：wsManager/forceoutManager 未初始化与初始化后的返回形态。
- VideoExtract：关键参数校验；对需要 `ffprobe` 的路径通过“假 ffprobe”脚本解耦（仅测试探测解析与创建任务的可达性，不运行 ffmpeg）。

2) 保持测试可重复、无外部依赖：
- 禁止连接外部网络服务；涉及外部接口的一律使用 `httptest` 或 `sqlmock`。

## 3. 知识库同步

- 更新 `helloagents/CHANGELOG.md` 记录本次测试体系加固。
- 视需要补充 `helloagents/wiki` 中“如何运行测试/常见测试环境问题”的说明。

