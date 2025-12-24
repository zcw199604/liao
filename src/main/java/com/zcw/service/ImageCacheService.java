package com.zcw.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 图片缓存服务
 * 管理用户上传图片的内存缓存
 */
@Service
public class ImageCacheService {

    private static final Logger log = LoggerFactory.getLogger(ImageCacheService.class);

    // 图片缓存：用户ID -> 图片URL列表及过期时间
    private final Map<String, CachedImages> imageCache = new ConcurrentHashMap<>();

    // 缓存过期时间：3小时（毫秒）
    private static final long CACHE_EXPIRE_TIME = 3 * 60 * 60 * 1000;

    /**
     * 缓存图片数据结构
     */
    public static class CachedImages {
        private final List<String> imageUrls;
        private long expireTime;

        public CachedImages(List<String> imageUrls, long expireTime) {
            this.imageUrls = imageUrls;
            this.expireTime = expireTime;
        }

        public List<String> getImageUrls() {
            return imageUrls;
        }

        public long getExpireTime() {
            return expireTime;
        }

        public void setExpireTime(long expireTime) {
            this.expireTime = expireTime;
        }

        public boolean isExpired() {
            return System.currentTimeMillis() > expireTime;
        }
    }

    /**
     * 添加图片到缓存
     *
     * @param userid   用户ID
     * @param imageUrl 图片本地路径
     */
    public void addImageToCache(String userid, String imageUrl) {
        long expireTime = System.currentTimeMillis() + CACHE_EXPIRE_TIME;

        CachedImages cached = imageCache.get(userid);
        if (cached == null || cached.isExpired()) {
            // 创建新缓存
            List<String> urls = new ArrayList<>();
            urls.add(imageUrl);
            imageCache.put(userid, new CachedImages(urls, expireTime));
            log.info("创建新缓存: userid={}, imageUrl={}", userid, imageUrl);
        } else {
            // 添加到现有缓存
            cached.getImageUrls().add(imageUrl);
            // 更新过期时间
            cached.setExpireTime(expireTime);
            log.info("添加到现有缓存: userid={}, imageUrl={}, 缓存大小={}", userid, imageUrl, cached.getImageUrls().size());
        }
    }

    /**
     * 获取用户的缓存图片
     *
     * @param userid 用户ID
     * @return 缓存对象，如果不存在或过期返回null
     */
    public CachedImages getCachedImages(String userid) {
        CachedImages cached = imageCache.get(userid);

        if (cached == null) {
            log.debug("用户 {} 没有缓存图片", userid);
            return null;
        }

        if (cached.isExpired()) {
            log.info("用户 {} 的缓存已过期，清除缓存", userid);
            imageCache.remove(userid);
            return null;
        }

        return cached;
    }

    /**
     * 重建用户的缓存
     *
     * @param userid    用户ID
     * @param imagePaths 图片路径列表
     */
    public void rebuildCache(String userid, List<String> imagePaths) {
        if (imagePaths == null || imagePaths.isEmpty()) {
            return;
        }

        long expireTime = System.currentTimeMillis() + CACHE_EXPIRE_TIME;
        imageCache.put(userid, new CachedImages(new ArrayList<>(imagePaths), expireTime));
        log.info("重建缓存: userid={}, 图片数量={}", userid, imagePaths.size());
    }

    /**
     * 清除用户的缓存
     *
     * @param userid 用户ID
     */
    public void clearCache(String userid) {
        imageCache.remove(userid);
        log.info("清除缓存: userid={}", userid);
    }
}
