# 技术设计: 修复列表横滑偏移卡住

## 技术方案

### 核心技术
- Vue 3
- `@vueuse/core` 的 `useSwipe`
- 在手势结束阶段执行 UI 清理/复位

### 实现要点
- `useSwipeAction`（`frontend/src/composables/useInteraction.ts`）
  - 保持现有 `onSwipeEnd` 的“超过 threshold 才触发”的语义不变，避免小幅滑动误触发业务切换。
  - 在 VueUse 的 `onSwipeEnd` 生命周期中，无条件触发新增的 `onSwipeFinish`（命名可调整），用于 UI 复位/清理。
  - `onSwipeFinish(deltaX, deltaY, isTriggered)`：
    - `deltaX/deltaY`: 结束时的最终位移（带符号）
    - `isTriggered`: 本次手势是否命中阈值并触发了 `onSwipeEnd`（用于消费端避免不必要的动画冲突）
  - 调用顺序：先触发 `onSwipeEnd`（如果命中阈值），再触发 `onSwipeFinish`（必触发），以保持“先业务切换、后收尾复位”的时序一致性。
- `ChatSidebar`（`frontend/src/components/chat/ChatSidebar.vue`）
  - `onSwipeProgress` 继续负责跟手更新 `listTranslateX`。
  - `onSwipeEnd` 仅负责 Tab 切换逻辑（消息 ↔ 收藏），并保持现有阈值体验。
  - 在 `onSwipeFinish` 中统一执行回弹复位（`listTranslateX = 0`），保证阈值内横滑也能复位；必要时可基于 `isTriggered` 做动画/时序微调。
  - 复位动画用 `isAnimating` 控制 `transition`；为避免连续滑动造成状态错乱，复位时清理旧的 timer。

## 架构决策 ADR

### ADR-001: 将“复位”从 `onSwipeEnd` 解耦（新增 `onSwipeFinish`）
**上下文:** 目前 `onSwipeEnd` 受 threshold 约束，阈值内横滑不会触发回调，导致跟手位移残留。业务侧既需要“阈值触发的动作”（切换 Tab），也需要“手势结束的清理”（复位位移）。

**决策:** 在 `useSwipeAction` 中新增 `onSwipeFinish`（可选），并保证在每次手势结束时都会调用；`onSwipeEnd` 仍保持阈值触发语义。

**理由:**
- 行为更可预测：业务动作与 UI 清理分离，避免复位逻辑依赖阈值判断。
- 向后兼容：新增可选回调不影响既有调用方。
- 可复用：后续其他跟手交互也可复用 `onSwipeFinish` 做清理。

**替代方案:** 在 `ChatSidebar` 内 `watch(isSwiping)` 做复位 → 拒绝原因: 仅修复单点，复位逻辑仍与手势抽象脱节，后续同类交互容易重复踩坑。

**影响:**
- 需要更新 `SwipeActionOptions` 类型定义，并在实现中增加一次回调触发。
- 调用方如需“阈值内也做收尾”，应使用 `onSwipeFinish` 承载清理逻辑。

## 安全与性能
- **安全:** 不涉及权限变更、不涉及外部请求/敏感信息处理
- **性能:** `onSwipeFinish` 仅在手势结束触发，开销可忽略；避免在 `onSwipeProgress` 内做重计算

## 测试与部署
- **测试:**
  - Vitest mock `@vueuse/core` 的 `useSwipe`，验证：
    - 阈值内：`onSwipeEnd` 不触发，但 `onSwipeFinish` 必触发，且 `isTriggered=false`
    - 阈值外：`onSwipeEnd` 与 `onSwipeFinish` 均触发，方向判定正确，且 `isTriggered=true`
    - 调用顺序：`onSwipeEnd`（若触发）应先于 `onSwipeFinish`
  - 组件侧验证：手动测试横滑阈值内/外，均可复位，不再出现残留偏移。
- **部署:** 按现有前端构建流程执行 `npm run build`，无额外发布步骤
