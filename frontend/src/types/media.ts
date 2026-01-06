export interface UploadedMedia {
  url: string
  type: 'image' | 'video' | 'file'
  localFilename?: string
  // 新增字段 - 用于显示详细信息
  originalFilename?: string
  fileSize?: number           // 字节数
  fileType?: string           // MIME类型
  fileExtension?: string      // 扩展名
  uploadTime?: string         // ISO时间字符串
  updateTime?: string         // ISO时间字符串
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
