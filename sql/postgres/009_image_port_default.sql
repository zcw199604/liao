-- PostgreSQL data migration: 009_image_port_default
-- 上游新版页面把媒体端口改为 Imgarr=['7001','7004']，图片/视频共用 7001/7004；旧端口 9006/8006 已不再监听。
-- 这里只迁移“仍是旧默认值 9006”的配置，避免覆盖用户手动设置过的端口。

UPDATE system_config
SET config_value = '7001', updated_at = NOW()
WHERE config_key = 'image_port_fixed' AND config_value = '9006';
