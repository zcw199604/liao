# CookieCloud（easychen/CookieCloud）Go 获取 Cookie / 解密方式整理

> 上游项目：`https://github.com/easychen/CookieCloud`  
> 版本基准：`79bea92a84cdf65a12308c187d8628089317ba4e`  
> 关键文件：
> - API 服务：`api/app.js`
> - 浏览器扩展：`ext/utils/functions.ts`
> - Go 解密示例（fixed IV）：`examples/fixediv/go/decrypt.go`
> - Python 拉取+解密示例（legacy + fixed IV）：`examples/decrypt.py`

---

## 1. CookieCloud 的“获取 Cookie”是什么意思

CookieCloud 本身并不“登录并抓取”目标站点 Cookie；它的核心是“同步/分发浏览器 Cookie”：

1. 浏览器扩展从浏览器 Cookie 存储中读取 Cookie（WebExtensions API：`browser.cookies.getAll`），按 domain 组织。
2. 扩展把 `{cookie_data, local_storage_data, update_time}` 组装成 JSON，并用 `uuid + password` 进行对称加密。
3. 扩展 `POST /update` 上传 `{uuid, encrypted, crypto_type}`，服务端落盘保存。
4. 客户端（扩展/脚本/服务）`GET /get/:uuid` 拉取 `{encrypted, crypto_type}`，在本地用同样算法解密得到 cookie 列表。

因此，Go 端所谓“获取 Cookie”，通常指：
- 调用 CookieCloud API 拉取密文；
- 用 Go 复现 CookieCloud 的解密算法；
- 取出目标 domain 的 cookie，拼成 HTTP Header `Cookie: ...` 使用。

---

## 2. CookieCloud API（`api/app.js`）

### 2.1 API_ROOT 前缀

服务端会读取环境变量 `API_ROOT` 作为统一前缀（并剥掉末尾 `/`），例如：
- `API_ROOT` 为空：`http://127.0.0.1:8088/update`
- `API_ROOT=/cookiecloud`：`http://127.0.0.1:8088/cookiecloud/update`

下文用 `/update`、`/get/:uuid` 表示“拼接 `API_ROOT` 后”的真实路径。

### 2.2 [POST] /update（上传加密数据）

请求体（JSON）：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| uuid | string | 是 | 用户唯一标识 |
| encrypted | string | 是 | 加密后的 payload（字符串） |
| crypto_type | string | 否 | 默认 `legacy`；也可 `aes-128-cbc-fixed` |

服务端行为：
- 写入 `api/data/{uuid}.json`，内容为 `{ encrypted, crypto_type }`
- 成功返回：`{"action":"done"}`

### 2.3 [ALL] /get/:uuid（下载加密数据；可选由服务端解密）

默认行为：返回 `api/data/{uuid}.json` 的内容：

```json
{
  "encrypted": "....",
  "crypto_type": "legacy"
}
```

可选行为（不建议）：如果请求 body 里带 `password`，服务端会直接返回解密后的 JSON 对象。

算法选择优先级（见 `api/app.js`）：
1. 请求 query `?crypto_type=...`
2. 存储文件里的 `crypto_type`
3. 默认 `'legacy'`

注意：
- 路由是 `app.all`，所以“带 body 的 GET”在很多 HTTP 客户端里不自然；如果你要用“服务端解密”模式，更稳妥是用 `POST /get/:uuid` 传 `{ "password": "..." }`。
- 把 `password` 发到服务端意味着服务端可以解密你的 Cookie（更高风险）；建议始终“客户端本地解密”。

---

## 3. 解密后的数据结构（明文 JSON）

扩展侧加密的原始数据形态（`ext/utils/functions.ts` 里 `data_to_encrypt`）：

```json
{
  "cookie_data": {
    ".example.com": [
      {
        "name": "a",
        "value": "1",
        "domain": ".example.com",
        "path": "/",
        "secure": true,
        "httpOnly": true,
        "sameSite": "lax"
      }
    ]
  },
  "local_storage_data": {
    "example.com": {
      "k": "v"
    }
  },
  "update_time": "2026-02-01T00:00:00.000Z"
}
```

其中：
- `cookie_data`：`map[domain][]Cookie`（domain 通常含前导 `.`）
- `local_storage_data`：`map[domain]map[key]value`（只有开启 `with_storage` 才会有）
- `update_time`：扩展端直接 `new Date()` 序列化，具体格式取决于浏览器实现

Go 侧如果只需要发 HTTP 请求，一般只用 `cookie_data`，把 cookie 列表转成：

```text
name=value; name2=value2
```

---

## 4. 加密/解密算法（`crypto_type`）

CookieCloud 当前至少存在两类加密格式：

### 4.1 `aes-128-cbc-fixed`（推荐）

来源：
- `api/app.js#cookie_decrypt` / `cookie_encrypt`
- `ext/utils/functions.ts#cookie_decrypt` / `cookie_encrypt`
- `examples/fixediv/go/decrypt.go`

算法规格：
- 算法：AES-128-CBC
- Key：`MD5(uuid + "-" + password)` 的 hex 字符串前 16 个字符（作为 UTF-8 字节直接使用）
- IV：固定 16 字节的 `0x00`
- Padding：PKCS7
- 密文：Base64（仅 ciphertext，不含 CryptoJS 的 OpenSSL/Salt 包装）

Go 侧实现要点（与 `examples/fixediv/go/decrypt.go` 一致）：
- 用 `md5.Sum([]byte(uuid+"-"+password))` 得到 16 字节摘要；
- `fmt.Sprintf("%x", sum)` 转 32 位 hex 字符串；
- `key := []byte(hexStr[:16])` 作为 AES key（16 字节）；
- Base64 decode -> CBC 解密 -> PKCS7 unpad -> `json.Unmarshal`。

