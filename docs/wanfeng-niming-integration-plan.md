# 晚风匿名聊天（wanfengniming.store）接入方案

> 目标：把该站点的「随机匹配 + 1v1 聊天」机制作为**第二条上游线路**接入本项目（现有上游为 chat2019.cn）。
> 状态：**P1/P2/P3 已实现**（握手 → 身份 → 登录 → 匹配 → 文字/图片/视频 → 历史/拉黑举报/匹配偏好），线路与现有 chat2019 并列可选。
> 最后更新：2026-11-07

## 实现状态（2026-11-07）

已完成（第一期：匹配 + 纯文字 + 并列线路）：

| 能力 | 状态 | 说明 |
|---|---|---|
| 加密层（ECDH P-256 + HKDF-SHA256 + AES-256-GCM） | ✅ | 纯标准库实现，零新依赖 |
| 会话握手/缓存/过期重建 | ✅ | 401/40310 自动重握手并重试一次 |
| 身份申请 + `identityToken` 持久化 | ✅ | 落库表 `identity_provider_binding` |
| WS 登录/心跳/断线重连/登录超时 | ✅ | 重连带 `identityToken` 且 `resetState=false` |
| 匹配 / 取消匹配 | ✅ | `random` / `randomOut` ↔ `join_match` / `cancel_match` |
| 纯文字收发 | ✅ | `touser_*` ↔ `send_message(text)`，回显用 md5 判定自身 |
| 对方离开 / 撤回 / 错误码映射 | ✅ | `code 16 / 9 / -4(forceout)` |
| 线路列表接口 + 前端线路选择 | ✅ | `/api/provider/list`，UI 在「设置 → 身份」面板 |
| 协议参数热修（不改代码） | ✅ | `WANFENG_PROTOCOL_OVERRIDES`（JSON 覆盖） |

### 二期：图片/视频（2026-11-07 已实现）

用户可以直接复用现有「本地上传 / mtPhoto 选图 / 抖音素材」的发送流程，后端在两条线路之间搬运字节：

| 能力 | 状态 | 说明 |
|---|---|---|
| 发送图片/视频 | ✅ | 前端照旧传到 `/api/upload` 得远程文件名 → 后端把文件搬到对方 `niming-file/upload` → 以 `type=image/video` + 对方 URL 发送 |
| 接收图片/视频 | ✅ | 后端下载对方媒体 → 上传到 chat2019 媒体服务器拿文件名 → 下发 `code=7` 的 `[文件名]`，前端渲染链路零改动 |
| 自己消息回显 | ✅ | 发送时记录「对方 URL → 本地文件名」映射，回显直接复用，前端乐观消息能正常合并 |
| 语音（audio） | ⚠️ 半支持 | 接收可转存，但前端语音渲染沿用 chat2019 编码，暂不保证体验 |
| 媒体开关 | ✅ | `WANFENG_MEDIA_ENABLED`（默认 true），关闭后媒体消息降级为系统提示 |

实现位置：
- `internal/upstream/wanfeng/media.go`：`[文件名]` 解析/还原、媒体消息载荷与 `MediaBridge` 接口；
- `internal/upstream/wanfeng/client.go`：下发队列（保序）+ 媒体搬运 + 接收转存 + 回显映射；
- `internal/upstream/wanfeng/httpx.go`：`UploadMedia` / `DownloadMedia`（含 Cookie Jar，对方按 Cookie 认定上传身份）；
- `internal/app/wanfeng_media.go`：chat2019 媒体服务器 ↔ 对方媒体服务的字节搬运与缓存。

#### 发送链路（图片/视频，前端零改动）

```
前端 uploadFile()               后端 /api/upload                chat2019 媒体服务器
   │  FormData(file)  ───────────►  uploadImgRandom  ──────────►  落盘
   │                              ◄── {state:OK, msg:"2026/11/07/Random/x.png"}
   │  ◄── {url: "http://img:7001/img/Upload/2026/.../x.png"}
   │
   │  WS 发送 {act:"touser_<uid>_<nick>", msg:"[2026/11/07/Random/x.png]"}
   ▼
Go 晚风线路（WanfengManager → Client.dispatchLoop）
   1) ParseMediaToken("[...]") → kind=image, path=2026/11/07/Random/x.png
   2) MediaBridge.MirrorOutbound：
      a. 先查 media_file 表读本地文件（命中就不用走网络）
      b. 否则 GET http://<imgServer>:<port>/img/Upload/<path>（端口按系统配置策略解析）
      c. POST <对方>/api/tools/niming-file/upload（multipart：file/userId/bucketName/fileHash，
         Cookie 携带身份，不加密） → {code:200,data:{filePath}}
      d. filePath 补成绝对 URL
   3) 发送帧 {type:"send_message", data:{targetUid, type:"image", content:<URL>, localTid}}
```

