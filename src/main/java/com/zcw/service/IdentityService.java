package com.zcw.service;

import com.zcw.model.Identity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Service;

import jakarta.annotation.PostConstruct;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.Random;
import java.util.UUID;

/**
 * 身份管理服务
 * 使用MySQL数据库存储身份数据
 */
@Service
public class IdentityService {

    private static final Logger logger = LoggerFactory.getLogger(IdentityService.class);

    /**
     * 日期时间格式化器
     */
    private static final DateTimeFormatter DATE_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss");

    /**
     * 随机名字池
     */
    private static final String[] RANDOM_NAMES = {
        "小明", "小红", "小华", "小美", "小强", "小芳", "小军", "小丽",
        "阿杰", "阿伟", "阿珍", "阿莲", "阿辉", "阿英", "阿龙", "阿凤",
        "大熊", "小鹿", "白兔", "黑猫", "金鱼", "银狐", "青蛙", "蝴蝶",
        "星辰", "月光", "阳光", "彩虹", "流云", "清风", "细雨", "白雪",
        "晨曦", "暮色", "春风", "夏雨", "秋叶", "冬雪", "花开", "叶落",
        "海浪", "山峰", "森林", "草原", "沙漠", "冰川", "火焰", "闪电",
        "孤独", "寂寞", "快乐", "忧伤", "温柔", "坚强", "勇敢", "善良",
        "路人甲", "过客乙", "行者丙", "旅人丁", "浪子", "游侠", "隐士", "剑客"
    };

    @Autowired
    private JdbcTemplate jdbcTemplate;

    /**
     * Identity行映射器
     */
    private final RowMapper<Identity> identityRowMapper = new RowMapper<Identity>() {
        @Override
        public Identity mapRow(ResultSet rs, int rowNum) throws SQLException {
            Identity identity = new Identity();
            identity.setId(rs.getString("id"));
            identity.setName(rs.getString("name"));
            identity.setSex(rs.getString("sex"));

            java.sql.Timestamp createdAt = rs.getTimestamp("created_at");
            if (createdAt != null) {
                identity.setCreatedAt(createdAt.toLocalDateTime().format(DATE_FORMATTER));
            }

            java.sql.Timestamp lastUsedAt = rs.getTimestamp("last_used_at");
            if (lastUsedAt != null) {
                identity.setLastUsedAt(lastUsedAt.toLocalDateTime().format(DATE_FORMATTER));
            }

            return identity;
        }
    };

    /**
     * 初始化服务，确保表存在
     */
    @PostConstruct
    public void init() {
        createTableIfNotExists();
        logger.info("身份服务初始化完成（MySQL存储）");
    }

    /**
     * 如果表不存在则创建
     */
    private void createTableIfNotExists() {
        String sql = """
            CREATE TABLE IF NOT EXISTS identity (
                id VARCHAR(32) PRIMARY KEY COMMENT '用户ID',
                name VARCHAR(50) NOT NULL COMMENT '名字',
                sex VARCHAR(10) NOT NULL COMMENT '性别',
                created_at DATETIME COMMENT '创建时间',
                last_used_at DATETIME COMMENT '最后使用时间',
                INDEX idx_last_used_at (last_used_at DESC)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户身份表'
            """;
        try {
            jdbcTemplate.execute(sql);
            logger.info("identity表检查/创建完成");
        } catch (Exception e) {
            logger.error("创建identity表失败", e);
        }
    }

    /**
     * 获取所有身份列表（按最后使用时间倒序）
     */
    public List<Identity> getAllIdentities() {
        String sql = "SELECT * FROM identity ORDER BY last_used_at DESC";
        return jdbcTemplate.query(sql, identityRowMapper);
    }

    /**
     * 根据ID获取身份
     */
    public Identity getIdentityById(String id) {
        String sql = "SELECT * FROM identity WHERE id = ?";
        List<Identity> results = jdbcTemplate.query(sql, identityRowMapper, id);
        return results.isEmpty() ? null : results.get(0);
    }

    /**
     * 创建新身份
     */
    public Identity createIdentity(String name, String sex) {
        String id = generateUniqueId();
        String now = LocalDateTime.now().format(DATE_FORMATTER);

        String sql = "INSERT INTO identity (id, name, sex, created_at, last_used_at) VALUES (?, ?, ?, ?, ?)";
        jdbcTemplate.update(sql, id, name, sex, now, now);

        logger.info("创建新身份: id={}, name={}, sex={}", id, name, sex);

        Identity identity = new Identity(id, name, sex);
        identity.setCreatedAt(now);
        identity.setLastUsedAt(now);
        return identity;
    }

    /**
     * 快速创建随机身份
     */
    public Identity quickCreateIdentity() {
        String name = generateRandomName();
        String sex = new Random().nextBoolean() ? "男" : "女";
        return createIdentity(name, sex);
    }

    /**
     * 更新身份信息
     */
    public Identity updateIdentity(String id, String name, String sex) {
        // 先检查是否存在
        Identity existing = getIdentityById(id);
        if (existing == null) {
            logger.warn("更新身份失败: 身份不存在, id={}", id);
            return null;
        }

        String now = LocalDateTime.now().format(DATE_FORMATTER);
        String sql = "UPDATE identity SET name = ?, sex = ?, last_used_at = ? WHERE id = ?";
        jdbcTemplate.update(sql, name, sex, now, id);

        logger.info("更新身份: id={}, name={}, sex={}", id, name, sex);

        existing.setName(name);
        existing.setSex(sex);
        existing.setLastUsedAt(now);
        return existing;
    }

    /**
     * 更新身份使用时间
     */
    public void updateLastUsedAt(String id) {
        String now = LocalDateTime.now().format(DATE_FORMATTER);
        String sql = "UPDATE identity SET last_used_at = ? WHERE id = ?";
        int updated = jdbcTemplate.update(sql, now, id);
        if (updated > 0) {
            logger.debug("更新身份使用时间: id={}", id);
        }
    }

    /**
     * 删除身份
     */
    public boolean deleteIdentity(String id) {
        String sql = "DELETE FROM identity WHERE id = ?";
        int deleted = jdbcTemplate.update(sql, id);

        if (deleted > 0) {
            logger.info("删除身份: id={}", id);
            return true;
        }

        logger.warn("删除身份失败: 身份不存在, id={}", id);
        return false;
    }

    /**
     * 生成唯一ID（32位十六进制字符串）
     */
    private String generateUniqueId() {
        return UUID.randomUUID().toString().replace("-", "");
    }

    /**
     * 生成随机名字
     */
    private String generateRandomName() {
        Random random = new Random();
        return RANDOM_NAMES[random.nextInt(RANDOM_NAMES.length)];
    }
}
