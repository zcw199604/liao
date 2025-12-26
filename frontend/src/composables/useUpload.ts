import { ref } from 'vue'
import { useMediaStore } from '@/stores/media'
import * as mediaApi from '@/api/media'
import { IMG_SERVER_IMAGE_PORT, IMG_SERVER_VIDEO_PORT } from '@/constants/config'
import type { UploadedMedia } from '@/types'
import { generateCookie } from '@/utils/cookie'

export const useUpload = () => {
  const mediaStore = useMediaStore()
  const uploadLoading = ref(false)

  const uploadFile = async (file: File, userId: string, userName: string): Promise<UploadedMedia | null> => {
    uploadLoading.value = true
    try {
      if (!mediaStore.imgServer) {
        await mediaStore.loadImgServer()
      }

      if (!mediaStore.imgServer) {
        console.error('图片服务器地址未获取')
        return null
      }

      const formData = new FormData()
      formData.append('file', file)
      formData.append('userid', userId)

      // 上游所需 headers 参数
      const cookieData = generateCookie(userId, userName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      formData.append('cookieData', cookieData)
      formData.append('referer', referer)
      formData.append('userAgent', userAgent)

      const res = await mediaApi.uploadMedia(formData)

      if (res?.state === 'OK' && res.msg) {
        const type = file.type.startsWith('video/') ? 'video' : 'image'
        const port = type === 'video' ? IMG_SERVER_VIDEO_PORT : IMG_SERVER_IMAGE_PORT
        const url = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`

        const media: UploadedMedia = {
          url,
          type,
          localFilename: res.localFilename
        }

        mediaStore.addUploadedMedia(media)
        return media
      }

      return null
    } catch (error) {
      console.error('上传失败:', error)
      return null
    } finally {
      uploadLoading.value = false
    }
  }

  const getMediaUrl = (input: string): string => {
    if (!input) return ''
    if (input.startsWith('http://') || input.startsWith('https://')) return input
    if (input.startsWith('/upload/')) return `${window.location.origin}${input}`
    if (input.startsWith('/images/') || input.startsWith('/videos/')) return `${window.location.origin}/upload${input}`
    return input
  }

  return {
    uploadLoading,
    uploadFile,
    getMediaUrl
  }
}
