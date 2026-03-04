# TikTokDownloader Web API（FastAPI）整理

> 上游项目：`https://github.com/JoeanAmier/TikTokDownloader`  
> 版本基准：`9fefb9a73bd4b64a243082be49b43c4803448a77`  
> 路由定义：`src/application/main_server.py`  
> 默认监听：`0.0.0.0:5555`（本机访问：`http://127.0.0.1:5555`）  
> 在线文档：`/docs`、`/redoc`

相关文档：
- 调用指南与 SDK 草稿：`helloagents/modules/external/tiktokdownloader-web-api-sdk.md`

---

## 1. 启动方式

项目运行后在主菜单选择 **Web API 模式** 启动服务器（文档提示为菜单项 `8`）。

启动后可访问：
- `http://127.0.0.1:5555/docs`
- `http://127.0.0.1:5555/redoc`

---

## 2. 认证与鉴权（token Header）

除 `GET /` 外，所有接口均通过 `token` Header 走统一校验依赖：
- 依赖位置：`src/application/main_server.py#token_dependency`
- 校验函数：`src/custom/function.py#is_valid_token(token)`

当前仓库默认 `is_valid_token()` 直接 `return True`，因此 **默认不需要 token**。  
如你公开部署，建议自行实现校验逻辑后，再在请求头中携带 `token`：

- Header：`token: <your-token>`

---

## 3. 通用请求字段（APIModel）

多数业务接口（抖音/TikTok 的 detail/account/mix/live/comment/reply/search）请求体均继承自 `src/models/base.py#APIModel`：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| cookie | string | 否 | `""` | 临时 Cookie；非空才会覆盖全局 Cookie |
| proxy | string | 否 | `""` | 临时代理；具体是否生效取决于内部实现与参数校验 |
| source | boolean | 否 | `false` | `true` 时尽量返回“原始响应数据”；`false` 返回提取/整形后的数据 |

---

## 4. 通用响应结构

### 4.1 DataResponse（大多数接口）

模型：`src/models/response.py#DataResponse`

| 字段 | 类型 | 说明 |
|---|---|---|
| message | string | 结果描述 |
| data | object / array / null | 业务数据（可能为单对象或列表） |
| params | object / null | 回显请求参数 |
| time | string | 服务端生成时间（格式化字符串） |

### 4.2 UrlResponse（share 接口）

模型：`src/models/response.py#UrlResponse`

| 字段 | 类型 | 说明 |
|---|---|---|
| message | string | 结果描述 |
| url | string / null | 解析后的完整链接 |
| params | object / null | 回显请求参数 |
| time | string | 服务端生成时间 |

---

## 5. 接口清单与调用示例

> 说明：示例使用 `curl`；如你的部署启用了 token 校验，请加上 `-H 'token: ...'`。

### 5.1 项目

#### [GET] /
重定向到项目 GitHub 仓库主页。

**示例：**
```bash
curl -i "http://127.0.0.1:5555/"
```

#### [GET] /token
测试 `token` 是否通过校验（默认永远通过）。

**示例：**
```bash
curl -s "http://127.0.0.1:5555/token" -H "token: "
```

---

### 5.2 配置

#### [GET] /settings
获取当前全局配置（对应 `settings.json` 的运行时视图）。

**示例：**
```bash
curl -s "http://127.0.0.1:5555/settings"
```

#### [POST] /settings
更新全局配置并返回更新后的完整配置。

**注意：**
- 接口入参为 `src/models/settings.py#Settings`（字段较多，建议先 `GET /settings` 再按需修改后回传）。
- 内部对部分字段使用“非空才覆盖”的策略：例如 `cookie=""`、`accounts_urls=[]` 不会清空现有配置。

**示例（仅更新 proxy / cookie）：**
```bash
curl -s "http://127.0.0.1:5555/settings" \
  -H "Content-Type: application/json" \
  -d '{"proxy":"http://127.0.0.1:7890","cookie":"__ac_nonce=...; __ac_signature=...;"}'
```

---

### 5.3 抖音（/douyin/*）

#### [POST] /douyin/share
把分享短链文本解析为重定向后的完整链接。

请求体模型：`src/models/share.py#ShortUrl`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| text | string | 是 | - |
| proxy | string | 否 | `""` |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/share" \
  -H "Content-Type: application/json" \
  -d '{"text":"https://v.douyin.com/xxxxxx/","proxy":""}'
```

#### [POST] /douyin/detail
获取单个作品数据。

请求体模型：`src/models/detail.py#Detail`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| detail_id | string | 是 | - |
| cookie | string | 否 | `""` |
| proxy | string | 否 | `""` |
| source | boolean | 否 | `false` |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/detail" \
  -H "Content-Type: application/json" \
  -d '{"detail_id":"0123456789","source":false}'
