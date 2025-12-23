import request from './request'
import type { ApiResponse, ConnectionStats } from '@/types'

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
