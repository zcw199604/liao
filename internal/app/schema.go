package app

import (
	"database/sql"
	"fmt"
)

func ensureSchema(db *sql.DB) error {
	statements := []string{
		// identity（与现有 init.sql/IdentityService 兼容）
		`CREATE TABLE IF NOT EXISTS identity (
			id VARCHAR(32) PRIMARY KEY COMMENT '用户ID（32位随机字符串）',
			name VARCHAR(50) NOT NULL COMMENT '名字',
			sex VARCHAR(10) NOT NULL COMMENT '性别（男/女）',
			created_at DATETIME COMMENT '创建时间',
			last_used_at DATETIME COMMENT '最后使用时间',
			INDEX idx_last_used_at (last_used_at DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户身份表'`,

		// chat_favorites（JPA ddl-auto=update 的等价兜底）
		`CREATE TABLE IF NOT EXISTS chat_favorites (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			identity_id VARCHAR(32) NOT NULL,
			target_user_id VARCHAR(64) NOT NULL,
			target_user_name VARCHAR(64) NULL,
			create_time DATETIME NOT NULL,
			INDEX idx_identity_id (identity_id),
			INDEX idx_target_user_id (target_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='本地聊天收藏'`,

		// media_file（媒体库）
		`CREATE TABLE IF NOT EXISTS media_file (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(32) NOT NULL,
			original_filename TEXT NOT NULL,
			local_filename TEXT NOT NULL,
			remote_filename TEXT NOT NULL,
			remote_url VARCHAR(500) NOT NULL,
			local_path VARCHAR(500) NOT NULL,
			file_size BIGINT NOT NULL,
			file_type VARCHAR(50) NOT NULL,
			file_extension VARCHAR(10) NOT NULL,
			file_md5 VARCHAR(32) NULL,
			upload_time DATETIME NOT NULL,
			update_time DATETIME NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_mf_user_id (user_id),
			INDEX idx_mf_file_md5 (file_md5),
			INDEX idx_mf_update_time (update_time DESC),
			INDEX idx_mf_local_path (local_path)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体文件库'`,

		// media_send_log（发送日志）
		`CREATE TABLE IF NOT EXISTS media_send_log (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(32) NOT NULL,
			to_user_id VARCHAR(32) NOT NULL,
			local_path VARCHAR(500) NOT NULL,
			remote_url VARCHAR(500) NOT NULL,
			send_time DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_msl_user_id (user_id),
			INDEX idx_msl_to_user_id (to_user_id),
			INDEX idx_msl_send_time (send_time DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体发送日志'`,

		// media_upload_history（历史遗留表：仅用于 MD5 -> local_path 复用查询）
		`CREATE TABLE IF NOT EXISTS media_upload_history (
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
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体上传历史记录表'`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("初始化数据表失败: %w", err)
		}
	}
	return nil
}