接收链路（对方推图片/视频）：

```
对方 WS 帧 receive_message {type:"image", payload:"/files/xxx.jpg", fromUid, tid}
   ▼
ParseReceiveMedia → kind=image, url=/files/xxx.jpg
   ▼
MediaBridge.MirrorInbound：
   1) GET <对方><url>（下载字节，≤80MB）
   2) POST http://<imgServer>/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=<uid>
      → {state:"OK", msg:"2026/11/07/Random/y.png"}
   3) 登记双向缓存
   ▼
下发前端 {code:7, type:"image", fromuser:{content:"[2026/11/07/Random/y.png]"}, ...}
   ▼
前端 parseMessageSegments → <img src="http://img:port/img/Upload/2026/.../y.png">
```

要点：

- **自己消息回显**必须复用发送时的 chat2019 文件名，否则前端「乐观消息」无法合并（`getOptimisticMatchKey` 按媒体路径 + 时间桶匹配）。实现方式：发送成功后在内存里记录 `对方 URL 键 → 本地文件名`（有界 256 条），回显命中就直接下发，不重复搬运。
- **多身份隔离**：对方用 Cookie 认定媒体归属，因此每个身份用 `APIClient.Clone()`（独立 Cookie Jar + 独立会话）并绑定自己的 `identityToken`，避免「图片传到别人名下」。
- **保序**：媒体搬运会阻塞，因此所有下发动作进 `outbox` 队列串行消费，保证「文字 → 图片」顺序不乱；搬运超时 60 秒，超时只影响该条。
- **接收不阻塞读循环**：`receive_message` 里是媒体的会放到 goroutine 处理，避免心跳/匹配事件被下载卡住。

#### 媒体相关的可热修参数（都在 `Protocol` 里）

| 字段 | 默认值 | 用途 |
|---|---|---|
| `uploadPath` | `/api/tools/niming-file/upload` | 媒体上传接口 |
| `uploadFileField` / `uploadUserField` / `uploadBucketField` / `uploadHashField` | `file` / `userId` / `bucketName` / `fileHash` | 表单字段名 |
| `bucketImage` / `bucketAudio` | `media-lib` / `chat` | 图片视频 / 语音的桶名 |
| `mediaUrlKeys` | `payload, mediaUrl, mediaURL, url, filePath, mediaId` | 接收消息里取媒体地址的候选键 |
| `uploadFilePathKeys` | `filePath, path, url, fileName` | 上传响应里取文件路径的候选键 |
| `refreshExpiryPath` | `/api/tools/niming-file/refresh-expiry` | 媒体过期时间续期（可选调用） |

对方改版时优先用 `WANFENG_PROTOCOL_OVERRIDES` 覆盖这些字段，例如：

```bash
WANFENG_PROTOCOL_OVERRIDES='{"uploadPath":"/api/v2/file/upload","uploadUserField":"uid","bucketImage":"chat-media"}'
```

### 三期：历史 / 拉黑举报 / 匹配偏好（2026-11-07 已实现）

这一期解决的是「线路功能残缺感」：晚风会话刷新后历史空白、拉黑举报发不出去、匹配偏好设不了。

#### ① 历史会话 / 联系人 / 聊天记录（后端分流，前端零改动）

老接口 `/api/getHistoryUserList`、`/api/getFavoriteUserList`、`/api/getMessageHistory`
不新增路径，而是**按身份绑定关系分流**：绑定过晚风的身份走晚风接口，其余继续走 chat2019。

判定依据是 `identity_provider_binding`（一个身份只绑一条线路），所以前端不需要把当前
线路传上来。

