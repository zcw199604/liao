# 晚风匿名聊天（wanfengniming.store）逆向知识库

> 用途：把对目标站点的协议逆向结果固化成可查的本地资料，供后续维护、排查、以及对方改版后快速定位差异。
>
> 抓取时间：2026-11-07
> 抓取方式：直接下载站点公开的前端静态资源（SPA bundle），静态分析得出；关键链路另行用我方 Go 客户端实测确认。
> 我方实现：`internal/upstream/wanfeng/`（协议层）+ `internal/app/wanfeng_*.go`（业务桥接）

## 目录

1. [站点技术栈](#一站点技术栈)
2. [HTTP 层：三道封装](#二http-层三道封装) — 握手、加密信封、POST 封装
3. [身份体系](#三身份体系) — uid/token/deviceId + 三套设备指纹
4. [WebSocket 协议](#四websocket-协议) — 42 个动作/事件全表、帧格式
5. [匹配与聊天状态机](#五匹配与聊天状态机) — 含拉黑时效、已读时机、隐藏已读、语音录制
6. [HTTP API 全集](#六http-api-全集47-个路径代码) — 47 个路径分组
7. [数据模型](#七数据模型) — 关系/收藏/消息字段
8. [风控与限制](#八风控与限制-这部分最容易踩) — 已实测撞到的墙
9. [我方实现映射表](#九我方实现映射表) — 对方机制 ↔ 我们的代码
10. [已知未接入的部分](#十已知未接入的部分)
11. [怎么重新抓取](#十一怎么重新抓取对方改版后)
12. [修订记录](#十二修订记录) — 本文修正过哪些错、还有哪些没验证

原始 JS 片段：[js-artifacts.md](./js-artifacts.md)

## 怎么读这份文档

每条结论都标了**来源可信度**，别把它们当成同等可靠：

| 标记 | 含义 |
|---|---|
| ✅ 实测 | 我方 Go 客户端真实连上对方服务端，跑通了 |
| 📖 代码 | 从对方前端 bundle 读出来的常量/逻辑，未在真实服务端验证 |
| ⚠️ 推断 | 从代码行为反推的语义，可能有别的解释 |

**改版应对优先级**：先看 `internal/upstream/wanfeng/protocol.go` 的默认值 → 用 `WANFENG_PROTOCOL_OVERRIDES` 热修 → 都不行再改代码。本文档的作用是让你知道"该改哪个字段"。

---

## 一、站点技术栈

| 项 | 值 | 来源 |
|---|---|---|
| 前端 | Vue 3 + Vite 打包的 SPA，hash 路由 | 📖 代码 |
| 入口 bundle | `index-*.js`（约 160KB），vendor 分包 `vendor-crypto-*.js` / `vendor-vue-*.js` | 📖 代码 |
| Web 加密 | WebCrypto API（`crypto.subtle`） | 📖 代码 |
| 站点标识 | 前端常量 `"wanfeng"` 会作为 `wanfeng` 字段塞进每个请求信封 | 📖 代码 |
| 客户端标记 | `wf_mark`（localStorage），默认值 `"xinghuosb"` | 📖 代码 |
| 版本号 | `appVersion = "3.0.94"` | 📖 代码 |

`wf_mark` 这个字段值得注意：它是 localStorage 里的一个可写值，默认 `xinghuosb`，每个加密请求都会带上。代码里没有任何地方校验它的值——看起来是**渠道/推广来源标记**，服务端可能按它做统计或风控分群。对方要求时可通过它识别来源，所以我方发 `wanfeng` 的默认值即可（我方实现里由 `Protocol` 决定，不硬编码）。

---

## 二、HTTP 层：三道封装

这是整个逆向里最关键的部分。对方不是把明文 JSON 直接 POST 上去，而是套了三层：

```
业务参数
  └─① 加密信封（ECDH 派生的 AES-GCM，或老 RSA）
      └─② JSON 包装 {"dataEnvelope": "<密文>"}
          └─③ HTTP POST + X-Get-Via-Post: 1
```

### 2.1 第一层：会话握手（ECDH P-256）

`POST /api/app/handshake`，请求体**明文**：

```json
{"v": 3, "pub": "<我方 P-256 公钥 raw 的 base64>"}
```

响应：

```json
{"success": true, "data": {"sid": "<会话ID字符串>", "pub": "<服务端 P-256 公钥 base64>", "exp": 1730000000000}}
```

密钥派生（📖 代码，✅ 我方已用 Go 标准库复现并通过真实握手）：

```
shared   = ECDH(P-256, 我方私钥, 服务端公钥)       // 32 字节
AESKey   = HKDF-SHA256(ikm=shared, salt=UTF8(sid), info=UTF8("wf-enc-v3"), len=32)
```

要点：

- **`salt` 就是 `sid` 字符串本身**（不是解码后的字节），`info` 固定 `"wf-enc-v3"`；
- 算法名 `"wf-enc-v3"` 和信封前缀 `"v3."` 是**两个独立常量**，改版时可能只改一个；
- 服务端下发的 `exp` 是毫秒时间戳；缺失时前端兜底 `now + 23h`；
- 前端缓存策略：`sessionStorage['wf_enc_v3']` 存 `{sid, k(base64 密钥), exp}`，**提前 5 分钟**过期重建（`oa = 5*60*1000`）；握手失败后 **60 秒**内不再重试（`Oo = 60*1000`）；
- 我方实现对应 `internal/upstream/wanfeng/crypto.go`，HKDF 是手写实现（为兼容 go 1.25 的 `crypto/hkdf` 缺失）。

### 2.2 第二层：加密信封

有两种形态，服务端**同时支持**，按能力探测自动选择。

**形态 A：v3 信封（正常路径，📖 代码 + ✅ 我方已实现）**

单个字符串三段，点号分隔：

```
v3.<base64(iv)>.<base64(ciphertext+tag)>
```

- `iv`：12 字节随机
- AES-GCM 参数：`tagLength: 128`，**AAD = UTF8(sid)**
- 明文 = `JSON.stringify(业务参数)`

⚠️ 注意区别：**HTTP 信封的密文是拼成 `v3.iv.ct` 字符串**，而 **WS 帧的加密信封是 `{v, sid, iv, ct}` 四字段对象**（见第三章）。两者密钥相同、AAD 相同，但序列化方式不同，别混用。

**形态 B：老 RSA 通道（降级路径，📖 代码）**

ECDH 不可用时回落。请求体变成：

```json
{"dataEnvelope": "<RSA 分块加密后以 | 连接的 base64>"}
```

明文是 `base64(encodeURIComponent(JSON.stringify({...业务参数, __enc: "wf-enc-v15", wanfeng: "xinghuosb"})))`，再按 **53 字节**分块 RSA 加密、`|` 连接、整体 base64。

⚠️ 这条通道的 RSA 私钥**明文硬编码在前端 JS 里**，等于没有加密保护。这意味着：任何拿到前端代码的人都能解开这层。我方的实现只走 v3 通道，不需要复现它；但如果哪天 v3 握手持续失败而站点还能用，说明对方把流量切回了 RSA 通道。

### 2.3 第三层：HTTP 封装

几乎所有"看起来是 GET"的业务接口，实际都是 **POST + 自定义头**：

```
POST /api/nimingChat/user/home-data
Content-Type: application/json
X-Get-Via-Post: 1
X-Enc-Sid: <sid>          // 仅当信封是 v3 形态时带

{"dataEnvelope": "v3.xxx.yyy"}
```

query 参数**不放在 URL 上**，而是解析进业务参数对象一起加密（📖 代码：`Lo()` 把 `?a=b` 拆出来放进 `params`，然后整个 `params` 进信封）。

⚠️ **`X-Get-Via-Post: 1` 的真实语义是「这个 POST 其实是个 GET」**（📖 代码 2026-11 版）：服务端看到该头就按 **GET 路由** 匹配。
因此对手方注册成 **POST-only** 的接口，带上这个头会直接

```
HTTP 404 {"success":false,"code":404,"message":"资源不存在: GET /api/xxm-nimingChat/account/identity-cookie"}
```

（`xxm-` 是内部 rewrite 前缀；消息里回显的方法/路径就是路由匹配时用的。）

判定依据是对方前端的两类调用方式：

| 前端封装 | 是否带 `X-Get-Via-Post` | 典型接口 |
|---|---|---|
| `ke()` / `$e.query()`（GET 风格业务接口） | 带 | `user/home-data`、`user/favorites`、`user/chat/messages`、`config/*`、`tools/util/generate` |
| `X()`（未包成 GET 的接口） | **不带** | `account/identity-cookie`（POST-only） |

对方前端把「必须按 POST 路由」的路径子串写死在 `_a` 常量里（`/health /app-info /user/identity /user/init /auth/ /config /account/identity-cookie /account/whoami /util/generate`），
但**只有 `identity-cookie` 是真正的 POST-only 路由**：`whoami` 相反，必须带该头，否则 404「资源不存在: POST /api/xxm-nimingChat/account/whoami」（已实测）。

我方对应实现：`Protocol.PostOnlyPaths`（默认 `["/account/identity-cookie"]`，可用 `WANFENG_PROTOCOL_OVERRIDES` 覆盖），
`httpx.go` 的 `post()` 命中这些路径时不再发该头。

**哪些路径必须加密**（📖 代码，正则 `^/api/(nimingChat|tools|bottle|risk|story)(/|$)`）：

必加密的 5 个前缀：`nimingChat` / `tools` / `bottle` / `risk` / `story`

**白名单不加密**（子串匹配）：

```
/health  /app-info  /user/identity  /user/init  /auth/
/config  /account/identity-cookie  /account/whoami  /util/generate
```

注意 `/util/generate` 在 `tools` 前缀下但被白名单放行 —— 这个接口不需要握手就能调，是用来拿新身份的，所以 onboarding 阶段能先拿到身份再建会话。

### 2.4 响应解密与软失败

响应统一 `{success, data, message, code?}`。`data` 是**加密字符串**时按同样方式解密（📖 代码 `Ca()`：看是否以 `v3.` 开头，是就走 v3，否则尝试老 RSA）。

⚠️ 一个坑，我方已经踩过：**`success:false` 不等于没有数据**。有些接口（如收藏相关）会返回 `success:false` 但 `code` 为空或 `"0"`，同时 `data` 里仍有正常内容。所以我方 `httpx.go` 的 `decodeResponseData` 加了 `hasDataField()` 判断：只有在没有 `data` 字段时才当成真失败。

### 2.5 错误码与重试（📖 代码）

| HTTP | code | 含义 | 前端行为 |
|---|---|---|---|
| 401 | `40310` | 会话密钥失效 | 清 sessionStorage 里的密钥 → 重握手 → 重试一次 |
| 403 | `40300` | 身份 Cookie 未就绪 / 与 uid 不匹配（媒体上传也用它表示 Cookie 失效） | 等身份写入完成（最多 10s）→ 重试 |
| 403 | `4031` | 同上另一分支 | 触发后台静默补发身份 Cookie |
| 404 | `404` | 路由不存在（常见原因：POST-only 接口带了 `X-Get-Via-Post`） | — |

我方实现：`internal/upstream/wanfeng/httpx.go` 对 401/40310 自动重握手并重试**一次**；
对 40300/4031（`IdentityNotReady`）由调用方重建身份 Cookie 后重试**一次**（不无限重试，避免触发风控）。

---

## 三、身份体系

### 3.1 三层标识

| 标识 | 生成方式 | 存储位置 | 作用 |
|---|---|---|---|
| `uid` | 服务端下发（`/api/tools/util/generate`） | localStorage + 服务端 Cookie | 业务主键，所有接口的主参数 |
| `identityToken` | 服务端下发（`identity-cookie` 接口） | localStorage `anon_identity` | 换回身份，避免重连变新用户 |
| `deviceId` | **客户端生成** UUID | cookie `_xxm_did`（10 年）+ localStorage `xdid` | 设备指纹的一部分，用于同设备风控 |

### 3.2 设备指纹（三者并存，📖 代码）

对方同时采集三套指纹，都很直白：

**① `deviceId`** — cookie `_xxm_did`，localStorage `xdid`，`crypto.randomUUID()`

**② `fingerprint`** — 格式 `fp4_<hash>`，参与哈希的字段：

```
["fpv4", canvas指纹, screen.colorDepth, "宽x高", devicePixelRatio,
 时区偏移, hardwareConcurrency, deviceMemory, maxTouchPoints,
 platform, userAgent, language, languages.join(",")].join("§")
```

canvas 指纹是标准的 240x60 画布 + 渐变 + 文字渲染后取图像数据。

**③ `hwfp`** — 格式 `hw1_<32位hex>`，存 localStorage `xhwfp`，参与字段：

```
["hw1", userAgent, platform, language, languages.join(","),
 "宽x高@色深", "dpr:x", "tz:时区名", "tzo:偏移",
 "cores:n", "mem:n", "touch:n", canvas指纹].join("|")
```

三者都随 `login` 帧上送。⚠️ 推断：`deviceId` 是强绑定（`DEVICE_DUPLICATE` 错误码说明同一浏览器只能登一个账号），`fingerprint`/`hwfp` 更可能是用于识别"同一台设备反复注册新身份"这类批量行为。

**我方实现的关键决策**：每个身份生成**独立且持久化**的 `deviceId`（存 `identity_provider_binding` 表），而不是每次连接重新随机。原因：随机 deviceId 会让对方看到"同一浏览器天天换设备"，反而更像批量号；固定 deviceId 才符合正常用户行为。

### 3.3 身份获取三件套

| 接口 | 作用 | 返回 |
|---|---|---|
| `GET/POST /api/tools/util/generate` | 申请全新身份（**免加密**） | `{uid, nickname}` |
| `POST /api/nimingChat/account/identity-cookie` | 用 `identityToken` 换回身份 Cookie（`Set-Cookie: wf_identity=<identityToken>`）；传 `{clear:true}` 是登出。**POST-only：不能带 `X-Get-Via-Post`，否则 404** | `{success}` |
| `POST /api/nimingChat/account/whoami` | 靠 Cookie 恢复当前身份 | `{uid, nickname}` |
| `GET /api/nimingChat/config/nickname/random` | 随机昵称 | `{nickname}` |

恢复顺序（📖 代码）：先试 `whoami`（靠 Cookie）→ 再试 `identityToken` → 都没有就 `generate` 新号。

### 3.3.1 身份 Cookie 是「历史/会话类接口」的前置条件 ⚠️

对方的 `user/home-data`、`user/favorites`、`user/relations`、`user/chat/messages` 全部按 **Cookie `wf_identity`** 认定身份，
`uid` 参数只是业务主键、不做鉴权。Cookie 缺失或与 `uid` 不匹配时统一返回：

```
HTTP 403 {"success":false,"code":40300,"message":"请求错误，请刷新页面","data":null}
```

推论与实测（2026-11-07，SCF 出口 106.53.99.253）：

1. 未绑定 Cookie → `home-data` 40300；`identity-cookie` 换到 Cookie（`wf_identity`）后同一请求立即 200；
2. `whoami` 靠 Cookie 反查身份，**不需要** `uid` 参数，它是判断「Cookie 是否已就绪」的最轻探针；
3. 对方前端遇到 `HTTP 403` 的处理是：「等身份写入完成（最多 10s）→ 重试一次」，不是换会话密钥；
   只有 `HTTP 401 + code 40310` 才是「会话密钥失效 → 重新握手」。

因此我方约定（`httpx.go` / `internal/app/wanfeng_client_cache.go`）：

- `APIError` 必须带上 HTTP 状态码，否则 40300 与 40310 无法区分；
- `wanfeng.IdentityNotReady(err)` 识别 `code=40300/4031` 或「带业务码的 HTTP 403」；
- **历史/收藏/聊天记录必须用「按身份克隆、独立 Cookie Jar、已绑定身份 Cookie」的客户端**（`wanfengIdentityClients`），
  不能用进程级共享客户端：多身份共用一份 Jar 会串号（A 的 Cookie 覆盖 B）；
- 收到身份类错误时重建 Cookie 再重试**一次**（对方前端同样只重试一次，避免风控）。

---

## 四、WebSocket 协议

### 4.1 连接

```
wss://<host>/ws?_t=<毫秒时间戳>
```

- `_t` 是**防缓存参数**，每次连接都不同（📖 代码 `buildWsUrl()`）；
- 无 token、无鉴权头 —— 身份完全靠连上之后发的 `login` 帧声明；
- 心跳：前端每 25s 发 `heartbeat`（我方 `Protocol.PingSeconds = 25`）；
- 登录超时：发 `login` 后 10s 没收到 `connected` 就断开重连（我方 `LoginTimeoutSeconds = 10`）。

### 4.2 帧格式

外层是明文 JSON（📖 代码，✅ 实测）：

```json
{"type": "login", "data": { ...业务字段... }}
```

📖 代码里 `FrameTypeField="type"`、`FrameDataField="data"`，另有 `sid` / `v` 字段但业务帧不使用。

**注意**：WS 的 `data` 是**明文对象**，不套 HTTP 那层 `v3.` 信封。加密只发生在 HTTP 层。（我方实现早期曾误以为 WS 也要信封，实测确认不需要。）

### 4.3 完整动作/事件枚举（42 个，📖 代码，已全量提取）

从 bundle 里的 TS enum 反编译，`K` 字段：

```
login                 connected             disconnect
heartbeat             user_info             update_profile
switch_identity       join_match            cancel_match
match_result          set_match_preference  direct_chat
send_message          receive_message       recall_message
message_recalled      read_messages         messages_read
view_flash_image      flash_image_viewed    exit_chat
partner_left          add_favorite          remove_favorite
block_user            unblock_user          report_user
bottle_notify         bottle_chat_invite    bottle_chat_response
bottle_chat_start     activate_code         activation_result
activation_kicked     check_code            check_code_result
story_notify          favorite_prompt       favorite_online
config_changed        public_column_event   error
```

⚠️ **重要：这里没有 `typing` / 输入状态**。整个 bundle 里 `typing`、`输入中`、`正在输入` 零命中。所以"对方能看到我在输入"这个功能物理上不存在，不是我们没接。

⚠️ 同样没有"在线状态查询"动作。唯一的在线信息是 `favorite_online`（收藏的人在线上线时服务端主动推），是**推送**不是**查询**。

### 4.4 login 帧（📖 代码 + ✅ 实测）

```json
{
  "type": "login",
  "data": {
    "uid": "<对方下发的 uid>",
    "nickname": "昵称",
    "gender": "male | female | secret",
    "age": 25,
    "resetState": true,
    "deviceId": "<持久化的 deviceId>",
    "fingerprint": "fp4_...",
    "hwfp": "hw1_...",
    "identityToken": "<换回身份的 token>"
  }
}
```

- `resetState`：**只有在 `state` 不是 MATCHING / IN_ROOM 时才是 `true`**。也就是说，如果重连时正处于匹配中或聊天中，要传 `false`，否则服务端会清掉你的房间状态。
- `identityToken` 是重连不变新号的关键，必须持久化。

我方 `Protocol.LoginPayload()` 另外注入了 `matchPreference` 字段（见第六章），对方前端自己的 login 帧里没有这个字段——但 `update_profile` 有，服务端能接受。

### 4.5 消息模型（📖 代码）

**发送** —— `send_message`：

```json
{"targetUid": "...", "type": "text", "content": "文字内容",
 "mediaId": "", "duration": 3000, "localTid": "local_xxx", "flash": false}
```

- `type` 取值：`text` / `image` / `video` / `audio` / `story` / `effect`
- `duration` 只对 `audio` 有意义（毫秒）
- `localTid` 是客户端自己生成的乐观消息 ID，格式 `local_<时间戳>_<4位随机>`
- `flash` 只对 `image` 有意义 = **闪图**（阅后即焚图片）

⚠️ **`mediaId` 是一个形同虚设的字段**。对方前端三处发送调用都显式传 `void 0`：

```js
// 图片/视频（MediaGalleryPanel 的 onSend）
U.sendMessage(_.filePath, c ? "video" : "image", void 0, void 0, oe);

// 语音（InputPanel 的 onSend）
U.sendMessage(_url, "audio", void 0, duration);
```

也就是说**媒体消息永远是 `content: <上传返回的 filePath URL>`，`mediaId` 一律不发**。
这是早期逆向时最容易搞反的一点：以为要"先传文件拿 ID 再发 ID"，实际发的是 URL 本身。
我方 `internal/upstream/wanfeng/media.go` 的 `SendMediaPayload()` 按真实行为实现（content 放 URL，不带 mediaId）。

**接收** —— `receive_message` 的 `data`：

```json
{"tid": "...", "fromCid": "self | <对方标识>", "fromUid": "...",
 "targetUid": "...", "type": "text", "payload": "内容或媒体URL",
 "ts": 1730000000000, "duration": 3000, "flash": false,
 "recalled": false, "nickname": "对方昵称"}
```

- `fromCid === "self"` 表示这条是自己发的（服务端回显）
- `payload` 对 text 是文字，对媒体类型是**媒体 URL**（不是 mediaId）
- `tid` 是服务端消息 ID

### 4.6 闪图机制（📖 代码）

`image` + `flash:true` 的消息：

1. 接收方看到模糊图，payload 里是真实 URL
2. 点开 → 发 `view_flash_image {tid}`
3. 服务端推 `flash_image_viewed` → 双方**清空本地 payload**，图片永久不可再看

我方未实现这个（前端没有对应 UI），但协议层是完整的。如果要加，需要一个"点开即焚"的组件。

### 4.7 WS 错误帧（📖 代码）

`error` 事件的 `data`：

```json
{"code": "...", "message": "...", "permanent": false,
 "clientMsg": "...", "remainingMs": 0, "expiresAt": "..."}
```

错误码全集：

| code | 含义 | 前端处理 |
|---|---|---|
| `IP_BANNED` | IP 被封禁 | 显示 `违反社区约定`，带剩余时长 |
| `USER_BANNED` | 用户被封禁 | 同上，`permanent` 决定是否永久 |
| `IDENTITY_INVALID` | 身份失效 | 停止重连，尝试恢复，失败则要求手动刷新 |
| `DEVICE_DUPLICATE` | 同设备重复登录 | 「同一浏览器只能登录一个账号」 |
| `ACTIVATION_EXPIRED` | 激活码过期 | 降到免费等级 |
| `ACTIVATION_BANNED` / `SUSPENDED` / `REVOKED` / `INVALID` | 激活码异常 | 降到免费等级 |
| `ACTIVATION_DUPLICATE_SESSION` | 激活码已在别的设备用 | 提示「同一时间仅允许一个用户使用」 |

另外三类**没有 code、靠 message 字符串匹配**的错误（对方用中文文案判断，很脆）：

| message 包含 | 处理 |
|---|---|
| `版本过旧` | 强制刷新页面 |
| `已被拉黑` 或 `拉黑关系` | 标记 `blocked=true`，清空这个人的会话和收藏，撤回本地乐观消息 |
| `敏感词` | 把本地最后一条乐观消息标为已撤回 |

⚠️ 对我方的启示：服务端**会按中文文案说话**，所以想完整复刻对方体验就必须做字符串匹配。

**我方现状（诚实记录）**：`adapter.go` 的 `Protocol.ErrorMessages()` 只处理**有 code 的**那些（`IP_BANNED`/`USER_BANNED`/`ACTIVATION_*`），其余按 message 原样透传给前端提示。**没有做**中文文案匹配，也就是说：

- 对方把我方拉黑后，我方发消息会收到「你与对方存在拉黑关系」的提示 —— 会正常弹给用户，但**不会**像对方前端那样把这个人从会话/收藏里清掉、也不会撤回本地乐观消息；
- 触敏感词时，我方那条乐观消息**不会**被标成已撤回，用户会看到一条永远"发送中"的消息。

这不是 bug，是没做的功能。要补齐的话，得在 `client.go` 的 `EventError` 分支里加文案判断。

### 4.8 多端互踢（📖 代码）

前端有个 `session_replaced` 回调，连续 3 次（`s=3`）触发就停止自动重连并提示「已在其他设备登录」。

⚠️ 这解释了为什么我方必须保证**一个身份只用一条上游连接**：两个下游会话各建一条上游连接，会互相踢下线，表现为无限重连风暴。

我方的处理：`WanfengManager` 按 identityID 复用同一条上游连接，多个下游会话共享。

---

## 五、匹配与聊天状态机

### 5.1 状态枚举（📖 代码）

```
IDLE      空闲
MATCHING  匹配中
IN_ROOM   聊天中
```

只有这三个状态。`resetState` 的取值取决于当前是不是后两个。

### 5.2 匹配流程

```
IDLE ── join_match ──► MATCHING ── match_result ──► IN_ROOM
  ▲                        │                          │
  └── cancel_match ────────┘                          │
  └────────────── exit_chat / partner_left ◄──────────┘
```

- `join_match {}` — 空 payload 进匹配池
- `cancel_match {}` — 取消
- `match_result` — 服务端推，`data` 里带 `{room: {targetUid, members[], ...}, ...}`（我方只取 `targetUid` 和成员昵称/性别/年龄）
- `exit_chat {}` — 主动退出
- `partner_left` — 对方退出（我方映射成 `code 16` 给前端）

### 5.2.1 状态机在服务端：房间没退就发 join_match 会被拒 ⚠️（✅ 实测）

状态不由客户端说了算，服务端按连接（uid）维护。实测（容器内直连 WS）：

| 场景 | 我方发送 | 服务端反应 |
|---|---|---|
| 登录后直接匹配 | `join_match {}` | 正常，推 `match_result` → IN_ROOM |
| IN_ROOM 时匹配 | `join_match {}` | `error {code:0, message:"当前状态不允许匹配"}`，房间不受影响 |
| IN_ROOM 时取消 | `cancel_match {}` | 无效果（它只作用于 MATCHING 队列，不离开房间） |
| IN_ROOM 时退出再匹配 | `exit_chat {}` → `join_match {}` | 正常退出并重新匹配成功 |

结论（这也是一类很隐蔽的线上故障）：

- **`cancel_match` 只取消「排队中」，`exit_chat` 才是「离开房间」**；两者不能混用。
- 前端「返回/退出聊天」如果只是清本地界面、不通知服务端，服务端仍认为这条连接在房间里——
  之后再点「匹配」就会一直回「当前状态不允许匹配」，直到连接断开。
- 我方处理：`client.go` 的 `deliver()` 在房间内重新匹配前先补发 `exit_chat`；房间内的
  `cancel_match`（前端 `act=randomOut`）改发 `exit_chat`；另外收到「不允许匹配」且确实刚发过
  `join_match` 时，自愈一次（`exit_chat` → 300ms → 重发 `join_match`，10 秒冷却），不把这个内部
  错误透给用户。

### 5.3 匹配偏好字段（📖 代码）

**真正生效的字段名**（这是本节最有价值的一条）：

```json
{
  "preferGender": "male | female",
  "preferSenderAgeMin": 22,
  "preferSenderAgeMax": 30,
  "preferSameProvince": true,
  "preferProvince": "广东"
}
```

⚠️ 注意是 `preferSenderAgeMin` / `preferSenderAgeMax`，**不是** `preferAgeMin`。这个命名先从 `/api/bottle/pick` 的参数里确认（📖 代码 `Ic()` 函数），再看 `update_profile` 用的是同一套。

⚠️ **`set_match_preference` 动作在枚举里存在，但对方前端从来不发它**。真正让偏好生效的路径是 `update_profile` 带 `matchPreference`。我方实测也确认：单独发 `set_match_preference` 不改变匹配结果。

偏好挂在**用户资料**上，不是独立配置。这就是为什么我方要本地落库、并在每次 `login` 时带上——它不会自动保存，必须每次声明。

### 5.4 拉黑 / 举报（📖 代码 + ✅ 我方已实现映射）

| 动作 | payload |
|---|---|
| `block_user` | `{targetUid}` |
| `unblock_user` | `{targetUid}` |
| `report_user` | `{targetUid, reason, remark}` |
| `add_favorite` | `{targetUid}` |
| `remove_favorite` | `{targetUid}` |

`report_user` 之后服务端回推 `report_user {targetUid, success}`。前端有个 `settleReport(targetUid, success)` —— 说明这是个**异步结算**：发出举报后挂起，收到回执才算完成。这个设计意味着举报有超时，不能假定立即成功。

⚠️ **拉黑不是永久的，每天零点自动解除**（📖 代码，来自前端确认弹窗文案原文）：

```
"拉黑后双方将无法继续聊天，每天零点自动解除。确认拉黑吗？"
```

这个「零点」是**服务端时区**的零点，不是浏览器本地零点。含义：

- 你拉黑某人 → 当天有效 → 次日 0 点后自动恢复可聊天状态；
- 所以「拉黑」在这个站点上是**当日静音**的语义，不是黑名单；
- 我方 UI 若按"永久拉黑"设计文案，会误导用户。

我方目前只是把前端 `warningreport` 动作直通成 `block_user`，没有实现"次日自动解除"的本地状态同步。如果要做得准确，需要在 relation 上记 `blockedAt`，跨过零点后自动清标记。

### 5.5 已读回执（📖 代码）

- `read_messages {targetUid}` — 告诉服务端"我读了这个人的消息"
- `messages_read` — 服务端推给**对方**，通知"他读完了"

**发送时机有个细节**（📖 代码）：对方前端不是收到消息就发，而是 `markVisibleMessagesRead(targetUid)` —— 只标记**当前视口内可见**的消息。这意味着：

- 打开会话但没滚到底 → 只对可见部分回已读；
- 页面在后台（`document.hidden`）→ 不回已读。

我方的实现是**收到即回**（`client.go` 的 `sendAutoRead`），比对方更"积极"。这在功能上没问题（对方看到已读得更快），但如果将来要做"隐藏已读"，就必须改成可控的。

### 5.6 「隐藏已读」机制（📖 代码）

这个功能存在两个层级，容易搞混：

**① 全局设置** —— `user.hideReadReceipts`，来自 `user_info` 帧，是**服务端字段**
**② 单人覆盖** —— `perUserHideRead[targetUid]`，存在 **localStorage** 的 `hide_read_prefs_<uid>`

```js
// 单人切换逻辑（原样）
function r(a){
  if(a in n) n[a] = !n[a];
  else { const s=we(); n[a] = !s.isHideReadReceipts }   // 首次以全局值取反
  o();  // 存 localStorage
}
```

⚠️ **单人覆盖完全是客户端本地行为**，服务端不知道。也就是说：

- 「隐藏已读」本质是"我方不回 `read_messages`"，是**客户端自制**的功能；
- 服务端没有对应开关，也无法校验；
- 所以任何客户端都可以选择不回已读 —— 这解释了为什么对方不提供"强制已读"。

对我方的启示：**我方的自动已读相当于强制"显示已读"，且无法被用户关闭**。如果用户从原站带过来"隐藏已读"的习惯，他会发现换到我方后已读藏不住了。要支持的话，需要在本地加一个开关，决定 `sendAutoRead` 是否跳过。

对方前端在打开会话时会发 `read_messages`。我方是收到消息即回一条（前端没有已读 UI，但要让对方看到已读）。

### 5.7 语音录制细节（📖 代码）

对方前端用 `MediaRecorder` 录，参数全在这里，我方可照抄：

**MIME 优先级**（逐个 `MediaRecorder.isTypeSupported` 试探，取第一个支持的）：

```js
["audio/mp4", "audio/webm;codecs=opus", "audio/webm",
 "audio/ogg;codecs=opus", "audio/wav"]
```

**文件命名与 bucket**：

```js
const ext = blob.type.includes("mp4") ? "m4a" : "webm";
form.append("file", blob, `voice_${Date.now()}.${ext}`);
form.append("userId", uid);
form.append("bucketName", "chat");   // 聊天输入面板录音用 chat
```

**时长上限**：60 秒（`Be = 60`）。

**上传后发送**：

```js
U.sendMessage(filePathURL, "audio", void 0, durationMs);
// 即 content=URL, type=audio, mediaId 不传, duration=毫秒
```

**采集约束**：

```js
getUserMedia({ audio: { echoCancellation: true, /* noiseSuppression 等 */ } })
```

**环境要求**：必须是安全上下文（HTTPS / localhost / 127.0.0.1），否则 `navigator.mediaDevices` 不可用。

**错误文案映射**（对方把 DOMException 翻成了中文，可复用）：

| 错误名 | 文案 |
|---|---|
| `NotAllowedError` / `PermissionDeniedError` | 麦克风权限被拒绝，请在浏览器设置中允许麦克风访问 |
| `NotFoundError` / `DevicesNotFoundError` | 未检测到麦克风设备 |
| `NotReadableError` / `TrackStartError` | 麦克风被其他应用占用，请关闭后重试 |
| `OverconstrainedError` | 麦克风不满足录音要求 |
| `AbortError` | 录音被中止，请重试 |

⚠️ **注意编解码格式跨端兼容性**：Chrome/Android 大概率选中 `audio/webm;codecs=opus`，Safari/iOS 会选中 `audio/mp4` → `.m4a`。所以**同一条语音在不同端上传的文件扩展名不同**。我方转存时必须按 blob 的真实 type 决定扩展名，不能硬编码 `.webm`，否则 iOS 用户录的语音在 Android 上播不出来。

---

## 六、HTTP API 全集（47 个路径，📖 代码）

从 bundle 里正则提取的全部 `/api/` 路径，按前缀分组。

### 6.1 握手与配置

```
/api/app/handshake                     会话握手（免加密）
/api/app/bootstrap                     启动引导
/api/nimingChat/config/app-info        应用信息（版本、公告开关）
/api/nimingChat/config/app-download-url
/api/nimingChat/config/notice          公告
/api/nimingChat/config/notice-detail   公告详情
/api/nimingChat/config/chat-effects    聊天特效清单
/api/nimingChat/config/public-column   公共栏目
/api/nimingChat/config/nickname/random 随机昵称
/api/nimingChat/config/user-tags       用户标签
/api/nimingChat/config/track-install-click
```

### 6.2 账号与身份

```
/api/tools/util/generate                     申请新身份（免加密）
/api/nimingChat/account/identity-cookie      身份 Token ↔ Cookie
/api/nimingChat/account/whoami               当前身份
/api/nimingChat/activation/my-codes          我的激活码
```

### 6.3 会话与消息（三期已接入）

```
/api/nimingChat/user/home-data               ⭐ 主页聚合：relations + favorites
/api/nimingChat/user/relations               关系（会话）列表
/api/nimingChat/user/relations/read          标记单个已读
/api/nimingChat/user/relations/read-all      全部已读
/api/nimingChat/user/relations/delete        删除单个会话
/api/nimingChat/user/relations/clear-all     清空所有会话
/api/nimingChat/user/favorites               收藏列表
/api/nimingChat/user/favorites/pin           收藏置顶
/api/nimingChat/user/chat/messages           ⭐ 历史消息（分页）
/api/nimingChat/user/notify-settings        通知设置
/api/nimingChat/user/personal-words          个人词库
/api/nimingChat/user/public-column
```

`home-data` 参数：`?uid=&paged=1&pageSize=20`，返回 `{relations[], relationsTotal, favorites[], favoritesTotal}`。
`chat/messages` 参数：`?uid=&targetUid=&limit=&before=`，返回 `{messages[], hasMore}`。**`before` 是游标**，不是页码。

### 6.4 分享名片（未接入）

```
/api/nimingChat/user/share/my|save|preview|reset|stats
/api/nimingChat/user/share/claim|decline
/api/nimingChat/user/share/rewards
/api/nimingChat/user/share/rewards/claim
```

机制：生成带 `?share=<code>` 的链接，别人打开能直接收藏你为好友。有"名额已满"和"已失效"两种失败态。

### 6.5 漂流瓶（未接入，独立功能线）

```
/api/bottle                    （前缀，具体路径动态拼接）
/api/bottle/throw              扔瓶子
/api/bottle/pick               捞瓶子
/api/bottle/quota              配额
/api/bottle/{type}/reply       回复
/api/bottle/my-thrown          我扔的
/api/bottle/my-picked          我捞的
```

⚠️ **`/api/bottle/pick` 的请求参数里藏了完整的匹配偏好字段**（`preferGender` / `preferSenderAgeMin` / `preferSenderAgeMax` / `preferSameProvince` / `preferProvince`）。如果你要确认偏好字段名对不对，这是最可靠的参照点。

### 6.6 文件与媒体（二期已接入）

```
/api/tools/niming-file/upload            ⭐ 上传（multipart）
/api/tools/niming-file/check-dedup       ⭐ 秒传检查（由 upload 路径字符串替换得到）
/api/tools/niming-file/list              文件列表
/api/tools/niming-file/my-media          我的媒体
/api/tools/niming-file/delete            删除
/api/tools/niming-file/copy-to-letter    复制到信箱
/api/tools/niming-file/refresh-expiry    续期（发消息前刷新有效期）
```

⚠️ `check-dedup` 在对方代码里不是独立常量，而是 `uploadUrl.replace("/upload", "/check-dedup")` 算出来的。**我方目前没有实现秒传**（每次上传都真实传字节），所以 `Protocol` 里没有这个路径字段；如果将来要加，建议做成独立常量而不是字符串替换。

**上传参数**（📖 代码 + ✅ 实测）：

| 字段 | 值 | 我方字段 |
|---|---|---|
| `file` | 文件本体 | `UploadFileField` |
| `userId` | uid | `UploadUserField` |
| `bucketName` | 见下 | `UploadBucketField` |
| `fileHash` | SHA-256 前 32 位 hex（可选） | `UploadHashField` |

⚠️ **`bucketName` 有四个实际取值**，别只记两个：

| bucket | 用在哪 | 我方常量 |
|---|---|---|
| `media-lib` | 图片、视频（媒体库选择器） | `BucketImage` |
| `chat` | 聊天输入面板录音 | `BucketAudio` |
| `voice` | 录音组件的默认兜底值 | — |
| （自定义） | 组件可接收 `bucket-name` prop | — |

**秒传流程**（📖 代码）：

```
1. SHA-256(文件) → 取 hex 前 32 位 → hash
2. POST /check-dedup   form: file=<空File占位,name=真实文件名>, hash, userId, bucketName
3. 返回 {code:200, data:{isDuplicate:true, filePath}} → 直接用 filePath，不传文件
4. isDuplicate 为假 → 走正式 upload，form 里额外带 fileHash
```

**续期**（📖 代码）：发图片前，若 `filePath` 的文件名匹配 `/^[0-9a-f]{24}$/`（这就是文件 ID），先调一次 `refresh-expiry`：

```js
const C = _.filePath.split("/").pop()?.split("?")[0];
if (C && /^[0-9a-f]{24}$/i.test(C)) {
  await fetch("/api/tools/niming-file/refresh-expiry", {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify(Qe({fileId: C, userId: Ze(), flash: oe}))
  });
}
```

业务参数：`{fileId: <24位hex>, userId: <uid>, flash: <bool>}`。`flash` 要和实际发送时的闪图标记一致（闪图的有效期规则不同）。不续期的话文件可能已过期，对方点开是 404。

⚠️ **这里有个未解之谜，写清楚免得后人踩**：这段代码用的是**裸 `fetch`**，只带 `Content-Type`，连 `X-Get-Via-Post` 都没有，但 `/api/tools/*` 属于需加密前缀。参数外面包了一层 `Qe()`，它是从共享 chunk 导入的（同 chunk 还导出 `Ke()` 取请求头、`De()` 解密响应），推测 `Qe()` = "把参数包成 `{dataEnvelope}` 并补齐加密所需的头"。

但那个 chunk（`index-Bmn6lqy2.js`）在我抓取时已经 404 —— 站点在此期间重新构建过，chunk hash 变了。所以 `Qe()` 的确切实现**没有验证**，上面这条标 ⚠️ 推断。

我方现状：**没有实现续期**（我们上传后立刻发送，文件不会过期，不需要续）。如果将来要支持"延迟发送已上传媒体"，必须先把这个接口打通，届时按需加密信封尝试（`X-Get-Via-Post: 1` + `{dataEnvelope: "v3..."}`）并实测。

响应 `{code:200, data:{filePath, isDuplicate?, expireAt?}}`（✅ 实测 filePath）。

⚠️ 上传响应的 `data` **一定是加密的**，而且不是 `v3.` 信封而是旧格式 RSA 分段（详见 [6.6.1](#661-媒体地址的加密回参--thumb-参数两个坑-️-实测-2026-11-07)）。我方 `httpx.go` 的 `decodeDataField` 已覆盖两套方案；自己写脚本调这个接口时，**必须解密**，否则 `filePath` 会是那一整段密文。

### 6.6.1 媒体地址的「加密回参 + thumb 参数」两个坑 ⚠️（✅ 实测 2026-11-07）

这两个坑分别是「我方发的图对方看不到」和「对方发的图我方加载失败」的根因，改媒体相关代码前必须先看：

**坑 1：上传接口的回参是「旧格式」RSA 密文，不是对象**

`/api/tools/niming-file/upload` 返回的 `data` 不是 `v3.*` 信封，而是站点**旧版**的加密形态：

```
data = base64( base64(RSA₁) | base64(RSA₂) | ... )      // 每段用 JS 里内嵌的 512 位私钥解
明文 = base64( JSON )                                    // {filePath, expireAt}
```

站点 JS（`app.js`）里对应 `Gs(e)`：`rn(e)` → 按 `|` 切开 → `ti.decrypt(o)` 逐段 RSA 解密 → 再 `rn()` 一次 → `JSON.parse`。
私钥（512 位，`BEGIN RSA PRIVATE KEY`）和公钥（`BEGIN PUBLIC KEY`）都**明文写在 JS 里**，任何访问者都能拿到。

实测解密结果：

```json
{"expireAt":1794046391353,"filePath":"/api/tools/niming-file/view/6aeef507c195fc86f9b8d57e"}
```

⚠️ 若把 `data` 当明文处理，`filePath` 就会变成那一大段 base64 密文，拼出来的"媒体地址"谁也打不开——
现象就是**我方发的图片对方看不到**（我们的日志里上传明明成功）。

我方实现：`internal/upstream/wanfeng/legacyrsa.go`（`openLegacyPayload`，用大数模幂避免 Go 1.24+ 对 512 位密钥的拒绝），
由 `httpx.go` 的 `decodeDataField` 统一分发（`v3.` 前缀走信封，其余走旧格式）。
对方换密钥时用 `WANFENG_LEGACY_RSA_KEY`（整段 PEM）覆盖，不用改代码。

**坑 2：取图片必须带 `thumb=1`，否则一律 403**

```
GET /api/tools/niming-file/view/<24位hex>          → 403 {"code":403,"message":"禁止访问"}
GET /api/tools/niming-file/view/<24位hex>?thumb=1  → 200 image/png   ✅
```

站点前端渲染图片时**自己会补**这个参数（`app.js` 的 `Ld()`/`F()`：`searchParams.set("thumb","1")`），所以浏览器里看不出问题；
但我们下载对方媒体时不补，就会吃到 403 —— 现象就是**对方发的图片我方加载失败**（视频地址不需要该参数，所以视频正常）。

- 发送侧：`content` 就是 `origin + filePath`（站点 JS：`Mt()`/`Ld()`），**不要**手动加 `thumb=1`，对方渲染时自己加。
- 接收侧：`httpx.go` 的 `DownloadMedia` 先按原样请求，失败且地址没有 `thumb` 参数时补 `thumb=1` 重试一次（`WithThumbParam`）。

**顺带排除的几个死路**（省得再试一遍）：

| 请求 | 结果 |
|---|---|
| `GET /api/tools/niming-file/download/<id>` | `40300 接口已升级，请使用 POST 方式调用` |
| `POST /api/tools/niming-file/download`（明文 JSON） | `40001 请求参数解密失败` |
| `POST /api/tools/niming-file/download`（v3 信封） | `404 资源不存在: POST /api/xxm-tools/niming-file/download` |
| `POST` + 旧格式 `{dataEnvelope}` | `40300 客户端版本过旧，请刷新页面更新` |
| `POST /api/tools/niming-file/list` | `404 资源不存在`（我方目前不需要，未深挖） |

结论：对方给浏览器的媒体访问路径就是 `view/<id>[?thumb=1]`，没有别的对外下载接口。

**bucket 归属**（📖 代码，`bucket-name` prop 实测）：随机聊天里图片/视频走 `media-lib`（`allow-video` 才允许视频），
语音走 `chat`，故事线/小屋走 `story` + `/api/tools/story-file/upload`，漂流瓶走 `bottle`。我方 `BucketImage=media-lib`、`BucketAudio=chat` 与此一致。

### 6.7 其他

```
/api/risk                   （前缀，风控相关，动态拼接）
/api/story                  故事线
/api/story/characters/avatar/{id}
/api/tools/public/topics/active
```

`story` 是一种特殊的消息类型：不是文字也不是媒体，而是"故事卡片"，`payload` 是 JSON 字符串 `{workId, title, plotTitle, characterAName, characterBName, messageCount}`。我方未接入（前端没有卡片渲染能力）。

---

## 七、数据模型（📖 代码，三期已按此翻译）

### 7.1 关系项（会话列表）

```json
{"uid": "自己", "targetUid": "对方", "targetNickname": "...",
 "targetGender": "male|female|secret", "targetAge": 25,
 "lastMessageAt": "ISO8601", "createdAt": "ISO8601",
 "unreadCount": 3, "pinned": false}
```

### 7.2 收藏项

```json
{"uid": "对方", "nickname": "...", "gender": "...", "age": 25,
 "lastMessageAt": "ISO8601", "pinned": false}
```

⚠️ 区别：关系项用 `targetUid` / `targetNickname`，收藏项用 `uid` / `nickname`。字段名不一致是对方的历史遗留，我方 `Protocol` 里用 `FavoriteUIDKeys` / `RelationTargetKeys` 两套键列表分别兜。

### 7.3 历史消息

```json
{"tid": "...", "fromCid": "self|...", "fromUid": "...", "targetUid": "...",
 "type": "text|image|audio|video|story|effect", "payload": "...",
 "ts": 1730000000000, "duration": 3000, "flash": false,
 "recalled": false, "nickname": "..."}
```

和 WS 的 `receive_message` payload **结构完全一致** —— 这是个好消息，同一套解析逻辑能复用。

---

## 八、风控与限制（⚠️ 这部分最容易踩）

### 8.1 已实测撞到的限制

| 现象 | 触发条件 | 我方对策 |
|---|---|---|
| `您的 IP 已被封禁（同 IP 短时间批量注册（自动风控）` | 短时间大量 `generate` 新身份 | 身份必须落库复用，禁止每次连接重新申请 |
| 握手返回 `HANDSHAKE_FAILED` | IP 被封期间 | 相同；已封期间任何握手都失败，只能等 |
| `DEVICE_DUPLICATE` | 同一 deviceId 上两个身份 | 每身份独立持久化 deviceId |
| 无限重连风暴 | 同身份两条上游连接互相踢 | 按 identityID 复用上游连接 |

### 8.2 推断的风控维度（⚠️ 推断，未验证）

- **IP + 注册频率**：已实测。批量注册是最敏感的指标。
- **deviceId 复用**：`DEVICE_DUPLICATE` 错误码证明存在设备维度限制。
- **fingerprint 相似度**：两个指纹都上送，很可能用于识别"同一设备换身份重注册"。
- **`wf_mark` 渠道标记**：可能用于分群统计，未验证是否影响限额。

### 8.3 我方当前策略

- 每个身份一个持久 `deviceId`；
- 身份 (`uid` + `identityToken`) 落 `identity_provider_binding` 表，重连复用；
- 握手失败 60s 内不重试（对齐对方前端行为）；
- 401/40310 只重试一次；
- 上游连接按身份复用。

---

## 九、我方实现映射表

| 对方机制 | 我方代码 | 可热修参数（`Protocol` 字段） |
|---|---|---|
| 握手 ECDH | `internal/upstream/wanfeng/crypto.go` | `HandshakePath`, `HKDFInfo`, `EnvelopeVersion`, `EnvelopePrefix` |
| HTTP 封装 | `internal/upstream/wanfeng/httpx.go` | `GetViaPostHeader`, `GetViaPostValue`, `SessionHeader` |
| 身份申请 | `internal/upstream/wanfeng/` + `Binding` 表 | `GenerateIdentityPath`, `IdentityCookiePath`, `UIDKeys`, `IdentityTokenKeys` |
| WS 收发 | `internal/upstream/wanfeng/client.go` | `WSPath`, `WSTimestampParam`, `PingSeconds`, `LoginTimeoutSeconds` |
| 帧翻译 | `internal/upstream/wanfeng/adapter.go` | `Action*`, `Event*`, `*Keys` 全部 |
| 历史列表 | `internal/app/wanfeng_history.go` | `HomeDataPath`, `RelationsPath`, `FavoritesPath`, `ChatMessagesPath` |
| 媒体搬运 | `internal/app/wanfeng_media.go` | `UploadPath`, `UploadFileField`, `BucketImage`, `BucketAudio` |
| 匹配偏好 | `internal/app/match_preference.go` | `ActionUpdateProfile` + `UpdateProfilePayload()` |

对方改版时的定位流程：

```bash
# 1. 先试热修（不用发版）
WANFENG_PROTOCOL_OVERRIDES='{"actionJoinMatch":"joinMatch2"}' go run ./cmd/liao

# 2. 重新抓 JS 对比常量（见第十一章脚本）
# 3. 改 protocol.go 默认值 + 补一个测试锁住新值
```

---

## 十、已知未接入的部分

按"是否值得做"排序：

| 功能 | 对方支持度 | 我方状态 | 备注 |
|---|---|---|---|
| 语音消息 | 完整（`MediaRecorder` + `type:audio` + `duration`） | 半支持（能收能转存，前端无音频类型所以按文件渲染） | 对方录制细节已逆向完（见 5.7），我方缺 `MediaRecorder` UI + 音频气泡组件 |
| 闪图 | 完整 | 未接入 | 需"点开即焚"组件 |
| 故事卡片 | 完整（`type:story`） | 未接入 | payload 是 JSON 字符串，需卡片渲染 |
| 聊天特效 | 完整（`type:effect` + `chat-effects` 清单） | 未接入 | 纯装饰，优先级低 |
| 漂流瓶 | 完整独立功能线 | 未接入 | 是一整套新 UI，不是聊天机制 |
| 分享名片 | 完整 | 未接入 | 有 `?share=` 深链机制 |
| 激活码体系 | 完整 | 未接入 | 涉及付费等级、特权 |
| 公共栏目 | 完整 | 未接入 | |
| 输入状态 | **对方没有** | 不做 | 全部 bundle 零命中 |
| 在线查询 | **对方没有** | 不做 | 只有 `favorite_online` 推送 |

---

## 十一、怎么重新抓取（对方改版后）

静态资源是公开的，直接下载即可。核心是拿到入口 HTML 引用的 JS 清单。

```bash
# 1. 取入口 HTML，找所有 <script src="...js">
curl -sS https://wanfengniming.store/ | grep -oE '/assets/[A-Za-z0-9_.\-]+\.js'

# 2. 逐个下载到本地（注意 vendor 分包也要拿，crypto 在里面）
mkdir -p /tmp/wf && cd /tmp/wf
for u in $(curl -sS https://wanfengniming.store/ | grep -oE '/assets/[A-Za-z0-9_.\-]+\.js'); do
  curl -sS "https://wanfengniming.store$u" -o "$(basename "$u")"
done

# 3. 提取关键常量，和本文档对照
grep -ohE '"wf-enc-v[0-9]+"|"/api/[a-zA-Z0-9_/-]+"|"v3\."' *.js | sort -u

# 4. 提取动作枚举（找 TS enum 反编译产物，特征是连续的 X="name" 赋值）
grep -oE '\(e\.\([A-Z_]+="[a-z_]+",' *.js | head -60

# 5. 确认没有新增输入状态事件
grep -c 'typing' *.js     # 期望 0
```

⚠️ 不要用真实账号跑自动化请求来"验证"改版。对方的 IP 风控很敏感（见 8.1），用单个持久化身份 + 低频探测就够。

---

## 附：原始代码片段

关键函数的压缩后原始 JS 片段存在 [`js-artifacts.md`](./js-artifacts.md)，用于对方改版时逐字对比。

---

## 十二、修订记录

### 2026-11-07 第三版（媒体两个坑：加密回参 + thumb）

| 项 | 之前 | 实测后 |
|---|---|---|
| 上传回参 `data` | 以为可能是 `v3.` 信封 | ❌ 实际是**旧格式 RSA 分段密文**（JS 内嵌 512 位私钥可解），必须解密才能拿到 `filePath`，详见 6.6.1 |
| 媒体访问地址 | 直接用 `origin + filePath` 下载 | 图片**必须**带 `?thumb=1`，裸地址 403「禁止访问」；站点前端自己补该参数 |
| `/niming-file/download` 等接口 | 以为是对外下载接口 | ❌ 浏览器不走它（GET 提示改用 POST、POST 要加密、旧格式又报版本过旧），媒体访问只有 `view/<id>?thumb=1` |
| `mediaId` | 第一版说"先上传拿 ID" | 维持第二版结论：聊天发送不带 `mediaId`，`content` 就是 URL |

### 2026-11-07 第二版（本次整理时发现并修正）

整理知识库过程中重新通读了全部 chunk，纠正/补充了以下内容。第一版（口口相传的结论）里的错误也标出来了：

| 项 | 第一版说法 | 修正后 |
|---|---|---|
| 媒体消息的 `mediaId` | "先上传拿 ID，再发 ID" | ❌ 错。真实客户端三处发送都传 `void 0`，`content` 放的就是 filePath URL |
| `bucketName` 取值 | 只有 `media-lib` / `chat` | 实际还有 `voice`（录音组件默认兜底） |
| 秒传 | 未记录 | 存在 `/check-dedup`，由 upload 路径字符串替换得到 |
| 续期 | 未记录 | 发图前需 `refresh-expiry`（实现细节未验证，见 6.6） |
| 拉黑时效 | 隐含"永久" | ❌ 实际**每天零点自动解除**，是"当日静音"语义 |
| 已读时机 | "打开会话就发" | 实际只对**视口内可见**消息发，且页面隐藏时不发 |
| 隐藏已读 | 未记录 | 存在，全局字段 + 单人 localStorage 覆盖，纯客户端行为 |
| 语音录制 | "对方支持但细节未知" | 已完整逆向：MIME 优先级、60s 上限、`voice_<ts>.m4a\|webm`、上传 bucket `chat` |

### 仍未验证的部分（明确列出，别当成结论用）

1. `Qe()` 信封包装器的确切实现（chunk 404，见 6.6）
2. `set_match_preference` 是否完全无效，还是仅在特定条件下生效（我方实测无效，但只测过一种场景）
3. `config/user-tags`、`user/personal-words`、`public-column` 三个接口的数据结构
4. `story` 故事线的完整接口族（只提取到 3 个路径）
5. `risk` 前缀下的具体路径（代码里是动态拼接，静态分析拿不到全集）
6. `wf_mark` 的具体用途（推断是渠道标记，无证据其影响风控）
7. 拉黑"零点自动解除"用的是哪个时区（推测服务端时区，未验证）
