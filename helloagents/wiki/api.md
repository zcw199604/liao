# API 手册（后端接口与处理逻辑 SSOT）

 > 本文档整理后端接口（原 Spring Boot：`src/main/java/com/zcw/`；现 Go 实现：`cmd/liao` + `internal/app/`）的全部 HTTP/WebSocket 接口及其关键处理逻辑，作为 100% 兼容实现的唯一真实来源（SSOT）。

---

## 1. 概述

- **API Base**：前端默认以 `/api` 为基地址（见 `frontend/src/constants/config.ts`）。
- **WebSocket**：前端通过 `/ws?token=...` 连接（见 `frontend/src/constants/config.ts`、`frontend/src/composables/useWebSocket.ts`）。
- **静态资源**：后端负责托管前端 SPA 静态资源与路由回退；上传文件通过 `/upload/**` 对外提供访问。
- **Go 代码入口**：
  - HTTP 路由：`internal/app/router.go`
  - JWT/CORS：`internal/app/middleware.go`、`internal/app/jwt.go`
  - WebSocket 代理：`internal/app/websocket_proxy.go`、`internal/app/websocket_manager.go`

---

## 2. 认证与鉴权

### 2.1 登录与 JWT

- 登录接口：`POST /api/auth/login`，使用访问码换取 JWT Token（HS256）。
- Token 验证接口：`GET /api/auth/verify`。
- Token 用法：HTTP 请求头 `Authorization: Bearer <token>`。

### 2.2 HTTP 拦截规则（强一致性要求）

拦截器（`com.zcw.interceptor.JwtInterceptor`）拦截所有 `/api/**`：
- **放行**：
  - `/api/auth/login`
  - `/api/auth/verify`
- **其余全部需要 Bearer Token**，否则返回 **HTTP 401**：
  - 缺失 Token：`{"code":401,"msg":"未登录或Token缺失"}`
  - Token 无效/过期：`{"code":401,"msg":"Token无效或已过期"}`

### 2.3 WebSocket 握手校验

握手拦截器（`com.zcw.websocket.JwtWebSocketInterceptor`）规则：
- 从 URL query 读取 `token` 参数
- Token 缺失或无效：握手拒绝（连接建立失败）

---

## 3. WebSocket 代理（/ws）行为

### 3.1 下游（前端）→ 后端

- 连接地址：`ws(s)://{host}/ws?token=<jwt>`
- 后端处理器：`com.zcw.websocket.ProxyWebSocketHandler`
- **第一条消息必须为登录 sign**（前端实现会在 `onopen` 发送）：
  - JSON 中要求至少包含：
    - `act`：`"sign"`
    - `id`：用户ID（作为后端 userId）
- 处理逻辑：
  1. 解析消息 JSON，读取 `act` 与 `id`
  2. `act == "sign"`：注册该 session → userId 映射，并将该用户的上游连接创建/复用，同时把 sign 原文转发到上游（用于上游登录）
  3. 其他消息：要求该 session 已完成注册，否则丢弃；随后将消息转发到上游（使用消息体内的 `id` 作为 userId）

### 3.2 上游（外部 WS）连接池与转发

管理器：`com.zcw.websocket.UpstreamWebSocketManager`，核心规则：

- **一人一条上游连接**：同一 `userId` 的多个下游 WebSocketSession 共享一条上游连接。
- **最多同时 2 个身份（userId）活跃**：`MAX_CONCURRENT_IDENTITIES = 2`
  - 超出上限时：FIFO 淘汰最早创建的 userId
  - 淘汰通知：向被淘汰的 userId 下游广播：
    - `{"code":-6,"content":"由于新身份连接，您已被自动断开","evicted":true}`
  - 通知后 1 秒：关闭该 userId 的上游连接与全部下游连接
- **上游地址获取**：
  - 每次创建上游连接时通过 `com.zcw.service.WebSocketAddressService#getUpstreamWebSocketUrl` 动态获取
  - 调用：`http://v1.chat2019.cn/Act/WebService.asmx/getRandServer?ServerInfo=serversdeskry&_=<ts>`
  - 解析失败/异常：降级 `ws://localhost:9999`