```
后端                                 晚风站点
/api/getHistoryUserList      ──►  POST /api/nimingChat/user/home-data?uid=&paged=1&pageSize=20
                                     → {relations[], relationsTotal, favorites[], favoritesTotal}
/api/getFavoriteUserList     ──►  POST /api/nimingChat/user/favorites?uid=&page=&pageSize=20
/api/getMessageHistory       ──►  POST /api/nimingChat/user/chat/messages?uid=&targetUid=&limit=&before=
                                     → {messages[], hasMore}
                                 ◄──  逐条翻译（媒体消息先转存成 [我方文件名]）
                                   ──►  前端认识的 contents_list
```

翻译要点：

- `relations[].targetUid/targetNickname/targetGender/targetAge` → 前端 `User` 字段；
  性别必须转成中文（`male→男`），前端按中文渲染图标。
- 历史消息里自己发的消息，`id` 填 `md5(身份ID)`（与一期实时消息一致），
  前端据此判定 `isSelf`。
- 历史消息里的图片/视频/语音先经二期媒体桥搬回我方媒体服务器，再以 `[文件名]` 下发，
  与实时消息走同一条渲染链路；转存失败的条目直接跳过（避免裂图）。
- 拉取后顺带写入聊天记录缓存（`chatHistoryCache`）与「最后一条消息」，
  保证与 chat2019 线路一致的离线可见体验。
- 上游失败时返回**空列表**而不是 5xx：前端 `loadHistoryUsers` 只处理数组，
  5xx 会让整块列表报错。

字段名与路径全部在 `Protocol` 里，对方改版时用 `WANFENG_PROTOCOL_OVERRIDES` 覆盖：

```bash
WANFENG_PROTOCOL_OVERRIDES='{"homeDataPath":"/api/v2/home","relationsPath":"/api/v2/relations"}'
```

#### ② 拉黑 / 举报 / 撤回 / 已读（WS 动作）

| 前端 act | 晚风动作 | payload |
|---|---|---|
| `warningreport`（拉黑，既有按钮） | `block_user` | `{targetUid}` |
| `unblockuser_<uid>` | `unblock_user` | `{targetUid}` |
| `reportuser_<uid>_<原因>_<备注>` | `report_user` | `{targetUid, reason, remark}` |
| `recall_<tid>` | `recall_message` | `{tid}` |
| `read_<uid>` | `read_messages` | `{targetUid}` |

- 举报调用后对方会推 `report_user {targetUid, success}`；**只有失败**才提示用户
  （成功由前端本地 toast，避免重复弹窗）。
- 收到对方消息时自动回一条 `read_messages`：我方前端没有已读 UI，但对方站点有，
  这条回执让对方看到「已读」。自己发的消息不触发。
- `messages_read` 事件只落日志（前端暂无已读展示），协议层已就位，后续加 UI 不用再改后端。
- 前端新增「举报该用户」菜单项 + 原因选择弹窗（`ChatHeader.vue` / `ChatRoomView.vue`）。

#### ③ 匹配偏好（`update_profile` 承载）

对方把偏好挂在用户资料上，只能通过 `update_profile` 下发生效，因此：

1. 本地落库 `identity_match_preference`（`prefer_gender / prefer_age_min / prefer_age_max / prefer_same_province`）；
2. 登录帧（`login`）与 `update_profile` 都带上 `matchPreference`；
3. 改偏好时若连接就绪，立即下发 `update_profile`；未就绪则下次登录生效。

```
GET  /api/getMatchPreference?identityId=        → {code, data, supported}
POST /api/updateMatchPreference                 → {code, data}
     identityId, preferGender(male/female/any), preferAgeMin, preferAgeMax, preferSameProvince
```

`supported` 表示该身份所在线路是否真支持偏好（chat2019 不支持 → 前端隐藏入口）。
前端入口在「匹配新用户」按钮的下拉菜单里（连续匹配下面），改完点保存才提交。
偏好归一化在服务端做：中文/大小写性别归一、负数归零、区间颠倒自动纠正，
避免脏值下发被对方拒绝。

#### ④ 收藏

前端「收藏/取消收藏」按钮原有的 `/api/toggleFavorite`、`/api/cancelFavorite`
在晚风线路上转成 WS 动作 `add_favorite` / `remove_favorite`，同时写本地收藏表
（对方列表有缓存延迟，本地立即生效）。响应码保持 `{code:"0", status:"true"}`
（前端判定条件是二者之一）。

#### 明确不支持（对方站点本身没有）

