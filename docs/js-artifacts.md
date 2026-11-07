# 晚风站点关键 JS 片段（原始压缩代码）

> 配套文档：[wanfeng-reverse-engineering.md](./wanfeng-reverse-engineering.md)
> 用途：对方改版时，把这些片段和线上最新 bundle 逐字对比，快速定位是哪个常量/字段变了。
> 抓取时间：2026-11-07，来源 `wanfengniming.store` 公开静态资源。

说明：以下是从 Vite 打包产物里原样摘出的压缩代码（变量已美化过命名的地方会注明）。压缩变量名（`e`/`t`/`n`/`o`）是混淆产物，每次发版都会变，**对比时只看字符串常量和结构，不要对比变量名**。

---

## 1. 加密通道核心常量

```js
const ta="/api/app/handshake",   // 握手路径
      na="wf-enc-v3",            // HKDF info
      ui="v3.",                  // 信封前缀
      cn="wf_enc_v3",            // sessionStorage key
      oa=5*60*1e3,               // 密钥提前过期余量 5 分钟
      Oo=60*1e3;                 // 握手失败后的冷却 60 秒
```

对比要点：改版时 `na` 和 `ui` 可能**不同步**变更（一个改 v4 另一个还是 v3）。两个都要查。

---

## 2. ECDH 握手与密钥派生

```js
async function la(){
  const e=wt();                                    // crypto.subtle
  if(!e)return null;
  try{
    const t=await e.generateKey({name:"ECDH",namedCurve:"P-256"},!1,["deriveBits"]),
          n=await e.exportKey("raw",t.publicKey),
          o=await fetch(`${fi}${ta}`,{
            method:"POST",credentials:"include",
            headers:{"Content-Type":"application/json"},
            body:JSON.stringify({v:3,pub:pt(n)})   // ← 请求体是明文
          });
    if(!o.ok)return null;
    const r=await o.json(),i=r&&r.data;
    if(!r.success||!i||typeof i.sid!="string"||typeof i.pub!="string")return null;
    const a=await e.importKey("raw",ln(i.pub),{name:"ECDH",namedCurve:"P-256"},!1,[]),
          s=await e.deriveBits({name:"ECDH",public:a},t.privateKey,256),
          c=await e.importKey("raw",s,"HKDF",!1,["deriveBits"]),
          l=new Uint8Array(await e.deriveBits({
              name:"HKDF",hash:"SHA-256",
              salt:rt.encode(i.sid),               // ← salt 就是 sid 字符串
              info:rt.encode(na)                   // ← "wf-enc-v3"
            },c,256)),
          m=typeof i.exp=="number"?i.exp:Date.now()+23*3600*1e3;
    return aa(i.sid,l,m),{sid:i.sid,key:await di(l),exp:m};
  }catch{return null}
}
```

⚠️ 注意 `salt: rt.encode(i.sid)` —— **直接把 sid 字符串做 UTF-8 编码当 salt**，不是 base64 解码。这是最容易实现错的地方。

---

## 3. 信封封装 / 解封（HTTP 与 WS 两套）

### 3.1 HTTP 信封（字符串形态）

```js
async function fa(e,t){
  const n=crypto.getRandomValues(new Uint8Array(12)),
        o=await wt().encrypt({
          name:"AES-GCM",iv:n,
          additionalData:rt.encode(e.sid),   // ← AAD = sid
          tagLength:128
        },e.key,rt.encode(JSON.stringify(t)));
  return{v:3,sid:e.sid,iv:pt(n),ct:pt(o)};  // 对象形态
}
```

但真正发出去时转成了字符串（另一个函数）：

```js
return `${ui}${pt(n)}.${pt(o)}`;   // → "v3.<b64(iv)>.<b64(ct)>"
```

解封：

```js
async function vi(e,t){
  const n=t.split(".");
  if(n.length!==3)throw new Error("bad v3 envelope");
  const o=await wt().decrypt({
        name:"AES-GCM",iv:ln(n[1]),
        additionalData:rt.encode(e.sid),tagLength:128
      },e.key,ln(n[2]));
  return JSON.parse(sa.decode(o));
}
```

### 3.2 WS 帧（对象形态，与 HTTP 不同）

```js
async function da(e,t){
  const n=crypto.getRandomValues(new Uint8Array(12)),
        o=await wt().encrypt({...同上...},e.key,rt.encode(JSON.stringify(t)));
  // 返回对象而非字符串
}
```

