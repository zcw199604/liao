-- MySQL schema migration: 007_chat_user_archive
-- Persist remote chat users locally so deleted upstream users can still be recovered.

CREATE TABLE IF NOT EXISTS chat_user_archive (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	owner_user_id VARCHAR(64) NOT NULL COMMENT '当前登录用户ID（myUserID）',
	target_user_id VARCHAR(64) NOT NULL COMMENT '对方用户ID',
	snapshot_json LONGTEXT NULL COMMENT '最近一次用户快照(JSON)',
	last_msg TEXT NULL COMMENT '最近消息摘要',
	last_time VARCHAR(64) NULL COMMENT '最近消息时间(原始字符串)',
	seen_in_history TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否出现在历史列表',
	seen_in_favorite TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否出现在收藏列表',
	first_seen_at DATETIME NOT NULL COMMENT '首次见到时间',
	last_seen_at DATETIME NOT NULL COMMENT '最近见到时间',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	UNIQUE KEY uk_chat_user_archive_owner_target (owner_user_id, target_user_id),
	INDEX idx_chat_user_archive_owner_history (owner_user_id, seen_in_history, updated_at DESC),
	INDEX idx_chat_user_archive_owner_favorite (owner_user_id, seen_in_favorite, updated_at DESC),
	INDEX idx_chat_user_archive_owner_seen (owner_user_id, last_seen_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天用户本地归档';