- **输入状态（正在输入）**：对方全部前端 bundle 里 `typing` / `输入中` / `正在输入` 零命中，
  42 个 WS 事件枚举里也没有这一项。协议层继续忽略 `inputStatusOn_*` 帧。
- **在线状态实时查询**：对方只有 `favorite_online`（收藏的人上线时推送），没有查询接口。

#### 仍待办

- 语音（录音 + 语音气泡渲染）：前端目前没有音频消息类型，晚风语音会被当作「文件」渲染。
  需要 `MediaRecorder` 录音 UI + `audio` 段类型 + 播放器组件，属独立一期。
- 线路健康度监控与自动熔断（当前仅日志 + 前端提示）。

---

## 一、结论

**可行**。该站点并非「匹配算法在客户端」的那种站，而是一个完整的服务端匹配服务：

- 传输层是 **一个 WebSocket（`/ws`）+ 一层 AES-GCM 加密信封**；
- 匹配、拉黑、撤回、已读、关系列表、聊天历史全部由服务端负责；
- 没有强依赖的服务端签名（客户端可自行实现同一套握手/加密），所以可以像现在代理 chat2019 一样，由 Go 后端做「协议适配器」。

已验证：用 Node 原型客户端（等价于 Go 实现）完成 `handshake → 取身份 → login → join_match`，成功进入匹配队列并真实收到 `match_result`。

---

## 二、对方站点机制（逆向结论）

### 2.1 HTTP 层：会话握手 + 加密信封

| 步骤 | 接口 | 说明 |
|---|---|---|
| 1 | `POST /api/app/handshake` | body `{"v":3,"pub":"<base64 客户端 ECDH P-256 公钥，65 字节未压缩点>"}`，返回 `{"data":{"sid","pub","exp"}}` |
| 2 | 派生密钥 | `shared = ECDH(P-256)`；`key = HKDF-SHA256(ikm=shared, salt=sid, info="wf-enc-v3", 32B)` |
| 3 | 业务接口 | 全部是 `POST`，body 为信封 `{"v":3,"sid","iv":"<b64 12B>","ct":"<b64 AES-GCM 密文+tag>"}`，AAD = `sid` |
| 4 | 响应解密 | `resp.data` 若为字符串且以 `v3.` 开头 → `v3.<b64 iv>.<b64 ct+tag>`，同 key / 同 AAD 解密得 JSON |

细节：

- 「GET 风格」接口靠请求头 **`X-Get-Via-Post: 1`** + 信封 body 实现（原参数放进明文 JSON 再加密）。直接 GET 会返回 `40300 接口已升级，请使用 POST 方式调用`。
- 会话有效期约 23 小时（`exp`），本地可缓存；`401 + code=40310` 时重建会话重试。
- 旧版本兼容层（RSA `JSEncrypt` 私钥 + `wf-enc-v15` 混淆信封）是**降级通道**，新实现不需要。
- 站点前面还有 WAF（ESA）：无浏览器 UA / 无 `Origin`、`Referer` 直接 GET `/api/**` 会被返回 403 HTML。`/ws` 的 Upgrade 请求本身不受影响。

### 2.2 身份体系

- `POST /api/tools/util/generate` → `{"uid":"WFU201NEX74Q7QYMVH0PEF1T","nickname":"柔软的企鹅"}`（uid 形如 `WFU2…`，另一类老身份形如 `WA…`）。
- 登录帧：`{"type":"login","data":{uid,nickname,gender,age,resetState,deviceId,fingerprint,hwfp,identityToken}}`
  - `gender`：`male` / `female` / `secret`；`age`：数字；
  - `deviceId`：随机 UUID（同设备多账号会被判 `DEVICE_DUPLICATE`）；
  - `identityToken`：首次为空，服务端在 `connected` 帧里下发（形如 `v1.xxx`），**重连时必须带上**；
  - `resetState`：重连时若本地处于匹配中/聊天中则传 false，保持服务端状态。
- 拿到 `identityToken` 后可 `POST /api/nimingChat/account/identity-cookie {identityToken}` 把身份绑到 Cookie（服务端「换发身份」机制，403 时用）。

### 2.3 WebSocket 协议

