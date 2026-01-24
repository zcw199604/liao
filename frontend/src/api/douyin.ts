import { douyinRequest, createFormData } from './request'

export const getDouyinDetail = (data: { input: string; cookie?: string }) => {
  return douyinRequest.post<any, any>('/douyin/detail', data)
}

export const getDouyinAccount = (data: {
  input: string
  cookie?: string
  tab?: 'post' | 'favorite'
  cursor?: number
  count?: number
}) => {
  return douyinRequest.post<any, any>('/douyin/account', data)
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
  return douyinRequest.post<any, any>('/douyin/import', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

export const listDouyinFavoriteUsers = () => {
  return douyinRequest.get<any, any>('/douyin/favoriteUser/list')
}

export const addDouyinFavoriteUser = (data: {
  secUserId: string
  sourceInput?: string
  displayName?: string
  avatarUrl?: string
  profileUrl?: string
  lastParsedCount?: number
  lastParsedRaw?: any
}) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/add', data)
}

export const removeDouyinFavoriteUser = (data: { secUserId: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/remove', data)
}

export const listDouyinFavoriteAwemes = () => {
  return douyinRequest.get<any, any>('/douyin/favoriteAweme/list')
}

export const addDouyinFavoriteAweme = (data: {
  awemeId: string
  secUserId?: string
  type?: string
  desc?: string
  coverUrl?: string
  rawDetail?: any
}) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/add', data)
}

export const removeDouyinFavoriteAweme = (data: { awemeId: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/remove', data)
}