- **延迟关闭**：
  - 当某个 userId 的最后一个下游断开后，不立即关闭上游；延迟 `CLOSE_DELAY_SECONDS = 80` 秒
  - 若延迟期间该 userId 有新下游连接加入，则取消关闭任务
- **上游断开处理**：
  - 上游断开后：关闭该 userId 的全部下游连接，让前端触发重连

### 3.3 Forceout（防重连/封禁）逻辑

- 触发来源：上游消息满足 `code = -3` 且 `forceout = true`（解析于 `com.zcw.websocket.UpstreamWebSocketClient#onMessage`）
- 处理逻辑（`UpstreamWebSocketManager#handleForceout` + `ForceoutManager`）：
  1. 将 userId 加入禁止列表 **5 分钟**
  2. 广播原始 forceout 消息到该 userId 的全部下游（让前端停止重连）
  3. 关闭该 userId 的上游连接
  4. 延迟 1 秒关闭全部下游连接
- 当被禁止 userId 再次尝试注册（sign）时：
  - 先向下游发送拒绝消息（code=-4）并立刻关闭连接：
    - `{"code":-4,"content":"由于重复登录，您的连接被暂时禁止，请{remainingSeconds}秒后再试","forceout":true}`

### 3.4 上游消息侧的缓存增强

`UpstreamWebSocketClient#onMessage` 会尝试解析 JSON 并执行：
- 若 `code=15`：提取匹配用户信息并缓存（`UserInfoCacheService.saveUserInfo`）
- 若 `code=7`：缓存最后一条消息（`UserInfoCacheService.saveLastMessage`），并包含“会话 key 归一化补写”以提升命中率
- 最终：将上游消息原文广播给该 userId 的所有下游

---

## 4. HTTP 接口清单（按模块）

> 说明：除 `/api/auth/*` 外，其余 `/api/**` 全部要求 `Authorization: Bearer <token>`。

### 4.1 Auth（认证）

#### [POST] /api/auth/login
**描述**：访问码登录，签发 JWT。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 说明 |
|---|---|---|
| accessCode | 是 | 访问码 |

**处理逻辑**
- accessCode 为空：HTTP 400，`{"code":-1,"msg":"访问码不能为空"}`
- accessCode 不匹配配置：HTTP 400，`{"code":-1,"msg":"访问码错误"}`
- 成功：HTTP 200，`{"code":0,"msg":"登录成功","token":"..."}`（Token 不绑定具体 userId）

#### [GET] /api/auth/verify
**描述**：验证 JWT 是否有效。

**请求头**
- `Authorization: Bearer <token>`

**处理逻辑**
- Header 缺失或不以 `Bearer ` 开头：HTTP 200，`{"code":-1,"msg":"Token缺失"}`
- Token 校验：
  - 有效：HTTP 200，`{"code":0,"msg":"Token有效","valid":true}`
  - 无效：HTTP 200，`{"code":-1,"msg":"Token无效","valid":false}`

---

### 4.2 Identity（本地身份管理，MySQL）

控制器：`com.zcw.controller.IdentityController`（Base：`/api`）  
存储：`com.zcw.service.IdentityService`（表：`identity`，启动时 `CREATE TABLE IF NOT EXISTS`）

#### [GET] /api/getIdentityList
**描述**：获取身份列表（按 `last_used_at` 倒序）。

**响应（HTTP 200）**
```json
{"code":0,"msg":"success","data":[{"id":"...","name":"...","sex":"男","createdAt":"...","lastUsedAt":"..."}]}
```

#### [POST] /api/createIdentity
**描述**：创建身份（生成 32 位随机 id，写入 MySQL）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 说明 |
|---|---|---|
| name | 是 | 名字，非空 |
| sex | 是 | 仅允许 `男` / `女` |

**错误**
- name 为空：HTTP 400，`{"code":-1,"msg":"名字不能为空"}`
- sex 非法：HTTP 400，`{"code":-1,"msg":"性别必须是男或女"}`

