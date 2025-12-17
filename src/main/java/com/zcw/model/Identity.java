package com.zcw.model;

/**
 * 用户身份数据模型
 * 用于存储公共身份池中的身份信息
 */
public class Identity {

    /**
     * 用户ID（32位随机字符串）
     */
    private String id;

    /**
     * 用户名字
     */
    private String name;

    /**
     * 性别（男/女）
     */
    private String sex;

    /**
     * 创建时间
     */
    private String createdAt;

    /**
     * 最后使用时间
     */
    private String lastUsedAt;

    public Identity() {
    }

    public Identity(String id, String name, String sex) {
        this.id = id;
        this.name = name;
        this.sex = sex;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getSex() {
        return sex;
    }

    public void setSex(String sex) {
        this.sex = sex;
    }

    public String getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(String createdAt) {
        this.createdAt = createdAt;
    }

    public String getLastUsedAt() {
        return lastUsedAt;
    }

    public void setLastUsedAt(String lastUsedAt) {
        this.lastUsedAt = lastUsedAt;
    }
}
