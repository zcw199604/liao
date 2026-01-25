<template>
  <teleport to="body">
    <div
      v-if="mtPhotoStore.showModal"
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
	        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
	          <div class="flex items-center gap-2 min-w-0">
            <button
              v-if="mtPhotoStore.view === 'album'"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a] flex-shrink-0"
              @click="mtPhotoStore.backToAlbums"
              title="返回相册列表"
            >
              <i class="fas fa-arrow-left"></i>
            </button>

            <i class="fas fa-photo-video text-pink-400 flex-shrink-0"></i>
            <h3 class="text-lg font-bold text-white truncate">
              {{ titleText }}
            </h3>
	            <span v-if="subTitleText" class="text-xs text-gray-500 ml-2 flex-shrink-0">
	              {{ subTitleText }}
	            </span>
	          </div>

	          <div class="flex items-center gap-2">
	            <button
	              v-if="mtPhotoStore.view === 'album'"
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

        <!-- 错误提示 -->
        <div v-if="mtPhotoStore.lastError" class="px-6 py-3 text-xs text-red-400 border-b border-gray-800">
          {{ mtPhotoStore.lastError }}
        </div>

        <!-- 相册列表 -->
        <div v-if="mtPhotoStore.view === 'albums'" class="flex-1 overflow-y-auto p-2 no-scrollbar">
          <div v-if="mtPhotoStore.albumsLoading" class="flex-1 flex items-center justify-center text-gray-500 text-sm">
            加载中...
          </div>

	          <div v-else-if="mtPhotoStore.albums.length > 0" class="grid grid-cols-2 sm:grid-cols-3 gap-2">
	            <button
	              v-for="album in mtPhotoStore.albums"
	              :key="album.id"
	              class="text-left rounded-xl overflow-hidden border border-gray-700 hover:border-pink-500 transition bg-[#111113]"
	              @click="mtPhotoStore.openAlbum(album)"
	            >
	              <div class="aspect-square bg-black/30 overflow-hidden">
	                <MediaTile
	                  v-if="album.cover"
	                  :src="getThumbUrl('s260', album.cover)"
	                  type="image"
	                  class="w-full h-full"
	                  :show-skeleton="false"
	                />
	                <div v-else class="w-full h-full flex items-center justify-center text-gray-500">
	                  <i class="fas fa-images text-3xl opacity-40"></i>
	                </div>
	              </div>
	              <div class="p-3">
                <div class="text-white font-medium text-sm truncate">{{ album.name }}</div>
                <div class="text-xs text-gray-500 mt-1">{{ album.count ?? 0 }} 个</div>
              </div>
            </button>
          </div>

          <div v-else class="flex-1 flex items-center justify-center text-gray-500 text-sm">
            暂无相册
          </div>
        </div>

        <!-- 相册媒体 -->
	        <InfiniteMediaGrid
	          v-else
	          :items="mtPhotoStore.mediaItems"
	          :loading="mtPhotoStore.mediaLoading"
	          :finished="mtPhotoStore.mediaTotalPages > 0 && mtPhotoStore.mediaPage >= mtPhotoStore.mediaTotalPages"
	          :total="mtPhotoStore.mediaTotal"
	          :layout-mode="layoutMode"
	          :item-key="(item, idx) => item.md5 + '-' + idx"
	          @load-more="mtPhotoStore.loadMore"
	        >
		          <template #default="{ item }">
		            <MediaTile
		              :src="getThumbUrl('h220', item.md5)"
		              type="image"
		              class="w-full rounded-xl overflow-hidden cursor-pointer border border-gray-700 hover:border-pink-500 transition bg-gray-800"
		              :class="layoutMode === 'grid' ? 'h-full' : ''"
		              :aspect-ratio="layoutMode === 'masonry' && item.width && item.height ? (Number(item.width) / Number(item.height)) : undefined"
		              :style="layoutMode === 'masonry' ? { contain: 'paint' } : {}"
		              :show-skeleton="false"
		              @click="handleMediaClick(item)"
		            >
		              <template v-if="item.type === 'video'" #center>
		                <div class="absolute inset-0 flex items-center justify-center bg-black/30">
		                  <i class="fas fa-play-circle text-white text-3xl"></i>
		                </div>
		              </template>
		            </MediaTile>
	          </template>

          <template #empty>
            <div class="flex items-center justify-center text-gray-500 text-sm h-full">
              暂无媒体
            </div>
          </template>

          <template #finished-text>
            已加载全部
          </template>
        </InfiniteMediaGrid>
      </div>

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="previewCanUpload"
        :media-list="previewMediaList"
        :resolve-original-filename="resolveMtPhotoOriginalFilename"
        @upload="confirmImportUpload"
        @media-change="handlePreviewMediaChange"
      />
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMtPhotoStore, type MtPhotoMediaItem } from '@/stores/mtphoto'
import { useUserStore } from '@/stores/user'
import { useToast } from '@/composables/useToast'
import { useModalFullscreen } from '@/composables/useModalFullscreen'
import * as mtphotoApi from '@/api/mtphoto'
	import MediaPreview from '@/components/media/MediaPreview.vue'
	import InfiniteMediaGrid from '@/components/common/InfiniteMediaGrid.vue'
	import MediaTile from '@/components/common/MediaTile.vue'
	import type { UploadedMedia } from '@/types'

