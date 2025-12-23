// 解析文件名从路径中提取
export const extractFileName = (localPath: string): string => {
  if (!localPath) return ''
  const parts = localPath.split('/')
  return parts[parts.length - 1] || ''
}

// 获取文件扩展名
export const getFileExtension = (filename: string): string => {
  if (!filename) return ''
  const parts = filename.split('.')
  return parts.length > 1 ? (parts[parts.length - 1] || '').toLowerCase() : ''
}

// 判断是否为图片文件
export const isImageFile = (filename: string): boolean => {
  const ext = getFileExtension(filename)
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp'].includes(ext)
}

// 判断是否为视频文件
export const isVideoFile = (filename: string): boolean => {
  const ext = getFileExtension(filename)
  return ['mp4', 'webm', 'ogg', 'mov', 'avi'].includes(ext)
}

// 格式化文件大小
export const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
