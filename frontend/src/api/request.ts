import axios, { AxiosError } from 'axios'
import { API_BASE } from '@/constants/config'
import type { ApiResponse } from '@/types'

const request = axios.create({
  baseURL: API_BASE,
  timeout: 30000
})

// 抖音专用 Axios 实例（超时时间 65 秒，适配后端 60 秒超时）
export const douyinRequest = axios.create({
  baseURL: API_BASE,
  timeout: 65000
})

export const navigation = {
  toLogin: () => {
    window.location.assign('/login')
  }
}

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
      navigation.toLogin()
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

// 为 douyinRequest 配置相同的拦截器
douyinRequest.interceptors.request.use(
  config => {
    const token = localStorage.getItem('authToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => Promise.reject(error)
)

douyinRequest.interceptors.response.use(
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
      navigation.toLogin()
    }
    return Promise.reject(error)
  }
)

export default request
