import request, { createFormData } from './request'

export const getMtPhotoAlbums = () => {
  return request.get<any, any>('/getMtPhotoAlbums')
}

export const getMtPhotoAlbumFiles = (albumId: number, page: number, pageSize: number) => {
  return request.get<any, any>('/getMtPhotoAlbumFiles', {
    params: { albumId, page, pageSize }
  })
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

