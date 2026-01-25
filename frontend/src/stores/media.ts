import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { UploadedMedia } from '@/types'
import * as mediaApi from '@/api/media'
import { extractUploadLocalPath, inferMediaTypeFromUrl } from '@/utils/media'
import { useSystemConfigStore } from '@/stores/systemConfig'

export const useMediaStore = defineStore('media', () => {
  const uploadedMedia = ref<UploadedMedia[]>([])
  const allUploadImages = ref<UploadedMedia[]>([])
  const allUploadSource = ref<'all' | 'local' | 'douyin'>('all')
  const allUploadDouyinSecUserId = ref('')
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
    const systemConfigStore = useSystemConfigStore()
    try {
      const res = await mediaApi.getCachedImages(userid)
      const cacheData = Array.isArray(res?.data) ? res.data : (Array.isArray(res) ? res : [])

      if (!Array.isArray(cacheData)) {
        uploadedMedia.value = []
        return
      }

      if (!systemConfigStore.loaded) {
        await systemConfigStore.loadSystemConfig()
      }

      uploadedMedia.value = await Promise.all(cacheData
        .filter((url: unknown) => typeof url === 'string' && !!url)
        .map(async (localUrl: string) => {
          const type = inferMediaTypeFromUrl(localUrl)
          // 将本地URL转换为上游URL：/upload/images/... -> /img/Upload/...
          const localPath = extractUploadLocalPath(localUrl)
          const filename = localPath.substring(localPath.lastIndexOf('/') + 1)
          const relativePath = localPath.replace(/^\//, '')
          const uploadPath = relativePath.replace(/^images\//, '').replace(/^videos\//, '')
          let url = localUrl
          if (imgServer.value) {
            const port = await systemConfigStore.resolveImagePort(uploadPath, imgServer.value)
            url = `http://${imgServer.value}:${port}/img/Upload/${uploadPath}`
          }
          return { url, type, localFilename: filename }
        }))
    } catch (error) {
      console.error('获取缓存图片失败', error)
      uploadedMedia.value = []
    }
  }

  const loadAllUploadImages = async (page: number = 1) => {
    allUploadLoading.value = true
    try {
      const source = allUploadSource.value || 'all'
      const douyinSecUserId = String(allUploadDouyinSecUserId.value || '').trim()
      const res = await mediaApi.getAllUploadImages(page, allUploadPageSize.value, {
        source,
        douyinSecUserId: douyinSecUserId || undefined
      })
      if (res && Array.isArray(res.data)) {
        // 后端现在返回MediaFileDTO对象数组，直接使用
        const newItems: UploadedMedia[] = res.data.map((item: any) => ({
          url: item.url,
          type: item.type,
          localFilename: item.localFilename,
          originalFilename: item.originalFilename,
          fileSize: item.fileSize,
          fileType: item.fileType,
          fileExtension: item.fileExtension,
          uploadTime: item.uploadTime,
          updateTime: item.updateTime
        }))

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
    allUploadSource,
    allUploadDouyinSecUserId,
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