- 地址：`wss://wanfengniming.store/ws?_t=<毫秒时间戳>`（cookie 域内，`Origin` 必须同站）。
- 帧格式：`{"type":"<动作>","data":"v3.<iv>.<ct>","v":3,"sid":"<会话 sid>"}` —— **收发双向 `data` 都是加密串**。
- 动作枚举（客户端 → 服务端）：

| type | data | 用途 |
|---|---|---|
| `login` | 见 2.2 | 建连后第一帧 |
| `update_profile` | 昵称/性别/年龄/城市 | 改资料 |
| `join_match` | `{}` | 开始匹配 |
| `cancel_match` | `{}` | 取消匹配 |
| `set_match_preference` | `{"preferGender","preferAgeMin","preferAgeMax","preferSameCity"}` | 匹配偏好 |
| `send_message` | `{"targetUid","type","content","mediaId","duration","localTid","flash"}` | 发消息（type: text/image/audio/video/story/effect） |
| `recall_message` | `{"tid"}` | 撤回 |
| `read_messages` | `{...}` | 已读回执 |
| `view_flash_image` | `{"tid"}` | 闪照已看 |
| `exit_chat` | `{}` | 结束会话，回到 IDLE |
| `add_favorite` / `remove_favorite` / `block_user` / `unblock_user` / `report_user` | `{targetUid...}` | 关系操作 |
| `heartbeat` / `disconnect` / `switch_identity` | — | 心跳 / 主动断开 / 切身份 |

- 服务端 → 客户端（关键几类）：

| type | data 关键字段 |
|---|---|
| `connected` | `uid,cid,state(IDLE/MATCHING/IN_ROOM),nickname,gender,age,city,matchPreference,identityToken,privileges,isRegistered` |
| `user_info` | 同 connected 的资料部分（可能多次推送，`state` 变化） |
| `match_result` | `{"type":"chat","targetUid","partner":{"uid","cid","nickname","gender","age","city","tags"}}`；也有 `type:"direct"` 的直接会话 |
| `receive_message` | `{"tid","fromCid","fromUid","targetUid","type","payload","ts","duration","flash","readByRecipient","nickname","gender","age"}` |
| `message_recalled` | `{"tid"}` |
| `messages_read` | `{...}` |
| `partner_left` | `{"partnerLeft":true,"fromUid"}` |
| `error` | `{"code","message","clientMsg","remainingMs","permanent"}`，code 含 `IP_BANNED` / `USER_BANNED` / `DEVICE_DUPLICATE` / `IDENTITY_INVALID` / `ACTIVATION_*` |

### 2.4 匹配与聊天状态机

```
IDLE --join_match--> MATCHING --match_result(chat)--> IN_ROOM
MATCHING --cancel_match--> IDLE      （客户端 30s 无结果自动 cancel）
IN_ROOM --exit_chat--> IDLE          （对端会收到 partner_left）
```

- 匹配是**服务端全局池**，无「同 IP 不互配」的硬限制观测，但也不保证优先同池；
- `match_result` 后双方各发 `send_message`，服务端回显 `receive_message`（`fromCid` 判定自己/对方）。

### 2.5 实测验证记录

原型脚本（Node，等价 Go 实现）流程与结果：

1. `POST /api/app/handshake` → 200，拿到 `sid/pub/exp`；
2. `POST /api/tools/util/generate`（信封） → 拿到 `uid` + `nickname`；
3. `wss://…/ws?_t=…` + 加密 `login` 帧 → `user_info` / `connected`（含 `identityToken`）；
4. `join_match` → 立刻收到真实用户的 `match_result`；
5. 对端退出时收到 `partner_left`。

结论：**协议可完整复刻，无需浏览器环境**。

---

## 三、与我们现有架构的差异

| 维度 | 现有（chat2019 上游） | 晚风 |
|---|---|---|
| 帧格式 | `{"act":"random","id":…}` 明文 | `{"type":"join_match","data":"v3.…"}` 加密 |
| 匹配 | 上游服务端，用 `code=15` 回传对象 | 服务端，`match_result` 回传 partner |
| 身份 | 我们自定义 id（`act:"sign"`） | 上游下发 uid + `identityToken` 绑定 |
| 消息 | `code=7` + `fromuser.content` | `receive_message` + `payload` |
| 历史/联系人 | chat2019 REST 接口 | `/api/nimingChat/user/{home-data,relations,favorites,chat/messages}`（`GET-via-POST` 加密） |
| 媒体 | 我们自己的 `/api/upload` + `/upload/**` | 需上传到对方 `/api/tools/niming-file/upload` 拿 `mediaId` |
| 风控 | 80s forceout 防重连 | `IP_BANNED` / `USER_BANNED` / `DEVICE_DUPLICATE` / 加密层 |