#### [POST] /api/quickCreateIdentity
**描述**：快速随机创建身份（随机名字池 + 随机性别）。

#### [POST] /api/updateIdentity
**描述**：更新身份信息（同时更新 `last_used_at`）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| id | 是 |
| name | 是 |
| sex | 是 |

**错误**
- id/name 为空或 sex 非法：HTTP 400（对应 msg：`身份ID不能为空`/`名字不能为空`/`性别必须是男或女`）
- 身份不存在：HTTP 400，`{"code":-1,"msg":"身份不存在"}`

#### [POST] /api/updateIdentityId
**描述**：更新身份 ID（删除旧 id，插入新 id，保留 createdAt，更新 lastUsedAt）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| oldId | 是 |
| newId | 是 |
| name | 是 |
| sex | 是 |

**错误**
- 参数校验失败：HTTP 400（对应 msg）
- 更新失败：HTTP 400，`{"code":-1,"msg":"更新失败，可能旧身份不存在或新ID已被使用"}`

#### [POST] /api/deleteIdentity
**描述**：删除身份。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| id | 是 |

**错误**
- id 为空：HTTP 400，`{"code":-1,"msg":"身份ID不能为空"}`
- 身份不存在：HTTP 400，`{"code":-1,"msg":"身份不存在"}`

#### [POST] /api/selectIdentity
**描述**：选择身份（更新 last_used_at），返回该 identity。

**请求参数（query 或 form）**
| 参数 | 必填 |
|---|---|
| id | 是 |

---

### 4.3 Favorite（本地聊天收藏，MySQL/JPA）

控制器：`com.zcw.controller.FavoriteController`（Base：`/api/favorite`）  
存储：`chat_favorites`（JPA 自动建表/更新：`ddl-auto=update`）

#### [POST] /api/favorite/add
**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 说明 |
|---|---|---|
| identityId | 是 | 本地身份ID |
| targetUserId | 是 | 目标用户ID |
| targetUserName | 否 | 目标昵称 |

**处理逻辑**
- 同 `(identityId,targetUserId)` 去重：存在则直接返回已存在记录

**响应**
```json
{"code":0,"msg":"success","data":{"id":1,"identityId":"...","targetUserId":"...","targetUserName":"...","createTime":"..."}}
```

#### [POST] /api/favorite/remove
#### [POST] /api/favorite/removeById
#### [GET] /api/favorite/listAll
#### [GET] /api/favorite/check
**说明**：接口响应统一为 `{code,msg[,data]}`；`check` 的 `data` 为 `{isFavorite:boolean}`。

---

### 4.4 User History（上游历史/收藏/消息代理 + 缓存增强）

控制器：`com.zcw.controller.UserHistoryController`（Base：`/api`）  
上游域：`v1.chat2019.cn`（HTTP 表单 + 特定 Header + Cookie 透传）

#### [POST] /api/getHistoryUserList
**描述**：代理上游历史用户列表，并在可用时增强返回列表：补齐用户信息 + lastMsg/lastTime。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 默认值 |
|---|---|---|
| myUserID | 否 | `5be810d731d340f090b098392f9f0a31` |
| vipcode | 否 | `""` |
| serverPort | 否 | `"1001"` |
| cookieData | 否 | `""` |
| referer | 否 | `http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj` |
| userAgent | 否 | `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36` |

**上游调用**
- `POST http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_Random`
- Headers：`Host/Origin/Referer/User-Agent` + `Cookie`（若 cookieData 非空）

**增强逻辑（仅当上游 HTTP 200 且缓存服务存在）**
- 解析响应为 JSON 数组：
  - 优先使用 `id` 作为 userIdKey；若不存在则尝试 `UserID` 或 `userid`
- 批量增强：
  1. `batchEnrichUserInfo(list, idKey)`：补齐 `nickname/sex/age/address`
  2. `batchEnrichWithLastMessage(list, myUserID)`：补齐 `lastMsg/lastTime`
- 返回增强后的 JSON 数组字符串

**注意**
- 接口会输出分段耗时日志（upstream/enrich/lastMsg/total），不影响响应内容。

