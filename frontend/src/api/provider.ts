import request from './request'
import type { ApiResponse } from '@/types'

// 上游线路信息（与后端 ProviderInfo 对应）
export interface ProviderInfo {
  id: string
  name: string
  description: string
  default: boolean
}

// 获取可用线路列表
export const getProviderList = () => {
  return request.get<any, ApiResponse<{ providers: ProviderInfo[] }>>('/provider/list')
}
