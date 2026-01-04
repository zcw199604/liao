package com.zcw.service.impl;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.zcw.model.CachedLastMessage;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.*;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("Redis缓存服务 - 最后消息功能测试")
class RedisUserInfoCacheServiceLastMessageTest {

    @Mock
    private StringRedisTemplate redisTemplate;

    @Mock
    private ValueOperations<String, String> valueOperations;

    @InjectMocks
    private RedisUserInfoCacheService cacheService;

    private ObjectMapper objectMapper;

    @BeforeEach
    void setUp() {
        objectMapper = new ObjectMapper();
        lenient().when(redisTemplate.opsForValue()).thenReturn(valueOperations);

        // 设置配置值
        ReflectionTestUtils.setField(cacheService, "lastMessagePrefix", "user:lastmsg:");
        ReflectionTestUtils.setField(cacheService, "expireDays", 7L);

        // 初始化本地缓存
        cacheService.init();
    }

    @Test
    @DisplayName("保存最后消息 - 成功写入Redis")
    void saveLastMessage_Success() throws Exception {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        );

        // Act
        cacheService.saveLastMessage(message);

        // Assert
        verify(valueOperations, times(1)).set(
            eq("user:lastmsg:user1_user2"),
            anyString(),
            eq(7L),
            eq(TimeUnit.DAYS)
        );
    }

    @Test
    @DisplayName("获取最后消息 - 从Redis读取")
    void getLastMessage_FromRedis() throws Exception {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "user1", "user2", "Test", "text", "2026-01-04 10:00:00"
        );
        String json = objectMapper.writeValueAsString(message);
        when(valueOperations.get("user:lastmsg:user1_user2")).thenReturn(json);

        // Act
        CachedLastMessage result = cacheService.getLastMessage("user1", "user2");

        // Assert
        assertNotNull(result);
        assertEquals("Test", result.getContent());
        assertEquals("user1", result.getFromUserId());
        verify(valueOperations, times(1)).get("user:lastmsg:user1_user2");
    }

    @Test
    @DisplayName("获取最后消息 - 双向会话使用相同key")
    void getLastMessage_BidirectionalSameKey() throws Exception {
        // Arrange
        CachedLastMessage message = new CachedLastMessage(
            "userA", "userB", "Message", "text", "2026-01-04 10:00:00"
        );
        String json = objectMapper.writeValueAsString(message);
        when(valueOperations.get("user:lastmsg:userA_userB")).thenReturn(json);

        // Act
        CachedLastMessage result1 = cacheService.getLastMessage("userA", "userB");
        // 第二次查询会从L1本地缓存读取，不会再调用Redis
        CachedLastMessage result2 = cacheService.getLastMessage("userB", "userA");

        // Assert
        assertNotNull(result1);
        assertNotNull(result2);
        assertEquals(result1.getConversationKey(), result2.getConversationKey());
        // 第一次查询调用Redis，第二次从L1缓存读取
        verify(valueOperations, times(1)).get("user:lastmsg:userA_userB");
    }

    @Test
    @DisplayName("获取最后消息 - 不存在返回null")
    void getLastMessage_NotFound_ReturnsNull() {
        // Arrange
        when(valueOperations.get(anyString())).thenReturn(null);

        // Act
        CachedLastMessage result = cacheService.getLastMessage("user1", "user2");

        // Assert
        assertNull(result);
    }

    @Test
    @DisplayName("批量增强用户列表 - 使用MultiGet优化")
    void batchEnrichWithLastMessage_UsesMultiGet() throws Exception {
        // Arrange
        CachedLastMessage msg1 = new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        );
        CachedLastMessage msg2 = new CachedLastMessage(
            "user1", "user3", "Hi", "text", "2026-01-04 10:05:00"
        );

        List<String> jsonList = Arrays.asList(
            objectMapper.writeValueAsString(msg1),
            objectMapper.writeValueAsString(msg2)
        );

        when(valueOperations.multiGet(anyList())).thenReturn(jsonList);

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
        assertEquals("我: Hi", result.get(1).get("lastMsg"));

        // 验证使用了multiGet（批量查询）
        verify(valueOperations, times(1)).multiGet(anyList());
    }

    @Test
    @DisplayName("批量增强 - 处理部分消息不存在")
    void batchEnrichWithLastMessage_PartialResults() throws Exception {
        // Arrange
        CachedLastMessage msg1 = new CachedLastMessage(
            "user1", "user2", "Hello", "text", "2026-01-04 10:00:00"
        );

        List<String> jsonList = Arrays.asList(
            objectMapper.writeValueAsString(msg1),
            null  // user3的消息不存在
        );

        when(valueOperations.multiGet(anyList())).thenReturn(jsonList);

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
        assertNull(result.get(1).get("lastMsg")); // user3没有lastMsg
    }

    @Test
    @DisplayName("保存null消息 - 不抛出异常")
    void saveLastMessage_NullMessage_NoException() {
        // Act & Assert
        assertDoesNotThrow(() -> cacheService.saveLastMessage(null));
        verify(valueOperations, never()).set(anyString(), anyString(), anyLong(), any(TimeUnit.class));
    }

    @Test
    @DisplayName("批量增强空列表 - 返回原列表")
    void batchEnrichWithLastMessage_EmptyList_ReturnsEmpty() {
        // Act
        List<Map<String, Object>> result = cacheService.batchEnrichWithLastMessage(
            new ArrayList<>(), "user1"
        );

        // Assert
        assertNotNull(result);
        assertTrue(result.isEmpty());
        verify(valueOperations, never()).multiGet(anyList());
    }

    @Test
    @DisplayName("Redis异常 - 优雅降级返回null")
    void getLastMessage_RedisException_ReturnsNull() {
        // Arrange
        when(valueOperations.get(anyString())).thenThrow(new RuntimeException("Redis连接失败"));

        // Act
        CachedLastMessage result = cacheService.getLastMessage("user1", "user2");

        // Assert
        assertNull(result);
    }
}
