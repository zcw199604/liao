package com.zcw.repository;

import com.zcw.model.MediaFile;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.Optional;
import java.util.List;

@Repository
public interface MediaFileRepository extends JpaRepository<MediaFile, Long> {

    Optional<MediaFile> findFirstByUserIdAndFileMd5(String userId, String fileMd5);

    Optional<MediaFile> findFirstByLocalFilenameAndUserId(String localFilename, String userId);
    Optional<MediaFile> findFirstByLocalFilename(String localFilename);

    Optional<MediaFile> findFirstByRemoteFilenameAndUserId(String remoteFilename, String userId);
    Optional<MediaFile> findFirstByRemoteFilename(String remoteFilename);

    Optional<MediaFile> findFirstByRemoteUrlAndUserId(String remoteUrl, String userId);
    Optional<MediaFile> findFirstByRemoteUrl(String remoteUrl);
    
    // 通过 local_path 查找（这是最准确的）
    Optional<MediaFile> findFirstByLocalPathAndUserId(String localPath, String userId);
    Optional<MediaFile> findFirstByLocalPath(String localPath);

    // 查询用户的所有文件，按更新时间倒序
    Page<MediaFile> findByUserIdOrderByUpdateTimeDesc(String userId, Pageable pageable);

    // 查询所有文件，按更新时间倒序（不区分用户）
    Page<MediaFile> findAllByOrderByUpdateTimeDesc(Pageable pageable);
    
    // 获取所有文件（用于全局列表，如果有去重需求，还是建议按 MD5 分组）
    // 但 MediaFile 表设计上应该是去重的（每个用户每个MD5一条），所以直接查即可
    @Query(value = """
        SELECT * FROM media_file 
        WHERE user_id = ?1
        ORDER BY update_time DESC
        """, countQuery = "SELECT count(*) FROM media_file WHERE user_id = ?1", nativeQuery = true)
    Page<MediaFile> findAllUserFiles(String userId, Pageable pageable);

    // 统计 MD5 引用计数（用于判断物理删除）
    long countByFileMd5(String fileMd5);

    // 统计用户文件总数
    int countByUserId(String userId);
    
    // 删除
    @Modifying
    @Query("DELETE FROM MediaFile m WHERE m.userId = ?1 AND m.localPath = ?2")
    int deleteByUserIdAndLocalPath(String userId, String localPath);
    
    @Modifying
    @Query("UPDATE MediaFile m SET m.updateTime = ?3 WHERE m.localPath = ?1 AND m.userId = ?2")
    int updateTimeByLocalPath(String localPath, String userId, LocalDateTime updateTime);

    @Modifying
    @Query("UPDATE MediaFile m SET m.updateTime = ?2 WHERE m.localPath = ?1")
    int updateTimeByLocalPathIgnoreUser(String localPath, LocalDateTime updateTime);
}
