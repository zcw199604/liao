-- MySQL schema migration: 006_mtphoto_folder_favorite
-- Introduce local favorites for mtPhoto folders.

CREATE TABLE IF NOT EXISTS mtphoto_folder_favorite (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	folder_id BIGINT NOT NULL COMMENT 'mtPhoto 文件夹 ID（全局唯一）',
	folder_name VARCHAR(255) NOT NULL COMMENT '文件夹名称',
	folder_path VARCHAR(1024) NOT NULL COMMENT '文件夹路径',
	cover_md5 VARCHAR(32) NULL COMMENT '收藏卡片封面 md5（可空）',
	tags_json LONGTEXT NOT NULL COMMENT '标签 JSON 数组字符串',
	note TEXT NULL COMMENT '备注',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	UNIQUE KEY uk_mtphoto_folder_favorite_folder_id (folder_id),
	INDEX idx_mtphoto_folder_favorite_updated_at (updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='mtPhoto 文件夹收藏（全局）';
