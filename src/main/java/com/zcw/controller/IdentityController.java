package com.zcw.controller;

import com.zcw.model.Identity;
import com.zcw.service.IdentityService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * 身份管理API控制器
 * 提供身份的增删改查接口
 */
@RestController
@RequestMapping("/api")
public class IdentityController {

    private static final Logger logger = LoggerFactory.getLogger(IdentityController.class);

    @Autowired
    private IdentityService identityService;

    /**
     * 获取所有身份列表
     * GET /api/getIdentityList
     */
    @GetMapping("/getIdentityList")
    public ResponseEntity<Map<String, Object>> getIdentityList() {
        logger.info("获取身份列表");
        List<Identity> identities = identityService.getAllIdentities();

        Map<String, Object> response = new HashMap<>();
        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identities);

        return ResponseEntity.ok(response);
    }

    /**
     * 创建新身份
     * POST /api/createIdentity
     * @param name 名字
     * @param sex 性别
     */
    @PostMapping("/createIdentity")
    public ResponseEntity<Map<String, Object>> createIdentity(
            @RequestParam String name,
            @RequestParam String sex) {
        logger.info("创建新身份: name={}, sex={}", name, sex);

        Map<String, Object> response = new HashMap<>();

        // 参数校验
        if (name == null || name.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "名字不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (sex == null || (!sex.equals("男") && !sex.equals("女"))) {
            response.put("code", -1);
            response.put("msg", "性别必须是男或女");
            return ResponseEntity.badRequest().body(response);
        }

        Identity identity = identityService.createIdentity(name.trim(), sex);

        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identity);

        return ResponseEntity.ok(response);
    }

    /**
     * 快速创建随机身份
     * POST /api/quickCreateIdentity
     */
    @PostMapping("/quickCreateIdentity")
    public ResponseEntity<Map<String, Object>> quickCreateIdentity() {
        logger.info("快速创建随机身份");

        Identity identity = identityService.quickCreateIdentity();

        Map<String, Object> response = new HashMap<>();
        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identity);

        return ResponseEntity.ok(response);
    }

    /**
     * 更新身份信息
     * POST /api/updateIdentity
     * @param id 身份ID
     * @param name 新名字
     * @param sex 新性别
     */
    @PostMapping("/updateIdentity")
    public ResponseEntity<Map<String, Object>> updateIdentity(
            @RequestParam String id,
            @RequestParam String name,
            @RequestParam String sex) {
        logger.info("更新身份: id={}, name={}, sex={}", id, name, sex);

        Map<String, Object> response = new HashMap<>();

        // 参数校验
        if (id == null || id.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "身份ID不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (name == null || name.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "名字不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (sex == null || (!sex.equals("男") && !sex.equals("女"))) {
            response.put("code", -1);
            response.put("msg", "性别必须是男或女");
            return ResponseEntity.badRequest().body(response);
        }

        Identity identity = identityService.updateIdentity(id.trim(), name.trim(), sex);

        if (identity == null) {
            response.put("code", -1);
            response.put("msg", "身份不存在");
            return ResponseEntity.badRequest().body(response);
        }

        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identity);

        return ResponseEntity.ok(response);
    }

    /**
     * 更新身份ID
     * POST /api/updateIdentityId
     * @param oldId 旧身份ID
     * @param newId 新身份ID
     * @param name 名字
     * @param sex 性别
     */
    @PostMapping("/updateIdentityId")
    public ResponseEntity<Map<String, Object>> updateIdentityId(
            @RequestParam String oldId,
            @RequestParam String newId,
            @RequestParam String name,
            @RequestParam String sex) {
        logger.info("更新身份ID: oldId={} -> newId={}, name={}, sex={}", oldId, newId, name, sex);

        Map<String, Object> response = new HashMap<>();

        // 参数校验
        if (oldId == null || oldId.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "旧身份ID不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (newId == null || newId.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "新身份ID不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (name == null || name.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "名字不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        if (sex == null || (!sex.equals("男") && !sex.equals("女"))) {
            response.put("code", -1);
            response.put("msg", "性别必须是男或女");
            return ResponseEntity.badRequest().body(response);
        }

        Identity identity = identityService.updateIdentityId(oldId.trim(), newId.trim(), name.trim(), sex);

        if (identity == null) {
            response.put("code", -1);
            response.put("msg", "更新失败，可能旧身份不存在或新ID已被使用");
            return ResponseEntity.badRequest().body(response);
        }

        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identity);

        return ResponseEntity.ok(response);
    }

    /**
     * 删除身份
     * POST /api/deleteIdentity
     * @param id 身份ID
     */
    @PostMapping("/deleteIdentity")
    public ResponseEntity<Map<String, Object>> deleteIdentity(@RequestParam String id) {
        logger.info("删除身份: id={}", id);

        Map<String, Object> response = new HashMap<>();

        if (id == null || id.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "身份ID不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        boolean success = identityService.deleteIdentity(id.trim());

        if (!success) {
            response.put("code", -1);
            response.put("msg", "身份不存在");
            return ResponseEntity.badRequest().body(response);
        }

        response.put("code", 0);
        response.put("msg", "success");

        return ResponseEntity.ok(response);
    }

    /**
     * 选择身份（更新最后使用时间）
     * POST /api/selectIdentity
     * @param id 身份ID
     */
    @PostMapping("/selectIdentity")
    public ResponseEntity<Map<String, Object>> selectIdentity(@RequestParam String id) {
        logger.info("选择身份: id={}", id);

        Map<String, Object> response = new HashMap<>();

        if (id == null || id.trim().isEmpty()) {
            response.put("code", -1);
            response.put("msg", "身份ID不能为空");
            return ResponseEntity.badRequest().body(response);
        }

        Identity identity = identityService.getIdentityById(id.trim());
        if (identity == null) {
            response.put("code", -1);
            response.put("msg", "身份不存在");
            return ResponseEntity.badRequest().body(response);
        }

        // 更新最后使用时间
        identityService.updateLastUsedAt(id.trim());

        response.put("code", 0);
        response.put("msg", "success");
        response.put("data", identity);

        return ResponseEntity.ok(response);
    }
}
