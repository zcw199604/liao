-- MySQL schema migration: 010_identity_provider_binding
-- 记录「我方身份 ↔ 上游线路账号」的绑定（当前用于 wanfeng 线路）。
-- 说明：wanfeng 会下发 uid 与 identityToken，必须持久化，否则重连会变成新用户。

CREATE TABLE IF NOT EXISTS identity_provider_binding (
	identity_id VARCHAR(64) NOT NULL COMMENT '我方身份ID（前端使用的 id）',
	provider VARCHAR(32) NOT NULL COMMENT '上游线路标识（chat2019/wanfeng）',
	upstream_uid VARCHAR(128) NOT NULL COMMENT '上游分配的用户ID',
	identity_token VARCHAR(1024) NULL COMMENT '上游身份令牌（重连必须携带）',
	device_id VARCHAR(64) NULL COMMENT '设备标识（每身份独立，避免触发同设备限制）',
	nickname VARCHAR(64) NULL COMMENT '上送昵称',
	gender VARCHAR(16) NULL COMMENT '上送性别（male/female/secret）',
	age INT NOT NULL DEFAULT 0 COMMENT '上送年龄',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	PRIMARY KEY (identity_id, provider),
	INDEX idx_ipb_provider_uid (provider, upstream_uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='身份与上游线路账号绑定';
