# 技术设计: 前端核心聊天模块测试补全

## 技术方案

### 核心技术
- Vitest（`jsdom` 环境）
- `@vue/test-utils`（SFC 渲染与事件触发）
- Pinia（`createPinia`/`setActivePinia`）

### 实现要点
- 组件测试以“可见 UI + emit 行为”为主，避免过度耦合内部实现细节。
- composables/store 单测以“输入 -> 状态/调用输出”为主，外部依赖（API、WebSocket、clipboard 等）通过 `vi.mock` 或 stub 方式隔离。
- 针对 WebSocket：实现 FakeWebSocket（构造 url、记录 send、手动触发 open/message/close）用于验证关键分支。
- 使用 `vi.useFakeTimers()` 覆盖长按/节流/重连等定时逻辑，并在 `afterEach` 统一恢复。

## 安全与性能
- **安全:** 测试中不写入任何真实凭据，不访问生产服务；仅 mock API 调用。
- **性能:** 控制测试用例数量与渲染深度，优先 shallow/mount 组合以保持运行速度。

## 测试与验证
- `cd frontend && npm test`
- `cd frontend && npm run build`

