package com.zcw.controller;

import com.zcw.service.MediaUploadService;
import com.zcw.service.ImageServerService;
import com.zcw.service.ImageCacheService;
import com.zcw.model.MediaUploadHistory;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.util.Collections;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(MediaHistoryController.class)
@DisplayName("媒体历史控制器测试")
class MediaHistoryControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private MediaUploadService mediaUploadService;

    @MockBean
    private ImageServerService imageServerService;

    @MockBean
    private ImageCacheService imageCacheService;

    @MockBean
    private com.zcw.util.JwtUtil jwtUtil;

    @BeforeEach
    void setUp() {
        // Mock Token validation to pass interceptor
        when(jwtUtil.validateToken(anyString())).thenReturn(true);
    }

    @Test
    @DisplayName("查询用户上传历史 - 成功")
    void getUserUploadHistory_ShouldReturnList() throws Exception {
        // Arrange
        when(mediaUploadService.getUserUploadHistory(eq("u1"), anyInt(), anyInt(), any()))
                .thenReturn(Collections.emptyList());
        when(mediaUploadService.getUserUploadCount("u1")).thenReturn(0);

        // Act & Assert
        mockMvc.perform(get("/api/getUserUploadHistory")
                        .header("Authorization", "Bearer mock_token")
                        .param("userId", "u1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.list").isArray());
    }

    @Test
    @DisplayName("记录图片发送 - 成功")
    void recordImageSend_ShouldReturnSuccess() throws Exception {
        // Arrange
        MediaUploadHistory history = new MediaUploadHistory();
        history.setId(1L);
        when(mediaUploadService.recordImageSend(anyString(), anyString(), anyString(), any()))
                .thenReturn(history);

        // Act & Assert
        mockMvc.perform(post("/api/recordImageSend")
                        .header("Authorization", "Bearer mock_token")
                        .param("remoteUrl", "http://url")
                        .param("fromUserId", "u1")
                        .param("toUserId", "u2"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true));
    }
}
