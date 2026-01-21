# TikTokDownloader Web API 调用指南与 SDK 草稿

> 上游项目：`https://github.com/JoeanAmier/TikTokDownloader`  
> 版本基准：`9fefb9a73bd4b64a243082be49b43c4803448a77`  
> 适用范围：对接 `FastAPI` Web API（默认 `http://127.0.0.1:5555`）

相关文档：
- 接口清单：`helloagents/wiki/external/tiktokdownloader-web-api.md`

---

## 1. 调用约定（建议）

### 1.1 Base URL

- 本机默认：`http://127.0.0.1:5555`
- 文档：`/docs`、`/redoc`

### 1.2 Header（token）

绝大多数接口都要求 `token` Header（实现上通过 FastAPI dependency 校验）。

当前上游默认 `is_valid_token()` 返回 `True`，因此 **默认无需 token**；若你自行启用校验逻辑：
- `token: <your-token>`

### 1.3 Content-Type

- 所有 `POST` 接口均使用 JSON Body：`Content-Type: application/json`

### 1.4 超时与重试（建议值）

- `timeout`: 30s（采集接口可能较慢，尤其是 `account/comment/search`）
- `retry`: 仅对网络错误/5xx 做重试；避免对 4xx（403/422）重试

---

## 2. 错误处理约定（推荐客户端策略）

### 2.1 HTTP 状态码

- `403`：token 校验失败（`{"detail":"验证失败！"}` 之类）
- `422`：请求体校验失败（FastAPI/Pydantic 自动返回，`detail` 为字段错误列表）
- `307/308`：`GET /` 的重定向（到 GitHub 仓库）

### 2.2 业务失败（HTTP 200 但 data 为空）

多数业务接口返回 `DataResponse`：
- 成功通常表现为：`data != null`
- 失败通常表现为：`data == null` 且 `message` 为“获取数据失败！”/“参数错误！”等

推荐客户端判断顺序：
1. `response.raise_for_status()`（先处理 4xx/5xx）
2. 解析 JSON
3. 若响应包含 `data` 字段：以 `data` 是否为空作为“业务成功”主判断

### 2.3 已知不一致：`POST /tiktok/live`

⚠️ 当前路由入参模型与处理逻辑存在不一致（代码读取 `room_id`，但路由使用的模型未声明该字段）。  
表现可能为 500/异常。建议以 `/docs` 的 OpenAPI 为准，必要时修复上游后再对接。

---

## 3. 字段行为补充（默认值 / 校验 / “传空不覆盖”）

### 3.1 通用字段（多数业务接口都支持）

- `cookie`（string，默认 `""`）：为空通常表示不覆盖全局 Cookie；非空则临时使用该 Cookie
- `proxy`（string，默认 `""`）：用于指定代理（是否完全覆盖全局代理取决于内部实现）
- `source`（bool，默认 `false`）：`true` 倾向返回原始响应数据；`false` 返回提取后的结构化数据

### 3.2 账号时间范围（/douyin/account、/tiktok/account）

`earliest/latest` 支持：
- `YYYY/MM/DD` 字符串
- 数字（int/float）：表示“距离 latest/today 的天数偏移”

### 3.3 校验（典型）

- `count`：大多为 `gt=0`（必须大于 0）
- `pages`：大多为 `gt=0`
- `keyword`：不能为空（否则 422）
- 搜索 `count`：要求 `>= 5`

---

## 4. Python SDK（草稿，httpx 同步版）

> 目的：提供“能直接用”的最小封装；你可以按需裁剪只保留用到的接口。

```python
from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any, Dict, Optional

import httpx


def extract_detail_id_from_url(url: str) -> Optional[str]:
    # 常见形态：
    # - https://www.douyin.com/video/<id>
    # - https://www.tiktok.com/@xxx/video/<id>
    match = re.search(r"/video/([0-9]+)", url)
    return match.group(1) if match else None


@dataclass
class TikTokDownloaderAPI:
    base_url: str = "http://127.0.0.1:5555"
    token: Optional[str] = None
    timeout: float = 30.0

    def __post_init__(self) -> None:
        self._client = httpx.Client(base_url=self.base_url, timeout=self.timeout)

    def close(self) -> None:
        self._client.close()

    def _headers(self) -> Dict[str, str]:
        # 说明：上游 token_dependency 从 Header("token") 取值；默认校验恒通过
        return {"token": self.token or ""}

    def _get(self, path: str) -> Dict[str, Any]:
        r = self._client.get(path, headers=self._headers())
        r.raise_for_status()
        return r.json()

    def _post(self, path: str, payload: Dict[str, Any]) -> Dict[str, Any]:
        r = self._client.post(path, json=payload, headers=self._headers())
        r.raise_for_status()
        return r.json()

    # --- Project / Settings ---

    def get_settings(self) -> Dict[str, Any]:
        return self._get("/settings")

    def update_settings(self, patch: Dict[str, Any]) -> Dict[str, Any]:
        return self._post("/settings", patch)

    # --- Douyin ---

    def douyin_share(self, text: str, proxy: str = "") -> Dict[str, Any]:
        return self._post("/douyin/share", {"text": text, "proxy": proxy})

    def douyin_detail(
        self,
        detail_id: str,
        *,
        cookie: str = "",
        proxy: str = "",
        source: bool = False,
    ) -> Dict[str, Any]:
        return self._post(
            "/douyin/detail",
            {"detail_id": detail_id, "cookie": cookie, "proxy": proxy, "source": source},
        )

    # --- TikTok ---

    def tiktok_share(self, text: str, proxy: str = "") -> Dict[str, Any]:
        return self._post("/tiktok/share", {"text": text, "proxy": proxy})

    def tiktok_detail(
        self,
        detail_id: str,
        *,
        cookie: str = "",
        proxy: str = "",
        source: bool = False,
    ) -> Dict[str, Any]:
        return self._post(
            "/tiktok/detail",
            {"detail_id": detail_id, "cookie": cookie, "proxy": proxy, "source": source},
        )


if __name__ == "__main__":
    api = TikTokDownloaderAPI()
    try:
        # 端到端示例：share -> 解析 detail_id -> detail
        share = api.douyin_share("https://v.douyin.com/xxxxxx/")
        url = share.get("url") or ""
        detail_id = extract_detail_id_from_url(url)
        if not detail_id:
            raise SystemExit(f"无法从重定向链接提取 detail_id: {url}")

        detail = api.douyin_detail(detail_id)
        ok = bool(detail.get("data"))
        print({"ok": ok, "message": detail.get("message")})
    finally:
        api.close()
```

---

## 5. Node.js 最小调用示例（fetch）

```js
const baseUrl = "http://127.0.0.1:5555";
const headers = { "Content-Type": "application/json", token: "" };

async function post(path, body) {
  const res = await fetch(`${baseUrl}${path}`, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${res.status} ${await res.text()}`);
  return res.json();
}

async function main() {
  const r = await post("/douyin/share", { text: "https://v.douyin.com/xxxxxx/", proxy: "" });
  console.log(r);
}

main().catch((e) => console.error(e));
```
