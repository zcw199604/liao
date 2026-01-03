package com.zcw.websocket;

import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.server.ServletServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.web.socket.WebSocketHandler;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
@DisplayName("WebSocket JWT拦截器测试")
class JwtWebSocketInterceptorTest {

    @Mock
    private JwtUtil jwtUtil;

    @InjectMocks
    private JwtWebSocketInterceptor interceptor;

    @Test
    @DisplayName("握手前 - Token有效时通过")
    void beforeHandshake_ShouldPass_WhenTokenValid() throws Exception {
        // Arrange
        MockHttpServletRequest servletRequest = new MockHttpServletRequest();
        servletRequest.setParameter("token", "valid_token");
        ServletServerHttpRequest request = new ServletServerHttpRequest(servletRequest);
        
        ServerHttpResponse response = mock(ServerHttpResponse.class);
        WebSocketHandler handler = mock(WebSocketHandler.class);
        Map<String, Object> attributes = new HashMap<>();

        when(jwtUtil.validateToken("valid_token")).thenReturn(true);

        // Act
        boolean result = interceptor.beforeHandshake(request, response, handler, attributes);

        // Assert
        assertTrue(result);
    }

    @Test
    @DisplayName("握手前 - Token无效时拒绝")
    void beforeHandshake_ShouldFail_WhenTokenInvalid() throws Exception {
        // Arrange
        MockHttpServletRequest servletRequest = new MockHttpServletRequest();
        servletRequest.setParameter("token", "invalid_token");
        ServletServerHttpRequest request = new ServletServerHttpRequest(servletRequest);
        
        ServerHttpResponse response = mock(ServerHttpResponse.class);
        WebSocketHandler handler = mock(WebSocketHandler.class);
        Map<String, Object> attributes = new HashMap<>();

        when(jwtUtil.validateToken("invalid_token")).thenReturn(false);

        // Act
        boolean result = interceptor.beforeHandshake(request, response, handler, attributes);

        // Assert
        assertFalse(result);
    }

    @Test
    @DisplayName("握手前 - Token缺失时拒绝")
    void beforeHandshake_ShouldFail_WhenTokenMissing() throws Exception {
        // Arrange
        MockHttpServletRequest servletRequest = new MockHttpServletRequest();
        // No token parameter
        ServletServerHttpRequest request = new ServletServerHttpRequest(servletRequest);
        
        ServerHttpResponse response = mock(ServerHttpResponse.class);
        WebSocketHandler handler = mock(WebSocketHandler.class);
        Map<String, Object> attributes = new HashMap<>();

        // Act
        boolean result = interceptor.beforeHandshake(request, response, handler, attributes);

        // Assert
        assertFalse(result);
    }
}
