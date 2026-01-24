# 数据模型（MySQL / Redis / 内存缓存）

> 本文档整理当前 Go 后端使用到的主要数据表与缓存结构；历史 Java(Spring Boot) 源码已从仓库移除，文中提到“原 Java 行为”仅用于兼容性说明。建表与缓存实现见 `internal/app/schema.go`、`internal/app/user_info_cache*.go`、`internal/app/image_cache.go`、`internal/app/forceout.go`。

---

## 1. MySQL 数据表

### 1.1 `identity`（身份表）

**创建位置**：`internal/app/schema.go`（启动时执行 `CREATE TABLE IF NOT EXISTS`）  
**CRUD 实现**：`internal/app/identity.go`、`internal/app/identity_handlers.go`  
**用途**：本地身份池管理（Identity CRUD + last_used_at 排序）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | VARCHAR(32) | PK | 用户ID（32位随机字符串） |
| name | VARCHAR(50) | NOT NULL | 名字 |
| sex | VARCHAR(10) | NOT NULL | 性别（男/女） |
| created_at | DATETIME | 可空 | 创建时间 |
| last_used_at | DATETIME | 可空 | 最后使用时间 |

**索引**
- `idx_last_used_at (last_used_at DESC)`

---

### 1.2 `chat_favorites`（本地聊天收藏）

**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）  
**实现**：`internal/app/favorite.go`、`internal/app/favorite_handlers.go`  
**用途**：前端“本地收藏列表”（与上游收藏接口无关）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| identity_id | VARCHAR(32) | NOT NULL | 本地身份ID |
| target_user_id | VARCHAR(64) | NOT NULL | 被收藏的对方用户ID |
| target_user_name | VARCHAR(64) | 可空 | 显示用昵称 |
| create_time | DATETIME | NOT NULL | 创建时间（@PrePersist 自动填充） |

**索引**
- `idx_identity_id (identity_id)`
- `idx_target_user_id (target_user_id)`

---

### 1.3 `media_file`（媒体库：物理文件与元数据）

**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）  
**相关实现**：`internal/app/media_upload.go`、`internal/app/media_history_handlers.go`、`internal/app/file_storage.go`  
**用途**：记录每个用户上传的媒体文件（按 MD5 去重的意图），用于“全站图片库/用户上传历史”等。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| user_id | VARCHAR(32) | NOT NULL | 上传用户ID |
| original_filename | TEXT/VARCHAR | NOT NULL | 原始文件名 |
| local_filename | TEXT/VARCHAR | NOT NULL | 本地存储文件名（basename） |
| remote_filename | TEXT/VARCHAR | NOT NULL | 上游返回文件名（相对路径） |
| remote_url | VARCHAR(500) | NOT NULL | 上游可访问 URL |
| local_path | VARCHAR(500) | NOT NULL | 本地相对路径（以 `/images/...` 或 `/videos/...` 开头） |
| file_size | BIGINT | NOT NULL | 字节大小 |
| file_type | VARCHAR(50) | NOT NULL | MIME |
| file_extension | VARCHAR(10) | NOT NULL | 扩展名（不含点） |
| file_md5 | VARCHAR(32) | 可空 | MD5（用于去重） |
| upload_time | DATETIME | NOT NULL | 首次上传时间 |
| update_time | DATETIME | 可空 | 最后活跃时间（用于排序） |
| created_at | DATETIME | NOT NULL | 创建时间 |

**索引（来自实体注解）**
- `idx_mf_user_id (user_id)`
- `idx_mf_file_md5 (file_md5)`
- `idx_mf_update_time (update_time DESC)`
- `idx_mf_local_path (local_path)`

**维护建议**
- `update_time` 用于全站图片库/上传历史的排序，应避免混用“应用侧 now”与“数据库 CURRENT_TIMESTAMP”等不同时间源；建议统一由服务端写入同一时间源（当前实现为应用侧 `now`），避免因时区/时钟漂移导致排序异常。
- 若遗留表 `media_upload_history` 存在 `file_md5` 为空或出现多条相同 `file_md5` 的情况，可使用 `/api/repairMediaHistory` 批量补齐与去重（默认 dry-run，需 `commit=true` 才会写入/删除）。

---

### 1.4 `media_send_log`（媒体发送日志）

**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）  
**相关实现**：`internal/app/media_upload.go`、`internal/app/media_history_handlers.go`  
**用途**：记录“fromUserId → toUserId”发送了哪个本地文件（local_path），用于聊天历史图片查询与去重。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| user_id | VARCHAR(32) | NOT NULL | 发送者 |
| to_user_id | VARCHAR(32) | NOT NULL | 接收者 |
| local_path | VARCHAR(500) | NOT NULL | 对应 `media_file.local_path` |
| remote_url | VARCHAR(500) | NOT NULL | 冗余上游 URL（用于去重/展示） |
| send_time | DATETIME | NOT NULL | 发送时间 |
| created_at | DATETIME | NOT NULL | 创建时间 |

