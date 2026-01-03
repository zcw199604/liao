package com.zcw.controller;

import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.forwardedUrl;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(SpaForwardController.class)
@DisplayName("SPA转发控制器测试")
class SpaForwardControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private JwtUtil jwtUtil;

    // JwtInterceptor 可能会拦截这些请求（除了 /login, /api/auth/verify 等排除路径）
    // 但 SpaForwardController 通常处理的是页面请求，不是 /api 请求。
    // 如果 JwtInterceptor 配置只拦截 /api/**，则这里无需 Mock Token。
    // 检查 WebMvcConfig，确实只拦截 /api/**。

    @Test
    @DisplayName("根路径转发到index.html")
    void root_ShouldForward() throws Exception {
        mockMvc.perform(get("/"))
                .andExpect(status().isOk())
                .andExpect(forwardedUrl("/index.html"));
    }

    @Test
    @DisplayName("前端路由转发到index.html")
    void chatRoute_ShouldForward() throws Exception {
        mockMvc.perform(get("/chat/123"))
                .andExpect(status().isOk())
                .andExpect(forwardedUrl("/index.html"));
    }
}
