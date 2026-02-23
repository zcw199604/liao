import request, { createFormData } from './request'

export interface MtPhotoFolderNode {
  id: number
  name: string
  hide?: boolean
  galleryName?: string
  galleryFolderNum?: number
  path?: string
  cover?: string
  s_cover?: string | null
  subFolderNum?: number
  subFileNum?: number
  trashNum?: number
  fileType?: string
}

export interface MtPhotoFolderFile {
  id: number
  fileName: string
  fileType?: string
  size?: string
  tokenAt?: string
  md5: string
  width?: number
  height?: number
  duration?: number | null
  status?: number
  type?: 'image' | 'video'
}

export interface MtPhotoFolderContentResponse {
  path: string
  folderList: MtPhotoFolderNode[]
  fileList: MtPhotoFolderFile[]
  trashNum?: number
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

export interface MtPhotoFolderFavoriteItem {
  id: number
  folderId: number
  folderName: string
  folderPath: string
  coverMd5?: string
  tags: string[]
  note?: string
  createTime?: string
  updateTime?: string
}

export interface MtPhotoFolderFavoritesQuery {
  tagKeyword?: string
  tagMode?: 'any' | 'all'
  sortBy?: 'updatedAt' | 'name' | 'tagCount'
  sortOrder?: 'asc' | 'desc'
  groupBy?: 'none' | 'tag'
}

export const getMtPhotoAlbums = () => {
  return request.get<any, any>('/getMtPhotoAlbums')
}

export const getMtPhotoAlbumFiles = (albumId: number, page: number, pageSize: number) => {
  return request.get<any, any>('/getMtPhotoAlbumFiles', {
    params: { albumId, page, pageSize }
  })
}

export const getMtPhotoFolderRoot = () => {
  return request.get<any, MtPhotoFolderContentResponse>('/getMtPhotoFolderRoot')
}

export const getMtPhotoFolderContent = (folderId: number, page: number, pageSize: number) => {
  return request.get<any, MtPhotoFolderContentResponse>('/getMtPhotoFolderContent', {
    params: { folderId, page, pageSize }
  })
}

export const getMtPhotoFolderBreadcrumbs = (folderId: number) => {
  return request.get<any, MtPhotoFolderContentResponse>('/getMtPhotoFolderBreadcrumbs', {
    params: { folderId }
  })
}

export const getMtPhotoFolderFavorites = (params?: MtPhotoFolderFavoritesQuery) => {
  return request.get<any, { items: MtPhotoFolderFavoriteItem[] }>('/getMtPhotoFolderFavorites', {
    params: params || undefined
  })
}

export const upsertMtPhotoFolderFavorite = (data: {
  folderId: number
  folderName: string
  folderPath: string
  coverMd5?: string
  tags?: string[]
  note?: string
}) => {
  return request.post<any, { success: boolean; item: MtPhotoFolderFavoriteItem | null }>(
    '/upsertMtPhotoFolderFavorite',
    data
  )
}

export const removeMtPhotoFolderFavorite = (folderId: number) => {
  return request.post<any, { success: boolean }>('/removeMtPhotoFolderFavorite', { folderId })
}

export const resolveMtPhotoFilePath = (md5: string) => {
  return request.get<any, any>('/resolveMtPhotoFilePath', { params: { md5 } })
}

export const importMtPhotoMedia = (data: {
  userid: string
  md5: string
  cookieData?: string
  referer?: string
  userAgent?: string
}) => {
  const formData = createFormData(data as any)
  return request.post<any, any>('/importMtPhotoMedia', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}
