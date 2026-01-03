package com.zcw.service;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("图片缓存服务测试")
class ImageCacheServiceTest {

    @Test
    @DisplayName("添加缓存 - 创建新缓存")
    void addImageToCache_ShouldCreateNew() {
        // Arrange
        ImageCacheService service = new ImageCacheService();
        String userId = "u1";
        String path = "/img/1.jpg";

        // Act
        service.addImageToCache(userId, path);

        // Assert
        ImageCacheService.CachedImages cached = service.getCachedImages(userId);
        assertNotNull(cached);
        assertEquals(1, cached.getImageUrls().size());
        assertEquals(path, cached.getImageUrls().get(0));
    }

    @Test
    @DisplayName("添加缓存 - 追加到现有")
    void addImageToCache_ShouldAppend() {
        // Arrange
        ImageCacheService service = new ImageCacheService();
        String userId = "u1";
        service.addImageToCache(userId, "/img/1.jpg");

        // Act
        service.addImageToCache(userId, "/img/2.jpg");

        // Assert
        ImageCacheService.CachedImages cached = service.getCachedImages(userId);
        assertEquals(2, cached.getImageUrls().size());
        assertTrue(cached.getImageUrls().contains("/img/2.jpg"));
    }

    @Test
    @DisplayName("清理缓存")
    void clearCache_ShouldRemove() {
        // Arrange
        ImageCacheService service = new ImageCacheService();
        service.addImageToCache("u1", "path");

        // Act
        service.clearCache("u1");

        // Assert
        assertNull(service.getCachedImages("u1"));
    }
}
