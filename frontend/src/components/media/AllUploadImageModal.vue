<template>
  <teleport to="body">
    <div
      v-if="mediaStore.showAllUploadImageModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div class="w-[90%] max-w-2xl h-[70vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col" @click.stop>
        <!-- 头部 -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div class="flex items-center gap-2">
            <i class="fas fa-images" :class="mediaStore.managementMode ? 'text-purple-500' : 'text-blue-500'"></i>
            <h3 class="text-lg font-bold text-white">
              {{ mediaStore.managementMode ? '管理已上传图片' : '所有上传图片' }}
            </h3>
            <span class="text-xs text-gray-500 ml-2">(共 {{ mediaStore.allUploadTotal }} 个)</span>
          </div>

          <div class="flex items-center gap-2">
            <button
              v-if="mediaStore.managementMode"
              @click="toggleSelectionMode"
              :class="mediaStore.selectionMode ? 'bg-purple-600' : 'bg-gray-700'"
              class="px-3 py-1.5 text-white text-sm rounded-lg transition flex items-center gap-1"
            >
              <i :class="mediaStore.selectionMode ? 'fas fa-check-square' : 'far fa-square'"></i>
              <span>{{ mediaStore.selectionMode ? '取消选择' : '选择' }}</span>
            </button>

            <button
              @click="close"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>

        <!-- 加载状态 -->
        <div v-if="mediaStore.allUploadLoading && mediaStore.allUploadImages.length === 0" class="flex-1 flex items-center justify-center">
          <div class="text-center">
            <div class="radar-spinner mx-auto mb-3"></div>
            <p class="text-gray-500 text-sm">加载中...</p>
          </div>
        </div>

        <!-- 网格 -->
        <div
          v-else-if="mediaStore.allUploadImages && mediaStore.allUploadImages.length > 0"
          class="flex-1 overflow-y-auto p-6 no-scrollbar"
          @scroll="handleScroll"
          ref="scrollContainer"
        >
          <div class="grid grid-cols-3 sm:grid-cols-4 gap-4">
            <div
              v-for="(media, idx) in mediaStore.allUploadImages"
              :key="'all-upload-' + idx"
              class="aspect-square rounded-xl overflow-hidden cursor-pointer border-2 transition-all relative group"
              :class="[
                mediaStore.selectedImages.includes(media.url) ? 'border-purple-500' : 'border-gray-700',
                deletingUrls.has(media.url) ? 'opacity-50' : 'hover:border-blue-500 hover:scale-105'
              ]"
              @click="handleMediaClick(media)"
            >
              <!-- 多选复选框（管理模式 + 选择模式下显示） -->
              <div
                v-if="mediaStore.managementMode && mediaStore.selectionMode"
                class="absolute top-2 left-2 z-10"
                @click.stop="toggleSelection(media.url)"
              >
                <div class="w-6 h-6 rounded bg-black/50 flex items-center justify-center">
                  <i
                    v-if="mediaStore.selectedImages.includes(media.url)"
                    class="fas fa-check-circle text-purple-500 text-lg"
                  ></i>
                  <i v-else class="far fa-circle text-white text-lg"></i>
                </div>
              </div>

              <!-- 删除按钮（管理模式 + 非选择模式） -->
              <button
                v-if="mediaStore.managementMode && !mediaStore.selectionMode"
                class="absolute top-2 right-2 z-10 w-7 h-7 rounded-full bg-black/60 text-red-400 hidden group-hover:flex items-center justify-center"
                @click.stop="confirmDelete([media.url])"
              >
                <i class="fas fa-trash text-xs"></i>
              </button>

              <img v-if="media.type === 'image'" :src="media.url" class="w-full h-full object-cover" />
              <video v-else :src="media.url" class="w-full h-full object-cover"></video>

              <div v-if="media.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/30">
                <i class="fas fa-play-circle text-white text-3xl"></i>
              </div>
            </div>
          </div>

          <div v-if="mediaStore.allUploadLoading" class="flex justify-center py-4 text-gray-500 text-sm">
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 border-2 border-gray-500 border-t-transparent rounded-full animate-spin"></span>
              <span>加载中...</span>
            </div>
          </div>

          <div
            v-else-if="mediaStore.allUploadPage >= mediaStore.allUploadTotalPages && mediaStore.allUploadImages.length > 0"
            class="flex justify-center py-4 text-gray-600 text-sm"
          >
            已加载全部
          </div>
        </div>

        <!-- 空状态 -->
        <div v-else class="flex-1 flex items-center justify-center">
          <div class="text-center text-gray-500">
            <i class="fas fa-image text-5xl mb-4 opacity-30"></i>
            <p>暂无上传记录</p>
          </div>
        </div>

        <!-- 底部 -->
        <div v-if="mediaStore.managementMode && mediaStore.selectionMode" class="px-6 py-4 border-t border-gray-800">
          <div class="flex items-center justify-between gap-3">
            <button
              @click="toggleSelectAll"
              class="px-4 py-2 bg-[#27272a] text-white rounded-lg hover:bg-[#333] transition text-sm"
            >
              {{ isAllSelected ? '取消全选' : '全选当前页' }}
            </button>

            <button
              @click="confirmDelete(mediaStore.selectedImages)"
              :disabled="mediaStore.selectedImages.length === 0"
              class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              <i class="fas fa-trash"></i>
              <span>删除选中 ({{ mediaStore.selectedImages.length }})</span>
            </button>
          </div>
          <p v-if="mediaStore.selectedImages.length > 50" class="text-xs text-amber-500 mt-2">
            单次最多删除50张，已自动按前50张处理
          </p>
        </div>

        <div v-else class="px-6 py-4 border-t border-gray-800 text-center text-xs text-gray-500">
          {{ mediaStore.managementMode ? '提示：点击图片预览，悬停显示删除按钮' : '点击图片预览，在预览中可上传/重新上传，再在上方\"已上传的文件\"中点击发送' }}
        </div>
      </div>

      <Dialog
        v-model:visible="showDeleteConfirm"
        title="确认删除"
        :content="deleteConfirmContent"
        show-warning
        @confirm="executeDelete"
      />

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="previewCanUpload"
        @upload="confirmPreviewUpload"
      />
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'
import { useToast } from '@/composables/useToast'
import { generateCookie } from '@/utils/cookie'
import { extractUploadLocalPath } from '@/utils/media'
import { IMG_SERVER_IMAGE_PORT, IMG_SERVER_VIDEO_PORT } from '@/constants/config'
import * as mediaApi from '@/api/media'
import Dialog from '@/components/common/Dialog.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import type { UploadedMedia } from '@/types'

