import request from './request'
import type { ApiResponse, ConnectionStats, SystemConfig } from '@/types'

// 获取连接统计
export const getConnectionStats = () => {
  return request.get<any, ApiResponse<ConnectionStats>>('/getConnectionStats')
}

// 断开所有连接
export const disconnectAllConnections = () => {
  return request.post<any, ApiResponse>('/disconnectAllConnections')
}

// 获取被禁止用户数量
export const getForceoutUserCount = () => {
  return request.get<any, ApiResponse<number>>('/getForceoutUserCount')
}

// 清除所有被禁止的用户
export const clearForceoutUsers = () => {
  return request.post<any, ApiResponse>('/clearForceoutUsers')
}

// 获取系统全局配置（所有用户共用）
export const getSystemConfig = () => {
  return request.get<any, ApiResponse<SystemConfig>>('/getSystemConfig')
}

// 更新系统全局配置（JSON）
export const updateSystemConfig = (data: Partial<SystemConfig>) => {
  return request.post<any, ApiResponse<SystemConfig>>('/updateSystemConfig', data)
}

// 解析图片端口（按后端配置策略）
export const resolveImagePort = (path: string) => {
  return request.post<any, ApiResponse<{ port: string }>>('/resolveImagePort', { path })
}
