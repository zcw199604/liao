package com.zcw.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

/**
 * WebSocket 服务地址获取服务
 */
@Service
public class WebSocketAddressService {

    private static final Logger log = LoggerFactory.getLogger(WebSocketAddressService.class);

    private final RestTemplate restTemplate;
    private final ObjectMapper objectMapper;

    // 获取 WebSocket 地址的接口 URL
    private static final String GET_SERVER_URL =
        "http://v1.chat2019.cn/Act/WebService.asmx/getRandServer?ServerInfo=serversdeskry&_=";

    public WebSocketAddressService(RestTemplate restTemplate, ObjectMapper objectMapper) {
        this.restTemplate = restTemplate;
        this.objectMapper = objectMapper;
    }

    /**
     * 动态获取上游 WebSocket 服务器地址
     * @return WebSocket 服务器地址
     */
    public String getUpstreamWebSocketUrl() {
        try {
            // 添加时间戳参数
            String url = GET_SERVER_URL + System.currentTimeMillis();

            log.info("正在获取 WebSocket 服务器地址: {}", url);

            // 调用接口
            String response = restTemplate.getForObject(url, String.class);
            log.info("接口返回数据: {}", response);

            // 解析响应，提取 WebSocket 地址
            String wsUrl = parseWebSocketUrl(response);

            if (wsUrl == null || wsUrl.isEmpty()) {
                log.error("解析 WebSocket 地址失败，使用默认地址");
                return "ws://localhost:9999";
            }

            log.info("获取到 WebSocket 地址: {}", wsUrl);
            return wsUrl;

        } catch (Exception e) {
            log.error("获取 WebSocket 服务器地址失败", e);
            // 返回默认地址作为降级方案
            return "ws://localhost:9999";
        }
    }

    /**
     * 解析接口响应，提取 WebSocket 地址
     * 接口返回格式: {"state":"OK","msg":{"server":"ws://xxx"},"code":0}
     */
    private String parseWebSocketUrl(String response) {
        try {
            // 解析 JSON
            JsonNode rootNode = objectMapper.readTree(response);

            // 检查返回状态
            if (rootNode.has("state") && "OK".equals(rootNode.get("state").asText())) {
                // 提取 msg.server 字段
                if (rootNode.has("msg")) {
                    JsonNode msgNode = rootNode.get("msg");
                    if (msgNode.has("server")) {
                        String wsUrl = msgNode.get("server").asText();
                        if (wsUrl != null && !wsUrl.isEmpty()) {
                            return wsUrl;
                        }
                    }
                }
            }

            log.warn("接口返回状态异常或未找到 server 字段，完整响应: {}", response);

        } catch (Exception e) {
            log.error("解析响应 JSON 失败", e);
        }

        return null;
    }
}
