-- MySQL schema migration: 013_identity_location_setting
-- 记录每个身份的「位置（城市）是否公开」。
-- 说明：晚风站点按访客 IP 归属地给身份打城市，匹配成功时以 match_result.partner.city 下发给对面；
-- 对方设置页的「城市：公开 / 不公开」= 用户资料字段 showCity（对方默认公开）。
-- 我方默认「不公开」（DEFAULT 0），用户可在身份设置里改成公开。

CREATE TABLE IF NOT EXISTS identity_location_setting (
	identity_id VARCHAR(64) NOT NULL COMMENT '我方身份ID',
	show_city TINYINT NOT NULL DEFAULT 0 COMMENT '是否公开位置/城市（0 不公开，1 公开）',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	PRIMARY KEY (identity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='身份位置公开设置';
