-- 用户身份表
-- 数据库: hot_img

CREATE TABLE IF NOT EXISTS identity (
    id VARCHAR(32) PRIMARY KEY COMMENT '用户ID（32位随机字符串）',
    name VARCHAR(50) NOT NULL COMMENT '名字',
    sex VARCHAR(10) NOT NULL COMMENT '性别（男/女）',
    created_at DATETIME COMMENT '创建时间',
    last_used_at DATETIME COMMENT '最后使用时间',
    INDEX idx_last_used_at (last_used_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户身份表';

-- 媒体上传历史记录表
CREATE TABLE IF NOT EXISTS media_upload_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    user_id VARCHAR(32) NOT NULL COMMENT '上传用户ID（发送者）',
    to_user_id VARCHAR(32) COMMENT '接收用户ID（发送时填充，上传时为NULL）',
    original_filename VARCHAR(255) NOT NULL COMMENT '原始文件名',
    local_filename VARCHAR(255) NOT NULL COMMENT '本地存储文件名（UUID命名）',
    remote_filename VARCHAR(255) NOT NULL COMMENT '上游返回的文件名',
    remote_url VARCHAR(500) NOT NULL COMMENT '完整的远程访问URL',
    local_path VARCHAR(500) NOT NULL COMMENT '本地存储相对路径',
    file_size BIGINT NOT NULL COMMENT '文件大小（字节）',
    file_type VARCHAR(50) NOT NULL COMMENT '文件MIME类型',
    file_extension VARCHAR(10) NOT NULL COMMENT '文件扩展名',
    upload_time DATETIME NOT NULL COMMENT '上传时间',
    send_time DATETIME COMMENT '发送时间（实际发送给某人的时间）',
    file_md5 VARCHAR(32) COMMENT '文件MD5哈希值（用于本地去重）',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
    INDEX idx_user_id (user_id),
    INDEX idx_to_user_id (to_user_id),
    INDEX idx_user_to_user (user_id, to_user_id, send_time DESC),
    INDEX idx_remote_url (remote_url),
    INDEX idx_upload_time (upload_time DESC),
    INDEX idx_file_md5 (file_md5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体上传历史记录表';

-- 抖音用户收藏（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_user (
    sec_user_id VARCHAR(128) PRIMARY KEY COMMENT '抖音 sec_user_id（sec_uid）',
    source_input TEXT NULL COMMENT '收藏时的原始输入（分享文本/链接/sec_uid）',
    display_name VARCHAR(128) NULL COMMENT '展示名（可选）',
    avatar_url VARCHAR(500) NULL COMMENT '头像URL（可选）',
    profile_url VARCHAR(500) NULL COMMENT '用户主页URL（可选）',
    last_parsed_at DATETIME NULL COMMENT '最后一次解析时间',
    last_parsed_count INT NULL COMMENT '最后一次解析得到的作品数量（best-effort）',
    last_parsed_raw LONGTEXT NULL COMMENT '最后一次解析的原始数据（JSON字符串，可选）',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    updated_at DATETIME NOT NULL COMMENT '更新时间',
    INDEX idx_dfu_updated_at (updated_at DESC),
    INDEX idx_dfu_created_at (created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音用户收藏（全局）';

-- 抖音作品收藏（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_aweme (
    aweme_id VARCHAR(64) PRIMARY KEY COMMENT '作品ID（aweme_id）',
    sec_user_id VARCHAR(128) NULL COMMENT '作者 sec_user_id（可选）',
    type VARCHAR(16) NULL COMMENT '作品类型（video/image）',
    description TEXT NULL COMMENT '作品描述/标题（best-effort）',
    cover_url VARCHAR(500) NULL COMMENT '封面URL（best-effort）',
    raw_detail LONGTEXT NULL COMMENT '作品解析原始数据（JSON字符串，可选）',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    updated_at DATETIME NOT NULL COMMENT '更新时间',
    INDEX idx_dfa_sec_user_id (sec_user_id),
    INDEX idx_dfa_updated_at (updated_at DESC),
    INDEX idx_dfa_created_at (created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音作品收藏（全局）';

-- 抖音用户收藏作品（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_user_aweme (
    sec_user_id VARCHAR(128) NOT NULL COMMENT '抖音 sec_user_id（sec_uid）',
    aweme_id VARCHAR(64) NOT NULL COMMENT '作品ID（aweme_id）',
    type VARCHAR(16) NULL COMMENT '作品类型（video/image）',
    description TEXT NULL COMMENT '作品描述/标题（best-effort）',
    cover_url VARCHAR(500) NULL COMMENT '封面URL（best-effort）',
    downloads LONGTEXT NULL COMMENT '资源下载链接列表（JSON数组字符串，可选）',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    updated_at DATETIME NOT NULL COMMENT '更新时间',
    PRIMARY KEY (sec_user_id, aweme_id),
    INDEX idx_dfua_user_created_at (sec_user_id, created_at DESC),
    INDEX idx_dfua_user_updated_at (sec_user_id, updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音用户收藏作品（全局）';

-- 抖音用户收藏标签（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_user_tag (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL COMMENT '标签名称（全局唯一）',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '展示顺序（越小越靠前）',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    updated_at DATETIME NOT NULL COMMENT '更新时间',
    UNIQUE KEY uk_dfut_name (name),
    INDEX idx_dfut_updated_at (updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音用户收藏标签（全局）';

-- 抖音用户收藏标签映射（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_user_tag_map (
    sec_user_id VARCHAR(128) NOT NULL COMMENT '抖音 sec_user_id（sec_uid）',
    tag_id BIGINT NOT NULL COMMENT '标签ID',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    PRIMARY KEY (sec_user_id, tag_id),
    INDEX idx_dfutm_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音用户收藏标签映射（全局）';

-- 抖音作品收藏标签（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_aweme_tag (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL COMMENT '标签名称（全局唯一）',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '展示顺序（越小越靠前）',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    updated_at DATETIME NOT NULL COMMENT '更新时间',
    UNIQUE KEY uk_dfat_name (name),
    INDEX idx_dfat_updated_at (updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音作品收藏标签（全局）';

-- 抖音作品收藏标签映射（全局）
CREATE TABLE IF NOT EXISTS douyin_favorite_aweme_tag_map (
    aweme_id VARCHAR(64) NOT NULL COMMENT '作品ID（aweme_id）',
    tag_id BIGINT NOT NULL COMMENT '标签ID',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    PRIMARY KEY (aweme_id, tag_id),
    INDEX idx_dfatm_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音作品收藏标签映射（全局）';
