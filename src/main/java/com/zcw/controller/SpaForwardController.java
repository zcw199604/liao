package com.zcw.controller;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

/**
 * 前端 SPA 路由回退控制器
 * 解决直接访问 /chat/... 等前端路由时服务端 404 的问题。
 */
@Controller
public class SpaForwardController {

    @GetMapping({"/", "/login", "/identity", "/list", "/chat", "/chat/**"})
    public String forwardToIndex() {
        return "forward:/index.html";
    }
}

