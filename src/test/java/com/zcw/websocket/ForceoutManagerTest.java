package com.zcw.websocket;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Forceout管理器测试")
class ForceoutManagerTest {

    @Test
    @DisplayName("禁止用户 - 添加并检查状态")
    void addForceoutUser_ShouldForbidUser() {
        // Arrange
        ForceoutManager manager = new ForceoutManager();
        String userId = "user1";

        // Act
        manager.addForceoutUser(userId);

        // Assert
        assertTrue(manager.isForbidden(userId));
        assertTrue(manager.getRemainingSeconds(userId) > 0);
        assertEquals(1, manager.getForbiddenUserCount());
    }

    @Test
    @DisplayName("过期检查 - 自动移除过期用户")
    void isForbidden_ShouldRemove_WhenExpired() {
        // Arrange
        ForceoutManager manager = new ForceoutManager();
        String userId = "user1";
        
        // 手动注入过期时间 (过去的时间)
        Map<String, Long> forceoutUsers = (Map<String, Long>) ReflectionTestUtils.getField(manager, "forceoutUsers");
        forceoutUsers.put(userId, System.currentTimeMillis() - 1000);

        // Act & Assert
        assertFalse(manager.isForbidden(userId)); // Should return false and remove
        assertEquals(0, manager.getForbiddenUserCount());
    }

    @Test
    @DisplayName("清除所有")
    void clearAllForceout_ShouldEmptyMap() {
        // Arrange
        ForceoutManager manager = new ForceoutManager();
        manager.addForceoutUser("u1");
        manager.addForceoutUser("u2");

        // Act
        int count = manager.clearAllForceout();

        // Assert
        assertEquals(2, count);
        assertEquals(0, manager.getForbiddenUserCount());
    }
}
