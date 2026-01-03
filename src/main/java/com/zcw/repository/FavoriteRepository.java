package com.zcw.repository;

import com.zcw.model.Favorite;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface FavoriteRepository extends JpaRepository<Favorite, Long> {
    
    /**
     * 查询某个身份的所有收藏
     */
    List<Favorite> findByIdentityIdOrderByCreateTimeDesc(String identityId);
    
    /**
     * 查询所有收藏（用于全局设置显示）
     */
    List<Favorite> findAllByOrderByCreateTimeDesc();

    /**
     * 检查是否已收藏
     */
    Optional<Favorite> findByIdentityIdAndTargetUserId(String identityId, String targetUserId);

    /**
     * 删除收藏
     */
    void deleteByIdentityIdAndTargetUserId(String identityId, String targetUserId);
}
