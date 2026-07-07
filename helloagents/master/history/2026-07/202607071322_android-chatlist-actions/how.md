# 技术设计: Android 会话项动作对齐

## 技术方案
### 核心技术
- Kotlin + Jetpack Compose
- Retrofit + FormUrlEncoded
- Room DAO
- ViewModel + coroutine
- JUnit + MockK + Compose UI Test

### 实现要点
- 在 `ChatApiService` 增加 `deleteUpstreamUser(myUserId, userToId)`，对齐 Vue `frontend/src/api/chat.ts` 的 `/deleteUpstreamUser`。
- 在 `ConversationDao` 增加按 ID 删除会话的方法，复用 `MessageDao.clearByPeer(peerId)` 清理消息。
- 在 `ChatListRepository` 增加 `deletePeer(peerId)`：读取当前身份、调用远端、成功后清理本地会话和消息缓存。
- `ChatListViewModel` 增加待删除会话、确认弹窗状态、清未读和删除动作的状态消息。
- `ChatListScreenContent` 为每个会话项增加清未读和删除入口，删除需要确认。

## 设计边界
- **范围内:** Android 会话列表单项动作、网络接口契约、DAO 方法和测试。
- **范围外:** 批量删除、长按上下文菜单完整复刻、查在线状态、全局收藏切换。
- **模块职责:** 网络层只声明接口；Repository 负责远端/本地一致性；ViewModel 负责 UI 状态；Compose 负责用户动作入口。
- **接口契约:** 新增 `POST /api/deleteUpstreamUser`，表单字段 `myUserId` 和 `userToId`。
- **数据边界:** 只增加 DAO 查询方法，不新增 Room 表字段，不提升数据库版本。
- **依赖边界:** 不新增第三方依赖。
- **大型项目最小改动:** 仅触达 chatlist、network、database 和对应测试，避免拆分大文件或重构导航。

## API设计
### POST /api/deleteUpstreamUser
- **请求:** `application/x-www-form-urlencoded`
  - `myUserId`: 当前身份 ID
  - `userToId`: 目标用户 ID
- **响应:** 复用 `ApiEnvelope<Unit>`；`code == 0` 视为成功。

## 安全与性能
- **安全:** 不新增敏感信息存储；删除操作需要当前身份会话存在。
- **性能:** 单条删除只触达目标会话和目标消息缓存，不扫描全表。

## 测试与部署
- **测试:** 优先补 `ChatListRepositoryTest` 和 `ChatListViewModelTest`；补 `ChatListScreenTest` 验证按钮/确认弹窗可触达。
- **部署:** Android 客户端常规构建；本轮无后端 schema 迁移。
