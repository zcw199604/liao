-- PostgreSQL schema migration: 010_identity_provider_binding
-- 记录「我方身份 ↔ 上游线路账号」的绑定（当前用于 wanfeng 线路）。

CREATE TABLE IF NOT EXISTS identity_provider_binding (
	identity_id VARCHAR(64) NOT NULL,
	provider VARCHAR(32) NOT NULL,
	upstream_uid VARCHAR(128) NOT NULL,
	identity_token VARCHAR(1024) NULL,
	device_id VARCHAR(64) NULL,
	nickname VARCHAR(64) NULL,
	gender VARCHAR(16) NULL,
	age INT NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY (identity_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_ipb_provider_uid
	ON identity_provider_binding (provider, upstream_uid);