const mediaStore = useMediaStore()
const userStore = useUserStore()
const { show } = useToast()

const scrollContainer = ref<HTMLElement | null>(null)
const deletingUrls = ref<Set<string>>(new Set())

const showDeleteConfirm = ref(false)
const deleteTargets = ref<string[]>([])

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewCanUpload = ref(false)
const previewTarget = ref<UploadedMedia | null>(null)

const isAllSelected = computed(() => {
  if (!mediaStore.allUploadImages.length) return false
  return mediaStore.selectedImages.length === mediaStore.allUploadImages.length
})

const deleteConfirmContent = computed(() => {
  if (deleteTargets.value.length <= 1) return '确定要删除该媒体文件吗？此操作无法撤销。'
  return `确定要删除选中的 ${deleteTargets.value.length} 个媒体文件吗？此操作无法撤销。`
})

const close = () => {
  mediaStore.showAllUploadImageModal = false
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
}

const handleScroll = async () => {
  const el = scrollContainer.value
  if (!el) return

  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 120
  if (!nearBottom) return

  if (mediaStore.allUploadLoading) return
  if (mediaStore.allUploadPage >= mediaStore.allUploadTotalPages) return
  if (!userStore.currentUser) return

  await mediaStore.loadAllUploadImages(userStore.currentUser.id, mediaStore.allUploadPage + 1)
}

