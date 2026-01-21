# 匿名匹配聊天前端

基于 Vue 3 + Vite + TypeScript + Pinia 构建的现代化前端项目。

## 技术栈

- **框架**: Vue 3 (Composition API + `<script setup>`)
- **构建工具**: Vite 7.3
- **语言**: TypeScript
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **HTTP客户端**: Axios
- **样式**: Tailwind CSS 4.0
- **工具库**: VueUse
- **图标**: FontAwesome 5

## 项目结构

```
src/
├── types/          # TypeScript类型定义
├── utils/          # 工具函数
├── constants/      # 常量配置
├── api/            # API请求封装
├── stores/         # Pinia状态管理
├── composables/    # 可组合函数（业务逻辑）
├── components/     # UI组件
│   ├── common/    # 通用组件
│   ├── chat/      # 聊天相关组件
│   ├── identity/  # 身份相关组件
│   ├── list/      # 列表相关组件
│   ├── media/     # 媒体相关组件
│   └── settings/  # 设置相关组件
└── views/          # 路由页面
```

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器（需要后端运行在8080端口）
npm run dev

# 访问 http://localhost:3000
```

## 构建

```bash
# 构建生产版本
npm run build

# 输出到 ../src/main/resources/static/
# Go 后端可直接托管
```

## 核心功能

- ✅ 访问码登录认证
- ✅ 多身份管理（创建、删除、切换）
- ✅ WebSocket实时通信
- ✅ 聊天消息（文本、图片、视频）
- ✅ 表情支持
- ✅ 随机匹配
- ✅ 用户收藏
- ✅ 媒体上传和管理
- ✅ 系统设置
- ✅ Forceout检测和防重连

## 从旧版本迁移

详见 [MIGRATION.md](./MIGRATION.md)

## License

MIT
