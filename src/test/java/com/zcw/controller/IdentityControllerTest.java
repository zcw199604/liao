package com.zcw.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.zcw.interceptor.JwtInterceptor;
import com.zcw.model.Identity;
import com.zcw.service.IdentityService;
import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.util.Arrays;
import java.util.Collections;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(IdentityController.class)
@DisplayName("身份控制器测试")
class IdentityControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private IdentityService identityService;

    // Mock dependencies for JwtInterceptor
    @MockBean
    private JwtUtil jwtUtil;

    // We can also mock the interceptor itself if needed, 
    // but mocking JwtUtil is enough if we pass the token.
    // However, @WebMvcTest will instantiate JwtInterceptor.
    
    @BeforeEach
    void setUp() {
        // Make sure token validation passes
        when(jwtUtil.validateToken(anyString())).thenReturn(true);
    }

    @Test
    @DisplayName("获取身份列表 - 成功")
    void getIdentityList_ShouldReturnList() throws Exception {
        Identity identity = new Identity("1", "User1", "男");
        when(identityService.getAllIdentities()).thenReturn(Arrays.asList(identity));

        mockMvc.perform(get("/api/getIdentityList")
                        .header("Authorization", "Bearer valid_token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data[0].name").value("User1"));
    }

    @Test
    @DisplayName("创建身份 - 成功")
    void createIdentity_ShouldReturnSuccess() throws Exception {
        Identity identity = new Identity("1", "NewUser", "女");
        when(identityService.createIdentity("NewUser", "女")).thenReturn(identity);

        mockMvc.perform(post("/api/createIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("name", "NewUser")
                        .param("sex", "女"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.name").value("NewUser"));
    }

    @Test
    @DisplayName("创建身份 - 参数验证失败(缺少名字)")
    void createIdentity_ShouldFail_WhenNameMissing() throws Exception {
        mockMvc.perform(post("/api/createIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("sex", "女"))
                .andExpect(status().isBadRequest()); // Required param missing throws 400
    }
    
    @Test
    @DisplayName("创建身份 - 参数验证失败(性别非法)")
    void createIdentity_ShouldFail_WhenSexInvalid() throws Exception {
        mockMvc.perform(post("/api/createIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("name", "User")
                        .param("sex", "Invalid"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.msg").value("性别必须是男或女"));
    }

    @Test
    @DisplayName("更新身份 - 成功")
    void updateIdentity_ShouldReturnSuccess() throws Exception {
        Identity identity = new Identity("1", "Updated", "男");
        when(identityService.updateIdentity("1", "Updated", "男")).thenReturn(identity);

        mockMvc.perform(post("/api/updateIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("id", "1")
                        .param("name", "Updated")
                        .param("sex", "男"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("更新身份 - 失败(身份不存在)")
    void updateIdentity_ShouldFail_WhenNotFound() throws Exception {
        when(identityService.updateIdentity("999", "Name", "男")).thenReturn(null);

        mockMvc.perform(post("/api/updateIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("id", "999")
                        .param("name", "Name")
                        .param("sex", "男"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.msg").value("身份不存在"));
    }

    @Test
    @DisplayName("删除身份 - 成功")
    void deleteIdentity_ShouldReturnSuccess() throws Exception {
        when(identityService.deleteIdentity("1")).thenReturn(true);

        mockMvc.perform(post("/api/deleteIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("id", "1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("快速创建身份 - 成功")
    void quickCreateIdentity_ShouldReturnSuccess() throws Exception {
        Identity identity = new Identity("quick_id", "RandomName", "男");
        when(identityService.quickCreateIdentity()).thenReturn(identity);

        mockMvc.perform(post("/api/quickCreateIdentity")
                        .header("Authorization", "Bearer valid_token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.id").value("quick_id"));
    }

    @Test
    @DisplayName("更新身份ID - 成功")
    void updateIdentityId_ShouldReturnSuccess() throws Exception {
        Identity identity = new Identity("new_id", "Name", "女");
        when(identityService.updateIdentityId("old_id", "new_id", "Name", "女")).thenReturn(identity);

        mockMvc.perform(post("/api/updateIdentityId")
                        .header("Authorization", "Bearer valid_token")
                        .param("oldId", "old_id")
                        .param("newId", "new_id")
                        .param("name", "Name")
                        .param("sex", "女"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("选择身份 - 成功")
    void selectIdentity_ShouldReturnSuccess() throws Exception {
        Identity identity = new Identity("id1", "Name", "男");
        when(identityService.getIdentityById("id1")).thenReturn(identity);

        mockMvc.perform(post("/api/selectIdentity")
                        .header("Authorization", "Bearer valid_token")
                        .param("id", "id1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }
}
