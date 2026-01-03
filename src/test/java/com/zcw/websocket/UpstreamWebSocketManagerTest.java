package com.zcw.websocket;

import com.zcw.service.WebSocketAddressService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.net.URISyntaxException;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("WebSocket 上游管理器测试")
class UpstreamWebSocketManagerTest {

    @Mock
    private WebSocketAddressService addressService;

    @Mock
    private ForceoutManager forceoutManager;
    
    @Mock
    private ScheduledExecutorService scheduler;

    private TestableUpstreamWebSocketManager manager;

    // 子类化以便 Mock 内部创建的 Client
    static class TestableUpstreamWebSocketManager extends UpstreamWebSocketManager {
        
        public UpstreamWebSocketClient mockClient;

        public TestableUpstreamWebSocketManager(WebSocketAddressService addressService, ForceoutManager forceoutManager, UpstreamWebSocketClient mockClient) {
            super(addressService, forceoutManager, null); // 传入 null 作为 cacheService
            this.mockClient = mockClient;
        }

        @Override
        protected UpstreamWebSocketClient createWebSocketClient(String url, String userId) {
            return mockClient;
        }
    }

    @BeforeEach
    void setUp() {
        UpstreamWebSocketClient mockClient = mock(UpstreamWebSocketClient.class);
        manager = new TestableUpstreamWebSocketManager(addressService, forceoutManager, mockClient);
        // 注入 mock scheduler
        ReflectionTestUtils.setField(manager, "scheduler", scheduler);
    }

    @Test
    @DisplayName("注册下游 - Forceout禁止时拒绝")
    void registerDownstream_ShouldReject_WhenForcedOut() throws Exception {
        // Arrange
        String userId = "bannedUser";
        WebSocketSession session = mock(WebSocketSession.class);
        when(forceoutManager.isForbidden(userId)).thenReturn(true);
        when(forceoutManager.getRemainingSeconds(userId)).thenReturn(300L);

        // Act
        manager.registerDownstream(userId, session, "sign_msg");

        // Assert
        verify(session, times(1)).sendMessage(any(TextMessage.class)); // Verify rejection message sent
        verify(session, times(1)).close();
        verify(addressService, never()).getUpstreamWebSocketUrl(); // Should not create upstream connection
    }

    @Test
    @DisplayName("广播消息 - 发送给所有下游")
    void broadcastToDownstream_ShouldSendToAllSessions() throws Exception {
        // Arrange
        String userId = "user1";
        WebSocketSession session1 = mock(WebSocketSession.class);
        WebSocketSession session2 = mock(WebSocketSession.class);
        when(session1.isOpen()).thenReturn(true);
        when(session2.isOpen()).thenReturn(true);
        when(session1.getId()).thenReturn("s1");
        when(session2.getId()).thenReturn("s2");

        // Inject sessions manually
        Map<String, List<WebSocketSession>> downstreamSessions = 
            (Map<String, List<WebSocketSession>>) ReflectionTestUtils.getField(manager, "downstreamSessions");
        List<WebSocketSession> sessions = new java.util.ArrayList<>();
        sessions.add(session1);
        sessions.add(session2);
        downstreamSessions.put(userId, sessions);

        // Act
        manager.broadcastToDownstream(userId, "message");

        // Assert
        verify(session1, times(1)).sendMessage(any(TextMessage.class));
        verify(session2, times(1)).sendMessage(any(TextMessage.class));
    }
    
    @Test
    @DisplayName("注销下游 - 最后连接断开时调度延迟关闭")
    void unregisterDownstream_ShouldScheduleClose_WhenLastSession() {
        // Arrange
        String userId = "user1";
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.getId()).thenReturn("s1");

        // Inject session
        Map<String, List<WebSocketSession>> downstreamSessions = 
            (Map<String, List<WebSocketSession>>) ReflectionTestUtils.getField(manager, "downstreamSessions");
        List<WebSocketSession> sessions = new java.util.ArrayList<>();
        sessions.add(session);
        downstreamSessions.put(userId, sessions);
        
        when(scheduler.schedule(any(Runnable.class), anyLong(), any(TimeUnit.class)))
                .thenReturn(mock(ScheduledFuture.class));

        // Act
        manager.unregisterDownstream(userId, session);

        // Assert
        assertTrue(downstreamSessions.isEmpty() || !downstreamSessions.containsKey(userId));
        verify(scheduler, times(1)).schedule(any(Runnable.class), eq(80L), eq(TimeUnit.SECONDS));
    }

    @Test
    @DisplayName("处理上游断开 - 关闭所有下游")
    void handleUpstreamDisconnect_ShouldCloseAllDownstream() throws Exception {
        // Arrange
        String userId = "user1";
        WebSocketSession session1 = mock(WebSocketSession.class);
        when(session1.isOpen()).thenReturn(true);

        // Inject session
        Map<String, List<WebSocketSession>> downstreamSessions = 
            (Map<String, List<WebSocketSession>>) ReflectionTestUtils.getField(manager, "downstreamSessions");
        List<WebSocketSession> sessions = new java.util.ArrayList<>();
        sessions.add(session1);
        downstreamSessions.put(userId, sessions);

        // Act
        manager.handleUpstreamDisconnect(userId);

        // Assert
        verify(session1, times(1)).close();
        assertFalse(downstreamSessions.containsKey(userId));
    }
    
    @Test
    @DisplayName("注册下游 - 正常连接流程(Mock验证)")
    void registerDownstream_ShouldCreateUpstream_WhenFirstConnection() throws Exception {
        // Arrange
        String userId = "user_new";
        WebSocketSession session = mock(WebSocketSession.class);
        when(forceoutManager.isForbidden(userId)).thenReturn(false);
        when(addressService.getUpstreamWebSocketUrl()).thenReturn("ws://mock-upstream");
        
        // Act
        manager.registerDownstream(userId, session, "sign_msg");
        
        // Assert
        Map<String, List<WebSocketSession>> downstreamSessions = 
            (Map<String, List<WebSocketSession>>) ReflectionTestUtils.getField(manager, "downstreamSessions");
        
        assertTrue(downstreamSessions.containsKey(userId));
        assertEquals(1, downstreamSessions.get(userId).size());
        verify(addressService, times(1)).getUpstreamWebSocketUrl();
        
        // 验证 mock client 被调用连接
        verify(manager.mockClient, times(1)).connect();
        verify(manager.mockClient, times(1)).sendMessage("sign_msg");
    }
}