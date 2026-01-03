package com.zcw.util;

import com.zcw.config.AuthProperties;
import io.jsonwebtoken.Claims;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("JWT工具类测试")
class JwtUtilTest {

    private JwtUtil jwtUtil;

    @BeforeEach
    void setUp() {
        AuthProperties properties = new AuthProperties();
        // 必须设置足够长度的密钥 (HS256 至少 32 bytes)
        properties.setJwtSecret("mySuperSecretKeyThatIsLongEnoughForHS256Algorithm");
        properties.setTokenExpireHours(1);
        jwtUtil = new JwtUtil(properties);
    }

    @Test
    @DisplayName("生成Token - 成功")
    void generateToken_ShouldReturnToken() {
        String token = jwtUtil.generateToken();
        assertNotNull(token);
        assertFalse(token.isEmpty());
    }

    @Test
    @DisplayName("验证Token - 有效Token返回True")
    void validateToken_ShouldReturnTrue_WhenValid() {
        String token = jwtUtil.generateToken();
        assertTrue(jwtUtil.validateToken(token));
    }

    @Test
    @DisplayName("验证Token - 无效Token返回False")
    void validateToken_ShouldReturnFalse_WhenInvalid() {
        assertFalse(jwtUtil.validateToken("invalid_token_string"));
    }

    @Test
    @DisplayName("解析Token - 获取Claims")
    void getClaimsFromToken_ShouldReturnSubject() {
        String token = jwtUtil.generateToken();
        Claims claims = jwtUtil.getClaimsFromToken(token);
        assertEquals("user", claims.getSubject());
    }
}
