package com.zcw.controller;

import com.zcw.websocket.ForceoutManager;
import com.zcw.websocket.UpstreamWebSocketManager;
import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.HttpEntity;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.web.client.RestTemplate;

import java.util.HashMap;
import java.util.Map;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(SystemController.class)
@DisplayName("系统控制器测试")
class SystemControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private UpstreamWebSocketManager upstreamWebSocketManager;

    @MockBean
    private ForceoutManager forceoutManager;

    @MockBean
    private RestTemplate restTemplate;

    @MockBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUp() {
        when(jwtUtil.validateToken(anyString())).thenReturn(true);
    }

    @Test
    @DisplayName("删除上游用户 - 成功")
    void deleteUpstreamUser_ShouldReturnSuccess() throws Exception {
        when(restTemplate.postForObject(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn("{\"success\":true}");

        mockMvc.perform(post("/api/deleteUpstreamUser")
                        .header("Authorization", "Bearer token")
                        .param("myUserId", "u1")
                        .param("userToId", "u2"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("获取连接统计")
    void getConnectionStats_ShouldReturnData() throws Exception {
        Map<String, Object> stats = new HashMap<>();
        stats.put("active", 10);
        when(upstreamWebSocketManager.getConnectionStats()).thenReturn(stats);

        mockMvc.perform(get("/api/getConnectionStats")
                        .header("Authorization", "Bearer token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.active").value(10));
    }

    @Test
    @DisplayName("清除被禁止用户")
    void clearForceoutUsers_ShouldReturnCount() throws Exception {
        when(forceoutManager.clearAllForceout()).thenReturn(5);

        mockMvc.perform(post("/api/clearForceoutUsers")
                        .header("Authorization", "Bearer token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.msg").value("已清除5个被禁止的用户"));
    }
}
