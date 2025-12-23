import request, { createFormData } from './request'
import type { ApiResponse } from '@/types'

// 登录（使用urlencoded格式）
export const login = (accessCode: string) => {
  const formData = createFormData({ accessCode })
  return request.post<any, ApiResponse>('/auth/login', formData, {
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    }
  })
}

// 验证Token
export const verifyToken = () => {
  return request.get<any, ApiResponse>('/auth/verify')
}
