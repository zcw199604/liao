# 晚风匿名聊天（wanfengniming.store）接入方案

> 目标：把该站点的「随机匹配 + 1v1 聊天」机制作为**第二条上游线路**接入本项目（现有上游为 chat2019.cn）。
> 状态：**P1 已实现并通过联调**（握手 → 身份 → 登录 → 匹配/文字收发链路可用），线路与现有 chat2019 并列可选。
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

二期（尚未实现，前端在该线路上会降级）：

- 历史会话/联系人列表：`/api/nimingChat/user/{home-data,relations,favorites,chat/messages}` 转发（当前历史列表只读 chat2019 上游，故晚风会话刷新后不回显）；
- 图片/语音：需转发对方 `niming-file/upload` 拿 `mediaId`；
- 输入状态、在线状态查询（`code 18/19/30`）、拉黑/举报、匹配偏好；
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
| 无图片/语音 | `mediaId` 需传到对方上传接口 | 一期只做文字 + emoji；二期转发上传并代理媒体 URL（注意其 URL 可能带签名过期） |
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
2. 第一期只做**匹配 + 纯文字**，图片/语音等富媒体放二期；
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
| `sql/{mysql,postgres}/010_identity_provider_binding.sql` | 绑定表迁移 |
| `frontend/src/stores/provider.ts`、`frontend/src/api/provider.ts` | 线路状态与接口 |
| `frontend/src/composables/useWebSocket.ts` | WS 连接带 `provider` 参数（切换线路自动重连） |
| `frontend/src/components/settings/SettingsDrawer.vue` | 线路选择 UI |

联调命令（不会进入匹配队列，不影响对方真实用户）：

```bash
# 手工检查握手 + 身份 + 登录
WANFENG_LIVE_TEST=1 go test ./internal/upstream/wanfeng/ -run TestLiveHandshakeAndLogin -v
# 手工检查「我方 /ws(provider=wanfeng) → 晚风」整条链路
WANFENG_LIVE_TEST=1 go test ./internal/app/ -run TestLiveWanfengWebSocketProxy -v
```
