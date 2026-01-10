# 变更提案: 测试覆盖补齐（认证/WebSocket/重连/CRUD）

## 需求背景
当前系统在认证与 WebSocket 消息核心上存在测试盲区，且前端断线重连与后端收藏/历史列表的关键分支缺少回归保护，导致改动时难以及时发现行为偏差。

## 变更内容
1. 补齐 Go 后端认证 handler 与 WebSocket 管理器单元测试。
2. 补齐前端 `useWebSocket` 断线自动重连、forceout 禁止重连与手动断开不重连的测试。
3. 补齐后端 `favorite`（本地收藏）与 `user_history`（历史/收藏列表代理）测试。

## 影响范围
- **模块:** Auth、WebSocket Proxy、User History、Favorite、Frontend Chat（useWebSocket）
- **文件:** 以新增/调整测试文件为主，少量可测试性改动（time delay 可注入、JWT 密钥缺失保护）
- **API:** 无
- **数据:** 无（使用 `sqlmock` 与自定义 HTTP Transport）

## 核心场景

### 需求: 认证接口测试覆盖
**模块:** Auth

#### 场景: 登录与验签
- 登录成功返回 `token`
- 访问码缺失/错误返回明确错误
- `Bearer` 缺失/无效/有效 Token 的 `verify` 分支可回归

### 需求: WebSocket 管理器核心行为测试覆盖
**模块:** WebSocket Proxy

#### 场景: 握手鉴权/代理/连接池/延迟关闭/踢下线/缓存写入
- `/ws` 握手 token 校验（缺失/无效拒绝）
- 下游发起 `sign` 后，上游可收到 sign；后续消息可正确代理到上游
- sign 绑定后仅允许转发与已绑定 `userId` 一致的消息，避免跨身份注入
- `sign` 消息可在上游连接建立后正确发送
- 下游全部断开后按延迟策略关闭上游连接；重新注册可取消关闭任务
- 超出最大并发身份时淘汰最旧身份连接
- 处理 `forceout` 时标记禁止重连并触发清理路径
- `code=15/7` 消息触发缓存写入（用户信息/最后消息）分支可回归
- 广播到下游时自动清理僵尸连接，避免阻塞/堆积

### 需求: 前端断线重连测试
**模块:** Frontend Chat

#### 场景: 自动重连与手动断开
- 非 forceout 的连接关闭会触发 3 秒后自动重连
- 手动 `disconnect(true)` 不应触发自动重连
- 收到 forceout 消息会清理 token，并跳过后续自动重连

### 需求: 收藏/历史列表测试
**模块:** Favorite、User History

#### 场景: CRUD 与代理增强
- `favorite` 的新增/查询/列表等基础分支可回归
- `user_history` 的上游失败与缓存增强分支可回归

## 风险评估
- **风险:** WebSocket/定时逻辑测试存在不稳定风险（时间敏感、异步 goroutine）。
- **缓解:** 将关键 delay 抽为可覆盖变量；测试中缩短 delay、统一清理连接与定时器，并在前端使用 fake timers 保证可控。
