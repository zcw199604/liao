-- PostgreSQL schema migration: 002_legacy_sort_order
-- Safe to re-run: statements use IF NOT EXISTS where possible.

ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS sort_order int NOT NULL DEFAULT 2147483647;
CREATE INDEX IF NOT EXISTS idx_dfua_user_sort_order ON douyin_favorite_user_aweme (sec_user_id, sort_order);
ALTER TABLE douyin_favorite_user_tag ADD COLUMN IF NOT EXISTS sort_order int NOT NULL DEFAULT 0;
ALTER TABLE douyin_favorite_aweme_tag ADD COLUMN IF NOT EXISTS sort_order int NOT NULL DEFAULT 0;
