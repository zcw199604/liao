import { isImageFile, isVideoFile } from './file'

export const extractUploadLocalPath = (url: string): string => {
  if (!url) return ''

  // URL格式：http://localhost:8080/upload/images/2025/12/19/xxx.jpg
  // 提取：/images/2025/12/19/xxx.jpg
  const match = url.match(/\/upload(\/.+)$/)
  if (match && match[1]) return match[1]

  // 已经是 /images/... 或 /videos/... 的情况
  if (url.startsWith('/images/') || url.startsWith('/videos/')) return url

  return url
}

export const extractRemoteFilePathFromImgUploadUrl = (url: string): string => {
  if (!url) return ''
  const match = url.match(/\/img\/Upload\/(.+)$/)
  if (match && match[1]) return match[1]
  return url
}

export const inferMediaTypeFromUrl = (url: string): 'image' | 'video' | 'file' => {
  if (!url) return 'file'
  // 移除 URL 参数干扰
  const cleanUrl = (url.split('?')[0] || '').split('#')[0] || ''
  if (isVideoFile(cleanUrl)) return 'video'
  if (isImageFile(cleanUrl)) return 'image'
  return 'file'
}

