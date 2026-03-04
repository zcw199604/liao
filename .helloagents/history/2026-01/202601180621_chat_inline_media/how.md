# 技术设计: 聊天消息图文混排渲染

## 技术方案

### 核心技术
- Vue 3 + Pinia（现有消息渲染组件）
- 统一媒体端口策略：`systemConfigStore.resolveImagePort`（现有逻辑复用）

### 实现要点
- **解析策略（前端）**
  - 在消息内容中扫描 `[...]` 片段，按顺序拆分为“文本段 + 媒体段”数组（支持多个媒体段）。
  - 表情兼容：若 `[...]` 片段完整命中 `emojiMap` key，则视为文本（交给 `parseEmoji` 渲染），不当作媒体。
  - 媒体识别：仅当 `[...]` 内字符串满足以下条件时才识别为媒体路径：
    - 不包含 `://`（避免把 URL 误当作上传路径）
    - 扩展名可识别（复用 `isImageFile/isVideoFile`；否则按文件处理需满足“扩展名为 2-10 位字母数字”）
  - 媒体 URL 拼接：对每个媒体路径 `path`：
    1) 确保 `mediaStore.imgServer` 已加载（必要时 `loadImgServer()`）
    2) 通过 `systemConfigStore.resolveImagePort(path, imgServer)` 获取端口
    3) 拼接 `http://{imgServer}:{port}/img/Upload/{path}`

- **数据模型（前端）**
  - 扩展 `ChatMessage`：增加 `segments?: MessageSegment[]`
  - `MessageSegment` 结构：
    - `text`: `{ kind: 'text', text: string }`
    - `image/video/file`: `{ kind: 'image'|'video'|'file', path: string, url: string }`
  - 保持现有字段以兼容现有逻辑：
    - `isImage/isVideo/isFile`：当消息包含对应媒体段时置为 true（用于预览画廊/筛选）
    - `imageUrl/videoUrl/fileUrl`：填充“第一条同类媒体”的 url（便于现有逻辑复用）
    - `content`：保留原始文本（用于复制/回退展示）

- **消息接收（WebSocket）**
  - 替换“整条消息为 `[path]` 才解析”的逻辑为“混排解析”：
    - 产出 `segments`
    - 依据 `segments` 计算 `isImage/isVideo/isFile` 与对应 URL 字段
  - `lastMsg` 预览文案改为基于 `segments` 生成：
    - 文本段拼接后截断（沿用 30 字规则）
    - 若包含图片/视频/文件段，追加对应标签（如 ` [图片]`）
    - 若无文本段，仅输出标签（如 `[图片]`）

- **历史消息解析（loadHistory）**
  - 在 `frontend/src/stores/message.ts` 的历史消息映射中复用同一套混排解析函数，保证历史与 WS 行为一致。
  - 语义去重：`getMessageRemoteMediaPath` 增强为可从 `segments` 中提取首个媒体 `path`，避免混排媒体在“WS 推送 + 历史拉取”时重复渲染。

- **渲染层（Vue 组件）**
  - `MessageList.vue` / `MessageBubble.vue` / `ChatHistoryPreview.vue`：
    - 若 `segments` 存在则按段渲染：
      - 文本段：`v-html="parseEmoji(text, emojiMap)"`
      - 图片段：`<img :src="getMediaUrl(url)" ... @click="preview">`
      - 视频段：`<video :src="getMediaUrl(url)" controls ...>`
      - 文件段：最小可用展示为可点击下载链接/块（复用 `MessageBubble` 的下载逻辑或简化实现）
    - 若 `segments` 不存在，保留现有渲染分支作为回退（避免历史缓存数据回归）。

## 架构决策 ADR
本变更为解析与渲染增强，不引入新模块与新依赖，不新增 ADR。

## API 设计
无新增/变更 API（复用现有 `/api/getImgServer` 与 `/api/resolveImagePort`）。

## 数据模型
无后端数据结构变更（仅增强后端推断与格式化逻辑）。

## 安全与性能
- **安全:**
  - 严格限制媒体识别条件（扩展名+无协议头），降低恶意内容被当作媒体渲染的概率。
  - 现有文本渲染使用 `v-html`，存在潜在 XSS 风险；本次变更不扩大该风险面，后续可单独评估“文本转义 + 表情替换”的安全改造。
- **性能:**
  - 解析使用线性扫描与简单正则，避免回溯型正则。
  - 端口解析使用 `systemConfigStore` 的缓存（TTL），避免每条消息重复探测端口。

## 测试与部署
- **测试:**
  - 前端：新增/补充单测覆盖混排解析与组件渲染（含 emoji 不误判）。
  - 后端：新增单测覆盖最后消息摘要的混排格式化。
- **部署:**
  - 无额外部署步骤；前端构建与后端构建流程保持不变。

