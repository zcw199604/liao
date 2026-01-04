package com.zcw.controller;

import com.zcw.service.FileStorageService;
import com.zcw.service.ImageCacheService;
import com.zcw.service.ImageServerService;
import com.zcw.service.MediaUploadService;
import com.zcw.service.UserInfoCacheService;
import com.zcw.model.CachedLastMessage;
import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.springframework.web.client.RestTemplate;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.http.ResponseEntity;
import org.springframework.http.HttpEntity;

import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;

@WebMvcTest(UserHistoryController.class)
@DisplayName("用户历史控制器测试")
class UserHistoryControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private ImageServerService imageServerService;

    @MockBean
    private ImageCacheService imageCacheService;

    @MockBean
    private FileStorageService fileStorageService;

    @MockBean
    private MediaUploadService mediaUploadService;

    @MockBean
    private JwtUtil jwtUtil;

    @MockBean
    private RestTemplate restTemplate;

    @MockBean
    private UserInfoCacheService userInfoCacheService;

    @BeforeEach
    void setUp() {
        when(jwtUtil.validateToken(anyString())).thenReturn(true);
        when(imageServerService.getImgServerHost()).thenReturn("127.0.0.1:8080");
    }

    @Test
    @DisplayName("获取缓存图片 - 空列表")
    void getCachedImages_ShouldReturnEmpty_WhenNoCache() throws Exception {
        when(imageCacheService.getCachedImages("u1")).thenReturn(null);

        mockMvc.perform(get("/api/getCachedImages")
                        .header("Authorization", "Bearer mock_token")
                        .param("userid", "u1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data").isArray())
                .andExpect(jsonPath("$.data").isEmpty());
    }

    @Test
    @DisplayName("更新图片服务器地址")
    void updateImgServer_ShouldReturnSuccess() throws Exception {
        mockMvc.perform(post("/api/updateImgServer")
                        .header("Authorization", "Bearer mock_token")
                        .param("server", "192.168.1.100"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true));
    }

    @Test
    @DisplayName("上传媒体 - 成功")
    void uploadMedia_ShouldReturnSuccess() throws Exception {
        MockMultipartFile file = new MockMultipartFile("file", "test.jpg", "image/jpeg", "content".getBytes());
        
        when(fileStorageService.isValidMediaType(anyString())).thenReturn(true);
        when(fileStorageService.calculateMD5(any())).thenReturn("md5");
        when(fileStorageService.saveFile(any(), anyString())).thenReturn("/path/to/file");
        
        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(ResponseEntity.ok("{\"state\":\"OK\",\"msg\":\"remote_path\"}"));

        mockMvc.perform(org.springframework.test.web.servlet.request.MockMvcRequestBuilders.multipart("/api/uploadMedia")
                        .file(file)
                        .header("Authorization", "Bearer mock_token")
                        .param("userid", "u1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.state").value("OK"));
    }

    @Test
    @DisplayName("收藏用户 - 成功")
    void toggleFavorite_ShouldReturnSuccess() throws Exception {
        when(restTemplate.exchange(anyString(), any(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(ResponseEntity.ok("{\"success\":true}"));

        mockMvc.perform(post("/api/toggleFavorite")
                        .header("Authorization", "Bearer mock_token")
                        .param("myUserID", "u1")
                        .param("UserToID", "u2"))
                .andExpect(status().isOk());
    }

    @Test
    @DisplayName("获取历史用户列表 - 成功")
    void getHistoryUserList_ShouldReturnList() throws Exception {
        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(ResponseEntity.ok("[]"));

        mockMvc.perform(post("/api/getHistoryUserList")
                        .header("Authorization", "Bearer mock_token")
                        .param("myUserID", "u1"))
                .andExpect(status().isOk());
    }

    @Test
    @DisplayName("获取消息历史 - 缓存最后消息时以myUserID修正会话Key")
    void getMessageHistory_ShouldCacheLastMessage_WithMyUserIdKey() throws Exception {
        String myUserId = "87d0b2019246400c84c2a390ead62cac";
        String peerUserId = "baa2f3eb5c058c4918633e4eef004aff";
        String aliasUserId = "e33dfdcec95e588658edaa0e23f1b3a7";

        String body = String.format(
            "{\"contents_list\":[{\"id\":\"%s\",\"toid\":\"%s\",\"content\":\"你特殊癖好是\",\"time\":\"2026-01-04 17:41:40.741\"}]}",
            peerUserId, aliasUserId
        );

        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(ResponseEntity.ok(body));

        mockMvc.perform(post("/api/getMessageHistory")
                        .header("Authorization", "Bearer mock_token")
                        .param("myUserID", myUserId)
                        .param("UserToID", peerUserId)
                        .param("isFirst", "1")
                        .param("firstTid", "0"))
                .andExpect(status().isOk());

        String expectedConversationKey = CachedLastMessage.generateConversationKey(myUserId, peerUserId);
        org.mockito.Mockito.verify(userInfoCacheService).saveLastMessage(argThat(m ->
            m != null && expectedConversationKey.equals(m.getConversationKey())
        ));
    }
}