⚠️ **两者容易混淆**。以我方实测为准：业务 WS 帧的 `data` 是**明文对象**，不套信封。这个对象形态的信封只用于某些内部路径。

---

## 4. 老 RSA 降级通道

```js
const xs=`-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAKUX1WcGR7eHEOCNhN/Jo4SmtONnLhZM
NxlxBEshxfNaMFW8nL0aeXNOX2ylJltSjDbVpfINeBT7MfXktULZI8sCAwEAAQ==
-----END PUBLIC KEY-----`;

const Vs=`-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBAKUX1WcGR7eHEOCNhN/Jo4SmtONnLhZMNxlxBEshxfNaMFW8nL0a
eXNOX2ylJltSjDbVpfINeBT7MfXktULZI8sCAwEAAQJAZvE6lvgahfzJcEuuO+C+
fnt4tYdj/HFfthR7qeGQThG/ev0qVv8JYuivTLeKFGQmt3uFuAAxPF7yyzSQQ2J7
cQIhAOeBF1jJBWIiBGo3dIBsYR15i9beqaHR9k/JmjLGruMjAiEAto/RxD//9vYq
I5ThuP8sTLgTIX0V76EuGMeq/jjquzkCIDoj3oQqwBJcLrWqRbT3AxDO9MK62ulL
/Mse+33jDgr3AiEAsMPQeBT/4j+3zXv6kwV//0i8PVDATKXj3Fp4Vm8F2QECIQDb
dOMuqVftSVHyAB4rHHRk0a1xSexYwx5gx9nMmQkGEw==
-----END RSA PRIVATE KEY-----`;   // ← 私钥硬编码在前端，等于没有保护

fr=53;  // 单块最大明文字节数
```

```js
function Ta(e){
  const t={...e, __enc:"wf-enc-v15", wanfeng:ai()},
        n=JSON.stringify(t),
        o=btoa(encodeURIComponent(n));
  return {dataEnvelope: Ws(o)};   // Ws = RSA 分块加密 + "|" 连接
}
```

对比要点：`"wf-enc-v15"` 是这条通道的版本号，和 v3 通道的 `"wf-enc-v3"` 完全无关。分块 53 字节 = 512bit RSA 的 PKCS#1 v1.5 上限（64 - 11）。

---

## 5. 请求封装（GET→POST 转换）

```js
const Ia="X-Get-Via-Post",
      ha="X-Enc-Sid",
      ga=/^\/api\/(nimingChat|tools|bottle|risk|story)(\/|$)/,   // 需加密的前缀
      _a=["/health","/app-info","/user/identity","/user/init","/auth/",
          "/config","/account/identity-cookie","/account/whoami","/util/generate"];  // 白名单
```

```js
async function it(e,t,n={}){
  const o=e.split("?")[0],
        r=n.method||"POST",                       // ← 默认 POST
        i={"Content-Type":"application/json"};
  n.getViaPost && (i[Ia]="1");                    // ← 关键头

  const a=async()=>{
    const m=await ka(t);                          // 把 params 加密
    return {
      res: await fetch(No(e),{
        method:r, credentials:"include", headers:i,
        body: JSON.stringify(m.body)
      }),
      sess:m.sess
    };
  };
  ...
  // 401 + code 40310 → 清密钥重握手重试一次
  let l=await s.json();
  return c && s.status===401 && l && l.code===ya && (ho(), {...重试...}, l=await s.json()),
         l && typeof l=="object" && (l.data=await Ca(l.data,o,c)),
         {status:s.status, ok:s.ok, json:l};
}
```

query 参数处理（证明参数走信封而非 URL）：

```js
function Lo(e){
  const t=e.indexOf("?");
  if(t===-1)return{path:e,params:{}};
  const n=e.slice(0,t),
        o=new URLSearchParams(e.slice(t+1)),
        r={};
  o.forEach((i,a)=>{r[a]=i});
  return{path:n,params:r};   // ← params 随后整个进信封
}
```

---

## 6. 客户端标记

```js
const ii="wanfeng",     // 塞进每个信封的字段名
      si="wf_mark",     // localStorage key
      tr="xinghuosb";   // 默认值

function ai(){
  if(lt)return lt;
  try{ lt=L.getItem(si)||tr }catch{ lt=tr }
  return lt;
}
```

每个加密请求的信封里都会带 `wanfeng: <这个值>`。代码中无任何地方校验它 → 推断是渠道/推广标记。

