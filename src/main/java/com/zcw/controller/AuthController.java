package com.zcw.controller;

import com.zcw.config.AuthProperties;
import com.zcw.util.JwtUtil;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

/**
 * 认证API控制器
 * 提供登录和Token验证接口
 */
@Slf4j
@RestController
@RequestMapping("/api/auth")
public class AuthController {

    @Autowired
    private AuthProperties authProperties;

    @Autowired
    private JwtUtil jwtUtil;

    /**
     * 访问码登录
     * POST /api/auth/login
     */
    @PostMapping("/login")
    public ResponseEntity<Map<String, Object>> login(@RequestParam String accessCode) {
        log.info("尝试登录，访问码长度: {}", accessCode != null ? accessCode.length() : 0);

        Map<String, Object> response = new HashMap<>();

        // 验证访问码
        if (accessCode == null || accessCode.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "访问码不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (!accessCode.equals(authProperties.getAccessCode())) {
            log.warn("访问码错误");
            response.put("code", -1);
            response.put("msg", "访问码错误");
            return ResponseEntity.badRequest().body(response);
        }

        // 生成Token
        String token = jwtUtil.generateToken();
        log.info("登录成功，生成Token");

        response.put("code", 0);
        response.put("msg", "登录成功");
        response.put("token", token);

        return ResponseEntity.ok(response);
    }

    /**
     * 验证Token
     * GET /api/auth/verify
     */
    @GetMapping("/verify")
    public ResponseEntity<Map<String, Object>> verifyToken(@RequestHeader(value = "Authorization", required = false) String authHeader) {
        Map<String, Object> response = new HashMap<>();

        if (authHeader == null || !authHeader.startsWith("Bearer ")) {
            response.put("code", -1);
            response.put("msg", "Token缺失");
            return ResponseEntity.ok(response);
        }

        String token = authHeader.substring(7);
        boolean valid = jwtUtil.validateToken(token);

        response.put("code", valid ? 0 : -1);
        response.put("msg", valid ? "Token有效" : "Token无效");
        response.put("valid", valid);

        return ResponseEntity.ok(response);
    }
}