---

## 四、推荐架构：Provider（线路）适配层

保留现有 chat2019 通路不动，**新增一个 provider 抽象**，前端 WS 上带 `provider` 参数：

```
frontend  ──/ws?token&provider=latewind──►  App.handleWebSocket
                                              │
                                    ┌─────────┴──────────┐
                              chat2019 代理        wanfeng 适配器
                            (现有 wsManager)   (internal/upstream/wanfeng)
                                                      │  握手/加密/身份/重连
                                                      ▼
                                              wss://wanfengniming.store/ws
```

适配器职责：**双向翻译**「我们前端的 act 协议」与「晚风的 type + 加密帧」，让前端 UI 不改（或极少改）。

后端关键组件：

1. `internal/upstream/wanfeng/crypto.go`：ECDH(P-256) + HKDF-SHA256 + AES-GCM 信封编解码（Go 1.25 标准库 `crypto/ecdh`、`crypto/hkdf`、`crypto/aes/cipher`，**零新依赖**）。
2. `internal/upstream/wanfeng/session.go`：会话缓存（sid/key/exp，401/40310 自动重建）。
3. `internal/upstream/wanfeng/client.go`：HTTP 客户端（`X-Get-Via-Post`、UA/Origin/Referer、信封 body、`v3.` 响应解密、RSA 降级可省略）+ WS 客户端（gorilla，Ping/Pong、断线重连、login 重放带 `identityToken`、`resetState=false`）。
4. `internal/upstream/wanfeng/adapter.go`：协议翻译（见第七节映射表），并把结果写入现有的 `userInfoCache` / `chatHistoryCache`。
5. `internal/app/wanfeng_manager.go`：仿 `UpstreamWebSocketManager` 的连接池，按我们自己的 `userId(identity id)` 复用一条晚风连接，多个下游会话共享。
6. DB：`identity` 表加列（或新表 `identity_provider_binding`）：`identity_id, provider, upstream_uid, identity_token, device_id, nickname, gender, age, created_at`。
7. 配置：`WANFENG_BASE_URL`（默认 `https://wanfengniming.store`）、`WANFENG_ENABLED`、`WANFENG_TIMEOUT`、可选的出站代理 `WANFENG_HTTP_PROXY`。

前端改动：

- 登录后拉 `/api/provider/list`（返回已启用的线路）；
- `connect()` 时带 `provider`；
- 匹配/取消/发消息沿用现有 `act`，不改；
- 线路切换 UI（身份可绑定到某线路）。

---

## 五、改造清单（后端）

| 文件 | 改动 |
|---|---|
| `internal/config/config.go` | 新增 `WanfengBaseURL`、`WanfengEnabled` 等 env |
| `internal/upstream/wanfeng/*.go` | 新增：crypto / session / client / adapter / types |
| `internal/app/wanfeng_manager.go` | 新增：连接池 + forceout 等价实现（对端断开/风控错误时冷却） |
| `internal/app/websocket_proxy.go` | `handleWebSocket` 按 `provider` 分派；保持现有 chat2019 路径不变 |
| `internal/app/router.go` | 新增 `/api/provider/list`、`/api/wanfeng/*`（历史/关系/上传转发） |
| `internal/app/identity_handlers.go` | 创建身份时可选「绑定线路」，落库上游 uid |
| `internal/app/user_info_cache.go` | 增加 provider 维度（或 uid 加前缀），避免两套 uid 冲突 |
| `internal/app/schema.go` | 新表/新列迁移 |
| `docs/websocket-code-reference.md` | 补充 wanfeng 帧→内部 code 的映射说明 |

## 六、字段映射表（适配器核心）

| 我方（下游 act） | 晚风帧 |
|---|---|
| `{"act":"sign","id":X}` | `login{uid: upstreamUID, nickname, gender, age, deviceId, identityToken}` |
| `{"act":"random","id":X,"userAge":"0"}` | `join_match{}` |
| `{"act":"randomOut",…}` | `cancel_match{}` |
| `{"act":"say","id":X,"touser":<peerUid>,"content":s}` | `send_message{targetUid:<peerUid>,type:"text",content:s,localTid}` |
| 断开 / 主动退出 | `exit_chat{}` |