```

#### [POST] /douyin/account
获取账号作品数据（发布/喜欢）。

请求体模型：`src/models/account.py#Account`

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| sec_user_id | string | 是 | - | 账号 `sec_uid` |
| tab | string | 否 | `"post"` | `"post"` / `"favorite"` |
| earliest | string/number/null | 否 | `null` | 支持 `YYYY/MM/DD` 或“距离 latest 的天数” |
| latest | string/number/null | 否 | `null` | 支持 `YYYY/MM/DD` 或“距离 today 的天数” |
| pages | integer/null | 否 | `null` | 最大请求次数（主要对喜欢页有效） |
| cursor | integer | 否 | `0` | 游标 |
| count | integer | 否 | `18` | 每页数量（>0） |
| cookie/proxy/source | - | 否 | - | 见通用字段 |

> 提示：如需结果中包含作品 `author`/封面等原始字段（例如用于“收藏作者/预览图”展示），请显式传 `source:true`。

**示例（发布作品）：**
```bash
curl -s "http://127.0.0.1:5555/douyin/account" \
  -H "Content-Type: application/json" \
  -d '{"sec_user_id":"MS4wLjABAAAA...","tab":"post","pages":1}'
```

#### [POST] /douyin/account/page
单页分页获取账号作品数据（包装镜像增强接口）。

请求体模型：`src/models/account.py#Account`（与 `/douyin/account` 基本一致；`pages` 可不传或传 `1`，接口仍只返回单页）。

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| sec_user_id | string | 是 | - | 账号 `sec_uid` |
| tab | string | 否 | `"post"` | `"post"` / `"favorite"` |
| earliest | string/number/null | 否 | `null` | 同 `/douyin/account` |
| latest | string/number/null | 否 | `null` | 同 `/douyin/account` |
| cursor | integer | 否 | `0` | 游标 |
| count | integer | 否 | `18` | 每页数量（>0） |
| cookie/proxy/source | - | 否 | - | 见通用字段 |

**响应（`data`）**
- `items`：本页列表
- `next_cursor`：下一页游标（int）
- `has_more`：是否还有更多（bool）

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/account/page" \
  -H "Content-Type: application/json" \
  -d '{"sec_user_id":"MS4wLjABAAAA...","tab":"post","cursor":0,"count":18,"source":true}'
```

#### [POST] /douyin/mix
获取合集作品数据（`mix_id` 与 `detail_id` 二选一）。

请求体模型：`src/models/mix.py#Mix`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| mix_id | string/null | 二选一 | `null` |
| detail_id | string/null | 二选一 | `null` |
| cursor | integer | 否 | `0` |
| count | integer | 否 | `12` |
| cookie/proxy/source | - | 否 | - |

**示例（按 mix_id）：**
```bash
curl -s "http://127.0.0.1:5555/douyin/mix" \
  -H "Content-Type: application/json" \
  -d '{"mix_id":"MIX123456789","cursor":0,"count":12}'
```

#### [POST] /douyin/live
获取直播数据（当前代码只使用 `web_rid`）。

请求体模型：`src/models/live.py#Live`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| web_rid | string/null | 是 | `null` |
| cookie/proxy/source | - | 否 | - |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/live" \
  -H "Content-Type: application/json" \
  -d '{"web_rid":"1234567890"}'
```

#### [POST] /douyin/comment
获取作品评论数据（可选递归抓取回复）。

请求体模型：`src/models/comment.py#Comment`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| detail_id | string | 是 | - |
| pages | integer | 否 | `1` |
| cursor | integer | 否 | `0` |
| count | integer | 否 | `20` |
| count_reply | integer | 否 | `3` |
| reply | boolean | 否 | `false` |
| cookie/proxy/source | - | 否 | - |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/comment" \
  -H "Content-Type: application/json" \
  -d '{"detail_id":"0123456789","pages":2,"reply":false}'
```

#### [POST] /douyin/reply
获取评论回复数据。

请求体模型：`src/models/reply.py#Reply`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| detail_id | string | 是 | - |
| comment_id | string | 是 | - |
| pages | integer | 否 | `1` |
| cursor | integer | 否 | `0` |
| count | integer | 否 | `3` |
| cookie/proxy/source | - | 否 | - |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/reply" \
  -H "Content-Type: application/json" \
  -d '{"detail_id":"0123456789","comment_id":"9876543210","pages":1}'
