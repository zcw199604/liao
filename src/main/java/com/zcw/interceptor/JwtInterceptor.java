package com.zcw.interceptor;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.zcw.util.JwtUtil;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

import java.util.HashMap;
import java.util.Map;

/**
 * JWT拦截器
 * 拦截所有API请求，验证Token
 */
@Slf4j
@Component
public class JwtInterceptor implements HandlerInterceptor {

    @Autowired
    private JwtUtil jwtUtil;

    @Autowired
    private ObjectMapper objectMapper;

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {
        // 处理预检请求
        if ("OPTIONS".equals(request.getMethod())) {
            return true;
        }

        // 获取Token
        String authHeader = request.getHeader("Authorization");

        if (authHeader == null || !authHeader.startsWith("Bearer ")) {
            log.warn("请求缺少Token: {} {}", request.getMethod(), request.getRequestURI());
            sendUnauthorizedResponse(response, "未登录或Token缺失");
            return false;
        }

        String token = authHeader.substring(7);

        // 验证Token
        if (!jwtUtil.validateToken(token)) {
            log.warn("Token验证失败: {} {}", request.getMethod(), request.getRequestURI());
            sendUnauthorizedResponse(response, "Token无效或已过期");
            return false;
        }

        return true;
    }

    /**
     * 发送401响应
     */
    private void sendUnauthorizedResponse(HttpServletResponse response, String message) throws Exception {
        response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
        response.setContentType("application/json;charset=UTF-8");

        Map<String, Object> result = new HashMap<>();
        result.put("code", 401);
        result.put("msg", message);

        response.getWriter().write(objectMapper.writeValueAsString(result));
    }
}