#### [POST] /api/getFavoriteUserList
**描述**：与 `/api/getHistoryUserList` 相同的增强逻辑，但上游接口不同。

**上游调用**
- `POST http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_My`

#### [POST] /api/reportReferrer
**描述**：上报访问记录（原样代理）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| referrerUrl | 是 |
| currUrl | 是 |
| userid | 是 |
| cookieData | 是 |
| referer | 否 |
| userAgent | 否 |

**上游调用**
- `POST http://v1.chat2019.cn/asmx/method.asmx/referrer_record`
- Headers 强制携带 Cookie：`headers.set("Cookie", cookieData)`

#### [POST] /api/getMessageHistory
**描述**：代理上游消息历史，并在新格式下缓存最后一条消息以提升列表页 lastMsg 命中率。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 默认值 |
|---|---|---|
| myUserID | 是 | - |
| UserToID | 是 | - |
| isFirst | 否 | `"1"` |
| firstTid | 否 | `"0"` |
| vipcode | 否 | `""` |
| serverPort | 否 | `"1001"` |
| cookieData | 否 | `""` |
| referer | 否 | 同上 |
| userAgent | 否 | 同上 |

**上游调用**
- `POST http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserMsgsPage`

**增强逻辑**
- 若响应 JSON 包含 `contents_list`（新格式）：
  - 取 `contents_list[0]` 作为“最新消息”，推断类型并写入最后消息缓存
  - 注意：当上游 `id/toid` 不含 `myUserID` 时，以请求参数 `(myUserID, UserToID)` 为准做补写归一化
  - **返回原始响应**（不修改 body）
- 若响应为 JSON 数组（旧格式）：
  - `batchEnrichUserInfo(list,"userid")` 后返回增强数组 JSON

#### [GET] /api/getImgServer
**描述**：原样转发上游获取图片服务器接口。

**上游调用**
- `GET http://v1.chat2019.cn/asmx/method.asmx/getImgServer?_=<ts>`

#### [POST] /api/updateImgServer
**描述**：仅更新本地 `ImageServerService` 内存中的 imgServerHost（不调用上游）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| server | 是 |

**响应（HTTP 200）**：`{"success":true}`

#### [POST] /api/uploadMedia
**描述**：上传媒体（图片/视频），本地落盘 + 上游上传 + DB 记录 + 缓存写入。

**请求（multipart/form-data）**
| 字段 | 必填 | 说明 |
|---|---|---|
| file | 是 | 图片/视频文件 |
| userid | 是 | 当前用户ID |
| cookieData | 否 | Cookie 字符串 |
| referer | 否 | 默认 referer |
| userAgent | 否 | 默认 UA |

**处理逻辑**
1. 校验 MIME：仅允许 `image/jpeg/png/gif/webp`、`video/mp4`
2. 计算 MD5（流式）
3. 通过 `FileStorageService.findLocalPathByMD5(md5)` 查找可复用的本地文件路径
4. 若无复用路径：按 MIME 分类保存到 `./upload/{images|videos}/yyyy/MM/dd/`，生成唯一文件名
5. 组装上游上传 URL：
   - `http://{imgServerHost}/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid={userid}`
6. 发送 multipart（字段名 `upload_file`），Headers：`Host/Origin/Referer/User-Agent` + 可选 Cookie
7. 若上游返回 JSON 且 `state=="OK"`：
   - `msg` 作为 `imagePath`（上游文件名）
   - 通过 `detectAvailablePort(imgServerHost)` 探测可用端口
   - 生成 `imageUrl`：`http://{host}:{port}/img/Upload/{imagePath}`
   - 构造 `MediaUploadHistory` 并调用 `mediaUploadService.saveUploadRecord(history)`（写入 `media_file`）
   - 写入图片缓存（按 localPath）
   - 返回增强后的 JSON（额外字段）：
     - `port`
     - `localFilename`（从 localPath 取 basename）
8. 若解析失败或非 OK：返回原始上游响应 body
9. 若上游上传失败：HTTP 500，`{"error":"上传媒体失败: ...","localPath":"..."}`（本地文件保留供重试）

