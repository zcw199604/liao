package com.zcw.service.impl;

import com.zcw.model.CachedLastMessage;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("内存缓存服务 - 最后消息功能测试")
class MemoryUserInfoCacheServiceLastMessageTest {

    private MemoryUserInfoCacheService cacheService;

    @BeforeEach
    void setUp() {
        cacheService = new MemoryUserInfoCacheService();
    }

    @Test
    @DisplayName("保存和获取最后消息 - 成功")
    void saveAndGetLastMessage_Success() {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", "Hello World", "text", "2026-01-04 10:00:00"
        );

        // Act
        cacheService.saveLastMessage(message);
        CachedLastMessage retrieved = cacheService.getLastMessage("user1", "user2");

        // Assert
        assertNotNull(retrieved);
        assertEquals("Hello World", retrieved.getContent());
        assertEquals("user1", retrieved.getFromUserId());
        assertEquals("user2", retrieved.getToUserId());
    }

    @Test
    @DisplayName("获取最后消息 - 双向会话共享")
    void getLastMessage_BidirectionalSharing() {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "userA", "userB", "Test Message", "text", "2026-01-04 10:00:00"
        );
        cacheService.saveLastMessage(message);

        // Act
        CachedLastMessage fromA = cacheService.getLastMessage("userA", "userB");
        CachedLastMessage fromB = cacheService.getLastMessage("userB", "userA");

        // Assert
        assertNotNull(fromA);
        assertNotNull(fromB);
        assertEquals(fromA.getConversationKey(), fromB.getConversationKey());
        assertEquals("Test Message", fromA.getContent());
        assertEquals("Test Message", fromB.getContent());
    }

    @Test
    @DisplayName("获取最后消息 - 不存在返回null")
    void getLastMessage_NotFound_ReturnsNull() {
        // Act
        CachedLastMessage result = cacheService.getLastMessage("user1", "user2");

        // Assert
        assertNull(result);
    }

    @Test
    @DisplayName("批量增强用户列表 - 补充lastMsg字段")
    void batchEnrichWithLastMessage_Success() {
        // Arrange
        cacheService.saveLastMessage(new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        ));
        cacheService.saveLastMessage(new CachedLastMessage(
            "user3", "user1", "Hi there", "text", "2026-01-04 10:05:00"
        ));

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user1 = new HashMap<>();
        user1.put("id", "user2");
        Map<String, Object> user2 = new HashMap<>();
        user2.put("id", "user3");
        userList.add(user1);
        userList.add(user2);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        assertEquals(2, result.size());
        assertEquals("我: Hello", result.get(0).get("lastMsg"));
        assertEquals("Hi there", result.get(1).get("lastMsg"));
    }

    @Test
    @DisplayName("批量增强用户列表 - 支持UserID字段")
    void batchEnrichWithLastMessage_UserIDKey_Success() {
        // Arrange
        cacheService.saveLastMessage(new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        ));

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user = new HashMap<>();
        user.put("UserID", "user2");
        userList.add(user);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        assertEquals(1, result.size());
        assertEquals("我: Hello", result.get(0).get("lastMsg"));
        assertEquals("2026-01-04 10:00:00", result.get(0).get("lastTime"));
    }

    @Test
    @DisplayName("格式化消息 - 图片消息显示标签")
    void formatLastMessage_ImageMessage() {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", "[20260104/image.jpg]", "image", "2026-01-04 10:00:00"
        );
        cacheService.saveLastMessage(message);

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user = new HashMap<>();
        user.put("id", "user2");
        userList.add(user);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        assertEquals("我: [图片]", result.get(0).get("lastMsg"));
    }

    @Test
    @DisplayName("格式化消息 - 视频消息显示标签")
    void formatLastMessage_VideoMessage() {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "user2", "user1", "[20260104/video.mp4]", "video", "2026-01-04 10:00:00"
        );
        cacheService.saveLastMessage(message);

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user = new HashMap<>();
        user.put("id", "user2");
        userList.add(user);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        assertEquals("[视频]", result.get(0).get("lastMsg"));
    }

    @Test
    @DisplayName("格式化消息 - 长文本截断")
    void formatLastMessage_LongTextTruncated() {
        // Arrange
        String longText = "这是一段很长的消息内容，超过三十个字符的限制，应该被截断显示省略号，还有更多内容在后面";
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", longText, "text", "2026-01-04 10:00:00"
        );
        cacheService.saveLastMessage(message);

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user = new HashMap<>();
        user.put("id", "user2");
        userList.add(user);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        String lastMsg = (String) result.get(0).get("lastMsg");
        assertTrue(lastMsg.startsWith("我: "));
        assertTrue(lastMsg.endsWith("..."));
        // 验证消息被截断了（不包含原始消息的后半部分）
        assertFalse(lastMsg.contains("还有更多内容在后面"));
    }

    @Test
    @DisplayName("批量增强 - 不覆盖已有lastMsg")
    void batchEnrichWithLastMessage_DoesNotOverrideExisting() {
        // Arrange
        cacheService.saveLastMessage(new CachedLastMessage(
            "user1", "user2", "Cached Message", "text", "2026-01-04 10:00:00"
        ));

        List<Map<String, Object>> userList = new ArrayList<>();
        Map<String, Object> user = new HashMap<>();
        user.put("id", "user2");
        user.put("lastMsg", "Existing Message");
        userList.add(user);

        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(userList, "user1");

        // Assert
        assertEquals("Existing Message", result.get(0).get("lastMsg"));
    }

    @Test
    @DisplayName("保存null消息 - 不抛出异常")
    void saveLastMessage_NullMessage_NoException() {
        // Act & Assert
        assertDoesNotThrow(() -> cacheService.saveLastMessage(null));
    }

    @Test
    @DisplayName("批量增强null列表 - 返回空列表")
    void batchEnrichWithLastMessage_NullList_ReturnsEmpty() {
        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(null, "user1");

        // Assert
        assertNotNull(result);
        assertTrue(result.isEmpty());
    }
}