| 晚风帧 | 我方（下游 code） |
|---|---|
| `connected{state:"IDLE"}` | `{code:12,content:"连接成功"}` |
| `connected{state:"MATCHING"}` | `{code:13}` |
| `match_result{type:"chat",targetUid,partner}` | `{code:15, sel_userid, sel_userNikename, sel_userSex, sel_userAge, sel_userAddress}` |
| `receive_message{type:"text"}` | `{code:7, tid, fromuser:{id,nickname,content,time}, touser:{id,nickname}}` |
| `message_recalled` | `{code:9}` |
| `partner_left` | `{code:16}` |
| `error{code:"IP_BANNED"/"USER_BANNED"/…}` | `{code:-4, forceout:true, content:"…"}`（复用现有拒绝逻辑） |
| `error{code:"DEVICE_DUPLICATE"}` | `{code:-3}` 或新 code，前端提示「同设备重复登录」 |

性别映射：`male/female/secret` ↔ 我们 `sex` 字段（`男/女/未知`）需统一。

---

## 七、风险与对策

| 风险 | 说明 | 对策 |
|---|---|---|
| 加密层变更 | 对方发版可能改 `wf-enc-v3` / 握手字段 / 帧结构 | 加密参数（AAD、info、v 号）全部配置化；握手失败时熔断降级到 chat2019 线路；加监控告警 |
| 风控/封 IP | 一个后端 IP 承载大量用户，最易触发 `IP_BANNED` | 每用户独立 `deviceId`；限流（单 IP 并发连接数、join_match 频率）；支持出站代理池；被封时自动禁用该线路一段时间 |
| `DEVICE_DUPLICATE` | 「同设备只能登录一个账号」 | 每身份生成并持久化独立 `deviceId`，绝不共用 |
| 身份被回收 | 长期不活跃或重置时 `IDENTITY_INVALID` | 持久化 `upstream_uid + identityToken`；失效时自动重建身份并提示用户 |
| 匹配不到（池子不同） | 用户期待「人多」，实际是别人家的池子 | 前端把「线路」做成可见选择项，并做线路健康度/命中率统计 |
| 无图片/语音 | 媒体要先上传到对方 `niming-file/upload` 再发 URL | 二期已实现：后端在 chat2019 媒体服务器与对方之间搬运字节（注意对方 URL 可能带签名过期，只做即时转发、不做长期代理） |
| 条款/合规 | 用其用户池为自己站点供流，可能违反对方服务条款，且双方用户会看到彼此内容 | 需业务侧确认授权/风险；技术上建议保留开关，随时可下线 |
| 心跳/断线 | 无心跳时服务端可能踢线（实测 `partner_left`） | 定时 `heartbeat` + gorilla ping；`connected` 后按 `resetState=false` 重连恢复 `IN_ROOM` |

---

## 八、分期计划

- **P0（1~2 天）**：`crypto.go` + `session.go` + 原型命令行（handshake → generate → login → join_match → 收到 match_result），补单测（固定向量）。
- **P1（2~3 天）**：`client.go` + `wanfeng_manager.go` + 适配器（匹配/取消/发消息/退出/错误映射），`/ws?provider=wanfeng` 打通真实聊天。
- **P2（1~2 天）**：身份绑定落库、`/api/provider/list`、前端线路选择、历史消息（`chat/messages`）与关系列表（`home-data`）转发。
- **P3（可选）**：图片/语音（转发上传）、匹配偏好、拉黑/举报、监控面板（线路状态/错误码统计）。

验证方法：

- `go test ./internal/upstream/wanfeng/...`（加密往返向量、错误码映射）；
- `go run ./cmd/liao` + 前端本地联调：匹配 → 收发文字 → 撤回 → 退出 → 重连恢复；
- 小流量灰度（`WANFENG_ENABLED` 白名单身份）。

---

## 九、已确认的产品决策（2026-11-07）

