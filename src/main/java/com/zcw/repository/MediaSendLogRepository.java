package com.zcw.repository;

import com.zcw.model.MediaSendLog;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface MediaSendLogRepository extends JpaRepository<MediaSendLog, Long> {

    // 查找已存在的发送记录（去重用）
    Optional<MediaSendLog> findFirstByRemoteUrlAndUserIdAndToUserId(String remoteUrl, String userId, String toUserId);

    // 查询发送记录（分页）
    Page<MediaSendLog> findByUserIdAndToUserIdOrderBySendTimeDesc(String userId, String toUserId, Pageable pageable);

    // 统计发送数
    int countByUserIdAndToUserId(String userId, String toUserId);

    // 获取聊天图片 (双向)
    // 注意：这里返回的是 local_path，用于去 MediaFile 表查详情或直接拼接 URL
    @Query(value = """
        SELECT local_path
        FROM media_send_log
        WHERE ((user_id = ?1 AND to_user_id = ?2) OR (user_id = ?2 AND to_user_id = ?1))
        GROUP BY local_path
        ORDER BY MAX(send_time) DESC
        LIMIT ?3
        """, nativeQuery = true)
    List<String> findChatImagePaths(String userId1, String userId2, int limit);
    
    // 删除关联记录
    @Modifying
    @Query("DELETE FROM MediaSendLog m WHERE m.userId = ?1 AND m.localPath = ?2")
    int deleteByUserIdAndLocalPath(String userId, String localPath);
}