#### [POST] /api/uploadImage（Deprecated）
**描述**：旧接口，内部转发到 `/api/uploadMedia`。

#### [GET] /api/getCachedImages
**描述**：返回用户缓存图片列表（本地 URL）以及探测到的可用端口。

**请求参数**
| 参数 | 必填 |
|---|---|
| userid | 是 |

**响应（HTTP 200）**
- 无缓存：
  - `{"port":"9006","data":[]}`
- 有缓存：
  - `{"port":"<detected>","data":["http://{host}/upload/...","..."]}`

#### [POST] /api/toggleFavorite
#### [POST] /api/cancelFavorite
**描述**：上游收藏/取消收藏（与本地 `/api/favorite/*` 无关）。

**上游调用**
- toggle：`POST http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Do`
- cancel：`POST http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Cancle`

**响应**
- 成功：返回上游响应 body（字符串）
- 异常：返回 `{"state":"ERROR","msg":"<err>"}`（HTTP 200）

---

### 4.5 Media History（本地媒体历史/图片库）

控制器：`com.zcw.controller.MediaHistoryController`（Base：`/api`）  
存储：`media_file` + `media_send_log`（由 `MediaUploadService` 管理）

#### [POST] /api/repairMediaHistory
**描述**：历史数据修复/清理接口，用于修复遗留表 `media_upload_history` 中 `file_md5` 缺失或重复的问题（补齐 MD5 + 按 MD5 去重）。

**能力**
- 可批量补齐 `media_upload_history.file_md5`（基于本地 `local_path` 读取文件计算 MD5）
- 可按 `file_md5` 全局去重（同一 MD5 仅保留 1 条记录）
- 可按 `local_path` 去重（仅针对 `file_md5` 仍为空的记录）
- 默认 dry-run：仅返回统计与样例；必须显式 `commit=true` 才会写入/删除（仅影响 DB，不删除物理文件）

**请求（application/json）**
| 字段 | 必填 | 默认 | 说明 |
|---|---|---|---|
| commit | 否 | false | 是否真实写入/删除（false=dry-run） |
| userId | 否 | 空 | **已废弃/忽略**（当前为全局修复，不按用户过滤） |
| fixMissingMd5 | 否 | true | 兼容字段：当前默认强制执行“补齐缺失 MD5” |
| deduplicateByMd5 | 否 | true | 兼容字段：当前默认强制执行“按 MD5 全局去重（同一 MD5 仅保留 1 条）” |
| deduplicateByLocalPath | 否 | false | 是否按本地路径去重（仅针对 MD5 为空；可选） |
| limitMissingMd5 | 否 | 500 | 单批扫描缺失 MD5 的数量（commit=true 会自动多批执行直至扫完整表） |
| maxDuplicateGroups | 否 | 500 | 单批处理重复分组数量（commit=true 会自动多批执行直至无重复分组） |
| sampleLimit | 否 | 20 | 返回样例数量上限 |

**响应（HTTP 200）**
- 返回统计：`missingMd5`、`duplicatesByMd5`、`duplicatesByLocalPath`，以及可选 `samples/warnings`。

#### [POST] /api/recordImageSend
**描述**：记录“媒体发送”关系，用于聊天历史图片查询。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 说明 |
|---|---|---|
| remoteUrl | 是 | 上游图片 URL |
| fromUserId | 是 | 发送者 |
| toUserId | 是 | 接收者 |
| localFilename | 否 | 本地文件名，用于更稳定关联 |

**处理逻辑（简述）**
- 优先 localFilename 匹配 `media_file`（先 userId 限定，再全局兜底）
- 其次按 remoteUrl / remoteFilename / basename 逐级兜底匹配
- 写入/更新 `media_send_log`，并刷新 `media_file.update_time`

**响应**
- `{"success":true,"message":"记录成功","data":{...}}`
- 找不到原始记录：`{"success":false,"message":"未找到原始上传记录"}`

#### [GET] /api/getUserUploadHistory
**描述**：查询某用户的上传历史（分页），并将 `remoteUrl` 改写为本地可访问 URL。

