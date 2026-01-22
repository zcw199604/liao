import request, { createFormData } from './request'

export const getDouyinDetail = (data: { input: string; cookie?: string }) => {
  return request.post<any, any>('/douyin/detail', data)
}

export const getDouyinAccount = (data: {
  input: string
  cookie?: string
  tab?: 'post' | 'favorite'
  cursor?: number
  count?: number
}) => {
  return request.post<any, any>('/douyin/account', data)
}

export const importDouyinMedia = (data: {
  userid: string
  key: string
  index: number
  cookieData?: string
  referer?: string
  userAgent?: string
}) => {
  const formData = createFormData(data as any)
  return request.post<any, any>('/douyin/import', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}
