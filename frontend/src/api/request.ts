import axios, { AxiosError } from 'axios'
import { API_BASE } from '@/constants/config'
import type { ApiResponse } from '@/types'

const request = axios.create({
  baseURL: API_BASE,
  timeout: 10000
})

// 请求拦截器：添加Token
request.interceptors.request.use(
  config => {
    const token = localStorage.getItem('authToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => Promise.reject(error)
)

// 响应拦截器：统一处理错误
request.interceptors.response.use(
  response => {
    const data = response.data
    if (typeof data === 'string') {
      const trimmed = data.trim()
      const looksLikeJson =
        (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'))
      if (looksLikeJson) {
        try {
          return JSON.parse(trimmed)
        } catch {
          // ignore parse error, return raw string
        }
      }
    }
    return data
  },
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('authToken')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// 辅助函数：创建URLSearchParams
export const createFormData = (data: Record<string, any>): URLSearchParams => {
  const params = new URLSearchParams()
  Object.keys(data).forEach(key => {
    if (data[key] !== undefined && data[key] !== null) {
      params.append(key, String(data[key]))
    }
  })
  return params
}

export default request