**索引（来自实体注解）**
- `idx_msl_user_id (user_id)`
- `idx_msl_to_user_id (to_user_id)`
- `idx_msl_send_time (send_time DESC)`

---

### 1.5 `media_upload_history`（历史遗留上传记录表）

**SQL 来源**：`sql/init.sql`  
**创建位置**：`internal/app/schema.go`（兜底建表；与 `sql/init.sql` 一致）  
**当前状态**：
- 业务主链路已迁移为 `media_file + media_send_log`（见 `internal/app/media_upload.go`）
- 但 `internal/app/file_storage.go` 的 `FindLocalPathByMD5` 仍查询该表：  
  `SELECT local_path FROM media_upload_history WHERE file_md5 = ? LIMIT 1`

> 为保持“现状兼容”，Go 实现仍保留对该表的读取逻辑（即便新上传记录主要写入 `media_file`）。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT AUTO_INCREMENT PK | 主键 |
| user_id | VARCHAR(32) | 上传用户 |
| to_user_id | VARCHAR(32) | 接收者（发送时填充） |
| original_filename | VARCHAR(255) | 原始文件名 |
| local_filename | VARCHAR(255) | 本地文件名 |
| remote_filename | VARCHAR(255) | 上游文件名 |
| remote_url | VARCHAR(500) | 上游 URL |
| local_path | VARCHAR(500) | 本地路径 |
| file_size | BIGINT | 文件大小 |
| file_type | VARCHAR(50) | MIME |
| file_extension | VARCHAR(10) | 扩展名 |
| upload_time | DATETIME | 上传时间 |
| send_time | DATETIME | 发送时间 |
| file_md5 | VARCHAR(32) | MD5 |
| created_at | DATETIME | 记录创建时间 |

---

### 1.6 `image_hash`（本地图片哈希索引表）

