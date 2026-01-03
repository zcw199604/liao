package com.zcw.websocket;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.handler.TextWebSocketHandler;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * WebSocket 代理处理器
 * 接收客户端连接，并转发消息到上游 WebSocket 服务
 */
@Component
public class ProxyWebSocketHandler extends TextWebSocketHandler {

    private static final Logger log = LoggerFactory.getLogger(ProxyWebSocketHandler.class);

    private final UpstreamWebSocketManager upstreamManager;
    private final ObjectMapper objectMapper;

    // 会话ID -> 用户ID 的映射
    private final Map<String, String> sessionUserMap = new ConcurrentHashMap<>();

    public ProxyWebSocketHandler(UpstreamWebSocketManager upstreamManager) {
        this.upstreamManager = upstreamManager;
        this.objectMapper = new ObjectMapper();
    }

    /**
     * 客户端连接建立时
     */
    @Override
    public void afterConnectionEstablished(WebSocketSession session) throws Exception {
        log.info("客户端连接建立: sessionId={}", session.getId());
        // 注意：此时还不知道用户ID，需要等待第一条消息（sign消息）
    }

    /**
     * 接收到客户端消息时，转发到上游服务
     */
    @Override
    protected void handleTextMessage(WebSocketSession session, TextMessage message) throws Exception {
        String payload = message.getPayload();
        log.info("收到客户端消息: sessionId={}, message={}", session.getId(), payload);

        try {
            // 解析消息，提取用户ID
            JsonNode jsonNode = objectMapper.readTree(payload);
            String act = jsonNode.has("act") ? jsonNode.get("act").asText() : null;
            String userId = jsonNode.has("id") ? jsonNode.get("id").asText() : null;

            log.info("解析消息: act={}, userId={}", act, userId);

            if (userId == null) {
                log.warn("消息中没有用户ID，无法转发");
                return;
            }

            // 如果是 sign 消息，注册客户端
            if ("sign".equals(act)) {
                log.info("收到登录消息: userId={}, sessionId={}", userId, session.getId());
                sessionUserMap.put(session.getId(), userId);
                upstreamManager.registerDownstream(userId, session, payload);
            } else {
                // 非 sign 消息，先确保该 session 已注册
                String mappedUserId = sessionUserMap.get(session.getId());
                if (mappedUserId == null) {
                    log.warn("会话未注册，无法转发消息: sessionId={}", session.getId());
                    return;
                }
                // 转发到上游服务
                upstreamManager.sendToUpstream(userId, payload);
            }

        } catch (Exception e) {
            log.error("处理消息失败", e);
        }
    }

    /**
     * 客户端断开连接时
     */
    @Override
    public void afterConnectionClosed(WebSocketSession session, CloseStatus status) throws Exception {
        log.info("客户端连接关闭: sessionId={}, status={}", session.getId(), status);

        // 从上游管理器注销
        String userId = sessionUserMap.remove(session.getId());
        if (userId != null) {
            upstreamManager.unregisterDownstream(userId, session);
        }
    }

    /**
     * 发生错误时
     */
    @Override
    public void handleTransportError(WebSocketSession session, Throwable exception) throws Exception {
        if (exception instanceof java.io.IOException) {
            log.error("WebSocket 传输错误: sessionId={}, error={}", session.getId(), exception.getMessage());
        } else {
            log.error("WebSocket 传输错误: sessionId={}", session.getId(), exception);
        }

        try {
            if (session.isOpen()) {
                session.close(CloseStatus.SERVER_ERROR);
            }
        } catch (Exception e) {
            // ignore
        }
    }
}

