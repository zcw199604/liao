# 数据模型

## 概述
当前服务启动时通过 `internal/app/schema.go` 调用 `internal/database/migrator.go`，按 `DB_URL` scheme 选择 `sql/mysql/*.sql` 或 `sql/postgres/*.sql`，并按文件名顺序执行迁移。缓存由内存实现或 Redis 实现承担，媒体文件落在 `./upload`，mtPhoto 本地文件由 `LSP_ROOT` 映射。

---

## 数据表

### `schema_migrations`
**描述:** 记录已执行迁移版本。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| version | VARCHAR(255) | 主键 | SQL 文件名去扩展名 |
| applied_at | TIMESTAMP | 非空 | 迁移记录时间 |

### `identity`
**描述:** 本地身份池。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(32) | 主键 | 本地身份 ID |
| name | VARCHAR(50) | 非空 | 名字 |
| sex | VARCHAR(10) | 非空 | 性别 |
| created_at | DATETIME/TIMESTAMP | 可空 | 创建时间 |
| last_used_at | DATETIME/TIMESTAMP | 可空，索引 | 最近使用时间 |

### `chat_favorites`
**描述:** 本地聊天收藏。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | 主键，自增 | 收藏 ID |
| identity_id | VARCHAR(32) | 非空，索引 | 本地身份 ID |
| target_user_id | VARCHAR(64) | 非空，索引 | 被收藏聊天对象 |
| target_user_name | VARCHAR(64) | 可空 | 展示昵称 |
| create_time | DATETIME/TIMESTAMP | 非空 | 收藏时间 |

### `chat_user_archive`
**描述:** 聊天用户本地归档，用于上游删除后仍能恢复列表信息，也作为跨身份联系人候选的本地数据源。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | 主键，自增 | 归档 ID |
| owner_user_id | VARCHAR(64) | 非空，联合唯一 | 当前身份 |
| target_user_id | VARCHAR(64) | 非空，联合唯一 | 对方用户 |
| snapshot_json | LONGTEXT/TEXT | 可空 | 最近用户快照 |
| last_msg | TEXT | 可空 | 最近消息摘要 |
| last_time | VARCHAR(64) | 可空 | 最近消息时间原始值 |
| seen_in_history | TINYINT/SMALLINT | 非空 | 是否历史列表可见 |
| seen_in_favorite | TINYINT/SMALLINT | 非空 | 是否收藏列表可见 |
| first_seen_at | DATETIME/TIMESTAMP | 非空 | 首次见到时间 |
| last_seen_at | DATETIME/TIMESTAMP | 非空，索引 | 最近见到时间 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |
| updated_at | DATETIME/TIMESTAMP | 非空 | 更新时间 |

**使用约束:**
- `owner_user_id + target_user_id` 表示某个本地身份与目标用户的归档关系。
- 历史/收藏列表代理和 WebSocket 匹配成功事件会写入该表。
- `GET /api/chat/contactCandidates` 复用该表读取来源身份候选，不新增联系人池表。
- 对外返回候选时，`snapshot_json` 会清理 cookie、token、JWT、Authorization、access code、password、secret 等敏感字段。

### `media_file`
**描述:** 本地媒体库主表。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | 主键，自增 | 媒体 ID |
| user_id | VARCHAR(32) | 非空，索引 | 上传用户 |
| original_filename | TEXT/VARCHAR | 非空 | 原始文件名 |
| local_filename | TEXT/VARCHAR | 非空 | 本地文件名 |
| remote_filename | TEXT/VARCHAR | 非空 | 上游返回文件名 |
| remote_url | VARCHAR(500) | 非空 | 上游访问 URL |
| local_path | VARCHAR(500) | 非空，索引 | 本地相对路径 |
| file_size | BIGINT | 非空 | 字节大小 |
| file_type | VARCHAR(50) | 非空 | MIME |
| file_extension | VARCHAR(10) | 非空 | 扩展名 |
| file_md5 | VARCHAR(32) | 可空，索引 | MD5 |
| media_width | INT | 可空 | 图片/媒体宽度，用于前端瀑布流布局 |
| media_height | INT | 可空 | 图片/媒体高度，用于前端瀑布流布局 |
| upload_time | DATETIME/TIMESTAMP | 非空 | 上传时间 |
| update_time | DATETIME/TIMESTAMP | 可空，索引 | 最近活跃时间 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |

### `douyin_media_file`
**描述:** 抖音导入媒体库，与普通媒体分表保存。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | 主键，自增 | 媒体 ID |
| user_id | VARCHAR(32) | 非空，索引 | 导入时身份，未选择时可为 `pre_identity` |
| sec_user_id | VARCHAR(128) | 可空，索引 | 抖音用户 sec_uid |
| detail_id | VARCHAR(64) | 可空，索引 | 作品 ID |
| author_unique_id | VARCHAR(64) | 可空 | 作者抖音号快照 |
| author_name | VARCHAR(128) | 可空 | 作者昵称快照 |
| original_filename/local_filename/remote_filename/remote_url/local_path | 多类型 | 非空 | 同 `media_file` |
| file_size/file_type/file_extension/file_md5 | 多类型 | 部分可空 | 文件元数据 |
| media_width/media_height | INT | 可空 | 图片/媒体尺寸，用于前端瀑布流布局 |
| upload_time/update_time/created_at | DATETIME/TIMESTAMP | 部分可空 | 时间字段 |

### `media_send_log`
**描述:** 媒体发送关系。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | 主键，自增 | 记录 ID |
| user_id | VARCHAR(32) | 非空，索引 | 发送者 |
| to_user_id | VARCHAR(32) | 非空，索引 | 接收者 |
| local_path | VARCHAR(500) | 非空 | 本地媒体路径 |
| remote_url | VARCHAR(500) | 非空 | 上游 URL |
| send_time | DATETIME/TIMESTAMP | 非空，索引 | 发送时间 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |

