<template>
  <teleport to="body">
    <div
      v-if="mediaStore.showAllUploadImageModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div
        :class="[
          'bg-[#18181b] flex flex-col min-h-0 transition-all duration-200 ease-out',
          isFullscreen
            ? 'w-full max-w-none h-full h-[100dvh] rounded-none shadow-none pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)]'
            : 'w-[95%] max-w-[1600px] h-[90vh] h-[90dvh] rounded-2xl shadow-2xl'
        ]"
        @click.stop
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-white/5">
          <div class="flex items-center gap-2">
            <i class="fas fa-images" :class="mediaStore.managementMode ? 'text-purple-500' : 'text-blue-500'"></i>
            <h3 class="text-lg font-bold text-white">
              {{ mediaStore.managementMode ? '管理已上传图片' : '所有上传图片' }}
            </h3>
            <span class="text-xs text-gray-500 ml-2">(共 {{ mediaStore.allUploadTotal }} 个)</span>

            <div class="flex items-center gap-1 ml-4 bg-[#27272a] border border-white/10 rounded-lg p-1">
              <button
                @click="changeAllUploadSource('all')"
                :class="mediaStore.allUploadSource === 'all' ? 'bg-gray-600 text-white' : 'text-gray-300 hover:text-white'"
                class="px-2 py-1 text-xs rounded-md transition"
              >
                全部
              </button>
              <button
                @click="changeAllUploadSource('local')"
                :class="mediaStore.allUploadSource === 'local' ? 'bg-gray-600 text-white' : 'text-gray-300 hover:text-white'"
                class="px-2 py-1 text-xs rounded-md transition"
              >
                本地
              </button>
              <button
                @click="changeAllUploadSource('douyin')"
                :class="mediaStore.allUploadSource === 'douyin' ? 'bg-gray-600 text-white' : 'text-gray-300 hover:text-white'"
                class="px-2 py-1 text-xs rounded-md transition"
              >
                抖音
              </button>
            </div>

            <div v-if="mediaStore.allUploadSource === 'douyin'" class="flex items-center gap-2 ml-3">
              <input
                v-model.trim="douyinSecUserIdInput"
                @keyup.enter="applyDouyinSecUserFilter"
                class="w-56 px-2 py-1 text-xs rounded-lg bg-[#0f0f12] border border-white/10 text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="抖音 sec_user_id（可选）"
              />
              <button
                @click="applyDouyinSecUserFilter"
                class="px-2 py-1 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition"
              >
                筛选
              </button>
              <button
                v-if="douyinSecUserIdInput"
                @click="clearDouyinSecUserFilter"
                class="px-2 py-1 text-xs rounded-lg bg-[#27272a] text-gray-200 hover:bg-[#333] transition"
              >
                清除
              </button>
            </div>
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
              @click="toggleFullscreen"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
              :title="isFullscreen ? '退出全屏' : '全屏'"
            >
              <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
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
	              <MediaTile
	                :src="media.url"
	                :type="media.type"
	                :reveal-top-right="true"
	                :fill="layoutMode === 'grid' || media.type === 'video'"
	                :media-class="layoutMode === 'grid' ? '' : (media.type === 'image' ? 'w-full h-auto' : '')"
	                class="rounded-xl overflow-hidden transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] bg-[#27272a]"
	                :class="[
                  mediaStore.selectedImages.includes(media.url) ? 'transform scale-95 ring-2 ring-purple-500' : '',
                  deletingUrls.has(media.url) ? 'opacity-50' : 'hover:brightness-110',
                  layoutMode === 'grid' ? 'w-full h-full' : (media.type === 'video' ? 'aspect-video' : 'w-full')
                ]"
                :show-skeleton="false"
                :muted="true"
                :indicator-size="'lg'"
	              >
	                <template #top-left>
	                  <MediaTileSelectMark
	                    v-if="mediaStore.managementMode && mediaStore.selectionMode"
	                    :checked="mediaStore.selectedImages.includes(media.url)"
	                    :interactive="true"
	                    tone="purple"
	                    size="md"
	                    @click="toggleSelection(media.url)"
	                  />
	                </template>

	                <template #top-right>
	                  <MediaTileActionButton
	                    v-if="mediaStore.managementMode && !mediaStore.selectionMode"
	                    tone="danger"
	                    size="md"
	                    title="删除"
	                    @click="confirmDelete([media.url])"
	                  >
	                    <i class="fas fa-trash text-sm"></i>
	                  </MediaTileActionButton>
	                </template>
	              </MediaTile>
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
        <div v-if="mediaStore.managementMode && mediaStore.selectionMode" class="px-6 py-4 border-t border-white/5">
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

        <div v-else class="px-6 py-4 border-t border-white/5 text-center text-xs text-gray-500">
          {{ mediaStore.managementMode ? '提示：点击图片预览，右上角可删除（桌面端悬停显示）' : '点击图片预览，在预览中可上传/重新上传，再在上方\"已上传的文件\"中点击发送' }}
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
import { useModalFullscreen } from '@/composables/useModalFullscreen'
import { generateCookie } from '@/utils/cookie'
import { extractUploadLocalPath } from '@/utils/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import * as mediaApi from '@/api/media'
import Dialog from '@/components/common/Dialog.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import MediaTile from '@/components/common/MediaTile.vue'
import MediaTileActionButton from '@/components/common/MediaTileActionButton.vue'
import MediaTileSelectMark from '@/components/common/MediaTileSelectMark.vue'
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

