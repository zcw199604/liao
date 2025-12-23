import { ref } from 'vue'
import { useMediaStore } from '@/stores/media'
import * as mediaApi from '@/api/media'
import { extractUploadLocalPath } from '@/utils/media'

export const useMediaManager = () => {
  const mediaStore = useMediaStore()
  const deleting = ref(false)
  const deletingImages = ref<string[]>([])

  const deleteSingle = async (url: string, userId: string) => {
    deleting.value = true
    try {
      const localPath = extractUploadLocalPath(url)
      const res = await mediaApi.deleteMedia(localPath, userId)
      if (res.code === 0) {
        // 从列表中移除
        mediaStore.allUploadImages = mediaStore.allUploadImages.filter(img => img.url !== url)
        mediaStore.allUploadTotal = Math.max(0, mediaStore.allUploadTotal - 1)
        return true
      }
      return false
    } finally {
      deleting.value = false
    }
  }

  const batchDelete = async (urls: string[], userId: string) => {
    if (urls.length === 0) return { success: 0, failed: 0 }

    deleting.value = true
    deletingImages.value = urls

    try {
      const localPaths = urls.map(extractUploadLocalPath).slice(0, 50)
      const res = await mediaApi.batchDeleteMedia(userId, localPaths)
      if (res.code === 0) {
        // 从列表中移除成功删除的
        mediaStore.allUploadImages = mediaStore.allUploadImages.filter(img => !urls.includes(img.url))
        mediaStore.allUploadTotal = Math.max(0, mediaStore.allUploadTotal - localPaths.length)

        // 清空选择
        mediaStore.selectedImages = []
        mediaStore.selectionMode = false

        return { success: urls.length, failed: 0 }
      }
      return { success: 0, failed: urls.length }
    } finally {
      deleting.value = false
      deletingImages.value = []
    }
  }

  const toggleSelection = (url: string) => {
    const index = mediaStore.selectedImages.indexOf(url)
    if (index > -1) {
      mediaStore.selectedImages.splice(index, 1)
    } else {
      mediaStore.selectedImages.push(url)
    }
  }

  const selectAll = () => {
    if (mediaStore.selectedImages.length === mediaStore.allUploadImages.length) {
      // 全部取消
      mediaStore.selectedImages = []
    } else {
      // 全选（单次最多50张）
      mediaStore.selectedImages = mediaStore.allUploadImages.slice(0, 50).map(img => img.url)
    }
  }

  const isSelected = (url: string) => {
    return mediaStore.selectedImages.includes(url)
  }

  return {
    deleting,
    deletingImages,
    deleteSingle,
    batchDelete,
    toggleSelection,
    selectAll,
    isSelected
  }
}