---

## 7. 动作 / 事件枚举（42 个）

```js
var me=(e=>(e.IDLE="IDLE",e.MATCHING="MATCHING",e.IN_ROOM="IN_ROOM",e))(me||{}),
    ze=(e=>(e.MALE="male",e.FEMALE="female",e.SECRET="secret",e))(ze||{}),
    K=(e=>(
      e.LOGIN="login", e.CONNECTED="connected", e.DISCONNECT="disconnect",
      e.HEARTBEAT="heartbeat", e.USER_INFO="user_info",
      e.UPDATE_PROFILE="update_profile", e.SWITCH_IDENTITY="switch_identity",
      e.JOIN_MATCH="join_match", e.CANCEL_MATCH="cancel_match",
      e.MATCH_RESULT="match_result", e.SET_MATCH_PREFERENCE="set_match_preference",
      e.DIRECT_CHAT="direct_chat", e.SEND_MESSAGE="send_message",
      e.RECEIVE_MESSAGE="receive_message", e.RECALL_MESSAGE="recall_message",
      e.MESSAGE_RECALLED="message_recalled", e.READ_MESSAGES="read_messages",
      e.MESSAGES_READ="messages_read", e.VIEW_FLASH_IMAGE="view_flash_image",
      e.FLASH_IMAGE_VIEWED="flash_image_viewed", e.EXIT_CHAT="exit_chat",
      e.PARTNER_LEFT="partner_left", e.ADD_FAVORITE="add_favorite",
      e.REMOVE_FAVORITE="remove_favorite", e.BLOCK_USER="block_user",
      e.UNBLOCK_USER="unblock_user", e.REPORT_USER="report_user",
      e.BOTTLE_NOTIFY="bottle_notify", e.BOTTLE_CHAT_INVITE="bottle_chat_invite",
      e.BOTTLE_CHAT_RESPONSE="bottle_chat_response",
      e.BOTTLE_CHAT_START="bottle_chat_start",
      e.ACTIVATE_CODE="activate_code", e.ACTIVATION_RESULT="activation_result",
      e.ACTIVATION_KICKED="activation_kicked", e.CHECK_CODE="check_code",
      e.CHECK_CODE_RESULT="check_code_result", e.STORY_NOTIFY="story_notify",
      e.FAVORITE_PROMPT="favorite_prompt", e.FAVORITE_ONLINE="favorite_online",
      e.CONFIG_CHANGED="config_changed",
      e.PUBLIC_COLUMN_EVENT="public_column_event", e.ERROR="error", e
    ))(K||{});
```

⚠️ **总数 42，无 `typing`，无在线查询动作**。这是"输入状态对方不支持"的原始证据。

---

## 8. 登录帧构造

```js
onOpen: async () => {
  const N=we(),                                       // 本地用户缓存
        le=N.user?.nickname||h,
        U =N.user?.gender||k,
        te=N.user?.age??b,
        D =N.user?.state==="MATCHING",
        ae=N.user?.state===me.IN_ROOM,
        Se=await Rs().catch(()=>""),                  // fingerprint
        ue=await Ms().catch(()=>""),                  // hwfp
        Re=L.getItem("anon_identity")||void 0;        // identityToken

  const Te={
    uid: F, nickname: le, gender: U, age: te,
    resetState: !D && !ae,        // ← 匹配中/聊天中必须 false
    deviceId: Ze(),               // ← 持久化 deviceId
    fingerprint: Se, hwfp: ue,
    identityToken: Re
  };
  d({type:K.LOGIN,data:Te});      // ← 明文帧，无信封
  f=window.setTimeout(()=>{...登录超时${_/1e3}s...}, _);   // _ = 5000? 见下
}
```

⚠️ 超时常量在上文出现过 `_=5e3`（5 秒）和别处的 10 秒。我方实现取 10 秒（更宽松，因为跨境网络慢）。这个值不影响协议正确性。

---

## 9. 设备指纹

