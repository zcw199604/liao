ALTER TABLE media_file ADD COLUMN IF NOT EXISTS media_width int NULL;
ALTER TABLE media_file ADD COLUMN IF NOT EXISTS media_height int NULL;

ALTER TABLE douyin_media_file ADD COLUMN IF NOT EXISTS media_width int NULL;
ALTER TABLE douyin_media_file ADD COLUMN IF NOT EXISTS media_height int NULL;
