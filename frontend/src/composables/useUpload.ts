import { ref } from 'vue'
import { useMediaStore } from '@/stores/media'
import * as mediaApi from '@/api/media'
import { IMG_SERVER_IMAGE_PORT, IMG_SERVER_VIDEO_PORT } from '@/constants/config'
import type { UploadedMedia } from '@/types'
import { generateCookie } from '@/utils/cookie'
import { useToast } from '@/composables/useToast'

export const useUpload = () => {
  const mediaStore = useMediaStore()
  const uploadLoading = ref(false)
  const { error: showError } = useToast()

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
        let type: 'image' | 'video' | 'file' = 'file'
        if (file.type.startsWith('image/')) type = 'image'
        else if (file.type.startsWith('video/')) type = 'video'

        const port = type === 'video'
          ? IMG_SERVER_VIDEO_PORT
          : Number(res?.port || IMG_SERVER_IMAGE_PORT) || IMG_SERVER_IMAGE_PORT
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
    } catch (error: any) {
      console.error('上传失败:', error)

      // 解析错误信息
      let errorMessage = '上传失败，请稍后重试'
      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error
        // 如果后端返回了localPath，说明本地文件已保存
        if (error.response.data.localPath) {
          errorMessage += '。文件已保存到本地，可在"全站图片库"中重试'
        }
      } else if (error?.message) {
        errorMessage = `上传失败: ${error.message}`
      }

      showError(errorMessage)
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
