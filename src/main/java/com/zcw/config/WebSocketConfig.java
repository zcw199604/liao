package com.zcw.config;

import com.zcw.websocket.JwtWebSocketInterceptor;
import com.zcw.websocket.ProxyWebSocketHandler;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.socket.config.annotation.EnableWebSocket;
import org.springframework.web.socket.config.annotation.WebSocketConfigurer;
import org.springframework.web.socket.config.annotation.WebSocketHandlerRegistry;

/**
 * WebSocket 配置类
 */
@Configuration
@EnableWebSocket
public class WebSocketConfig implements WebSocketConfigurer {

    private final ProxyWebSocketHandler proxyWebSocketHandler;
    private final JwtWebSocketInterceptor jwtWebSocketInterceptor;

    public WebSocketConfig(ProxyWebSocketHandler proxyWebSocketHandler, JwtWebSocketInterceptor jwtWebSocketInterceptor) {
        this.proxyWebSocketHandler = proxyWebSocketHandler;
        this.jwtWebSocketInterceptor = jwtWebSocketInterceptor;
    }

    @Override
    public void registerWebSocketHandlers(WebSocketHandlerRegistry registry) {
        // 注册 WebSocket 端点，客户端通过 ws://localhost:8080/ws 连接
        registry.addHandler(proxyWebSocketHandler, "/ws")
                .setAllowedOrigins("*")      // 允许跨域
                .addInterceptors(jwtWebSocketInterceptor);  // 添加JWT拦截器
    }
}
