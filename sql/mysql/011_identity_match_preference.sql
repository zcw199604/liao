-- MySQL schema migration: 011_identity_match_preference
-- 记录每个身份的「匹配偏好」（期望对方性别/年龄区间等）。
-- 说明：晚风线路把匹配偏好挂在用户资料上，通过 update_profile 下发生效；
-- 我方需要在本地保留一份，才能在重连/登录帧里带上。

CREATE TABLE IF NOT EXISTS identity_match_preference (
	identity_id VARCHAR(64) NOT NULL COMMENT '我方身份ID',
	prefer_gender VARCHAR(16) NOT NULL DEFAULT 'any' COMMENT '期望对方性别：male/female/any',
	prefer_age_min INT NOT NULL DEFAULT 0 COMMENT '期望对方最小年龄（0 表示不限）',
	prefer_age_max INT NOT NULL DEFAULT 0 COMMENT '期望对方最大年龄（0 表示不限）',
	prefer_same_province TINYINT NOT NULL DEFAULT 0 COMMENT '是否限制同省（1 是）',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	PRIMARY KEY (identity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='身份匹配偏好';
