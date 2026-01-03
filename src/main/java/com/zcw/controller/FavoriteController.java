package com.zcw.controller;

import com.zcw.model.Favorite;
import com.zcw.service.FavoriteService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/favorite")
public class FavoriteController {

    private static final Logger logger = LoggerFactory.getLogger(FavoriteController.class);

    @Autowired
    private FavoriteService favoriteService;

    /**
     * 添加收藏
     */
    @PostMapping("/add")
    public ResponseEntity<Map<String, Object>> addFavorite(
            @RequestParam String identityId,
            @RequestParam String targetUserId,
            @RequestParam(required = false) String targetUserName) {
        
        logger.info("添加收藏: identityId={}, targetUserId={}", identityId, targetUserId);
        Favorite favorite = favoriteService.addFavorite(identityId, targetUserId, targetUserName);
        
        return success(favorite);
    }

    /**
     * 移除收藏
     */
    @PostMapping("/remove")
    public ResponseEntity<Map<String, Object>> removeFavorite(
            @RequestParam String identityId,
            @RequestParam String targetUserId) {
        
        logger.info("移除收藏: identityId={}, targetUserId={}", identityId, targetUserId);
        favoriteService.removeFavorite(identityId, targetUserId);
        
        return success(null);
    }

    /**
     * 根据ID移除收藏
     */
    @PostMapping("/removeById")
    public ResponseEntity<Map<String, Object>> removeFavoriteById(@RequestParam Long id) {
        logger.info("移除收藏ID: {}", id);
        favoriteService.removeFavoriteById(id);
        return success(null);
    }

    /**
     * 获取所有收藏（全局）
     */
    @GetMapping("/listAll")
    public ResponseEntity<Map<String, Object>> listAllFavorites() {
        List<Favorite> favorites = favoriteService.getAllFavorites();
        return success(favorites);
    }

    /**
     * 检查是否收藏
     */
    @GetMapping("/check")
    public ResponseEntity<Map<String, Object>> checkFavorite(
            @RequestParam String identityId,
            @RequestParam String targetUserId) {
        boolean isFav = favoriteService.isFavorite(identityId, targetUserId);
        Map<String, Object> data = new HashMap<>();
        data.put("isFavorite", isFav);
        return success(data);
    }

    private ResponseEntity<Map<String, Object>> success(Object data) {
        Map<String, Object> response = new HashMap<>();
        response.put("code", 0);
        response.put("msg", "success");
        if (data != null) {
            response.put("data", data);
        }
        return ResponseEntity.ok(response);
    }
}