```js
// ① deviceId
const sn="xdid", jr="_xxm_did", Kr=10*365*24*60*60;
function Ze(){
  let e=qr()||localStorage.getItem(sn);
  return e||(e=crypto.randomUUID?crypto.randomUUID():
    `${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`),
  Yr(e), localStorage.setItem(sn,e), e;
}

// ② fingerprint
async function Rs(){
  if(Wt)return Wt;
  const e=navigator;
  let t=""; try{t=await Ts()}catch{}        // Ts = canvas 指纹
  const n=["fpv4",t,String(screen.colorDepth),`${screen.width}x${screen.height}`,
    String(devicePixelRatio??1),String(new Date().getTimezoneOffset()),
    String(e.hardwareConcurrency??0),String(e.deviceMemory??0),
    String(e.maxTouchPoints??0),e.platform||"",e.userAgent||"",
    e.language||"",(e.languages||[]).join(",")].join("§");
  return Wt="fp4_"+await Xr(n), Wt;         // Xr = 哈希
}

// ③ hwfp
async function Ms(){
  if(Gt)return Gt;
  try{const r=localStorage.getItem(Yo);
      if(r&&/^hw1_[0-9a-f]{32}$/.test(r))return Gt=r,r}catch{}
  const e=navigator;
  let t=""; try{t=Intl.DateTimeFormat().resolvedOptions().timeZone||""}catch{}
  const n=["hw1",e.userAgent||"",e.platform||"",e.language||"",
    (e.languages||[]).join(","),`${screen.width}x${screen.height}@${screen.colorDepth}`,
    `dpr:${devicePixelRatio??1}`,`tz:${t}`,
    `tzo:${new Date().getTimezoneOffset()}`,
    `cores:${e.hardwareConcurrency??"?"}`,`mem:${e.deviceMemory??"?"}`,
    `touch:${e.maxTouchPoints??"?"}`,bs()].join("|");   // bs = canvas
  const o="hw1_"+await Xr(n);
  Gt=o; try{localStorage.setItem(Yo,o)}catch{} return o;
}
```

三套指纹并存，全部随 login 上送。

---

## 10. 消息发送

```js
function a(d, y="text", g, E, C){
  if(!t.value)return;
  if(o.value){G({message:"你与对方存在拉黑关系，无法发送消息",...});return}

  const B=t.value.targetUid,
        R=`local_${Date.now()}_${Math.random().toString(36).slice(2,6)}`;

  n.value.push({                     // ← 本地乐观插入
    tid:R, fromCid:"self", targetUid:B, type:y,
    payload: y==="text"?d:g||d,
    ts:Date.now(),
    duration: y==="audio"?E:void 0,
    flash: y==="image"&&C?!0:void 0
  });

  i({type:K.SEND_MESSAGE,data:{
    targetUid:B, type:y, content:d, mediaId:g,
    duration:E, localTid:R,
    flash: y==="image"&&C?!0:void 0
  }});
}
```

`localTid` 格式 `local_<Date.now()>_<4位36进制>`。我方必须发同样的格式，否则对方回执匹配不上（我方实测确认服务端会原样回传 `localTid`）。

⚠️ **三处调用都显式传 `void 0` 给 mediaId**，媒体消息实际发的是 URL：

```js
// 图片/视频
U.sendMessage(_.filePath, c ? "video" : "image", void 0, void 0, oe);
// 语音
U.sendMessage(_url, "audio", void 0, duration);
// 特效
i({type:K.SEND_MESSAGE,data:{targetUid:g,type:"effect",content:y,localTid:E}});
```

顺带的媒体发送路径：

```js
function s(d){   // 发故事卡片
  const E={workId:d.workId,title:d.title,plotTitle:d.plotTitle,
    characterAName:d.characterAName,characterBName:d.characterBName,
    messageCount:d.messageCount};
  n.value.push({tid:g,fromCid:"self",targetUid:y,type:"story",
    payload:JSON.stringify(E),ts:Date.now()});
  i({type:K.SEND_MESSAGE,data:{targetUid:y,type:"story",content:d.workId,localTid:g}});
}
```

⚠️ 故事卡片的 `payload` 是 JSON 字符串，但发送时 `content` 只传 `workId`。

---

## 11. 服务端错误处理（中文文案匹配）

```js
function ql(e,t,n){
  if(e.message&&e.message.includes("版本过旧")){ /* 强制刷新 */ return }

  if(e.code==="IDENTITY_INVALID"){ n.stopReconnect(); ... return }

  if(e.code==="IP_BANNED"||e.code==="USER_BANNED"){
    const s=e.permanent??!1,
          c=typeof e.clientMsg=="string"?e.clientMsg.trim():"",
          l=typeof e.remainingMs=="number"?e.remainingMs:
            e.expiresAt?Math.max(0,new Date(e.expiresAt).getTime()-Date.now()):null;
    n.markBanned({message:"违反社区约定",reason:"违反社区约定",
      remark:null,clientMsg:c||null,permanent:s,remainingMs:l});
    ...
  }

  if(e.code==="DEVICE_DUPLICATE"){ /* 同一浏览器只能登录一个账号 */ return }
  if(e.code==="ACTIVATION_EXPIRED"){ ... }
  if(e.code==="ACTIVATION_BANNED"||e.code==="ACTIVATION_SUSPENDED"
   ||e.code==="ACTIVATION_REVOKED"||e.code==="ACTIVATION_INVALID"){ ... }
  if(e.code==="ACTIVATION_DUPLICATE_SESSION"){ ... }

  // ⚠️ 以下三类靠中文文案，没有 code
  if(e.message&&(e.message.includes("已被拉黑")||e.message.includes("拉黑关系"))){
    t.blocked=!0; ... return
  }
  if(e.message&&e.message.includes("敏感词")){ /* 标记乐观消息已撤回 */ }
}
```

⚠️ **这是对方的设计缺陷**，但意味着我方也必须做字符串匹配才能正确处理拉黑和敏感词。文案是"已被拉黑"/"拉黑关系"/"敏感词"/"版本过旧"/"版本过旧"这类固定短语，中文，不带标点变化。

---

## 12. 匹配偏好字段（最可靠的参照）

`/api/bottle/pick` 的参数构造，直接暴露了完整字段名：

```js
async function Ic(e,t,n,o,r,i,a){
  const s=Ze(), c={uid:e,nickname:t,deviceId:s};
  n&&(c.preferGender=n),
  o!=null&&(c.preferSenderAgeMin=o),      // ← 注意：preferSenderAgeMin
  r!=null&&(c.preferSenderAgeMax=r),      // ← 不是 preferAgeMin
  i&&(c.preferSameProvince=!0),
  a&&(c.preferProvince=a);
  En();
  const l=await Nt(pe("/bottle/pick"),c);
  ...
}
```

`pe(e)` 就是 `` `/api/bottle${e}` ``。

⚠️ `set_match_preference` 在枚举里，但**全站搜索没有一处发送它**。偏好实际通过 `update_profile` 的 `matchPreference` 字段生效。

---

## 13. 上传响应与路径

```js
const xi=He("config",()=>{
  const e=A(""),                                  // wsUrl
        t=A("/api/tools/niming-file/upload"),     // ← 上传路径
        n=A(""),                                  // appName
        o=A("3.0.94"),                            // ← appVersion
        r=A(!1);
  ...
});

function a(){                                     // buildWsUrl
  const c=e.value; if(!c)return"";
  const l=c.includes("?")?"&":"?";
  return `${c}${l}_t=${Date.now()}`;              // ← 防缓存参数
}
```

---

## 14. 多端互踢

```js
onSessionReplaced: () => {
  if(f&&(clearTimeout(f),f=null), c&&(clearTimeout(c),c=null), a++,
     W("WS",`连续 session_replaced 第 ${a} 次`), a>=s)      // s=3
    return i.value=!0, n.value=!1, y(),
           W("WS","已在其他设备登录，停止自动重连"), !0;
}
```

连续 3 次才停止。我方对策是根本不让它发生：**一个身份只复用一条上游连接**。

---

## 15. 复现用速查命令

```bash
# 拉全站 JS 到本地
mkdir -p /tmp/wf && cd /tmp/wf
curl -sS https://wanfengniming.store/ \
  | grep -oE '/assets/[A-Za-z0-9_.\-]+\.js' \
  | while read u; do curl -sS "https://wanfengniming.store$u" -o "$(basename "$u")"; done

# 版本常量是否变化
grep -ohE '"wf-enc-v[0-9]+"|"v3\."|"wf-enc-v15"' *.js | sort -u

# 动作枚举是否有新增（数个数，现在应该是 42）
grep -oE 'e\.[A-Z_]+="[a-z_]+"' *.js | wc -l

# 确认依然没有输入状态
grep -c 'typing\|正在输入\|输入中' *.js

# API 路径全集
grep -ohE '"/api/[a-zA-Z0-9_/-]+"' *.js | sort -u

# 匹配偏好字段名是否改了
grep -ohE 'prefer(Gender|SenderAgeMin|SenderAgeMax|SameProvince|Province)' *.js | sort -u
```

把输出和本文各节对照，差异就是改版点。

---

## 16. 上传 / 秒传 / 续期

```js
// 正式上传
const P=new FormData;
P.append("file", D, D.name);
P.append("userId", _);
P.append("bucketName", d);
ee && P.append("fileHash", ee);            // SHA-256 前 32 位 hex
const W=await(await fetch(F.uploadUrl,{method:"POST",credentials:"include",
  headers:Ke(),body:P})).json();
W.data = await De(W.data, F.uploadUrl);    // ← 响应 data 也可能加密
if(W.code===200 && W.data?.filePath){ ... }

// 秒传检查（路径是字符串替换出来的，不是常量）
const U = F.uploadUrl.replace("/upload","/check-dedup");
const p=new FormData;
p.append("file", new File([], D.name, {type:D.type}), D.name);   // ← 空文件占位
p.append("hash", ee);
p.append("userId", _);
p.append("bucketName", d);
const z=await(await fetch(U,{...})).json();
if(z.code===200 && z.data?.isDuplicate && z.data?.filePath){ /* 直接用它 */ }
```

续期（发图前，文件名是 24 位 hex 时）：

```js
const C = _.filePath.split("/").pop()?.split("?")[0];
if(C && /^[0-9a-f]{24}$/i.test(C)){
  const G = Ze();
  if(G) await fetch("/api/tools/niming-file/refresh-expiry",{
    method:"POST", headers:{"Content-Type":"application/json"},
    body: JSON.stringify(Qe({fileId:C, userId:G, flash:oe}))
  });
}
```

`bucketName` 实际取值：`media-lib`（图片/视频）、`chat`（聊天录音）、`voice`（录音组件默认兜底）。

---

## 17. 语音录制

```js
const Be = 60;   // 时长上限 60 秒

function Re(){   // 选 MIME
  const s=["audio/mp4","audio/webm;codecs=opus","audio/webm",
           "audio/ogg;codecs=opus","audio/wav"];
  for(const r of s) try{ if(MediaRecorder.isTypeSupported(r)) return r }catch{}
  return "";
}

async function D(C){                       // C = Blob
  const m = X();                           // uid
  if(!m) return null;
  const _ = C.type.includes("mp4") ? "m4a" : "webm";   // ← 扩展名随 MIME 变
  const P = new FormData;
  P.append("file", C, `voice_${Date.now()}.${_}`);
  P.append("userId", m);
  P.append("bucketName", r?.bucketName ?? "voice");
  const V = await(await fetch(F.uploadUrl,{method:"POST",credentials:"include",
    headers:Q(),body:P})).json();
  V.data = await Z(V.data, F.uploadUrl);
  if(V.code===200 && V.data?.filePath){
    const N = V.data.filePath;
    return N.startsWith("http") ? N : `${window.location.origin}${N}`;
  }
  return null;
}
```

发送（`InputPanel` 里的 handler）：

```js
function A(_, c){ U.sendMessage(_, "audio", void 0, c) }
// _ = filePath URL, c = 时长毫秒
```

---

## 18. 隐藏已读（纯客户端行为）

```js
// localStorage key: hide_read_prefs_<uid>
function r(a){                                  // 切换某个人的隐藏已读
  if(a in n) n[a] = !n[a];
  else { const s = we(); n[a] = !s.isHideReadReceipts }   // 首次取反全局值
  o();                                          // 写 localStorage
}

const i = computed(()=>{                        // 当前会话是否隐藏已读
  const a = e.value?.targetUid;
  return a ? (a in n ? n[a] : we().isHideReadReceipts) : false;
});
```

判定"要不要回已读"的地方：

```js
function g(){
  const e = s.room?.targetUid;
  if(e){
    if(s.currentHideReadReceipts){ s.markRelationRead(e); return }  // 只本地标记
    s.markVisibleMessagesRead(e);                                   // 真回已读
  }
}
```

⚠️ 单人覆盖只存 localStorage，**服务端完全不知情**。「隐藏已读」= 客户端选择不发 `read_messages`。

---

## 19. 拉黑自动解除的文案（唯一证据）

```js
await Le({
  title:"确认拉黑",
  message:"拉黑后双方将无法继续聊天，每天零点自动解除。确认拉黑吗？",
  confirmButtonText:"拉黑",
  confirmButtonColor:"var(--red, #e53e3e)",
  cancelButtonText:"取消"
});
s.blockUser(m), F({message:"已拉黑",position:"bottom"}), $("close");
```

「每天零点自动解除」是服务端语义，前端只是复述。我方按永久拉黑设计文案会误导用户。
