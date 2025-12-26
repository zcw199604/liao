export interface UploadedMedia {
  url: string
  type: 'image' | 'video' | 'file'
  localFilename?: string
}

export interface MediaItem {
  localPath: string
  fileName: string
  uploadTime?: string
  userId?: string
  md5?: string
}

export interface MediaPreview {
  url: string
  type: 'image' | 'video' | 'file'
  canUpload: boolean
  uploadTarget?: any
}
