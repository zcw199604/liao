# 数据模型（MySQL / Redis / 内存缓存）

> 本文档整理后端使用到的主要数据表与缓存结构，用于保持 Java/Go 之间 100% 行为兼容（含已知“历史遗留/不一致”点的说明）。Go 侧建表与缓存实现见 `internal/app/schema.go`、`internal/app/user_info_cache*.go`、`internal/app/image_cache.go`、`internal/app/forceout.go`。

---

## 1. MySQL 数据表

### 1.1 `identity`（身份表）

**创建位置**：`com.zcw.service.IdentityService#createTableIfNotExists`（启动时执行 `CREATE TABLE IF NOT EXISTS`）  
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

**实体/仓库**：`com.zcw.model.Favorite` + `com.zcw.repository.FavoriteRepository`（JPA 自动建表/更新）  
**用途**：前端“本地收藏列表”（与上游收藏接口无关）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| identity_id | VARCHAR(32) | NOT NULL | 本地身份ID |
| target_user_id | VARCHAR(64) | NOT NULL | 被收藏的对方用户ID |
| target_user_name | VARCHAR(64) | 可空 | 显示用昵称 |
| create_time | DATETIME | NOT NULL | 创建时间（@PrePersist 自动填充） |

**索引**
- `idx_identity_id (identityId)`
- `idx_target_user_id (targetUserId)`

---

### 1.3 `media_file`（媒体库：物理文件与元数据）

**实体/仓库**：`com.zcw.model.MediaFile` + `com.zcw.repository.MediaFileRepository`（JPA 自动建表/更新）  
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

**实体/仓库**：`com.zcw.model.MediaSendLog` + `com.zcw.repository.MediaSendLogRepository`（JPA 自动建表/更新）  
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
**实体/仓库**：`com.zcw.model.MediaUploadHistory` + `com.zcw.repository.MediaUploadHistoryRepository`  
**当前状态**：
- 业务主链路已迁移为 `media_file + media_send_log`（见 `com.zcw.service.MediaUploadService`）
- 但 `com.zcw.service.FileStorageService#findLocalPathByMD5` 仍查询该表：  
  `SELECT local_path FROM media_upload_history WHERE file_md5 = ? LIMIT 1`

> Go 重构需注意：为保持“现状兼容”，需要保留对该表的读取逻辑（即便新上传记录主要写入 `media_file`）。

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

### 2.3 内存缓存补充

- `ImageCacheService`：缓存 `userId -> local_path[]`，过期 3 小时（用于上传弹窗快速展示）
- `ForceoutManager`：缓存被 forceout 的 userId 与过期时间戳（5 分钟）
