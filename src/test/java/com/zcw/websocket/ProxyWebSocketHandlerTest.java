package com.zcw.websocket;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.util.HashMap;
import java.util.Map;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("WebSocket 代理处理器测试")
class ProxyWebSocketHandlerTest {

    @Mock
    private UpstreamWebSocketManager upstreamManager;

    @InjectMocks
    private ProxyWebSocketHandler proxyHandler;

    private final ObjectMapper objectMapper = new ObjectMapper();

    @Test
    @DisplayName("处理消息 - 登录(Sign)消息注册客户端")
    void handleTextMessage_ShouldRegister_WhenSignMessage() throws Exception {
        // Arrange
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.getId()).thenReturn("session1");
        
        Map<String, String> msgMap = new HashMap<>();
        msgMap.put("act", "sign");
        msgMap.put("id", "user1");
        String payload = objectMapper.writeValueAsString(msgMap);
        TextMessage message = new TextMessage(payload);

        // Act
        proxyHandler.handleTextMessage(session, message);

        // Assert
        verify(upstreamManager, times(1)).registerDownstream(eq("user1"), eq(session), eq(payload));
    }

    @Test
    @DisplayName("处理消息 - 普通消息转发到上游")
    void handleTextMessage_ShouldForward_WhenNormalMessage() throws Exception {
        // Arrange
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.getId()).thenReturn("session1");
        
        // First send sign message to register session in map
        Map<String, String> signMap = new HashMap<>();
        signMap.put("act", "sign");
        signMap.put("id", "user1");
        proxyHandler.handleTextMessage(session, new TextMessage(objectMapper.writeValueAsString(signMap)));

        // Now send normal message
        Map<String, String> msgMap = new HashMap<>();
        msgMap.put("act", "chat");
        msgMap.put("id", "user1");
        msgMap.put("content", "hello");
        String payload = objectMapper.writeValueAsString(msgMap);
        TextMessage message = new TextMessage(payload);

        // Act
        proxyHandler.handleTextMessage(session, message);

        // Assert
        verify(upstreamManager, times(1)).sendToUpstream(eq("user1"), eq(payload));
    }

    @Test
    @DisplayName("处理消息 - 未注册会话的消息不转发")
    void handleTextMessage_ShouldNotForward_WhenSessionNotRegistered() throws Exception {
        // Arrange
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.getId()).thenReturn("unknown_session");
        
        Map<String, String> msgMap = new HashMap<>();
        msgMap.put("act", "chat");
        msgMap.put("id", "user1");
        String payload = objectMapper.writeValueAsString(msgMap);
        TextMessage message = new TextMessage(payload);

        // Act
        proxyHandler.handleTextMessage(session, message);

        // Assert
        verify(upstreamManager, never()).sendToUpstream(anyString(), anyString());
    }

    @Test
    @DisplayName("连接关闭 - 注销客户端")
    void afterConnectionClosed_ShouldUnregister() throws Exception {
        // Arrange
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.getId()).thenReturn("session1");
        
        // First register
        Map<String, String> signMap = new HashMap<>();
        signMap.put("act", "sign");
        signMap.put("id", "user1");
        proxyHandler.handleTextMessage(session, new TextMessage(objectMapper.writeValueAsString(signMap)));

        // Act
        proxyHandler.afterConnectionClosed(session, CloseStatus.NORMAL);

        // Assert
        verify(upstreamManager, times(1)).unregisterDownstream(eq("user1"), eq(session));
    }
}
