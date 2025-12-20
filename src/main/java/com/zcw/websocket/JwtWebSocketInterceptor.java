package com.zcw.websocket;

import com.zcw.util.JwtUtil;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.server.ServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.http.server.ServletServerHttpRequest;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketHandler;
import org.springframework.web.socket.server.HandshakeInterceptor;

import java.util.Map;

/**
 * WebSocket JWT握手拦截器
 * 在WebSocket连接建立前验证Token
 */
@Slf4j
@Component
public class JwtWebSocketInterceptor implements HandshakeInterceptor {

    private final JwtUtil jwtUtil;

    public JwtWebSocketInterceptor(JwtUtil jwtUtil) {
        this.jwtUtil = jwtUtil;
    }

    @Override
    public boolean beforeHandshake(ServerHttpRequest request, ServerHttpResponse response,
                                   WebSocketHandler wsHandler, Map<String, Object> attributes) throws Exception {

        if (request instanceof ServletServerHttpRequest) {
            ServletServerHttpRequest servletRequest = (ServletServerHttpRequest) request;

            // 从URL参数获取Token
            String token = servletRequest.getServletRequest().getParameter("token");

            if (token == null || token.trim().isEmpty()) {
                log.warn("WebSocket连接缺少Token");
                return false;
            }

            // 验证Token
            if (!jwtUtil.validateToken(token)) {
                log.warn("WebSocket连接Token无效");
                return false;
            }

            log.info("WebSocket连接Token验证成功");
            return true;
        }

        return false;
    }

    @Override
    public void afterHandshake(ServerHttpRequest request, ServerHttpResponse response,
                              WebSocketHandler wsHandler, Exception exception) {
        // 握手完成后的处理(可选)
    }
}
