# System Config

## 目的
提供全局配置、图片端口解析和系统连接管理能力。

## 模块概述
- **职责:** 系统配置 Key-Value、图片端口策略、连接统计、断开连接、forceout 状态清理。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: 全局配置
**模块:** System Config  
配置存储在 `system_config`，由服务启动时确保默认值。

#### 场景: 前端加载配置
- `getSystemConfig` 返回全部配置。
- `updateSystemConfig` 更新后返回最新配置。

### 需求: 图片端口解析
**模块:** System Config  
图片端口可通过固定、探测或真实请求策略解析，供媒体 URL 拼接使用。

#### 场景: 前端展示媒体
- 前端缓存解析结果，后端以当前配置和路径判断端口。

### 需求: 连接管理
**模块:** System Config  
系统面板可查询当前 WS 连接数、断开全部连接、查询和清空 forceout 列表。

## API接口
- `GET /api/getSystemConfig`
- `POST /api/updateSystemConfig`
- `POST /api/resolveImagePort`
- `GET /api/getConnectionStats`
- `POST /api/disconnectAllConnections`
- `GET /api/getForceoutUserCount`
- `POST /api/clearForceoutUsers`

## 数据模型
- `system_config`
- 运行时 `UpstreamWebSocketManager`
- 运行时 `ForceoutManager`

## 依赖
- `internal/app/system_config.go`
- `internal/app/system_config_handlers.go`
- `internal/app/system_handlers.go`
- `internal/app/image_port_resolver.go`
- `frontend/src/api/system.ts`
- `frontend/src/stores/systemConfig.ts`
