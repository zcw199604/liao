package com.zcw.service;

import com.zcw.model.Identity;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.Collections;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("身份服务测试")
class IdentityServiceTest {

    @Mock
    private JdbcTemplate jdbcTemplate;

    @InjectMocks
    private IdentityService identityService;

    @Test
    @DisplayName("获取所有身份 - 成功返回列表")
    void getAllIdentities_ShouldReturnList() {
        // Arrange
        Identity identity = new Identity("1", "TestUser", "男");
        when(jdbcTemplate.query(anyString(), any(RowMapper.class)))
                .thenReturn(Collections.singletonList(identity));

        // Act
        List<Identity> result = identityService.getAllIdentities();

        // Assert
        assertNotNull(result);
        assertEquals(1, result.size());
        assertEquals("TestUser", result.get(0).getName());
        verify(jdbcTemplate, times(1)).query(contains("SELECT * FROM identity"), any(RowMapper.class));
    }

    @Test
    @DisplayName("创建身份 - 成功保存并返回对象")
    void createIdentity_ShouldSaveAndReturnIdentity() {
        // Arrange
        String name = "NewUser";
        String sex = "女";
        
        // Act
        Identity result = identityService.createIdentity(name, sex);

        // Assert
        assertNotNull(result);
        assertNotNull(result.getId());
        assertEquals(name, result.getName());
        assertEquals(sex, result.getSex());
        assertNotNull(result.getCreatedAt());
        
        // Verify SQL execution
        verify(jdbcTemplate, times(1)).update(
                contains("INSERT INTO identity"),
                eq(result.getId()),
                eq(name),
                eq(sex),
                anyString(), // createdAt
                anyString()  // lastUsedAt
        );
    }

    @Test
    @DisplayName("更新身份 - ID存在时更新成功")
    void updateIdentity_ShouldUpdate_WhenIdExists() {
        // Arrange
        String id = "123";
        String newName = "UpdatedName";
        String newSex = "男";
        Identity existingIdentity = new Identity(id, "OldName", "女");
        
        // Mock getIdentityById (which calls jdbcTemplate.query)
        when(jdbcTemplate.query(contains("SELECT * FROM identity WHERE id"), any(RowMapper.class), eq(id)))
                .thenReturn(Collections.singletonList(existingIdentity));

        // Act
        Identity result = identityService.updateIdentity(id, newName, newSex);

        // Assert
        assertNotNull(result);
        assertEquals(newName, result.getName());
        assertEquals(newSex, result.getSex());
        
        verify(jdbcTemplate, times(1)).update(
                contains("UPDATE identity SET name = ?, sex = ?, last_used_at = ? WHERE id = ?"),
                eq(newName),
                eq(newSex),
                anyString(),
                eq(id)
        );
    }

    @Test
    @DisplayName("更新身份 - ID不存在时返回Null")
    void updateIdentity_ShouldReturnNull_WhenIdDoesNotExist() {
        // Arrange
        String id = "999";
        
        // Mock getIdentityById returning empty list
        when(jdbcTemplate.query(contains("SELECT * FROM identity WHERE id"), any(RowMapper.class), eq(id)))
                .thenReturn(Collections.emptyList());

        // Act
        Identity result = identityService.updateIdentity(id, "Name", "男");

        // Assert
        assertNull(result);
        // Ensure no update SQL is executed
        verify(jdbcTemplate, never()).update(startsWith("UPDATE identity"), any(), any(), any(), any());
    }

    @Test
    @DisplayName("删除身份 - 成功删除")
    void deleteIdentity_ShouldReturnTrue_WhenDeleted() {
        // Arrange
        String id = "123";
        when(jdbcTemplate.update(contains("DELETE FROM identity"), eq(id))).thenReturn(1);

        // Act
        boolean result = identityService.deleteIdentity(id);

        // Assert
        assertTrue(result);
        verify(jdbcTemplate, times(1)).update(contains("DELETE FROM identity"), eq(id));
    }

    @Test
    @DisplayName("删除身份 - 身份不存在时返回False")
    void deleteIdentity_ShouldReturnFalse_WhenNotFound() {
        // Arrange
        String id = "999";
        when(jdbcTemplate.update(contains("DELETE FROM identity"), eq(id))).thenReturn(0);

        // Act
        boolean result = identityService.deleteIdentity(id);

        // Assert
        assertFalse(result);
    }

    @Test
    @DisplayName("快速创建身份 - 成功生成随机信息")
    void quickCreateIdentity_ShouldGenerateRandomData() {
        // Act
        Identity result = identityService.quickCreateIdentity();

        // Assert
        assertNotNull(result);
        assertNotNull(result.getName());
        assertTrue(result.getSex().equals("男") || result.getSex().equals("女"));
        verify(jdbcTemplate, times(1)).update(startsWith("INSERT INTO identity"), any(), any(), any(), any(), any());
    }
}
