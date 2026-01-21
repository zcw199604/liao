import request, { createFormData } from './request'

export const getDouyinDetail = (data: { input: string; cookie?: string; proxy?: string }) => {
  return request.post<any, any>('/douyin/detail', data)
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