const mtPhotoStore = useMtPhotoStore()
const userStore = useUserStore()
const { show } = useToast()

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
	const previewCanUpload = ref(true)
	const previewMediaList = ref<UploadedMedia[]>([])
	const previewMD5 = ref('')

	// 真实文件名解析缓存：md5 -> basename(filename)
	const mtPhotoOriginalFilenameCache = new Map<string, string>()

	const extractBasename = (value: string): string => {
	  const raw = String(value || '').trim()
	  if (!raw) return ''
	  const normalized = raw.replace(/\\/g, '/')
	  const withoutQuery = normalized.split('?')[0] || ''
	  const withoutHash = withoutQuery.split('#')[0] || ''
	  const parts = withoutHash.split('/').filter(Boolean)
	  return parts[parts.length - 1] || ''
	}

	const resolveMtPhotoOriginalFilename = async (media: UploadedMedia): Promise<string> => {
	  const md5Value = String(media.md5 || '').trim()
	  if (!md5Value) return ''
	  const cached = mtPhotoOriginalFilenameCache.get(md5Value)
	  if (cached) return cached

	  try {
	    const res = await mtphotoApi.resolveMtPhotoFilePath(md5Value)
	    const filename = extractBasename(String(res?.filePath || ''))
	    if (filename) {
	      mtPhotoOriginalFilenameCache.set(md5Value, filename)
	      return filename
	    }
	  } catch (e) {
	    console.warn('解析 mtPhoto 文件名失败:', e)
	  }

	  return ''
	}

	// 布局模式：'masonry' | 'grid'（与“全站图片库”保持一致）
	const layoutMode = ref<'masonry' | 'grid'>(
	  (localStorage.getItem('media_layout_mode') as 'masonry' | 'grid') || 'masonry'
	)

	const toggleLayout = () => {
	  layoutMode.value = layoutMode.value === 'masonry' ? 'grid' : 'masonry'
	  localStorage.setItem('media_layout_mode', layoutMode.value)
	}

	const titleText = computed(() => {
	  if (mtPhotoStore.view === 'albums') return 'mtPhoto 相册'
	  return mtPhotoStore.selectedAlbum?.name || 'mtPhoto 相册'
	})

const subTitleText = computed(() => {
  if (mtPhotoStore.view === 'albums') return mtPhotoStore.albums.length ? `(共 ${mtPhotoStore.albums.length} 个)` : ''
  if (mtPhotoStore.selectedAlbum) return `(共 ${mtPhotoStore.selectedAlbum.count ?? 0} 个)`
  return ''
})

