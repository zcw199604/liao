package com.zcw.controller;

import com.zcw.config.AuthProperties;
import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(AuthController.class)
@DisplayName("认证控制器测试")
class AuthControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private AuthProperties authProperties;

    @MockBean
    private JwtUtil jwtUtil;

    @Test
    @DisplayName("登录 - 访问码正确时成功")
    void login_ShouldReturnToken_WhenCodeCorrect() throws Exception {
        // Arrange
        String code = "123456";
        when(authProperties.getAccessCode()).thenReturn(code);
        when(jwtUtil.generateToken()).thenReturn("mock_token");

        // Act & Assert
        mockMvc.perform(post("/api/auth/login")
                        .param("accessCode", code))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.token").value("mock_token"));
    }

    @Test
    @DisplayName("登录 - 访问码错误时失败")
    void login_ShouldFail_WhenCodeIncorrect() throws Exception {
        // Arrange
        when(authProperties.getAccessCode()).thenReturn("correct_code");

        // Act & Assert
        mockMvc.perform(post("/api/auth/login")
                        .param("accessCode", "wrong_code"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value(-1))
                .andExpect(jsonPath("$.msg").value("访问码错误"));
    }

    @Test
    @DisplayName("验证Token - Token有效")
    void verifyToken_ShouldReturnValid_WhenTokenValid() throws Exception {
        // Arrange
        String token = "valid_token";
        when(jwtUtil.validateToken(token)).thenReturn(true);

        // Act & Assert
        mockMvc.perform(get("/api/auth/verify")
                        .header("Authorization", "Bearer " + token))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.valid").value(true));
    }

    @Test
    @DisplayName("验证Token - Token缺失")
    void verifyToken_ShouldFail_WhenTokenMissing() throws Exception {
        // Act & Assert
        mockMvc.perform(get("/api/auth/verify"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(-1))
                .andExpect(jsonPath("$.msg").value("Token缺失"));
    }
}
