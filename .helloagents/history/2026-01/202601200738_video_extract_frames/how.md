# 技术设计: 视频抽帧（关键帧/固定FPS/逐帧）与任务管理

## 技术方案

### 核心技术
- 后端：Go（`exec.CommandContext` 调用 `ffmpeg/ffprobe`）
- 前端：Vue3 + TS（复用 `MediaPreview`、`InfiniteMediaGrid`）
- 存储：MySQL（任务与帧索引）+ 本地文件系统（帧图目录位于 `./upload` 下）

### 推荐方案（落地优先）
在 Go 后端内置“抽帧任务管理器”（队列 + worker pool），使用 DB 持久化任务元数据与帧索引；前端通过轮询获取任务状态/增量帧列表（后续可平滑升级 SSE）。

#### 方案对比（复杂任务-强制对比）
- 方案1：内置任务队列 + 轮询（推荐）
  - 优点：改动可控；不影响现有聊天 WS；实现成本低；可逐步演进
  - 缺点：轮询带来额外请求；极端实时性不如推送
- 方案2：新增独立任务 WS（/ws/tasks）推送进度
  - 优点：实时性好、网络更节省
  - 缺点：新增 WS 协议/鉴权；需要处理断线重连与多端订阅；复杂度更高
- 方案3：外部 worker/队列（如独立服务或 MQ）
  - 优点：可扩展性好
  - 缺点：引入新基础设施与运维成本；不符合当前项目体量与目标

默认按方案1规划实现；若后续确需强实时，再引入方案2。

### 实现要点

#### 1) 视频输入源解析（Source Resolver）
支持两类输入源：
- upload 视频：前端传 `localPath`（形如 `/videos/...` 或 `/upload/videos/...` 或完整 URL），后端复用现有 `normalizeUploadLocalPathInput` 解析为统一 localPath，再拼接到 `upload` 目录得到 absPath
- mtPhoto 视频：前端传 `md5`（必要时传 `id`），后端 `mtPhoto.ResolveFilePath` → `resolveLspLocalPath(LSP_ROOT, /lsp/...)` 得到 absPath

若 mtPhoto 源无法直接读取本地文件（例如仅可 HTTP 下载），后端先下载缓存到受控目录（如 `upload/mtphoto_cache/{md5}.mp4`），并在任务元数据中记录实际输入路径。

#### 2) 视频探测（Probe）
新增 `ffprobe` 调用获取：
- durationSec、width、height、avgFps（可选）、codecName（可选）
用于：
- 任务详情展示（前端 aspectRatio=width/height）
- 进度计算（以 out_time / duration 为主）

#### 3) 抽帧执行器（FFmpeg Runner）
不同模式的过滤器建议：
- 关键帧-I 帧：`select='eq(pict_type\\,I)'` + `-vsync vfr`
- 关键帧-场景变化：`select='gt(scene\\,THRESH)'` + `-vsync vfr`
- 固定 FPS：`fps=FPS`
- 逐帧：默认输出全帧（配合时间区间与 `-frames:v` 限制；必要时可按 `-vsync 0`）

通用约束：
- 时间区间：`-ss startSec` + `-to endSec`（或 `-t duration`）
- 最大帧数：`-frames:v maxFramesRemaining`

进度采集：
- 使用 `-progress pipe:1` 解析 `out_time_ms`/`frame`/`speed`，将最新进度写入内存 + DB（DB 写入节流，如 1s 一次）

#### 4) 任务管理（Queue/Worker/Cancellation）
- 一个任务可多次“运行”（continue），但 taskId 不变；每次运行使用同一输出目录并追加帧
- 取消：保存 `context.CancelFunc` 与 `*exec.Cmd`，由 `cancel` API 触发；取消后任务状态更新为 `PAUSED_USER` 或 `CANCELLED`
- 继续：根据任务游标计算新的 `startSec`（从上次 `cursorOutTimeSec + epsilon` 起），并基于用户扩展后的 endSec/maxFrames 继续

状态建议（更清晰的状态机）：
- `PENDING`：排队中
- `PREPARING`：准备中（如缓存下载）
- `RUNNING`：运行中
- `PAUSED_USER`：用户暂停/终止（可继续）
- `PAUSED_LIMIT`：达到限制（必须扩展限制后才能继续）
- `FINISHED`：自然结束（到 endSec 或视频结束且未触发 maxFrames）
- `FAILED`：失败（记录 error）

