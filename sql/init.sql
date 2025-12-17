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