### 4.2 `legacy`（兼容旧版）

来源：
- `api/app.js#cookie_decrypt`（CryptoJS passphrase 模式）
- `examples/decrypt.py`（复现 OpenSSL `EVP_BytesToKey` 的实现）

算法特征（以 Python 示例为准，更便于在 Go 里复刻）：
- `encrypted` 是 Base64 字符串，decode 后格式为：
  - 前 8 字节：`"Salted__"`
  - 接着 8 字节：salt
  - 剩余：ciphertext
- key/iv 推导：OpenSSL `EVP_BytesToKey`（MD5 循环堆叠）从 `passphraseBytes + salt` 得到 48 字节材料：
  - key：前 32 字节（AES-256）
  - iv：后 16 字节
- 解密：AES-256-CBC + PKCS7 unpad

其中 `passphraseBytes` 仍然是：
- `MD5(uuid + "-" + password)` 的 hex 字符串前 16 个字符（UTF-8 字节）

---

## 5. Go 侧“获取 Cookie”的最小流程（建议实现）

如果你要在 Go 服务里用 CookieCloud 的 Cookie 去请求第三方站点，建议按下面的最小闭环实现：

1. `GET {server}/get/{uuid}`（可选加 `?crypto_type=...` 覆盖），拿到 `{encrypted, crypto_type}`；
2. 选择解密分支：
   - `crypto_type == "aes-128-cbc-fixed"`：走 AES-128-CBC fixed IV（标准库可实现）；
   - 否则：按 legacy 兼容实现；
3. 取 `cookie_data` 中目标 domain 的 cookies，拼接成 `Cookie` Header。

示例（仅展示拉取 + fixed IV 解密的骨架；legacy 可按 `examples/decrypt.py` 迁移到 Go）：

```go
type CookieCloudGetResponse struct {
	Encrypted  string `json:"encrypted"`
	CryptoType string `json:"crypto_type"`
}

type CookieCloudDecrypted struct {
	CookieData map[string][]struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"cookie_data"`
}

// 1) GET /get/:uuid -> 2) Decrypt (fixed) -> 3) build "name=value; ..."
func FetchCookieHeaderFixed(serverURL, uuid, password, domain string) (string, error) {
	// 省略：用 net/http GET JSON，解析到 CookieCloudGetResponse
	// 省略：调用 fixed 解密函数得到 CookieCloudDecrypted
	// 然后拼 Cookie header
	return "", nil
}
```

---

## 6. 实务注意点

- `domain` 匹配：CookieCloud 的 `cookie_data` key 通常是 cookie 的 `domain` 字段（可能带前导 `.`），建议在你自己的匹配逻辑里做一次归一化（例如同时尝试 `.example.com` / `example.com`）。
- `sameSite`：扩展里针对 Firefox 把 `unspecified` 映射成 `no_restriction` 才能成功写回浏览器；Go 侧如果只是发 HTTP 请求，一般可以忽略 `sameSite`。
- 安全建议：尽量不要把 CookieCloud 的 `password` 交给服务端（不要用“服务端解密”模式），避免明文 Cookie 被服务端/日志/代理持有。

---

## 7. 本仓库实现（`internal/cookiecloud`）

本仓库已提供可直接复用的 Go 包 `internal/cookiecloud`，包含：
- 拉取：`Client.GetEncrypted`（`GET /get/:uuid`）
- 解密：`Client.GetDecrypted` / `Decrypt`（支持 `legacy` + `aes-128-cbc-fixed`）
- 组装 Header：`Client.GetCookieHeader` / `BuildCookieHeader`

示例（拉取并组装 HTTP `Cookie` Header）：

```go
cc, err := cookiecloud.NewClient("http://127.0.0.1:8088")
if err != nil {
  // handle error
}

cookieHeader, err := cc.GetCookieHeader(ctx, uuid, password, "example.com", cookiecloud.CryptoTypeAES128CBCFixed)
if err != nil {
  // handle error
}

req.Header.Set("Cookie", cookieHeader)
```

### 7.1 环境变量配置（`internal/config`）

本仓库的配置加载器 `internal/config` 已支持通过环境变量注入 CookieCloud 相关配置（未配置时不启用）。

| 环境变量 | 必填 | 说明 |
|---|---:|---|
| `COOKIECLOUD_BASE_URL` | 是（启用时） | CookieCloud 服务地址；可包含 `API_ROOT` 前缀，例如 `http://127.0.0.1:8088/cookiecloud` |
| `COOKIECLOUD_UUID` | 是（启用时） | 你的 CookieCloud UUID |
| `COOKIECLOUD_PASSWORD` | 是（启用时） | 你的 CookieCloud 解密密码（建议仅在本地/私有环境使用） |
| `COOKIECLOUD_CRYPTO_TYPE` | 否 | 留空表示使用服务端返回的 `crypto_type`；也可显式指定 `legacy` / `aes-128-cbc-fixed` |
| `COOKIECLOUD_DOMAIN` | 否 | 需要提取的域名（例如 `example.com`）；用于业务层生成 `Cookie` Header 的默认值 |
| `COOKIECLOUD_COOKIE_EXPIRE_HOURS` | 否 | CookieCloud cookie 的缓存 TTL（小时）；默认 `72`（3 天）；`CACHE_TYPE=redis` 时写入 Redis 并带 TTL，否则仅进程内缓存 |

兼容别名（任选其一即可）：
- `COOKIE_CLOUD_BASE_URL` / `COOKIE_CLOUD_UUID` / `COOKIE_CLOUD_PASSWORD` / `COOKIE_CLOUD_CRYPTO_TYPE` / `COOKIE_CLOUD_DOMAIN`
- `COOKIE_CLOUD_COOKIE_EXPIRE_HOURS`
