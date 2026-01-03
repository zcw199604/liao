import request, { createFormData } from './request'
import type { ApiResponse, Favorite } from '@/types'

export const addFavorite = (identityId: string, targetUserId: string, targetUserName?: string) => {
  const formData = createFormData({ identityId, targetUserId, targetUserName: targetUserName || '' })
  return request.post<any, ApiResponse<Favorite>>('/favorite/add', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

export const removeFavorite = (identityId: string, targetUserId: string) => {
  const formData = createFormData({ identityId, targetUserId })
  return request.post<any, ApiResponse>('/favorite/remove', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

export const removeFavoriteById = (id: number) => {
  const formData = createFormData({ id })
  return request.post<any, ApiResponse>('/favorite/removeById', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

export const listAllFavorites = () => {
  return request.get<any, ApiResponse<Favorite[]>>('/favorite/listAll')
}

export const checkFavorite = (identityId: string, targetUserId: string) => {
  return request.get<any, ApiResponse<{ isFavorite: boolean }>>('/favorite/check', {
    params: { identityId, targetUserId }
  })
}
