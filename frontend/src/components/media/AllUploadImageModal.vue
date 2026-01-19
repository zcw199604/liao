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
              @click="toggleLayout"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
              :title="layoutMode === 'masonry' ? '切换到网格视图' : '切换到瀑布流视图'"
            >
              <i :class="layoutMode === 'masonry' ? 'fas fa-th' : 'fas fa-stream'"></i>
            </button>

            <button
              @click="close"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>

        <!-- 列表容器 -->
        <InfiniteMediaGrid
          :items="mediaStore.allUploadImages"
          :loading="mediaStore.allUploadLoading"
          :finished="mediaStore.allUploadPage >= mediaStore.allUploadTotalPages"
          :total="mediaStore.allUploadTotal"
          :layout-mode="layoutMode"
          :item-key="(item, idx) => 'all-upload-' + idx"
          @load-more="loadMore"
        >
          <template #default="{ item: media }">
            <div class="h-full cursor-pointer" @click="handleMediaClick(media)">
              <!-- 容器：处理缩放和圆角 -->
              <div 
                class="rounded-xl overflow-hidden transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] relative bg-[#27272a]"
                :class="[
                  mediaStore.selectedImages.includes(media.url) ? 'transform scale-95 ring-2 ring-purple-500' : '',
                  deletingUrls.has(media.url) ? 'opacity-50' : 'hover:brightness-110',
                  layoutMode === 'grid' ? 'w-full h-full' : ''
                ]"
              >
                 <!-- 多选复选框 -->
                <div
                  v-if="mediaStore.managementMode && mediaStore.selectionMode"
                  class="absolute top-2 left-2 z-20 transition-transform duration-300"
                  :class="mediaStore.selectedImages.includes(media.url) ? 'scale-100' : 'scale-90 opacity-80'"
                  @click.stop="toggleSelection(media.url)"
                >
                  <div class="w-6 h-6 rounded-full bg-black/40 backdrop-blur-sm flex items-center justify-center shadow-lg">
                    <i
                      v-if="mediaStore.selectedImages.includes(media.url)"
                      class="fas fa-check-circle text-purple-400 text-lg drop-shadow-md transform transition-transform duration-300 scale-110"
                    ></i>
                    <i v-else class="far fa-circle text-white/80 text-lg hover:text-white"></i>
                  </div>
                </div>

                <!-- 删除按钮 -->
                <button
                  v-if="mediaStore.managementMode && !mediaStore.selectionMode"
                  class="absolute top-2 right-2 z-20 w-8 h-8 rounded-full bg-black/60 text-red-400 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center hover:bg-black/80"
                  @click.stop="confirmDelete([media.url])"
                >
                  <i class="fas fa-trash text-sm"></i>
                </button>
                
                <!-- 媒体内容 -->
                <LazyImage
                  v-if="media.type === 'image'"
                  :src="media.url"
                  :container-class="layoutMode === 'grid' ? 'w-full h-full bg-[#27272a]' : 'w-full bg-[#27272a]'"
                  :img-class="layoutMode === 'grid' ? 'w-full h-full object-cover' : 'w-full h-auto block'"
                />
                
                <div v-else class="w-full bg-[#27272a] relative" :class="layoutMode === 'grid' ? 'h-full' : 'aspect-video'">
                    <video :src="media.url" class="w-full h-full object-cover"></video>
                    <div class="absolute inset-0 flex items-center justify-center bg-black/30">
                        <i class="fas fa-play-circle text-white text-4xl drop-shadow-lg"></i>
                    </div>
                </div>
              </div>
            </div>
          </template>

          <template #empty>
            <div class="text-center text-gray-500">
              <i class="fas fa-image text-5xl mb-4 opacity-30"></i>
              <p>暂无上传记录</p>
            </div>
          </template>

          <template #finished-text>
            已加载全部 {{ mediaStore.allUploadTotal }} 张图片
          </template>
        </InfiniteMediaGrid>

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
        :media-list="mediaStore.allUploadImages"
        @upload="confirmPreviewUpload"
        @media-change="handlePreviewMediaChange"
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
import { useSystemConfigStore } from '@/stores/systemConfig'
import * as mediaApi from '@/api/media'
import Dialog from '@/components/common/Dialog.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import LazyImage from '@/components/common/LazyImage.vue'
import InfiniteMediaGrid from '@/components/common/InfiniteMediaGrid.vue'
import type { UploadedMedia } from '@/types'

const mediaStore = useMediaStore()
const userStore = useUserStore()
const systemConfigStore = useSystemConfigStore()
const { show } = useToast()

const deletingUrls = ref<Set<string>>(new Set())

const showDeleteConfirm = ref(false)
const deleteTargets = ref<string[]>([])

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewCanUpload = ref(false)
const previewTarget = ref<UploadedMedia | null>(null)

// 布局模式：'masonry' | 'grid'
const layoutMode = ref<'masonry' | 'grid'>(
  (localStorage.getItem('media_layout_mode') as 'masonry' | 'grid') || 'masonry'
)

const toggleLayout = () => {
  layoutMode.value = layoutMode.value === 'masonry' ? 'grid' : 'masonry'
  localStorage.setItem('media_layout_mode', layoutMode.value)
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  // 预览切换后同步当前媒体，避免“切换后仍对首张执行上传/重传”等不一致行为。
  previewTarget.value = media
  previewUrl.value = media.url
  previewType.value = media.type
}

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

const loadMore = async () => {
  if (mediaStore.allUploadLoading) return
  if (mediaStore.allUploadPage >= mediaStore.allUploadTotalPages) return
  if (!userStore.currentUser) return

  await mediaStore.loadAllUploadImages(mediaStore.allUploadPage + 1)
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
      const port = await systemConfigStore.resolveImagePort(res.msg, mediaStore.imgServer)
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