const getThumbUrl = (size: 's260' | 'h220', md5: string) => {
  const safeMD5 = encodeURIComponent(md5 || '')
  return `/api/getMtPhotoThumb?size=${size}&md5=${safeMD5}`
}

const getOriginalDownloadUrl = (id: number, md5: string) => {
  const safeID = encodeURIComponent(String(id || ''))
  const safeMD5 = encodeURIComponent(md5 || '')
  return `/api/downloadMtPhotoOriginal?id=${safeID}&md5=${safeMD5}`
}

const close = () => {
  mtPhotoStore.close()
  showPreview.value = false
  previewUrl.value = ''
  previewMediaList.value = []
  previewMD5.value = ''
}

const { isFullscreen, toggleFullscreen } = useModalFullscreen({
  isModalOpen: () => mtPhotoStore.showModal,
  isBlocked: () => showPreview.value,
  onRequestClose: close
})

const handleMediaClick = async (item: MtPhotoMediaItem) => {
  previewMD5.value = item.md5
  previewType.value = item.type
  previewCanUpload.value = !!userStore.currentUser

  // 图片直接用网关缩略图预览；视频则解析本地路径以便播放
  previewUrl.value = getThumbUrl('h220', item.md5)
  previewMediaList.value = []
  if (item.type === 'image') {
    // 仅在“点图片”时启用画廊模式：左右切换浏览当前已加载的相册图片列表。
    const list: UploadedMedia[] = mtPhotoStore.mediaItems
      .filter(m => m.type === 'image')
      .map(m => ({
        url: getThumbUrl('h220', m.md5),
        type: 'image',
        downloadUrl: getOriginalDownloadUrl(m.id, m.md5),
        md5: m.md5,
        originalFilename: mtPhotoOriginalFilenameCache.get(m.md5),
        fileExtension: m.fileType ? String(m.fileType).trim().toLowerCase() : undefined,
        width: m.width,
        height: m.height,
        day: m.day
      }))
    previewMediaList.value = list
  }
  if (item.type === 'video') {
    try {
      const res = await mtphotoApi.resolveMtPhotoFilePath(item.md5)
      if (res?.filePath) {
        previewUrl.value = res.filePath
        const filename = extractBasename(res.filePath)
        if (filename) {
          mtPhotoOriginalFilenameCache.set(item.md5, filename)
        }
      }
    } catch {
      // ignore
    }
    previewMediaList.value = [
      {
        url: previewUrl.value,
        type: 'video',
        md5: item.md5,
        originalFilename: mtPhotoOriginalFilenameCache.get(item.md5),
        fileExtension: item.fileType ? String(item.fileType).trim().toLowerCase() : undefined,
        width: item.width,
        height: item.height,
        duration: item.duration,
        day: item.day
      }
    ]
  }

  showPreview.value = true
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  // 预览内部切换后，同步当前媒体，确保“上传此图片”导入的是当前所见内容。
  previewUrl.value = media.url || previewUrl.value
  previewType.value = media.type || previewType.value
  if (media.md5) {
    previewMD5.value = media.md5
  }
}

const confirmImportUpload = async () => {
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return
  }
  if (!previewMD5.value) return

  try {
    const res = await mtphotoApi.importMtPhotoMedia({
      userid: userStore.currentUser.id,
      md5: previewMD5.value
    })

    if (res?.state === 'OK' && res.localPath) {
      const dedup = !!res.dedup
      show(dedup ? '已存在（去重复用）' : '已导入到本地（去“所有图片”里手动上传后发送）')
      showPreview.value = false
      mtPhotoStore.close()
      return
    }

    show(`导入失败: ${res?.error || res?.msg || '未知错误'}`)
  } catch (e: any) {
    console.error('导入失败:', e)
    show('导入失败')
  }
}
</script>
