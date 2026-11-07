import request from './request'
import type { ApiResponse } from '@/types'

export const createWorkerPreviewSession = () => {
  return request.post<any, ApiResponse<{ url: string }>>('/workerPreview/session')
}

export const clearWorkerPreviewSession = (token: string) => {
  // Axios 的请求拦截器异步执行；退出后 localStorage 已清空，必须保留退出前的凭据。
  return request.delete<any, ApiResponse>('/workerPreview/session', {
    headers: { Authorization: `Bearer ${token}` }
  })
}
