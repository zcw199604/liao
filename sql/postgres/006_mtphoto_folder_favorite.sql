-- PostgreSQL schema migration: 006_mtphoto_folder_favorite
-- Introduce local favorites for mtPhoto folders.

CREATE TABLE IF NOT EXISTS mtphoto_folder_favorite (
	id BIGSERIAL PRIMARY KEY,
	folder_id BIGINT NOT NULL,
	folder_name VARCHAR(255) NOT NULL,
	folder_path VARCHAR(1024) NOT NULL,
	cover_md5 VARCHAR(32) NULL,
	tags_json TEXT NOT NULL DEFAULT '[]',
	note TEXT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	UNIQUE (folder_id)
);

CREATE INDEX IF NOT EXISTS idx_mtphoto_folder_favorite_updated_at
	ON mtphoto_folder_favorite (updated_at DESC);
