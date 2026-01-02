package com.zcw.repository;

import com.zcw.model.MediaUploadHistory;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface MediaUploadHistoryRepository extends JpaRepository<MediaUploadHistory, Long> {

    // 通过本地文件名查找（原始上传记录）
    Optional<MediaUploadHistory> findFirstByLocalFilenameAndUserIdAndToUserIdIsNull(String localFilename, String userId);

    // 通过远程文件名查找（原始上传记录）
    Optional<MediaUploadHistory> findFirstByRemoteFilenameAndUserIdAndToUserIdIsNull(String remoteFilename, String userId);

    // 通过远程URL查找（原始上传记录）
    Optional<MediaUploadHistory> findFirstByRemoteUrlAndUserIdAndToUserIdIsNull(String remoteUrl, String userId);

    // 查找已存在的发送记录
    Optional<MediaUploadHistory> findFirstByRemoteUrlAndUserIdAndToUserId(String remoteUrl, String userId, String toUserId);
    Optional<MediaUploadHistory> findFirstByRemoteFilenameAndUserIdAndToUserId(String remoteFilename, String userId, String toUserId);

    // 查找MD5记录（原始上传记录）
    Optional<MediaUploadHistory> findFirstByUserIdAndFileMd5AndToUserIdIsNull(String userId, String fileMd5);

    // 查询用户上传的所有图片（原始上传记录，按更新时间排序）
    Page<MediaUploadHistory> findByUserIdAndToUserIdIsNullOrderByUploadTimeDesc(String userId, Pageable pageable);

    // 查询用户发送给特定用户的图片
    Page<MediaUploadHistory> findByUserIdAndToUserIdOrderBySendTimeDesc(String userId, String toUserId, Pageable pageable);

    // 统计用户上传总数
    int countByUserIdAndToUserIdIsNull(String userId);

    // 统计发送给特定用户的图片数
    int countByUserIdAndToUserId(String userId, String toUserId);

    // 获取聊天图片（去重逻辑比较复杂，可能需要自定义查询）
    // 原SQL:
    // SELECT local_path FROM media_upload_history
    // WHERE ((user_id = ? AND to_user_id = ?) OR (user_id = ? AND to_user_id = ?))
    // AND send_time IS NOT NULL
    // GROUP BY local_path
    // ORDER BY MAX(send_time) DESC
    // LIMIT ?
    @Query(value = """
        SELECT local_path
        FROM media_upload_history
        WHERE ((user_id = ?1 AND to_user_id = ?2) OR (user_id = ?2 AND to_user_id = ?1))
          AND send_time IS NOT NULL
        GROUP BY local_path
        ORDER BY MAX(send_time) DESC
        LIMIT ?3
        """, nativeQuery = true)
    List<String> findChatImagePaths(String userId1, String userId2, int limit);

    // 根据 local_path 获取原始文件名
    @Query(value = "SELECT original_filename FROM media_upload_history WHERE local_path = ?1 LIMIT 1", nativeQuery = true)
    String findOriginalFilenameByLocalPath(String localPath);

    // 获取所有上传图片（去重 MD5）
    // 原SQL:
    // SELECT local_path FROM media_upload_history
    // WHERE file_md5 IS NOT NULL
    // AND id IN (SELECT MAX(id) FROM media_upload_history WHERE file_md5 IS NOT NULL GROUP BY file_md5)
    // ORDER BY update_time DESC LIMIT ? OFFSET ?
    @Query(value = """
        SELECT local_path
        FROM media_upload_history
        WHERE file_md5 IS NOT NULL
          AND id IN (
              SELECT MAX(id)
              FROM media_upload_history
              WHERE file_md5 IS NOT NULL
              GROUP BY file_md5
          )
        ORDER BY update_time DESC, upload_time DESC
        """, countQuery = "SELECT COUNT(DISTINCT file_md5) FROM media_upload_history WHERE file_md5 IS NOT NULL", nativeQuery = true)
    Page<String> findAllUploadImagePaths(Pageable pageable);

    // 统计MD5去重后的总数
    @Query(value = "SELECT COUNT(DISTINCT file_md5) FROM media_upload_history WHERE file_md5 IS NOT NULL", nativeQuery = true)
    int countDistinctByFileMd5IsNotNull();

    // 查找文件路径对应的所有记录
    List<MediaUploadHistory> findByLocalPathAndUserId(String localPath, String userId);

    // 删除用户的文件记录
    @Modifying
    @Query("DELETE FROM MediaUploadHistory m WHERE m.userId = ?1 AND m.localPath = ?2")
    int deleteByUserIdAndLocalPath(String userId, String localPath);

    // 检查MD5是否还有其他引用
    long countByFileMd5(String fileMd5);

    // 更新文件时间
    @Modifying
    @Query("UPDATE MediaUploadHistory m SET m.updateTimeRaw = CURRENT_TIMESTAMP WHERE m.localPath = ?1 AND m.userId = ?2 AND m.toUserId IS NULL")
    int updateTimeByLocalPath(String localPath, String userId);
}