1. 线路**并列可选**（前端可选），不替换现有 chat2019 线路；
2. 第一期只做**匹配 + 纯文字**；第二期已补上**图片/视频**（后端搬运字节，前端零改动）；
3. 首要约束是**对方改版时我方改动要小** → 因此：
   - 所有易变参数（接口路径、帧动作名、加密版本/前缀/HKDF info、字段候选键）集中在 `Protocol`，支持 `WANFENG_PROTOCOL_OVERRIDES` JSON 热修；
   - 未知帧、解密失败、未知 act 一律降级忽略，不中断线路；
   - 帧 → 前端 code 的翻译集中在 `adapter.go`（纯函数 + 单测），改协议不牵动连接池与业务。

---

## 附：代码落地位置（P1）

| 文件 | 作用 |
|---|---|
| `internal/upstream/wanfeng/protocol.go` | 协议参数集中定义 + JSON 覆盖 + 容错取值（对方改版只改这里/配置） |
| `internal/upstream/wanfeng/crypto.go` | ECDH/HKDF/AES-GCM 信封 |
| `internal/upstream/wanfeng/session.go` | 会话缓存（singleflight 合并并发握手） |
| `internal/upstream/wanfeng/httpx.go` | HTTP 客户端（握手/身份/信封/响应解密/重试） |
| `internal/upstream/wanfeng/adapter.go` | act 协议 ↔ 晚风帧翻译（纯函数，单测覆盖） |
| `internal/upstream/wanfeng/client.go` | 单身份长连接（登录/心跳/重连/帧分发） |
| `internal/upstream/wanfeng/binding.go` | 绑定结构与存储接口 |
| `internal/app/wanfeng_manager.go` | 连接池、下游注册/广播、致命错误处理、统计 |
| `internal/app/wanfeng_binding.go` | 绑定落库（MySQL/PostgreSQL upsert） |
| `internal/app/websocket_wanfeng.go` | `/ws?provider=wanfeng` 入口 |
| `internal/app/provider_handlers.go` | `/api/provider/list` |
| `internal/upstream/wanfeng/media.go` | `[文件名]` ↔ 对方媒体 URL、媒体消息拼装、`MediaBridge` 接口 |
| `internal/app/wanfeng_media.go` | 媒体字节搬运（chat2019 媒体服务器 ↔ 对方上传接口）与双向缓存 |
| `sql/{mysql,postgres}/010_identity_provider_binding.sql` | 绑定表迁移 |
| `frontend/src/stores/provider.ts`、`frontend/src/api/provider.ts` | 线路状态与接口 |
| `frontend/src/composables/useWebSocket.ts` | WS 连接带 `provider` 参数（切换线路自动重连） |
| `frontend/src/components/settings/SettingsDrawer.vue` | 线路选择 UI |

联调命令（进入匹配队列前会先登录，注意别频繁跑，对方有同 IP 注册风控）：

```bash
# 手工检查握手 + 身份 + 登录
WANFENG_LIVE_TEST=1 go test ./internal/upstream/wanfeng/ -run TestLiveHandshakeAndLogin -v
# 手工检查「我方 /ws(provider=wanfeng) → 晚风」整条链路
WANFENG_LIVE_TEST=1 go test ./internal/app/ -run TestLiveWanfengWebSocketProxy -v
# 手工检查媒体上传/下载（会申请一个新身份）
WANFENG_LIVE_TEST=1 go test ./internal/upstream/wanfeng/ -run TestLiveMediaUploadAndDownload -v
```

> 注意：对方对「同 IP 短时间批量注册」有自动风控（`HANDSHAKE_FAILED` + `您的 IP 已被封禁（同 IP 短时间批量注册）`）。
> 频繁跑联调测试会触发；线上多人共用出口 IP 时也需要提前准备代理/限流（见风险表）。
> 触发风控后媒体联调会直接 `SKIP`，此时用已有身份复跑：
>
> ```bash
> WANFENG_LIVE_TEST=1 WANFENG_TEST_UID=WFU... WANFENG_TEST_TOKEN=v1.xxx \
>   go test ./internal/upstream/wanfeng/ -run TestLiveMediaUploadAndDownload -v
> ```
>
> **待补验证**：媒体上传接口（`niming-file/upload`）的真实链路目前只在本地 mock 站点验证过
> （`e2e_media_test.go` 用同一套握手/加密实现模拟服务端），真实站点因上述注册风控暂时无法复测。
> 复测点：①上传返回 `{code:200,data:{filePath}}`；②返回的 URL 能匿名下载；③`send_message(type=image)`
> 在对方页面能正常出图。
