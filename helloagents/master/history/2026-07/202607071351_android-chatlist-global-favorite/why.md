# 变更提案: Android 会话列表全局收藏切换

## 需求背景
Vue 会话侧边栏支持在单个会话上直接执行“全局收藏 / 取消全局收藏”，使用 `/api/favorite/add` 与 `/api/favorite/remove` 维护跨身份收藏。Android 会话列表目前已经补齐清未读和删除动作，但还不能直接切换全局收藏，用户需要进入独立全局收藏页才能管理，移动端操作闭环仍弱于 Web 端。

## 变更内容
1. Android 会话列表加载当前身份的全局收藏状态。
2. Android 会话项提供“全局收藏 / 取消全局收藏”动作。
3. 操作成功后即时更新会话列表按钮状态和提示消息。
4. 为 Repository、ViewModel 和 Compose 页面补充测试。

## 范围边界
- **范围内:** Android 会话列表单项全局收藏切换；读取 `/api/favorite/listAll`；调用 `/api/favorite/add` 和 `/api/favorite/remove`。
- **范围外:** WebSocket 在线状态查询、批量全局收藏、全局收藏页重构、聊天页头部收藏语义调整。
- **拆分说明:** 会话侧上下文动作仍有在线状态查询未补齐，本方案包先覆盖 HTTP/本地状态闭环清晰的全局收藏切换。

## 影响范围
- **模块:** Android Client
- **文件:** `ChatListFeature.kt`, `ChatListRepositoryTest.kt`, `ChatListViewModelTest.kt`, `ChatListScreenTest.kt`, `AndroidUiTestTags.kt`
- **API:** `GET /api/favorite/listAll`, `POST /api/favorite/add`, `POST /api/favorite/remove`
- **数据:** 不新增 Room schema；会话列表 ViewModel 持有当前身份下全局收藏目标 ID 集合。

## 核心场景

### 需求: 会话列表全局收藏切换
**模块:** Android Client
Android 会话列表应支持与 Vue 会话侧边栏一致的全局收藏切换动作。

#### 场景: 已收藏用户取消全局收藏
用户在 Android 会话列表点击已收藏会话的“取消全局收藏”。
- 调用 `POST /api/favorite/remove`，参数包含当前身份 ID 与目标用户 ID。
- 成功后当前会话按钮切换为“全局收藏”。
- 显示“已取消全局收藏”提示。

#### 场景: 未收藏用户加入全局收藏
用户在 Android 会话列表点击未收藏会话的“全局收藏”。
- 调用 `POST /api/favorite/add`，参数包含当前身份 ID、目标用户 ID 和目标用户名称。
- 成功后当前会话按钮切换为“取消全局收藏”。
- 显示“已加入全局收藏”提示。

#### 场景: 操作失败
接口返回失败或当前身份缺失。
- 不改变当前按钮状态。
- 显示后端错误或兜底错误文案。

## 风险评估
- **风险:** 全局收藏和上游普通收藏是两套语义，误用 `/toggleFavorite` 会导致 Web/Android 行为不一致。
- **缓解:** Android 会话列表只调用 `/favorite/*` 全局收藏 API，保留聊天页已有普通收藏逻辑。
