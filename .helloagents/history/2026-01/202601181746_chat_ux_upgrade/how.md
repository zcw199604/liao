# 技术设计: 聊天体验优化（乐观发送 / 骨架屏 / 虚拟滚动 / 媒体组件化）

## 技术方案

### 核心技术
- Vue 3 + Vite + TypeScript（现有）
- Pinia（现有）
- WebSocket（现有）
- `vue-virtual-scroller`（新增，用于虚拟滚动）
- IntersectionObserver（新增使用，用于媒体懒加载）
- Tailwind CSS（现有）

### 实现要点

#### 1) 乐观发送（Optimistic UI）

**核心目标**：在调用 WebSocket 发送时立即插入本地消息，并在回显/超时后更新状态。

**数据模型（前端本地扩展）**
- `ChatMessage` 增加字段（不影响后端协议）：
  - `clientId`: 本地生成的唯一标识（用于定位与更新 UI）
  - `sendStatus`: `sending` | `sent` | `failed`
  - `sendError?`: 失败原因（可选，用于提示）
  - `optimistic?`: 标记是否为本地插入（可选）

**发送流程（useMessage）**
1. 校验 `currentUser/targetUser/wsConnected`
2. 生成 `clientId` 与本地 `ChatMessage`（`sendStatus=sending`），插入 `messageStore`
3. 同时通过 `useWebSocket.send()` 发送原有协议消息
4. 启动超时计时器：超时未确认则标记 `failed`

**回显确认（useWebSocket.onmessage）**
- 继续保留现有 “按 `tid` 去重” 逻辑
- 新增“回显合并”逻辑：
  - 仅对 `isSelf === true` 的私信回显尝试合并
  - 匹配条件：同会话（peerId）+ 同消息类型（文本/媒体）+ 归一化内容或媒体 remotePath 一致 + 时间窗口（如 10~20s）+ 队列顺序（优先匹配最早 pending）
  - 匹配成功：更新原本地消息为 `sent`，并补全 `tid/time/segments` 等字段（避免重复渲染）
  - 匹配失败：按现有逻辑作为新消息追加

**失败与重试**
- UI 在 `sendStatus=failed` 时提供“重试”
- 重试复用原始内容/媒体信息重新发送，并将状态重置为 `sending`（可生成新 `clientId` 或复用旧值，需在实现中确定一致性）

#### 2) 骨架屏加载（Skeleton Loading）

**新增组件**
- `Skeleton.vue`：通用灰色脉冲块（支持圆角/尺寸/行数）

**应用策略**
- 消息列表：当 `messageStore.isLoadingHistory === true` 且当前会话消息为空时，渲染若干“气泡骨架”
- 侧边栏/收藏列表：首次加载期间渲染“头像 + 文本”骨架条目；刷新（pull-to-refresh）可保留现有顶部进度条

#### 3) 消息列表虚拟滚动（Virtual Scrolling）

**技术选型**
- 使用 `vue-virtual-scroller` 的 `DynamicScroller` / `DynamicScrollerItem` 支持不定高消息（文本/图片/视频）。

**集成要点**
- Key：优先 `tid`，其次 `clientId`，再兜底 key（避免重排与渲染抖动）
- 兼容现有交互：
  - 继续支持“回到底部/新消息”悬浮按钮
  - 继续支持“查看历史消息/加载更多”
  - 图片加载导致高度变化：通过 `DynamicScrollerItem` 的测量机制自动更新（必要时由媒体组件主动触发刷新）

#### 4) 媒体组件化（ChatMedia）

**新增组件**
- `ChatMedia.vue`：统一渲染图片/视频（供消息气泡调用）

**能力**
- 加载占位：媒体未加载完成前显示骨架占位
- 自适应宽高：优先用已知元数据；无元数据时先用默认比例占位，加载后再校正
- 错误兜底：加载失败显示自定义占位与提示
- 懒加载：进入视口后再设置 `src` 发起请求（IntersectionObserver；不支持时降级为立即加载）

## 架构决策 ADR

### ADR-20260118-01: 引入 `vue-virtual-scroller` 实现消息列表虚拟滚动
**上下文:** `MessageList.vue` 通过 `v-for` 渲染全部消息，长对话导致 DOM 过多、滚动卡顿与内存升高；消息高度不固定（文本/媒体混排）。
**决策:** 引入 `vue-virtual-scroller`，使用 `DynamicScroller` 支持不定高虚拟列表。
**理由:**
- 成熟方案，较低实现与维护成本
- 对不定高场景支持更好，能覆盖媒体加载导致高度变化
- 可在不改变后端协议的前提下优化性能
**替代方案:** 自研虚拟列表 → 拒绝原因: 不定高测量与滚动定位复杂，回归风险与维护成本高
**影响:**
- 新增前端依赖与样式引入
- 需要重构 `MessageList.vue` 的滚动逻辑与“新消息提示”判断

## API设计
- 不变更后端 API/协议
- 前端回显确认通过本地匹配策略实现（未来可扩展：在协议中增加 `clientMsgId` 以实现强一致对账）

## 数据模型
- 不新增/修改后端数据表
- 仅扩展前端 `ChatMessage` 的本地字段（`clientId/sendStatus` 等）

## 安全与性能
- **安全:**
  - 不新增敏感信息存储；不在本地持久化明文凭证
  - 保持消息渲染的 XSS 风险可控（继续复用现有 emoji/富文本渲染策略；必要时补充输入清理）
- **性能:**
  - 虚拟滚动显著降低 DOM 数量与重排成本
  - 媒体懒加载减少首屏请求与解码压力
  - 骨架屏减少布局跳动带来的“感知卡顿”

## 测试与部署
- **测试:**
  - `vitest`：覆盖“乐观消息插入/超时失败/回显合并/重试”核心逻辑
  - 组件测试：MessageList 虚拟列表 key 稳定性、失败状态 UI 展示
- **部署:**
  - 仅前端改动；确保 `npm run build` 通过即可（后端无需改动）
