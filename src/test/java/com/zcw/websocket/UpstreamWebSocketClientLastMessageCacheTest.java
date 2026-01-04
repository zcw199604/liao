package com.zcw.websocket;

import com.zcw.model.CachedLastMessage;
import com.zcw.service.UserInfoCacheService;
import com.zcw.service.WebSocketAddressService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.verify;

@ExtendWith(MockitoExtension.class)
@DisplayName("上游WebSocket客户端 - 最后消息缓存兼容性测试")
class UpstreamWebSocketClientLastMessageCacheTest {

    @Mock
    private WebSocketAddressService addressService;

    @Mock
    private ForceoutManager forceoutManager;

    @Mock
    private UserInfoCacheService cacheService;

    private UpstreamWebSocketClient client;

    @BeforeEach
    void setUp() {
        UpstreamWebSocketManager manager = new UpstreamWebSocketManager(addressService, forceoutManager, cacheService);
        client = new UpstreamWebSocketClient("ws://localhost:9999", "87d0b2019246400c84c2a390ead62cac", manager);
    }

    @Test
    @DisplayName("聊天消息code=7 - 当上游toUserId不包含本地userId时仍应补写可命中的会话Key")
    void onMessage_ShouldSaveNormalizedKey_WhenUserIdNotPresentInFromTo() {
        String myUserId = "87d0b2019246400c84c2a390ead62cac";
        String peerUserId = "baa2f3eb5c058c4918633e4eef004aff";
        String aliasUserId = "e33dfdcec95e588658edaa0e23f1b3a7";

        String message = String.format(
            "{\"code\":7,\"fromuser\":{\"id\":\"%s\",\"content\":\"你特殊癖好是\",\"time\":\"2026-01-04 17:41:40.741\",\"type\":\"text\"},\"touser\":{\"id\":\"%s\"}}",
            peerUserId, aliasUserId
        );

        client.onMessage(message);

        String expectedConversationKey = CachedLastMessage.generateConversationKey(myUserId, peerUserId);
        verify(cacheService).saveLastMessage(argThat(m ->
            m != null && expectedConversationKey.equals(m.getConversationKey())
        ));
    }
}