**请求参数**
| 参数 | 必填 | 默认值 |
|---|---|---|
| userId | 是 | - |
| page | 否 | `1` |
| pageSize | 否 | `20` |

**处理逻辑**
- 调用 `mediaUploadService.getUserUploadHistory(userId,page,pageSize,hostHeader)`：
  - 查询 `media_file`（native SQL，按 `update_time DESC`）
  - 将每条记录转换为 `MediaUploadHistory` DTO，并把 `remoteUrl` 替换为：
    - `http://{HostHeader}/upload{localPath}`
    - 若 Host 头缺失：`http://localhost:{server.port}/upload{localPath}`
- `total` 使用 `mediaUploadService.getUserUploadCount(userId)`
  - **现状说明**：该方法当前实现为 `mediaFileRepository.count()`（未按 userId 过滤）

**响应（HTTP 200）**
```json
{"success":true,"message":"查询成功","data":{"list":[...],"total":0,"page":1,"pageSize":20,"totalPages":0}}
```

#### [GET] /api/getUserSentImages
**描述**：查询某用户发送给某对方的媒体（分页）。

**请求参数**
| 参数 | 必填 | 默认值 |
|---|---|---|
| fromUserId | 是 | - |
| toUserId | 是 | - |
| page | 否 | `1` |
| pageSize | 否 | `20` |

**处理逻辑**
- 调用 `mediaUploadService.getUserSentImages(fromUserId,toUserId,page,pageSize,hostHeader)`：
  - 分页查询 `media_send_log`（按 `send_time DESC`）
  - 对每条 log 再查 `media_file` 补齐文件信息（找不到时用空对象兜底避免 NPE）
  - 同样将返回的 `remoteUrl` 改写为本地 URL：`http://{host}/upload{localPath}`
- `total` 使用 `mediaUploadService.getUserSentCount(fromUserId,toUserId)`

**响应（HTTP 200）**
```json
{"success":true,"message":"查询成功","data":{"list":[...],"total":0,"page":1,"pageSize":20,"totalPages":0}}
```

#### [GET] /api/getUserUploadStats
**描述**：查询上传统计。

**请求参数**
| 参数 | 必填 |
|---|---|
| userId | 是 |

**响应（HTTP 200）**
```json
{"success":true,"message":"查询成功","data":{"totalCount":0}}
```

> 现状说明：`totalCount` 取自 `mediaUploadService.getUserUploadCount(userId)`（当前未按 userId 过滤）。

#### [GET] /api/getChatImages
**描述**：获取两人双向聊天历史图片（用于上传弹出框历史图片预览）。

**请求参数**
| 参数 | 必填 | 默认值 |
|---|---|---|
| userId1 | 是 | - |
| userId2 | 是 | - |
| limit | 否 | `20` |

**响应**
- 成功：HTTP 200，返回字符串数组（本地 URL 列表）
- 失败：HTTP 500，返回 `[]`

#### [POST] /api/reuploadHistoryImage
**描述**：从本地文件重新上传到上游服务器（用于历史图片点击“重传”）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 | 默认值 |
|---|---|---|
| userId | 是 | - |
| localPath | 是 | - |
| cookieData | 否 | `""` |
| referer | 否 | 同上 |
| userAgent | 否 | 同上 |

**处理逻辑**
- `mediaUploadService.reuploadLocalFile(...)`：
  - 读取本地文件（`./upload`）
  - 构造上游上传 URL（同 `/api/uploadMedia`）
  - 上传成功后按 `localPath` 执行 `media_file.update_time` 刷新（忽略 userId，兼容不同存储形式）
- 若返回 JSON 且 `state=="OK"`：将该 `localPath` 写入 `ImageCacheService`（按 userId）

**响应**
- 成功：HTTP 200，返回上游响应 body（字符串）
- 失败：HTTP 500，`{"state":"ERROR","msg":"<err>"}`（字符串）

#### [GET] /api/getAllUploadImages
**描述**：“全站图片库”分页接口，返回对象数组（含文件元数据）+ 探测端口。