const douyinSecUserIdInput = ref(String(mediaStore.allUploadDouyinSecUserId || '').trim())

const changeAllUploadSource = async (source: 'all' | 'local' | 'douyin') => {
  if (mediaStore.allUploadSource === source) return
  mediaStore.allUploadSource = source
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  if (source !== 'douyin') {
    douyinSecUserIdInput.value = ''
    mediaStore.allUploadDouyinSecUserId = ''
  }
  await mediaStore.loadAllUploadImages(1)
}

const applyDouyinSecUserFilter = async () => {
  if (mediaStore.allUploadSource !== 'douyin') {
    await changeAllUploadSource('douyin')
  }
  mediaStore.allUploadDouyinSecUserId = String(douyinSecUserIdInput.value || '').trim()
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  await mediaStore.loadAllUploadImages(1)
}

const clearDouyinSecUserFilter = async () => {
  douyinSecUserIdInput.value = ''
  mediaStore.allUploadDouyinSecUserId = ''
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  await mediaStore.loadAllUploadImages(1)
}

const close = () => {
  mediaStore.showAllUploadImageModal = false
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  showPreview.value = false
  previewUrl.value = ''
  previewType.value = 'image'
  previewCanUpload.value = false
  previewTarget.value = null
}

const { isFullscreen, toggleFullscreen } = useModalFullscreen({
  isModalOpen: () => mediaStore.showAllUploadImageModal,
  isBlocked: () => showPreview.value,
  onRequestClose: close
})

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

const loadMore = async () => {
  if (mediaStore.allUploadLoading) return
  if (mediaStore.allUploadPage >= mediaStore.allUploadTotalPages) return

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
  previewCanUpload.value = !mediaStore.managementMode && !!userStore.currentUser
  showPreview.value = true
}

const confirmPreviewUpload = async () => {
  if (!previewTarget.value) return
  if (!userStore.currentUser) {
    show('请先选择身份后再重新上传')
    return
  }

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
  if (!userStore.currentUser?.id) {
    show('请先登录后再删除')
    return
  }
  if (!deleteTargets.value.length) return

  const userId = userStore.currentUser?.id || 'pre_identity'
  const urls = deleteTargets.value.slice(0, 50)
  urls.forEach(u => deletingUrls.value.add(u))

  try {
    const localPaths = urls.map(extractUploadLocalPath)

    if (localPaths.length === 1) {
      const res = await mediaApi.deleteMedia(localPaths[0]!, userId)
      if (res.code === 0) {
        mediaStore.allUploadImages = mediaStore.allUploadImages.filter(m => m.url !== urls[0])
        mediaStore.allUploadTotal = Math.max(0, mediaStore.allUploadTotal - 1)
        show('删除成功')
      } else {
        show(res.msg || '删除失败')
      }
      return
    }

    const res = await mediaApi.batchDeleteMedia(userId, localPaths)
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
