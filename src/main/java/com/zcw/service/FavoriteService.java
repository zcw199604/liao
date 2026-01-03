package com.zcw.service;

import com.zcw.model.Favorite;
import com.zcw.repository.FavoriteRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Optional;

@Service
public class FavoriteService {

    @Autowired
    private FavoriteRepository favoriteRepository;

    /**
     * 添加收藏
     * @param identityId 本地身份ID
     * @param targetUserId 目标用户ID
     * @param targetUserName 目标用户名
     * @return 保存的收藏实体
     */
    @Transactional
    public Favorite addFavorite(String identityId, String targetUserId, String targetUserName) {
        // 查重
        Optional<Favorite> existing = favoriteRepository.findByIdentityIdAndTargetUserId(identityId, targetUserId);
        if (existing.isPresent()) {
            return existing.get();
        }

        Favorite favorite = new Favorite();
        favorite.setIdentityId(identityId);
        favorite.setTargetUserId(targetUserId);
        favorite.setTargetUserName(targetUserName);
        return favoriteRepository.save(favorite);
    }

    /**
     * 删除收藏
     * @param identityId 本地身份ID
     * @param targetUserId 目标用户ID
     */
    @Transactional
    public void removeFavorite(String identityId, String targetUserId) {
        favoriteRepository.deleteByIdentityIdAndTargetUserId(identityId, targetUserId);
    }

    /**
     * 删除指定ID的收藏
     * @param id 收藏记录ID
     */
    @Transactional
    public void removeFavoriteById(Long id) {
        favoriteRepository.deleteById(id);
    }

    /**
     * 获取所有收藏（全局）
     */
    public List<Favorite> getAllFavorites() {
        return favoriteRepository.findAllByOrderByCreateTimeDesc();
    }

    /**
     * 获取指定身份的收藏
     */
    public List<Favorite> getFavoritesByIdentity(String identityId) {
        return favoriteRepository.findByIdentityIdOrderByCreateTimeDesc(identityId);
    }

    /**
     * 检查是否已收藏
     */
    public boolean isFavorite(String identityId, String targetUserId) {
        return favoriteRepository.findByIdentityIdAndTargetUserId(identityId, targetUserId).isPresent();
    }
}
