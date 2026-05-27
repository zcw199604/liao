ALTER TABLE media_file
  ADD COLUMN media_width INT NULL COMMENT '媒体宽度（用于前端瀑布流布局）';

ALTER TABLE media_file
  ADD COLUMN media_height INT NULL COMMENT '媒体高度（用于前端瀑布流布局）';

ALTER TABLE douyin_media_file
  ADD COLUMN media_width INT NULL COMMENT '媒体宽度（用于前端瀑布流布局）';

ALTER TABLE douyin_media_file
  ADD COLUMN media_height INT NULL COMMENT '媒体高度（用于前端瀑布流布局）';
