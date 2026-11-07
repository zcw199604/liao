-- PostgreSQL schema migration: 011_identity_match_preference
-- 记录每个身份的「匹配偏好」（期望对方性别/年龄区间等）。

CREATE TABLE IF NOT EXISTS identity_match_preference (
	identity_id VARCHAR(64) NOT NULL,
	prefer_gender VARCHAR(16) NOT NULL DEFAULT 'any',
	prefer_age_min INT NOT NULL DEFAULT 0,
	prefer_age_max INT NOT NULL DEFAULT 0,
	prefer_same_province SMALLINT NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY (identity_id)
);
