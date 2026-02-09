-- PostgreSQL schema migration: 005_douyin_media_author_snapshot
-- Safe to re-run: statements use IF NOT EXISTS where possible.

ALTER TABLE douyin_media_file ADD COLUMN IF NOT EXISTS author_unique_id varchar(64) NULL;
ALTER TABLE douyin_media_file ADD COLUMN IF NOT EXISTS author_name varchar(128) NULL;
