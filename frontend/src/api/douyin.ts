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

export const listDouyinFavoriteUserAwemes = (params: { secUserId: string; cursor?: number; count?: number }) => {
  return douyinRequest.get<any, any>('/douyin/favoriteUser/aweme/list', { params })
}

export const upsertDouyinFavoriteUserAwemes = (data: {
  secUserId: string
  items: { awemeId: string; type?: string; desc?: string; coverUrl?: string; downloads?: string[] }[]
}) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/aweme/upsert', data)
}

export const pullLatestDouyinFavoriteUserAwemes = (data: { secUserId: string; cookie?: string; count?: number }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/aweme/pullLatest', data)
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

export const listDouyinFavoriteUserTags = () => {
  return douyinRequest.get<any, any>('/douyin/favoriteUser/tag/list')
}

export const addDouyinFavoriteUserTag = (data: { name: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/tag/add', data)
}

export const updateDouyinFavoriteUserTag = (data: { id: number; name: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/tag/update', data)
}

export const removeDouyinFavoriteUserTag = (data: { id: number }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/tag/remove', data)
}

export const applyDouyinFavoriteUserTags = (data: { secUserIds: string[]; tagIds: number[]; mode?: 'set' | 'add' | 'remove' }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/tag/apply', data)
}

export const reorderDouyinFavoriteUserTags = (data: { tagIds: number[] }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteUser/tag/reorder', data)
}

export const listDouyinFavoriteAwemeTags = () => {
  return douyinRequest.get<any, any>('/douyin/favoriteAweme/tag/list')
}

export const addDouyinFavoriteAwemeTag = (data: { name: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/tag/add', data)
}

export const updateDouyinFavoriteAwemeTag = (data: { id: number; name: string }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/tag/update', data)
}

export const removeDouyinFavoriteAwemeTag = (data: { id: number }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/tag/remove', data)
}

export const applyDouyinFavoriteAwemeTags = (data: { awemeIds: string[]; tagIds: number[]; mode?: 'set' | 'add' | 'remove' }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/tag/apply', data)
}

export const reorderDouyinFavoriteAwemeTags = (data: { tagIds: number[] }) => {
  return douyinRequest.post<any, any>('/douyin/favoriteAweme/tag/reorder', data)
}