**用途**：存储本地图片的路径与哈希信息（MD5 + pHash），用于查重与相似图片检索（见 `/api/checkDuplicateMedia`）。  
**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`；实际数据通常由外部扫描/入库流程写入）。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | INT | PK, AUTO_INCREMENT | 主键 |
| file_path | VARCHAR(1000) | NOT NULL | 图片完整路径（或统一的存储路径标识） |
| file_name | VARCHAR(255) | NOT NULL | 文件名 |
| file_dir | VARCHAR(500) | 可空 | 文件目录 |
| md5_hash | VARCHAR(32) | NOT NULL | 文件 MD5（32位 hex） |
| phash | BIGINT | NOT NULL | 图片 pHash（64位，BIGINT 存储） |
| file_size | BIGINT | 可空 | 文件大小（字节） |
| created_at | DATETIME | NOT NULL | 记录创建时间 |

**索引**
- `idx_ih_md5_hash (md5_hash)`
- `idx_ih_phash (phash)`

---

### 1.7 `system_config`（系统全局配置表）

**用途**：存储系统级全局配置（所有用户共用），以 Key-Value 形式落库，供前端 Settings 面板与后端逻辑读取。  
**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| config_key | VARCHAR(64) | PK | 配置键 |
| config_value | TEXT | NOT NULL | 配置值（字符串化） |
| created_at | DATETIME | NOT NULL | 创建时间 |
| updated_at | DATETIME | NOT NULL | 更新时间 |

**已定义配置键（节选）**
- `image_port_mode`：图片端口策略（`fixed/probe/real`）
- `image_port_fixed`：固定模式图片端口（默认 `9006`）
- `image_port_real_min_bytes`：真实图片请求最小字节阈值（默认 `2048`）

---

### 1.8 `video_extract_task`（视频抽帧任务表）

**用途**：记录视频抽帧任务的来源、参数、状态与进度信息；产物目录落在 `./upload/extract/{taskId}/frames/`，由前端“任务中心”分页预览。  
**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| task_id | VARCHAR(64) | UNIQUE | 任务ID（uuid） |
| user_id | VARCHAR(32) | 可空 | 创建者用户ID（前端可传；非强鉴权） |
| source_type | VARCHAR(16) | NOT NULL | 来源类型：`upload` / `mtPhoto` |
| source_ref | VARCHAR(500) | NOT NULL | 来源引用：upload=localPath；mtPhoto=md5 |
| input_abs_path | TEXT | NOT NULL | 输入文件绝对路径（仅服务端内部使用） |
| output_dir_local_path | VARCHAR(500) | NOT NULL | 输出目录（相对 upload，形如 `/extract/{taskId}`） |
| output_format | VARCHAR(8) | NOT NULL | 输出图片格式：`jpg`/`png` |
| jpg_quality | INT | 可空 | JPG 质量（ffmpeg `-q:v`，1-31） |
| mode | VARCHAR(16) | NOT NULL | 抽帧模式：`keyframe`/`fps`/`all` |
| keyframe_mode | VARCHAR(16) | 可空 | 关键帧模式：`iframe`/`scene` |
| fps | DOUBLE | 可空 | 固定 FPS（mode=fps） |
| scene_threshold | DOUBLE | 可空 | 场景阈值（keyframe_mode=scene） |
| start_sec | DOUBLE | 可空 | 起始秒 |
| end_sec | DOUBLE | 可空 | 结束秒 |
| max_frames_total | INT | NOT NULL | 最大帧数上限（总） |
| frames_extracted | INT | NOT NULL | 已输出帧数 |
| video_width | INT | NOT NULL | 视频宽 |
| video_height | INT | NOT NULL | 视频高 |
| duration_sec | DOUBLE | 可空 | 视频时长（秒） |
| cursor_out_time_sec | DOUBLE | 可空 | 续跑游标（秒，绝对时间） |
| status | VARCHAR(16) | NOT NULL | `PENDING/PREPARING/RUNNING/PAUSED_USER/PAUSED_LIMIT/FINISHED/FAILED` |
| stop_reason | VARCHAR(32) | 可空 | `MAX_FRAMES/END_SEC/EOF/USER/ERROR` |
| last_error | TEXT | 可空 | 最后错误 |
| last_logs | TEXT | 可空 | 最后日志片段（JSON 数组字符串） |
| created_at | DATETIME | NOT NULL | 创建时间 |
| updated_at | DATETIME | NOT NULL | 更新时间 |

**索引**
- `uk_vet_task_id (task_id)`
- `idx_vet_updated_at (updated_at DESC)`
- `idx_vet_user_id (user_id)`

---

### 1.9 `video_extract_frame`（视频抽帧帧索引表）

**用途**：记录每个任务生成的帧图文件列表，支持按 `seq` 增量分页返回（用于运行中实时预览）。  
**创建位置**：`internal/app/schema.go`。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| task_id | VARCHAR(64) | NOT NULL | 任务ID |
| seq | INT | NOT NULL | 帧序号（从 1 开始；跨续跑单调递增） |
| rel_path | VARCHAR(500) | NOT NULL | 相对 upload 路径（`/extract/{taskId}/frames/frame_000001.jpg`） |
| time_sec | DOUBLE | 可空 | 帧时间（秒，可选） |
| created_at | DATETIME | NOT NULL | 创建时间 |

**索引**
- `uk_vef_task_seq (task_id, seq)`
- `idx_vef_task_id (task_id)`

---

### 1.10 `douyin_favorite_user`（抖音用户收藏，全局）

**用途**：存储“已解析后手动收藏”的抖音用户（全局一份，不按本地身份隔离），用于前端收藏列表与一键再次解析。  
**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）。  
**实现**：`internal/app/douyin_favorite.go`、`internal/app/douyin_favorite_handlers.go`。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| sec_user_id | VARCHAR(128) | PK | 抖音 `sec_user_id`（sec_uid） |
| source_input | TEXT | 可空 | 收藏时原始输入（分享文本/链接/sec_uid） |
| display_name | VARCHAR(128) | 可空 | 展示名（best-effort，可由前端/后端补齐） |
| avatar_url | VARCHAR(500) | 可空 | 头像 URL（best-effort） |
| profile_url | VARCHAR(500) | 可空 | 用户主页 URL（best-effort） |
| last_parsed_at | DATETIME | 可空 | 最后一次解析时间 |
| last_parsed_count | INT | 可空 | 最后一次解析得到的作品数量（best-effort） |
| last_parsed_raw | LONGTEXT | 可空 | 最后一次解析原始数据（JSON 字符串，可选） |
| created_at | DATETIME | NOT NULL | 创建时间 |
| updated_at | DATETIME | NOT NULL | 更新时间 |

**索引**
- `idx_dfu_updated_at (updated_at DESC)`
- `idx_dfu_created_at (created_at DESC)`

---

### 1.11 `douyin_favorite_aweme`（抖音作品收藏，全局）

**用途**：存储“已解析后手动收藏”的抖音作品（全局一份），用于前端收藏列表展示与一键再次解析。  
**创建位置**：`internal/app/schema.go`（启动时 `CREATE TABLE IF NOT EXISTS`）。  
**实现**：`internal/app/douyin_favorite.go`、`internal/app/douyin_favorite_handlers.go`。

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| aweme_id | VARCHAR(64) | PK | 作品 ID（aweme_id） |
| sec_user_id | VARCHAR(128) | 可空 | 作者 `sec_user_id`（可选） |
| type | VARCHAR(16) | 可空 | `video/image`（best-effort） |
| description | TEXT | 可空 | 作品描述/标题（best-effort） |
| cover_url | VARCHAR(500) | 可空 | 封面 URL（best-effort） |
| raw_detail | LONGTEXT | 可空 | 作品解析原始数据（JSON 字符串，可选） |
| created_at | DATETIME | NOT NULL | 创建时间 |
| updated_at | DATETIME | NOT NULL | 更新时间 |

**索引**
- `idx_dfa_sec_user_id (sec_user_id)`
- `idx_dfa_updated_at (updated_at DESC)`
- `idx_dfa_created_at (created_at DESC)`

## 2. 缓存（内存 / Redis）

### 2.1 用户信息缓存（UserInfoCacheService）

接口：`internal/app.UserInfoCacheService`  
实现：
- 内存：`internal/app.MemoryUserInfoCacheService`（默认启用，`CACHE_TYPE=memory` 或未配置）
- Redis：`internal/app.RedisUserInfoCacheService`（`CACHE_TYPE=redis`）

Redis 连接方式（优先级从高到低）：
- `UPSTASH_REDIS_URL` / `REDIS_URL`：支持 `redis://` / `rediss://`（注意 URL 可能包含密码，文档示例必须用占位符）
- `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB`：传统四元组配置