**请求参数**
| 参数 | 必填 | 默认值 |
|---|---|---|
| userId | 否 | - |
| page | 否 | `1` |
| pageSize | 否 | `20` |

**处理逻辑**
- `data`：`mediaUploadService.getAllUploadImagesWithDetails(page,pageSize,hostHeader)`
  - **现状说明**：该实现按 `media_file.update_time DESC` 查询全表，不按 userId 过滤（userId 参数仅为兼容旧调用）
  - 每条返回为 `MediaFileDTO`：`url/type/localFilename/originalFilename/fileSize/fileType/fileExtension/uploadTime/updateTime`
- `total`：`mediaUploadService.getAllUploadImagesCount()`（全表 count）
- `port`：通过 `detectAvailablePort(imageServerService.getImgServerHost())` 探测可用端口

**响应（HTTP 200）**
```json
{"port":"9006","data":[...],"total":0,"page":1,"pageSize":20,"totalPages":0}
```

#### [POST] /api/deleteMedia
**描述**：删除单个媒体文件（DB 记录 + 发送日志 + 可能的物理文件删除）。

**请求参数（application/x-www-form-urlencoded）**
| 参数 | 必填 |
|---|---|
| localPath | 是 |
| userId | 是 |

**兼容性说明**
- `localPath` 允许传入以下形式（后端会自动归一化后再执行删除）：`/images/...`、`images/...`、`/upload/images/...`、`http(s)://{host}/upload/images/...`（视频同理）。
- 当前 Go 实现中：`/api/getAllUploadImages` 不按 `userId` 过滤，因此 `/api/deleteMedia` 也不校验上传者归属（`userId` 参数仅用于兼容旧调用）。

**响应（成功，HTTP 200）**
```json
{"code":0,"msg":"删除成功","data":{"deletedRecords":1,"fileDeleted":true}}
```

**错误（HTTP 状态 + body）**
- 参数错误：HTTP 400，`{"code":400,"msg":"参数错误：..."}`
- 权限/业务错误：HTTP 403，`{"code":403,"msg":"..."}`
- 其他异常：HTTP 500，`{"code":500,"msg":"删除失败：..."}`

#### [POST] /api/batchDeleteMedia
**描述**：批量删除媒体文件（最多 50 个）。

**请求（application/json）**
```json
{"userId":"...","localPaths":["/images/...","/videos/..."]}
```

**参数校验**
- userId 为空：HTTP 400，`{"code":400,"msg":"用户ID不能为空"}`
- localPaths 为空：HTTP 400，`{"code":400,"msg":"文件路径列表不能为空"}`
- localPaths > 50：HTTP 400，`{"code":400,"msg":"单次最多删除50张图片"}`

**响应（HTTP 200）**
```json
{"code":0,"msg":"批量删除完成","data":{"successCount":0,"failCount":0,"failedItems":[]}}
```

---

### 4.6 System（系统管理）

控制器：`com.zcw.controller.SystemController`（Base：`/api`）

#### [POST] /api/deleteUpstreamUser
**描述**：调用上游删除用户接口。

**上游调用**
- `POST http://v1.chat2019.cn/asmx/method.asmx/Del_User`
- form：`myUserID`、`UserToID`、`vipcode=""`、`serverPort="1001"`

#### [GET] /api/getConnectionStats
**描述**：返回 WS 连接统计：`active/upstream/downstream/maxIdentities/availableSlots`。

#### [POST] /api/disconnectAllConnections
**描述**：断开所有上游与下游 WS 连接，清理延迟任务。

#### [GET] /api/getForceoutUserCount
**描述**：返回当前被 forceout 禁止的 userId 数量。

#### [POST] /api/clearForceoutUsers
**描述**：清空 forceout 禁止列表。

---

## 5. 静态资源与 SPA 回退

- 路由回退控制器：`com.zcw.controller.SpaForwardController`
  - `GET /`、`/login`、`/identity`、`/list`、`/chat`、`/chat/**` → `forward:/index.html`
- 上传文件访问：
  - `GET /upload/**` → 映射到本地目录 `./upload/`
