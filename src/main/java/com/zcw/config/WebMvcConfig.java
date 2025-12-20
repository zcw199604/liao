package com.zcw.config;

import com.zcw.interceptor.JwtInterceptor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.CorsRegistry;
import org.springframework.web.servlet.config.annotation.InterceptorRegistry;
import org.springframework.web.servlet.config.annotation.ResourceHandlerRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

/**
 * Web MVC 配置类
 * 配置静态资源访问和跨域
 */
@Configuration
public class WebMvcConfig implements WebMvcConfigurer {

    @Autowired
    private JwtInterceptor jwtInterceptor;

    /**
     * 配置静态资源访问路径
     */
    @Override
    public void addResourceHandlers(ResourceHandlerRegistry registry) {
        // 配置 /static/** 路径访问静态资源
        registry.addResourceHandler("/static/**")
                .addResourceLocations("classpath:/static/");

        // 配置 /public/** 路径访问公共资源
        registry.addResourceHandler("/public/**")
                .addResourceLocations("classpath:/public/");

        // 配置 /resources/** 路径访问资源文件
        registry.addResourceHandler("/resources/**")
                .addResourceLocations("classpath:/resources/");

        // 配置 /upload/** 路径访问上传文件（如果有文件上传功能）
        registry.addResourceHandler("/upload/**")
                .addResourceLocations("file:./upload/");
    }

    /**
     * 配置跨域访问
     */
    @Override
    public void addCorsMappings(CorsRegistry registry) {
        registry.addMapping("/**")
                .allowedOriginPatterns("*")
                .allowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS")
                .allowedHeaders("*")
                .allowCredentials(true)
                .maxAge(3600);
    }

    /**
     * 配置拦截器
     */
    @Override
    public void addInterceptors(InterceptorRegistry registry) {
        registry.addInterceptor(jwtInterceptor)
                .addPathPatterns("/api/**")                    // 拦截所有API请求
                .excludePathPatterns(
                        "/api/auth/login",                     // 放行登录接口
                        "/api/auth/verify"                     // 放行验证接口(可选)
                );
    }
}
