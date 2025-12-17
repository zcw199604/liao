package com.zcw.websocket;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.java_websocket.client.WebSocketClient;
import org.java_websocket.handshake.ServerHandshake;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URI;
import java.util.concurrent.ConcurrentLinkedQueue;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * 上游 WebSocket 客户端
 * 连接到上游 WebSocket 服务，并将消息转发回下游客户端
 */
public class UpstreamWebSocketClient extends WebSocketClient {

    private static final Logger log = LoggerFactory.getLogger(UpstreamWebSocketClient.class);

    // 用户ID
    private final String userId;

    // 上游连接管理器
    private final UpstreamWebSocketManager manager;

    // JSON 解析器
    private final ObjectMapper objectMapper;

    // 消息缓存队列，用于缓存连接建立前的消息
    private final ConcurrentLinkedQueue<String> messageQueue = new ConcurrentLinkedQueue<>();

    // 连接是否已建立
    private volatile boolean connected = false;

    // 心跳定时任务
    private ScheduledExecutorService heartbeatExecutor;

    public UpstreamWebSocketClient(String serverUri, String userId, UpstreamWebSocketManager manager) {
        super(URI.create(serverUri));
        this.userId = userId;
        this.manager = manager;
        this.objectMapper = new ObjectMapper();

        // 设置连接丢失检测超时：60秒没响应则认为连接丢失
        this.setConnectionLostTimeout(700);
    }

    /**
     * 连接建立成功
     */
    @Override
    public void onOpen(ServerHandshake handshake) {
        log.info("上游 WebSocket 连接建立成功: userId={}, status={}", userId, handshake.getHttpStatus());
        connected = true;

        // 启动心跳发送
        startHeartbeat();

        // 发送缓存的消息
        flushMessageQueue();
    }

    /**
     * 启动心跳发送任务
     */
    private void startHeartbeat() {
        if (heartbeatExecutor == null || heartbeatExecutor.isShutdown()) {
            heartbeatExecutor = Executors.newSingleThreadScheduledExecutor();
            heartbeatExecutor.scheduleAtFixedRate(() -> {
                try {
                    if (isOpen()) {
                        sendPing();
                        log.debug("发送心跳ping: userId={}", userId);
                    }
                } catch (Exception e) {
                    log.error("发送心跳失败: userId={}", userId, e);
                }
            }, 600, 600, TimeUnit.SECONDS); // 每30秒发送一次

            log.info("心跳任务已启动: userId={}", userId);
        }
    }

    /**
     * 停止心跳任务
     */
    private void stopHeartbeat() {
        if (heartbeatExecutor != null && !heartbeatExecutor.isShutdown()) {
            heartbeatExecutor.shutdown();
            log.info("心跳任务已停止: userId={}", userId);
        }
    }

    /**
     * 发送消息（如果未连接则缓存）
     */
    public void sendMessage(String message) {
        if (connected && isOpen()) {
            send(message);
            log.info("消息已发送到上游服务: userId={}", userId);
        } else {
            log.info("上游连接未建立，消息已加入队列: userId={}", userId);
            messageQueue.offer(message);
        }
    }

    /**
     * 发送队列中的所有缓存消息
     */
    private void flushMessageQueue() {
        String message;
        while ((message = messageQueue.poll()) != null) {
            send(message);
            log.info("发送缓存消息到上游服务: {}", message);
        }
    }

    /**
     * 收到上游服务的消息，直接转发给当前用户的所有下游客户端
     */
    @Override
    public void onMessage(String message) {
        log.info("收到上游服务消息: userId={}, message={}", userId, message);

        // 直接广播到当前用户的所有下游客户端
        // 不需要解析消息内容，因为一个上游连接属于一个用户
        manager.broadcastToDownstream(userId, message);
    }

    /**
     * 上游连接关闭
     */
    @Override
    public void onClose(int code, String reason, boolean remote) {
        log.info("上游 WebSocket 连接关闭: userId={}, code={}, reason={}, remote={}", userId, code, reason, remote);
        connected = false;

        // 停止心跳任务
        stopHeartbeat();

        // 通知管理器处理重连
        manager.handleUpstreamDisconnect(userId);
    }

    /**
     * 发生错误
     */
    @Override
    public void onError(Exception ex) {
        log.error("上游 WebSocket 连接错误: userId={}", userId, ex);
    }

    /**
     * 关闭连接时清理资源
     */
    @Override
    public void close() {
        stopHeartbeat();
        super.close();
    }
}

