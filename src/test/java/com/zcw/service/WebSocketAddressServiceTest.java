package com.zcw.service;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import static org.junit.jupiter.api.Assertions.assertEquals;

@DisplayName("WebSocket地址服务测试")
class WebSocketAddressServiceTest {

    @Test
    @DisplayName("解析WebSocket地址 - 成功")
    void parseWebSocketUrl_ShouldReturnUrl() {
        // Arrange
        WebSocketAddressService service = new WebSocketAddressService(null, new com.fasterxml.jackson.databind.ObjectMapper());
        String json = "{\"state\":\"OK\",\"msg\":{\"server\":\"ws://1.2.3.4:8080\"},\"code\":0}";

        // Act
        // 使用反射调用私有方法
        String url = ReflectionTestUtils.invokeMethod(service, "parseWebSocketUrl", json);

        // Assert
        assertEquals("ws://1.2.3.4:8080", url);
    }

    @Test
    @DisplayName("解析WebSocket地址 - 失败返回Null")
    void parseWebSocketUrl_ShouldReturnNull_WhenError() {
        // Arrange
        WebSocketAddressService service = new WebSocketAddressService(null, new com.fasterxml.jackson.databind.ObjectMapper());
        String json = "{\"state\":\"ERROR\"}";

        // Act
        String url = ReflectionTestUtils.invokeMethod(service, "parseWebSocketUrl", json);

        // Assert
        assertEquals(null, url);
    }
}
