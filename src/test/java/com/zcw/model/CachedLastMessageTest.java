package com.zcw.model;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("最后消息缓存模型测试")
class CachedLastMessageTest {

    @Test
    @DisplayName("生成会话Key - 按字典序排序")
    void generateConversationKey_ShouldSortByDictionary() {
        // Arrange & Act
        String key1 = CachedLastMessage.generateConversationKey("user123", "user456");
        String key2 = CachedLastMessage.generateConversationKey("user456", "user123");

        // Assert
        assertEquals("user123_user456", key1);
        assertEquals("user123_user456", key2);
        assertEquals(key1, key2, "双向会话应该生成相同的key");
    }

    @Test
    @DisplayName("生成会话Key - 处理null值")
    void generateConversationKey_ShouldHandleNull() {
        // Act & Assert
        assertEquals("", CachedLastMessage.generateConversationKey(null, "user123"));
        assertEquals("", CachedLastMessage.generateConversationKey("user123", null));
        assertEquals("", CachedLastMessage.generateConversationKey(null, null));
    }

    @Test
    @DisplayName("构造函数 - 自动生成conversationKey和updateTime")
    void constructor_ShouldGenerateKeyAndTime() {
        // Arrange
        long beforeTime = System.currentTimeMillis();

        // Act
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        );

        long afterTime = System.currentTimeMillis();

        // Assert
        assertNotNull(message.getConversationKey());
        assertEquals("user1_user2", message.getConversationKey());
        assertNotNull(message.getUpdateTime());
        assertTrue(message.getUpdateTime() >= beforeTime && message.getUpdateTime() <= afterTime);
        assertEquals("user1", message.getFromUserId());
        assertEquals("user2", message.getToUserId());
        assertEquals("Hello", message.getContent());
        assertEquals("text", message.getType());
        assertEquals("2026-01-04 10:00:00", message.getTime());
    }

    @Test
    @DisplayName("会话Key生成 - 相同用户ID")
    void generateConversationKey_SameUserId() {
        // Act
        String key = CachedLastMessage.generateConversationKey("user123", "user123");

        // Assert
        assertEquals("user123_user123", key);
    }
}
