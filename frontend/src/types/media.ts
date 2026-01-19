export interface UploadedMedia {
  url: string
  type: 'image' | 'video' | 'file'
  // 用于下载（与展示 URL 解耦，例如 mtPhoto 预览展示缩略图，但下载原图）
  downloadUrl?: string
  localFilename?: string
  // 新增字段 - 用于显示详细信息
  originalFilename?: string
  fileSize?: number           // 字节数
  fileType?: string           // MIME类型
  fileExtension?: string      // 扩展名
  uploadTime?: string         // ISO时间字符串
  updateTime?: string         // ISO时间字符串
  md5?: string
  pHash?: string
  similarity?: number
  width?: number
  height?: number
  duration?: number
  day?: string
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

export interface DuplicateCheckItem {
  id: number
  filePath: string
  fileName: string
  fileDir?: string
  md5Hash: string
  pHash: string
  fileSize?: number
  createdAt?: string
  distance: number
  similarity: number
}

export interface CheckDuplicateData {
  matchType: 'md5' | 'phash' | 'none'
  md5: string
  pHash?: string
  thresholdType: string
  similarityThreshold: number
  distanceThreshold: number
  limit: number
  items: DuplicateCheckItem[]
  reason?: string
}
