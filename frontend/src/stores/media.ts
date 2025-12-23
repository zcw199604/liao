import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { UploadedMedia } from '@/types'
import * as mediaApi from '@/api/media'
import { extractUploadLocalPath, inferMediaTypeFromUrl } from '@/utils/media'

export const useMediaStore = defineStore('media', () => {
  const uploadedMedia = ref<UploadedMedia[]>([])
  const allUploadImages = ref<UploadedMedia[]>([])
  const allUploadTotal = ref(0)
  const allUploadPage = ref(1)
  const allUploadPageSize = ref(20)
  const allUploadTotalPages = ref(0)
  const allUploadLoading = ref(false)
  const imgServer = ref('')
  const openUploadMenuSeq = ref(0)

  const showAllUploadImageModal = ref(false)
  const managementMode = ref(false)
  const selectionMode = ref(false)
  const selectedImages = ref<string[]>([])

  const loadImgServer = async () => {
    try {
      const res = await mediaApi.getImgServerAddress()
      if (res?.state === 'OK' && res.msg?.server) {
        imgServer.value = res.msg.server
        await mediaApi.updateImgServerAddress(imgServer.value)
      }
    } catch (error) {
      console.error('获取图片服务器地址失败', error)
    }
  }

  const loadCachedImages = async (userid: string) => {
    try {
      const res = await mediaApi.getCachedImages(userid)
      const cacheData = Array.isArray(res?.data) ? res.data : (Array.isArray(res) ? res : [])

      if (!Array.isArray(cacheData)) {
        uploadedMedia.value = []
        return
      }

      uploadedMedia.value = cacheData
        .filter((url: unknown) => typeof url === 'string' && !!url)
        .map((localUrl: string) => {
          const type = inferMediaTypeFromUrl(localUrl)
          // 将本地URL转换为上游URL：/upload/images/... -> /img/Upload/...
          const localPath = extractUploadLocalPath(localUrl)
          const relativePath = localPath.replace(/^\//, '')
          const uploadPath = relativePath.replace(/^images\//, '').replace(/^videos\//, '')
          const port = type === 'video' ? '8006' : '9006'
          const url = imgServer.value ? `http://${imgServer.value}:${port}/img/Upload/${uploadPath}` : localUrl
          return { url, type }
        })
    } catch (error) {
      console.error('获取缓存图片失败', error)
      uploadedMedia.value = []
    }
  }

  const loadAllUploadImages = async (userId: string, page: number = 1) => {
    allUploadLoading.value = true
    try {
      const res = await mediaApi.getAllUploadImages(userId, page, allUploadPageSize.value)
      if (res && Array.isArray(res.data)) {
        const newItems: UploadedMedia[] = res.data
          .filter((url: unknown) => typeof url === 'string' && !!url)
          .map((url: string) => ({ url, type: inferMediaTypeFromUrl(url) }))

        if (page === 1) {
          allUploadImages.value = newItems
        } else {
          allUploadImages.value.push(...newItems)
        }

        allUploadTotal.value = Number(res.total || 0)
        allUploadPage.value = Number(res.page || page)
        allUploadPageSize.value = Number(res.pageSize || allUploadPageSize.value)
        allUploadTotalPages.value = Number(res.totalPages || Math.ceil(allUploadTotal.value / allUploadPageSize.value))
      }
    } finally {
      allUploadLoading.value = false
    }
  }

  const addUploadedMedia = (media: UploadedMedia) => {
    uploadedMedia.value.unshift(media)
  }

  const removeUploadedMedia = (url: string) => {
    uploadedMedia.value = uploadedMedia.value.filter(m => m.url !== url)
  }

  const clearUploadedMedia = () => {
    uploadedMedia.value = []
  }

  const requestOpenUploadMenu = () => {
    openUploadMenuSeq.value += 1
  }

  return {
    uploadedMedia,
    allUploadImages,
    allUploadTotal,
    allUploadPage,
    allUploadPageSize,
    allUploadTotalPages,
    allUploadLoading,
    imgServer,
    openUploadMenuSeq,
    showAllUploadImageModal,
    managementMode,
    selectionMode,
    selectedImages,
    loadImgServer,
    loadCachedImages,
    loadAllUploadImages,
    addUploadedMedia,
    removeUploadedMedia,
    clearUploadedMedia,
    requestOpenUploadMenu
  }
})