```

---

### 5.4 抖音搜索（/douyin/search/*）

搜索接口共用基础模型：`src/models/search.py#BaseSearch`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| keyword | string | 是 | - |
| pages | integer | 否 | `1` |
| offset | integer | 否 | `0` |
| count | integer | 否 | `10`（≥5） |
| cookie/proxy/source | - | 否 | - |

> 说明：各枚举参数（如 `sort_type/publish_time/duration/...`）的语义映射在上游文档中有更详细说明；路由描述中给出了 GitHub Wiki 链接。

#### [POST] /douyin/search/general
请求体模型：`src/models/search.py#GeneralSearch`

附加字段（均为数值枚举）：
- `sort_type`: `0|1|2`
- `publish_time`: `0|1|7|180`
- `duration`: `0|1|2|3`
- `search_range`: `0|1|2|3`
- `content_type`: `0|1|2`

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/search/general" \
  -H "Content-Type: application/json" \
  -d '{"keyword":"测试","pages":1,"count":10,"sort_type":0}'
```

#### [POST] /douyin/search/video
请求体模型：`src/models/search.py#VideoSearch`

附加字段：
- `sort_type`: `0|1|2`
- `publish_time`: `0|1|7|180`
- `duration`: `0|1|2|3`
- `search_range`: `0|1|2|3`

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/search/video" \
  -H "Content-Type: application/json" \
  -d '{"keyword":"测试","pages":1,"count":10,"publish_time":7}'
```

#### [POST] /douyin/search/user
请求体模型：`src/models/search.py#UserSearch`

附加字段：
- `douyin_user_fans`: `0|1|2|3|4|5`
- `douyin_user_type`: `0|1|2|3`

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/search/user" \
  -H "Content-Type: application/json" \
  -d '{"keyword":"测试","pages":1,"count":10,"douyin_user_fans":0}'
```

#### [POST] /douyin/search/live
请求体模型：`src/models/search.py#LiveSearch`

**示例：**
```bash
curl -s "http://127.0.0.1:5555/douyin/search/live" \
  -H "Content-Type: application/json" \
  -d '{"keyword":"测试","pages":1,"count":10}'
```

---

### 5.5 TikTok（/tiktok/*）

#### [POST] /tiktok/share
与 `/douyin/share` 相同：`src/models/share.py#ShortUrl`

**示例：**
```bash
curl -s "http://127.0.0.1:5555/tiktok/share" \
  -H "Content-Type: application/json" \
  -d '{"text":"https://vt.tiktok.com/xxxxxx/"}'
```

#### [POST] /tiktok/detail
请求体模型：`src/models/detail.py#DetailTikTok`（同 `Detail`）

**示例：**
```bash
curl -s "http://127.0.0.1:5555/tiktok/detail" \
  -H "Content-Type: application/json" \
  -d '{"detail_id":"0123456789","source":false}'
```

#### [POST] /tiktok/account
请求体模型：`src/models/account.py#AccountTiktok`（同 `Account`）

**示例：**
```bash
curl -s "http://127.0.0.1:5555/tiktok/account" \
  -H "Content-Type: application/json" \
  -d '{"sec_user_id":"MS4wLjABAAAA...","tab":"post","pages":1}'
```

#### [POST] /tiktok/mix
请求体模型：`src/models/mix.py#MixTikTok`

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| mix_id | string/null | 是 | `null` |
| cursor | integer | 否 | `0` |
| count | integer | 否 | `30` |
| cookie/proxy/source | - | 否 | - |

**示例：**
```bash
curl -s "http://127.0.0.1:5555/tiktok/mix" \
  -H "Content-Type: application/json" \
  -d '{"mix_id":"MIX123456789","cursor":0,"count":30}'
```

#### [POST] /tiktok/live
用于获取直播数据（逻辑上需要 `room_id`）。

⚠️ 注意：当前路由签名使用的请求模型为 `src/models/live.py#Live`，但处理逻辑读取 `extract.room_id`。  
如你调用该接口遇到异常，建议优先以 `/docs` 的 OpenAPI 为准并结合源码修复（将路由入参改为 `LiveTikTok` 或为 `Live` 补充 `room_id` 字段）。

**预期请求体字段（按代码描述）：**

| 字段 | 类型 | 必填 | 默认值 |
|---|---|---|---|
| room_id | string | 是 | - |
| cookie/proxy/source | - | 否 | - |

**示例（可能受上述问题影响）：**
```bash
curl -s "http://127.0.0.1:5555/tiktok/live" \
  -H "Content-Type: application/json" \
  -d '{"room_id":"1234567890","source":false}'
```
