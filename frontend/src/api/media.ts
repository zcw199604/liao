import request, { createFormData } from './request'
import type { ApiResponse } from '@/types'

// 上传图片/视频（FormData格式，不是urlencoded）
export const uploadMedia = (formData: FormData) => {
  // 注意：后端返回为上游 JSON（可能是 text/plain），在 request 拦截器里会尝试 JSON.parse
  return request.post<any, any>('/uploadMedia', formData)
}

// 上传文件并在本地 image_hash 表中查重（先 MD5，后 pHash 相似度）
export const checkDuplicateMedia = (formData: FormData) => {
  return request.post<any, ApiResponse>('/checkDuplicateMedia', formData)
}

// 获取图片服务器地址
export const getImgServerAddress = () => {
  return request.get<any, any>('/getImgServer')
}

// 更新图片服务器地址（urlencoded）
export const updateImgServerAddress = (server: string) => {
  const formData = createFormData({ server })
  return request.post<any, any>('/updateImgServer', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 获取所有上传的图片（全站图片库，不按 userId 过滤）
export const getAllUploadImages = (
  page: number,
  pageSize: number,
  opts?: { source?: 'all' | 'local' | 'douyin'; douyinSecUserId?: string }
) => {
  return request.get<any, any>('/getAllUploadImages', {
    params: { page, pageSize, ...(opts || {}) }
  })
}

// 删除单个媒体（urlencoded）
export const deleteMedia = (localPath: string, userId: string) => {
  const formData = createFormData({ localPath, userId })
  return request.post<any, ApiResponse>('/deleteMedia', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 批量删除媒体
export const batchDeleteMedia = (userId: string, localPaths: string[]) => {
  return request.post<any, ApiResponse>('/batchDeleteMedia', { userId, localPaths })
}

// 获取缓存图片
export const getCachedImages = (userid: string) => {
  return request.get<any, any>('/getCachedImages', { params: { userid } })
}

// 获取聊天历史图片/视频（本地URL列表）
export const getChatImages = (userId1: string, userId2: string, limit: number = 20) => {
  return request.get<any, any>('/getChatImages', { params: { userId1, userId2, limit } })
}

// 从本地文件重新上传到上游服务器
export const reuploadHistoryImage = (data: {
  userId: string
  localPath: string
  cookieData?: string
  referer?: string
  userAgent?: string
}) => {
  const formData = createFormData(data)
  return request.post<any, any>('/reuploadHistoryImage', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 记录媒体发送关系（用于聊天历史图片查询）
export const recordImageSend = (data: { remoteUrl: string; fromUserId: string; toUserId: string; localFilename?: string }) => {
  const formData = createFormData(data)
  return request.post<any, any>('/recordImageSend', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}