#### 5) 输出目录与静态访问
输出目录放在 `./upload` 下，天然可通过现有静态路由访问：
- `upload/extract/{taskId}/frames/*.jpg`
- 可选：`upload/extract/{taskId}/thumbs/*.jpg`（用于列表预览，减少带宽与 DOM 压力）

后端返回对外 URL：`http://{host}/upload/extract/{taskId}/frames/...`

## 架构设计（增量）
```mermaid
flowchart TD
  FE[Vue 前端] -->|POST 创建任务| API[Go /api]
  FE -->|GET 轮询状态/分页帧列表| API
  API --> DB[(MySQL: video_extract_task/frame)]
  API --> FS[(./upload/extract/...)]
  API --> FF[ffprobe/ffmpeg]
  API --> MTP[mtPhoto: ResolveFilePath/下载缓存(可选)]
```

## 架构决策 ADR

### ADR-002: 采用“内置任务队列 + 轮询”的抽帧实现（推荐）
**上下文:** 现有项目的 WS 主要用于聊天代理，直接复用推送容易与聊天消息混淆；同时抽帧属于 CPU/IO 密集任务，需要异步化与可取消能力。  
**决策:** 后端内置任务队列与 worker pool，前端通过轮询获取状态/增量结果；后续如需强实时再新增独立任务 WS/SSE。  
**理由:** 复杂度低、集成自然、风险可控、可演进。  
**替代方案:** 复用聊天 WS 推送 → 拒绝原因: 协议混杂且易破坏现有消息处理。  
**影响:** 新增 DB 表与文件输出目录；需要处理任务幂等、取消与续跑游标。

## API设计（拟定）

### [GET] /api/probeVideo
- **请求:** `sourceType` + source 参数（upload: localPath；mtPhoto: md5）
- **响应:** `{code:0,data:{durationSec,width,height,avgFps,sourceResolved:{...}}}`

### [POST] /api/createVideoExtractTask
- **请求(JSON):**
  - `sourceType`: `upload|mtPhoto`
  - `source`: `{localPath}` 或 `{md5}`
  - `mode`: `keyframe|fps|all`
  - `keyframeMode`: `iframe|scene`（mode=keyframe）
  - `sceneThreshold`: number（可选）
  - `fps`: number（mode=fps）
  - `startSec?`, `endSec?`
  - `maxFrames`: number（必填）
  - `outputFormat`: `jpg|png`（默认 jpg）
  - `jpgQuality?`: 1-31（可选）
- **响应:** `{code:0,data:{taskId}}`

### [GET] /api/getVideoExtractTaskList
- **请求:** `page,pageSize`
- **响应:** `{code:0,data:{items:[...],total,page,pageSize}}`

### [GET] /api/getVideoExtractTaskDetail
- **请求:** `taskId`, `frameCursor?`, `framePageSize?`, `logOffset?`
- **响应:** `{code:0,data:{task:{...},frames:{items:[...],nextCursor},logs:{items:[...],nextOffset}}}`

### [POST] /api/cancelVideoExtractTask
- **请求:** `{taskId}`
- **响应:** `{code:0,data:{status}}`

### [POST] /api/continueVideoExtractTask
- **请求:** `{taskId, extendEndSec?, addMaxFrames?}`（或 `newEndSec/newMaxFrames`）
- **响应:** `{code:0,data:{status}}`

### [POST] /api/deleteVideoExtractTask
- **请求:** `{taskId, deleteFiles:boolean}`
- **响应:** `{code:0,data:{deleted:boolean}}`