写入策略（Redis 模式）：
- L1 本地缓存：立即写入（同实例内读优先命中本地缓存）
- L2 Redis：写入进入队列，按 `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` 批量 flush（默认 60 秒；同一 key 在间隔内多次更新会合并为最后一次）

**缓存内容**
- `CachedUserInfo`：`userId/nickname/gender/age/address/updateTime`
- `CachedLastMessage`：`conversationKey/fromUserId/toUserId/content/type/time/updateTime`

**会话 key**
- `conversationKey = min(userId1,userId2) + "_" + max(userId1,userId2)`（字典序排序，双向一致）

### 2.2 Redis Key 约定（当启用 Redis 模式）

由 `RedisUserInfoCacheService` 读取配置：
- `CACHE_REDIS_PREFIX`（默认：`user:info:`）
- `CACHE_REDIS_LASTMSG_PREFIX`（默认：`user:lastmsg:`）
- `CACHE_REDIS_EXPIRE_DAYS`（默认：7 天）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS`（默认：60 秒；写入队列批量 flush 间隔，用于降低写入频率/成本）
- `CACHE_REDIS_LOCAL_TTL_SECONDS`（默认：3600 秒；Redis L1 本地缓存 TTL，用于降低读频率）

Key 示例：
- 用户信息：`user:info:{userId}` → JSON（CachedUserInfo）
- 最后消息：`user:lastmsg:{conversationKey}` → JSON（CachedLastMessage）

### 2.3 聊天记录缓存（ChatHistoryCacheService，Redis 模式）

接口：`internal/app.ChatHistoryCacheService`  
实现：
- Redis：`internal/app.RedisChatHistoryCacheService`（`CACHE_TYPE=redis` 时启用；best-effort）

**缓存格式**
- 对齐上游 `/api/getMessageHistory` 新格式的 `contents_list`：单条消息 JSON 中保留 `Tid/id/toid/content/time` 等字段（字段可能随上游变化，缓存以“尽量不丢字段”为原则）。

**Redis Key 约定**
由 `RedisChatHistoryCacheService` 读取配置：
- `CACHE_REDIS_CHAT_HISTORY_PREFIX`（默认：`user:chathistory:`）
- `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS`（默认：30 天）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS`（同上；用于批量写入降低成本）

Key 示例（以 `conversationKey=min(u1,u2)+"_"+max(u1,u2)`）：
- 单 key：`user:chathistory:{conversationKey}` → ZSET
  - score：`tid`（可解析为 int64）
  - member：`"{tid}|{json}"`（`json` 为上游 `contents_list` 单条消息对象；用于保持字段兼容）
  - ttl：`CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS`

**清理策略**
- 每次写入会对相同 `tid` 先做 `ZREMRANGEBYSCORE(tid,tid)` 再 `ZADD`，避免重复记录同一条消息。
- 会话 key 使用 TTL 过期：在最近一次写入后 `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS` 天过期。

**读取策略（与接口行为关联）**
- `/api/getMessageHistory` 最新页（`firstTid=0`）始终请求上游以保证最新消息，同时读取 Redis 合并去重。
- 历史翻页（`firstTid>0`）若 Redis 命中足够覆盖页大小（默认 20）可跳过上游直接返回 Redis；不足时再拉取上游并合并。

### 2.4 内存缓存补充

- `ImageCacheService`：缓存 `userId -> local_path[]`，过期 3 小时（用于上传弹窗快速展示）
- `ForceoutManager`：缓存被 forceout 的 userId 与过期时间戳（5 分钟）
