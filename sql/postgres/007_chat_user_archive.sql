-- PostgreSQL schema migration: 007_chat_user_archive
-- Persist remote chat users locally so deleted upstream users can still be recovered.

CREATE TABLE IF NOT EXISTS chat_user_archive (
	id BIGSERIAL PRIMARY KEY,
	owner_user_id VARCHAR(64) NOT NULL,
	target_user_id VARCHAR(64) NOT NULL,
	snapshot_json TEXT NULL,
	last_msg TEXT NULL,
	last_time VARCHAR(64) NULL,
	seen_in_history SMALLINT NOT NULL DEFAULT 0,
	seen_in_favorite SMALLINT NOT NULL DEFAULT 0,
	first_seen_at TIMESTAMP NOT NULL,
	last_seen_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	UNIQUE (owner_user_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_user_archive_owner_history
	ON chat_user_archive (owner_user_id, seen_in_history, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_chat_user_archive_owner_favorite
	ON chat_user_archive (owner_user_id, seen_in_favorite, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_chat_user_archive_owner_seen
	ON chat_user_archive (owner_user_id, last_seen_at DESC);