## 数据模型（拟定）
```sql
-- 抽帧任务表
CREATE TABLE IF NOT EXISTS video_extract_task (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(32) NULL,
  source_type VARCHAR(16) NOT NULL,
  source_ref VARCHAR(500) NOT NULL,
  input_abs_path TEXT NOT NULL,
  output_dir_local_path VARCHAR(500) NOT NULL,
  mode VARCHAR(16) NOT NULL,
  keyframe_mode VARCHAR(16) NULL,
  fps DOUBLE NULL,
  scene_threshold DOUBLE NULL,
  start_sec DOUBLE NULL,
  end_sec DOUBLE NULL,
  max_frames_total INT NOT NULL,
  frames_extracted INT NOT NULL,
  video_width INT NOT NULL,
  video_height INT NOT NULL,
  duration_sec DOUBLE NULL,
  cursor_out_time_sec DOUBLE NULL,
  status VARCHAR(16) NOT NULL,
  stop_reason VARCHAR(32) NULL,
  last_error TEXT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_task_id (task_id),
  INDEX idx_vet_updated_at (updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频抽帧任务';

-- 帧索引表（受 maxFrames 保护，预计规模可控）
CREATE TABLE IF NOT EXISTS video_extract_frame (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  seq INT NOT NULL,
  rel_path VARCHAR(500) NOT NULL,
  time_sec DOUBLE NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_task_seq (task_id, seq),
  INDEX idx_vef_task_id (task_id),
  INDEX idx_vef_id (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频抽帧帧索引';
```

## 安全与性能
- **安全:**
  - 禁止 `sh -c` 拼接命令；使用 `exec.CommandContext("ffmpeg", args...)`
  - 输入校验：`localPath` 仅允许落在 `./upload`；mtPhoto md5 仅允许 `[0-9a-fA-F]{32}`；输出目录由服务端生成
  - 访问控制：接口走既有 JWT；任务若按 userId 隔离，需要校验任务归属（初期可按全局任务，但建议绑定 userId）
- **性能:**
  - 任务 worker pool 限制并发（即使用户“暂时不限制资源”，也建议设置默认并发=CPU 核数或固定小值，避免拖垮主 API）
  - 前端避免一次性渲染大量图片：分页/增量；必要时引入虚拟滚动与缩略图
  - 轮询自适应：running 1s；后台/不可见降级 5s 或暂停

## 测试与部署
- **测试:**
  - Go：为任务状态流转、输入解析、取消/继续幂等增加单测
  - 前端：以 `npm run build` 作为编译验证；关键逻辑（store/参数校验）可增量添加单测（如已有测试框架）
- **部署:**
  - Docker 最终镜像安装 `ffmpeg`（包含 `ffprobe`）
  - 生产环境需确保容器内 `./upload` 挂载持久卷（否则任务产物随容器销毁而丢失）

## 前端页面设计（含 Gemini 审查结论）

### 入口与信息架构
- **入口 1（就近）:** 在 `MediaPreview` 预览视频时增加“抽帧”按钮（覆盖 upload 库视频与 mtPhoto 视频）
- **入口 2（推荐）:** 在 `SettingsDrawer` 增加“任务中心/抽帧任务”入口，避免用户必须回到某个视频才能查看任务（减少 Modal 迷失）
- **避免弹窗套娃:** 提交任务后优先 Toast 提示“任务已开始”，提供“查看详情”按钮；不强制自动打开任务详情（Gemini 建议）

### CreateModal（VideoExtractCreateModal）
- 模式选择：关键帧 / 固定FPS / 逐帧
- 参数区：
  - 关键帧：I 帧 / sceneThreshold（场景阈值）
  - 固定FPS：fps
  - 通用：startSec/endSec（建议提供双滑块 Range Slider；同时保留数字输入）
  - maxFrames（必填；结合预估输出量进行强提醒）
- 展示 probe 信息：duration/width/height/avgFps
- **预估与风险提示（Gemini 建议）:** 实时估算输出帧数量；过大时给出红色警告，必要时禁用提交

### TaskModal（VideoExtractTaskModal）
- 任务列表：状态、模式、进度、已输出帧数、限制摘要（timeRange/maxFrames）、更新时间
- 任务详情：
  - 顶部显示参数与输出目录（可复制）
  - 操作按钮：停止（需二次确认）、继续（当 `PAUSED_LIMIT` 时必须弹 ContinueModal）、删除（运行中需先 cancel 再清理）
  - 帧图预览：复用 `InfiniteMediaGrid`（分页/增量加载；后续数据量大时引入虚拟滚动）
  - 日志区：显示最近 N 条；自动滚动到底部，用户手动上滚后暂停自动滚动（Gemini 建议）

### 状态机（Gemini 建议吸收）
- 区分 `PAUSED_USER` 与 `PAUSED_LIMIT`，避免“继续”无效造成困惑
- 按钮禁用/防抖：创建/停止/继续/删除均需请求中禁用，避免重复点击
