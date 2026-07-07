# 技术设计: Android 会话列表全局收藏切换

## 技术方案
### 核心技术
- Kotlin + Jetpack Compose
- Retrofit `FavoriteApiService`
- ViewModel + coroutine
- JUnit + MockK + Compose UI Test

### 实现要点
- `ChatListRepository` 注入 `FavoriteApiService`，新增 `loadGlobalFavoriteTargetIds()` 与 `toggleGlobalFavorite(peer, isGlobalFavorite)`。
- 初始加载时通过 `/api/favorite/listAll` 得到所有全局收藏，筛选当前身份的 `targetUserId` 集合。
- 切换时根据当前状态调用 `/api/favorite/add` 或 `/api/favorite/remove`，成功后由 ViewModel 更新集合。
- `ChatListScreenContent` 会话项增加全局收藏按钮，文案基于 `globalFavoriteTargetIds` 切换。

## 设计边界
- **范围内:** 会话列表全局收藏状态加载、单项添加/移除、UI 提示和测试。
- **范围外:** 在线状态查询、聊天页普通收藏重构、全局收藏页缓存策略重构。
- **模块职责:** Repository 负责读取当前身份与调用全局收藏 API；ViewModel 负责集合状态和提示；Compose 负责按钮展示与点击回调。
- **接口契约:** 复用 `GET /api/favorite/listAll`、`POST /api/favorite/add`、`POST /api/favorite/remove`。
- **数据边界:** 不新增 Room 表字段或数据库版本；本轮状态保存在 `ChatListUiState`。
- **依赖边界:** 不新增第三方依赖。
- **大型项目最小改动:** 仅触达 chatlist 与对应测试，不拆分大文件，不调整导航。

## API设计
### GET /api/favorite/listAll
- **请求:** 无额外参数。
- **响应:** `ApiEnvelope<List<JsonElement>>`，Android 使用 `toFavoriteItemOrNull()` 解析。

### POST /api/favorite/add
- **请求:** `identityId`, `targetUserId`, `targetUserName`
- **响应:** `ApiEnvelope<JsonElement>`

### POST /api/favorite/remove
- **请求:** `identityId`, `targetUserId`
- **响应:** `ApiEnvelope<Unit>`

## 安全与性能
- **安全:** 不新增敏感信息存储；操作依赖当前身份会话。
- **性能:** 初始加载只拉取一次全局收藏列表；切换成功后 ViewModel 局部更新集合，避免每次点击都强制全量刷新。

## 测试与部署
- **测试:** 补 Repository 全局收藏加载/添加/移除/失败，补 ViewModel 状态更新，补 Compose 按钮回调测试。
- **部署:** Android 客户端常规构建；本轮无数据库迁移。
