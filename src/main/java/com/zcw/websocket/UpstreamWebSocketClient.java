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
     * 收到上游服务的消息，检测forceout消息后转发给下游客户端
     */
    @Override
    public void onMessage(String message) {
        log.info("收到上游服务消息: userId={}, message={}", userId, message);

        try {
            com.fasterxml.jackson.databind.JsonNode jsonNode = objectMapper.readTree(message);
            
            // 1. 检查是否是forceout消息
            if (jsonNode.has("code") && jsonNode.get("code").asInt() == -3 &&
                jsonNode.has("forceout") && jsonNode.get("forceout").asBoolean()) {

                log.warn("收到forceout消息，将添加到禁止列表: userId={}", userId);
                // 通知管理器处理forceout
                manager.handleForceout(userId, message);
                return;
            }

            // 2. 检查是否是匹配成功消息 (code=15)，捕获用户信息
            // 格式: {"code":15,"sel_userid":"...","sel_userNikename":"...","sel_userSex":"...","sel_userAge":"...","sel_userAddress":"..."}
            if (jsonNode.has("code") && jsonNode.get("code").asInt() == 15) {
                try {
                    String targetUserId = jsonNode.has("sel_userid") ? jsonNode.get("sel_userid").asText() : null;
                    if (targetUserId != null && !targetUserId.isEmpty()) {
                        com.zcw.model.CachedUserInfo userInfo = new com.zcw.model.CachedUserInfo(
                            targetUserId,
                            jsonNode.has("sel_userNikename") ? jsonNode.get("sel_userNikename").asText() : "",
                            jsonNode.has("sel_userSex") ? jsonNode.get("sel_userSex").asText() : "",
                            jsonNode.has("sel_userAge") ? jsonNode.get("sel_userAge").asText() : "",
                            jsonNode.has("sel_userAddress") ? jsonNode.get("sel_userAddress").asText() : ""
                        );

                        // 异步保存到缓存
                        if (manager.getCacheService() != null) {
                            manager.getCacheService().saveUserInfo(userInfo);
                            log.info("捕获并缓存用户信息: userId={}", targetUserId);
                        }
                    }
                } catch (Exception e) {
                    log.error("解析匹配用户信息失败", e);
                }
            }

            // 3. 检查是否是聊天消息 (code=7)，缓存最后一条消息
            // 格式: {"code":7,"fromuser":{"id":"...","nickname":"...","content":"...","time":"..."},"touser":{"id":"...","nickname":"..."}}
            if (jsonNode.has("code") && jsonNode.get("code").asInt() == 7) {
                try {
                    String fromUserId = jsonNode.path("fromuser").path("id").asText();
                    String toUserId = jsonNode.path("touser").path("id").asText();

                    // 优先从fromuser提取content、time、type，如果不存在则从顶级fallback
                    com.fasterxml.jackson.databind.JsonNode fromuserNode = jsonNode.path("fromuser");
                    String content = fromuserNode.has("content") ?
                                     fromuserNode.path("content").asText() :
                                     jsonNode.path("content").asText();
                    String time = fromuserNode.has("time") ?
                                  fromuserNode.path("time").asText() :
                                  jsonNode.path("time").asText();
                    String type = fromuserNode.has("type") ?
                                  fromuserNode.path("type").asText() :
                                  (jsonNode.has("type") ? jsonNode.path("type").asText() : "text");

                    // 确保所有必要字段都存在
                    if (!fromUserId.isEmpty() && !toUserId.isEmpty() && !content.isEmpty() && !time.isEmpty()) {
                        com.zcw.model.CachedLastMessage lastMsg = new com.zcw.model.CachedLastMessage(
                            fromUserId, toUserId, content, type, time
                        );

                        if (manager.getCacheService() != null) {
                            manager.getCacheService().saveLastMessage(lastMsg);
                            log.debug("缓存最后消息: {} -> {}, content={}, time={}", fromUserId, toUserId, content, time);
                        }
                    } else {
                        log.warn("聊天消息字段不完整: fromUserId={}, toUserId={}, content={}, time={}",
                                 fromUserId, toUserId, content, time);
                    }
                } catch (Exception e) {
                    log.error("缓存聊天消息失败: userId={}, message={}", userId, message, e);
                }
            }

        } catch (Exception e) {
            // 解析失败，按普通消息处理
            log.debug("消息不是JSON格式或解析失败，按普通消息处理: userId={}", userId);
        }

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

