package com.zcw.controller;

import com.zcw.model.Favorite;
import com.zcw.service.FavoriteService;
import com.zcw.util.JwtUtil;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import java.util.Collections;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(FavoriteController.class)
@DisplayName("收藏控制器测试")
class FavoriteControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private FavoriteService favoriteService;

    @MockBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUp() {
        when(jwtUtil.validateToken(anyString())).thenReturn(true);
    }

    @Test
    @DisplayName("添加收藏 - 成功")
    void addFavorite_ShouldReturnSuccess() throws Exception {
        Favorite favorite = new Favorite();
        favorite.setId(1L);
        when(favoriteService.addFavorite("id1", "target1", "name"))
                .thenReturn(favorite);

        mockMvc.perform(post("/api/favorite/add")
                        .header("Authorization", "Bearer token")
                        .param("identityId", "id1")
                        .param("targetUserId", "target1")
                        .param("targetUserName", "name"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("移除收藏 - 成功")
    void removeFavorite_ShouldReturnSuccess() throws Exception {
        mockMvc.perform(post("/api/favorite/remove")
                        .header("Authorization", "Bearer token")
                        .param("identityId", "id1")
                        .param("targetUserId", "target1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("检查收藏 - 返回状态")
    void checkFavorite_ShouldReturnStatus() throws Exception {
        when(favoriteService.isFavorite("id1", "target1")).thenReturn(true);

        mockMvc.perform(get("/api/favorite/check")
                        .header("Authorization", "Bearer token")
                        .param("identityId", "id1")
                        .param("targetUserId", "target1"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.isFavorite").value(true));
    }

    @Test
    @DisplayName("根据ID移除收藏 - 成功")
    void removeFavoriteById_ShouldReturnSuccess() throws Exception {
        mockMvc.perform(post("/api/favorite/removeById")
                        .header("Authorization", "Bearer token")
                        .param("id", "123"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));
    }

    @Test
    @DisplayName("列出所有收藏 - 成功")
    void listAllFavorites_ShouldReturnList() throws Exception {
        when(favoriteService.getAllFavorites()).thenReturn(Collections.emptyList());

        mockMvc.perform(get("/api/favorite/listAll")
                        .header("Authorization", "Bearer token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data").isArray());
    }
}