### `media_upload_history`
**描述:** 历史兼容上传记录表。当前主链路以 `media_file` 与 `media_send_log` 为主，但部分查找逻辑仍兼容此表。

### `image_hash`
**描述:** 本地图片 MD5/pHash 索引，用于 `/api/checkDuplicateMedia`。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | INT | 主键，自增 | 索引 ID |
| file_path | VARCHAR(1000) | 非空 | 图片路径 |
| file_name | VARCHAR(255) | 非空 | 文件名 |
| file_dir | VARCHAR(500) | 可空 | 目录 |
| md5_hash | VARCHAR(32) | 非空，索引 | MD5 |
| phash | BIGINT | 非空，索引 | 64 位感知哈希 |
| file_size | BIGINT | 可空 | 字节大小 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |

### `system_config`
**描述:** 全局系统配置 Key-Value 表。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| config_key | VARCHAR(64) | 主键 | 配置键 |
| config_value | TEXT | 非空 | 配置值 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |
| updated_at | DATETIME/TIMESTAMP | 非空 | 更新时间 |

### `video_extract_task`
**描述:** 视频抽帧任务。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| task_id | VARCHAR(64) | 唯一 | 任务 ID |
| user_id | VARCHAR(32) | 可空，索引 | 创建者 |
| source_type | VARCHAR(16) | 非空 | `upload` 或 `mtPhoto` |
| source_ref | VARCHAR(500) | 非空 | 来源引用 |
| input_abs_path | TEXT | 非空 | 服务端输入绝对路径 |
| output_dir_local_path | VARCHAR(500) | 非空 | 输出目录相对路径 |
| output_format | VARCHAR(8) | 非空 | `jpg` 或 `png` |
| mode/keyframe_mode/fps/scene_threshold/start_sec/end_sec | 多类型 | 部分可空 | 抽帧参数 |
| max_frames_total/frames_extracted | INT | 非空 | 帧数限制与进度 |
| video_width/video_height/duration_sec | 多类型 | 部分可空 | 源视频元数据 |
| cursor_out_time_sec | DOUBLE | 可空 | 续跑游标 |
| status | VARCHAR(16) | 非空 | 任务状态 |
| stop_reason/last_error/last_logs | 多类型 | 可空 | 停止原因与日志 |
| created_at/updated_at | DATETIME/TIMESTAMP | 非空 | 时间字段 |

### `video_extract_frame`
**描述:** 抽帧产物索引。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| task_id | VARCHAR(64) | 非空，联合唯一 | 任务 ID |
| seq | INT | 非空，联合唯一 | 帧序号 |
| rel_path | VARCHAR(500) | 非空 | 相对 `upload` 路径 |
| time_sec | DOUBLE | 可空 | 帧时间 |
| created_at | DATETIME/TIMESTAMP | 非空 | 创建时间 |

### 抖音收藏表
**描述:** 全局抖音用户/作品收藏和标签体系。

| 表名 | 用途 |
|------|------|
| `douyin_favorite_user` | 收藏抖音用户及最近解析快照 |
| `douyin_favorite_aweme` | 收藏抖音作品及元数据 |
| `douyin_favorite_user_aweme` | 收藏用户的作品列表，含置顶/发布时间/作者快照 |
| `douyin_favorite_user_tag` | 用户收藏标签定义 |
| `douyin_favorite_user_tag_map` | 用户收藏与标签映射 |
| `douyin_favorite_aweme_tag` | 作品收藏标签定义 |
| `douyin_favorite_aweme_tag_map` | 作品收藏与标签映射 |

### `mtphoto_folder_favorite`
**描述:** mtPhoto 文件夹收藏。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| folder_id | BIGINT | 唯一 | mtPhoto 文件夹 ID |
| folder_name | VARCHAR(255) | 非空 | 文件夹名称 |
| folder_path | VARCHAR(1024) | 非空 | 文件夹路径 |
| cover_md5 | VARCHAR(32) | 可空 | 封面 MD5 |
| tags_json | LONGTEXT/TEXT | 非空 | 标签 JSON 数组 |
| note | TEXT | 可空 | 备注 |
| created_at/updated_at | DATETIME/TIMESTAMP | 非空 | 时间字段 |

---

## 缓存模型

### UserInfoCacheService
- **内存实现:** 默认启用。
- **Redis 实现:** `CACHE_TYPE=redis` 时启用。
- **内容:** 用户信息缓存与最后消息缓存。
- **Key 约定:** 用户信息默认 `user:info:{userId}`，最后消息默认 `user:lastmsg:{conversationKey}`。

### ChatHistoryCacheService
- **实现:** Redis 模式下启用，保存聊天消息 `contents_list`。
- **Key 约定:** 默认 `user:chathistory:{conversationKey}`。
- **读取策略:** 最新页仍请求上游；历史翻页可在 Redis 命中足够时跳过上游。

### 进程内缓存
- `ImageCacheService`: 缓存用户上传图片路径。
- `ForceoutManager`: 记录 5 分钟内禁止重连的 userId。
- 抖音下载缓存: 用随机 key 映射可下载资源，避免前端传任意 URL。

### 前端消息缓存
- Pinia `message` store 的 `chatHistory` 和 `firstTidMap` 使用 `conversationKey(ownerUserId, targetUserId)` 作为新路径 key，格式为 `{当前身份ID}:{目标用户ID}`。
- 该 key 只存在于前端内存状态，用于隔离不同身份联系同一目标时的消息、乐观发送状态、超时和回显合并。
- 旧单参数读取在唯一匹配时可兼容回读，但新增写入应使用当前身份 ID 或显式会话 key。
