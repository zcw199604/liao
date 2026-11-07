-- PostgreSQL schema migration: 013_identity_location_setting
-- 记录每个身份的「位置（城市）是否公开」，默认不公开。

CREATE TABLE IF NOT EXISTS identity_location_setting (
	identity_id VARCHAR(64) NOT NULL,
	show_city SMALLINT NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY (identity_id)
);
