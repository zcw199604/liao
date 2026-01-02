package com.zcw.controller;

import com.zcw.websocket.ForceoutManager;
import com.zcw.websocket.UpstreamWebSocketManager;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;

import java.util.HashMap;
import java.util.Map;

/**
 * 系统管理接口
 */
@Slf4j
@RestController
@RequestMapping("/api")
public class SystemController {

    @Autowired
    private UpstreamWebSocketManager upstreamWebSocketManager;

    @Autowired
    private ForceoutManager forceoutManager;

    private final RestTemplate restTemplate = new RestTemplate();

    /**
     * 删除上游用户
     * POST /api/deleteUpstreamUser
     */
    @PostMapping("/deleteUpstreamUser")
    public Map<String, Object> deleteUpstreamUser(@RequestParam String myUserId, @RequestParam String userToId) {
        log.info("删除上游用户: myUserId={}, userToId={}", myUserId, userToId);
        Map<String, Object> response = new HashMap<>();

        try {
            String url = "http://v1.chat2019.cn/asmx/method.asmx/Del_User";
            
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

            MultiValueMap<String, String> map = new LinkedMultiValueMap<>();
            map.add("myUserID", myUserId);
            map.add("UserToID", userToId);
            map.add("vipcode", "");
            map.add("serverPort", "1001");

            HttpEntity<MultiValueMap<String, String>> request = new HttpEntity<>(map, headers);

            String result = restTemplate.postForObject(url, request, String.class);
            log.info("删除上游用户结果: {}", result);

            response.put("code", 0);
            response.put("msg", "success");
            response.put("data", result);

        } catch (Exception e) {
            log.error("删除上游用户失败", e);
            response.put("code", -1);
            response.put("msg", "删除失败: " + e.getMessage());
        }

        return response;
    }

    /**
     * 获取连接统计信息
     */
    @GetMapping("/getConnectionStats")
    public Map<String, Object> getConnectionStats() {
        log.info("获取连接统计信息");

        Map<String, Object> response = new HashMap<>();
        try {
            Map<String, Object> stats = upstreamWebSocketManager.getConnectionStats();

            response.put("code", 0);
            response.put("msg", "success");
            response.put("data", stats);
        } catch (Exception e) {
            log.error("获取连接统计失败", e);
            response.put("code", -1);
            response.put("msg", "获取统计信息失败: " + e.getMessage());
        }

        return response;
    }

    /**
     * 断开所有WebSocket连接
     */
    @PostMapping("/disconnectAllConnections")
    public Map<String, Object> disconnectAllConnections() {
        log.info("执行断开所有连接操作");

        Map<String, Object> response = new HashMap<>();
        try {
            upstreamWebSocketManager.closeAllConnections();

            response.put("code", 0);
            response.put("msg", "所有连接已断开");
        } catch (Exception e) {
            log.error("断开所有连接失败", e);
            response.put("code", -1);
            response.put("msg", "操作失败: " + e.getMessage());
        }

        return response;
    }

    /**
     * 获取被forceout禁止的用户数量
     */
    @GetMapping("/getForceoutUserCount")
    public Map<String, Object> getForceoutUserCount() {
        log.info("获取被禁止用户数量");

        Map<String, Object> response = new HashMap<>();
        try {
            int count = forceoutManager.getForbiddenUserCount();
            log.info("📊 当前被禁止用户数量: {}", count);

            response.put("code", 0);
            response.put("data", count);
        } catch (Exception e) {
            log.error("获取被禁止用户数量失败", e);
            response.put("code", -1);
            response.put("msg", "获取失败: " + e.getMessage());
        }

        return response;
    }

    /**
     * 清除所有被forceout禁止的用户
     */
    @PostMapping("/clearForceoutUsers")
    public Map<String, Object> clearForceoutUsers() {
        log.info("执行清除所有被禁止用户操作");

        Map<String, Object> response = new HashMap<>();
        try {
            int count = forceoutManager.clearAllForceout();

            response.put("code", 0);
            response.put("msg", String.format("已清除%d个被禁止的用户", count));
        } catch (Exception e) {
            log.error("清除被禁止用户失败", e);
            response.put("code", -1);
            response.put("msg", "操作失败: " + e.getMessage());
        }

        return response;
    }
}
