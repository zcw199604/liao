package com.zcw.service;

import com.zcw.model.Favorite;
import com.zcw.repository.FavoriteRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDateTime;
import java.util.Collections;
import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("收藏服务测试")
class FavoriteServiceTest {

    @Mock
    private FavoriteRepository favoriteRepository;

    @InjectMocks
    private FavoriteService favoriteService;

    @Test
    @DisplayName("添加收藏 - 记录不存在时保存")
    void addFavorite_ShouldSave_WhenNotExists() {
        // Arrange
        String identityId = "id1";
        String targetUserId = "target1";
        String targetUserName = "TargetUser";
        
        when(favoriteRepository.findByIdentityIdAndTargetUserId(identityId, targetUserId))
                .thenReturn(Optional.empty());
        
        when(favoriteRepository.save(any(Favorite.class))).thenAnswer(invocation -> {
            Favorite f = invocation.getArgument(0);
            f.setId(1L);
            f.setCreateTime(LocalDateTime.now());
            return f;
        });

        // Act
        Favorite result = favoriteService.addFavorite(identityId, targetUserId, targetUserName);

        // Assert
        assertNotNull(result);
        assertEquals(identityId, result.getIdentityId());
        assertEquals(targetUserId, result.getTargetUserId());
        assertEquals(targetUserName, result.getTargetUserName());
        verify(favoriteRepository, times(1)).save(any(Favorite.class));
    }

    @Test
    @DisplayName("添加收藏 - 记录已存在时直接返回")
    void addFavorite_ShouldReturnExisting_WhenExists() {
        // Arrange
        String identityId = "id1";
        String targetUserId = "target1";
        Favorite existing = new Favorite(1L, identityId, targetUserId, "Name", LocalDateTime.now());
        
        when(favoriteRepository.findByIdentityIdAndTargetUserId(identityId, targetUserId))
                .thenReturn(Optional.of(existing));

        // Act
        Favorite result = favoriteService.addFavorite(identityId, targetUserId, "NewName");

        // Assert
        assertEquals(existing, result);
        verify(favoriteRepository, never()).save(any(Favorite.class));
    }

    @Test
    @DisplayName("移除收藏 - 调用仓库删除方法")
    void removeFavorite_ShouldCallDelete() {
        // Arrange
        String identityId = "id1";
        String targetUserId = "target1";

        // Act
        favoriteService.removeFavorite(identityId, targetUserId);

        // Assert
        verify(favoriteRepository, times(1)).deleteByIdentityIdAndTargetUserId(identityId, targetUserId);
    }

    @Test
    @DisplayName("检查收藏状态 - 返回True")
    void isFavorite_ShouldReturnTrue_WhenExists() {
        // Arrange
        when(favoriteRepository.findByIdentityIdAndTargetUserId("id1", "target1"))
                .thenReturn(Optional.of(new Favorite()));

        // Act
        boolean result = favoriteService.isFavorite("id1", "target1");

        // Assert
        assertTrue(result);
    }

    @Test
    @DisplayName("检查收藏状态 - 返回False")
    void isFavorite_ShouldReturnFalse_WhenNotExists() {
        // Arrange
        when(favoriteRepository.findByIdentityIdAndTargetUserId("id1", "target1"))
                .thenReturn(Optional.empty());

        // Act
        boolean result = favoriteService.isFavorite("id1", "target1");

        // Assert
        assertFalse(result);
    }
}
