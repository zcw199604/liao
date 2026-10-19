import axios, { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
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
const attachToken = (config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('authToken')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
}

// 响应拦截器：上游可能以 text/plain 返回 JSON，这里尽量解析
const parseResponseData = (response: AxiosResponse) => {
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
}

// 响应错误：401 统一跳转登录
const handleResponseError = (error: AxiosError) => {
  if (error.response?.status === 401) {
    localStorage.removeItem('authToken')
    navigation.toLogin()
  }
  return Promise.reject(error)
}

// 两个实例共用同一套拦截器
for (const instance of [request, douyinRequest]) {
  instance.interceptors.request.use(attachToken, error => Promise.reject(error))
  instance.interceptors.response.use(parseResponseData, handleResponseError)
}

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
