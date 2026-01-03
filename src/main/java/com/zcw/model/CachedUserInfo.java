package com.zcw.model;

import lombok.Data;
import java.io.Serializable;

/**
 * 缓存的用户信息
 */
@Data
public class CachedUserInfo implements Serializable {
    private String userId;
    private String nickname;
    private String gender;
    private String age;
    private String address;
    private Long updateTime;

    public CachedUserInfo() {}

    public CachedUserInfo(String userId, String nickname, String gender, String age, String address) {
        this.userId = userId;
        this.nickname = nickname;
        this.gender = gender;
        this.age = age;
        this.address = address;
        this.updateTime = System.currentTimeMillis();
    }
}