const toggleSelectionMode = () => {
  mediaStore.selectionMode = !mediaStore.selectionMode
  if (!mediaStore.selectionMode) {
    mediaStore.selectedImages = []
  }
}

const toggleSelection = (url: string) => {
  const idx = mediaStore.selectedImages.indexOf(url)
  if (idx >= 0) {
    mediaStore.selectedImages.splice(idx, 1)
  } else {
    mediaStore.selectedImages.push(url)
  }
}

const toggleSelectAll = () => {
  if (isAllSelected.value) {
    mediaStore.selectedImages = []
    return
  }

  // 单次最多50张
  mediaStore.selectedImages = mediaStore.allUploadImages.slice(0, 50).map(m => m.url)
}

const handleMediaClick = (media: UploadedMedia) => {
  if (mediaStore.managementMode && mediaStore.selectionMode) {
    toggleSelection(media.url)
    return
  }

  previewTarget.value = media
  previewUrl.value = media.url
  previewType.value = media.type
  previewCanUpload.value = !mediaStore.managementMode
  showPreview.value = true
}

const confirmPreviewUpload = async () => {
  if (!previewTarget.value || !userStore.currentUser) return

  if (!mediaStore.imgServer) {
    await mediaStore.loadImgServer()
  }
  if (!mediaStore.imgServer) {
    show('图片服务器地址未获取')
    return
  }

  const localPath = extractUploadLocalPath(previewTarget.value.url)
  const cookieData = generateCookie(userStore.currentUser.id, userStore.currentUser.name)
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
  const userAgent = navigator.userAgent

  try {
    const res = await mediaApi.reuploadHistoryImage({
      userId: userStore.currentUser.id,
      localPath,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.msg) {
      const port = previewTarget.value.type === 'video' ? IMG_SERVER_VIDEO_PORT : IMG_SERVER_IMAGE_PORT
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`
      
      const filename = localPath.substring(localPath.lastIndexOf('/') + 1)
      mediaStore.addUploadedMedia({ 
        url: remoteUrl, 
        type: previewTarget.value.type,
        localFilename: filename
      })
      
      show('图片已加载，点击可发送')
      showPreview.value = false
      close()
      mediaStore.requestOpenUploadMenu()
      return
    }

    show(`重新上传失败: ${res?.msg || res?.error || '未知错误'}`)
  } catch (e: any) {
    console.error('重新上传失败:', e)
    show('重新上传失败')
  }
}

const confirmDelete = (targets: string[]) => {
  if (!targets || targets.length === 0) return
  deleteTargets.value = targets
  showDeleteConfirm.value = true
}

const executeDelete = async () => {
  if (!userStore.currentUser) return
  if (!deleteTargets.value.length) return

  const urls = deleteTargets.value.slice(0, 50)
  urls.forEach(u => deletingUrls.value.add(u))

  try {
    const localPaths = urls.map(extractUploadLocalPath)

    if (localPaths.length === 1) {
      const res = await mediaApi.deleteMedia(localPaths[0]!, userStore.currentUser.id)
      if (res.code === 0) {
        mediaStore.allUploadImages = mediaStore.allUploadImages.filter(m => m.url !== urls[0])
        mediaStore.allUploadTotal = Math.max(0, mediaStore.allUploadTotal - 1)
        show('删除成功')
      } else {
        show(res.msg || '删除失败')
      }
      return
    }

    const res = await mediaApi.batchDeleteMedia(userStore.currentUser.id, localPaths)
    if (res.code === 0) {
      mediaStore.allUploadImages = mediaStore.allUploadImages.filter(m => !urls.includes(m.url))
      mediaStore.allUploadTotal = Math.max(0, mediaStore.allUploadTotal - localPaths.length)
      mediaStore.selectedImages = []
      mediaStore.selectionMode = false
      show('批量删除完成')
    } else {
      show(res.msg || '批量删除失败')
    }
  } catch (e) {
    console.error('删除失败:', e)
    show('删除失败')
  } finally {
    urls.forEach(u => deletingUrls.value.delete(u))
    deleteTargets.value = []
  }
}
</script>
