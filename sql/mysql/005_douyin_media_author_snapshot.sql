-- MySQL schema migration: 005_douyin_media_author_snapshot
-- Safe to re-run: duplicate column errors are ignored by the migrator.

ALTER TABLE douyin_media_file ADD COLUMN author_unique_id VARCHAR(64) NULL COMMENT '作者抖音号（unique_id）';
ALTER TABLE douyin_media_file ADD COLUMN author_name VARCHAR(128) NULL COMMENT '作者名称/昵称（快照）';
