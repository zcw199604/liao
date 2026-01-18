<template>
  <teleport to="body">
    <div
      v-if="mtPhotoStore.showModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div class="w-[90%] max-w-2xl h-[70vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col" @click.stop>
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

          <button
            @click="close"
            class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>

        <!-- 错误提示 -->
        <div v-if="mtPhotoStore.lastError" class="px-6 py-3 text-xs text-red-400 border-b border-gray-800">
          {{ mtPhotoStore.lastError }}
        </div>

        <!-- 相册列表 -->
        <div v-if="mtPhotoStore.view === 'albums'" class="flex-1 overflow-y-auto p-6 no-scrollbar">
          <div v-if="mtPhotoStore.albumsLoading" class="flex-1 flex items-center justify-center text-gray-500 text-sm">
            加载中...
          </div>

          <div v-else-if="mtPhotoStore.albums.length > 0" class="grid grid-cols-2 sm:grid-cols-3 gap-4">
            <button
              v-for="album in mtPhotoStore.albums"
              :key="album.id"
              class="text-left rounded-xl overflow-hidden border border-gray-700 hover:border-pink-500 transition bg-[#111113]"
              @click="mtPhotoStore.openAlbum(album)"
            >
              <div class="aspect-square bg-black/30 overflow-hidden">
                <img
                  v-if="album.cover"
                  :src="getThumbUrl('s260', album.cover)"
                  class="w-full h-full object-cover"
                  loading="lazy"
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
        <div
          v-else
          ref="scrollContainer"
          class="flex-1 overflow-y-auto p-6 no-scrollbar"
          @scroll="handleScroll"
        >
          <div v-if="mtPhotoStore.mediaItems.length > 0" class="grid grid-cols-3 sm:grid-cols-4 gap-4">
            <div
              v-for="(item, idx) in mtPhotoStore.mediaItems"
              :key="item.md5 + '-' + idx"
              class="aspect-square rounded-xl overflow-hidden cursor-pointer border border-gray-700 hover:border-pink-500 transition relative group"
              @click="handleMediaClick(item)"
            >
              <img
                v-if="item.type === 'image'"
                :src="getThumbUrl('h220', item.md5)"
                class="w-full h-full object-cover"
                loading="lazy"
              />
              <img
                v-else
                :src="getThumbUrl('h220', item.md5)"
                class="w-full h-full object-cover"
                loading="lazy"
              />

              <div v-if="item.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/30">
                <i class="fas fa-play-circle text-white text-3xl"></i>
              </div>
            </div>
          </div>

          <div v-else class="flex-1 flex items-center justify-center text-gray-500 text-sm">
            暂无媒体
          </div>

          <div v-if="mtPhotoStore.mediaLoading" class="flex justify-center py-4 text-gray-500 text-sm">
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 border-2 border-gray-500 border-t-transparent rounded-full animate-spin"></span>
              <span>加载中...</span>
            </div>
          </div>

          <div
            v-else-if="mtPhotoStore.mediaTotalPages > 0 && mtPhotoStore.mediaPage >= mtPhotoStore.mediaTotalPages"
            class="text-center text-gray-600 text-xs py-4"
          >
            已加载全部
          </div>
        </div>
      </div>

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="previewCanUpload"
        :media-list="previewMediaList"
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
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useToast } from '@/composables/useToast'
import { generateCookie } from '@/utils/cookie'
import * as mtphotoApi from '@/api/mtphoto'
import MediaPreview from '@/components/media/MediaPreview.vue'
import type { UploadedMedia } from '@/types'

const mtPhotoStore = useMtPhotoStore()
const userStore = useUserStore()
const mediaStore = useMediaStore()
const systemConfigStore = useSystemConfigStore()
const { show } = useToast()

const scrollContainer = ref<HTMLElement | null>(null)

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewCanUpload = ref(true)
const previewMediaList = ref<UploadedMedia[]>([])
const previewMD5 = ref('')

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

const close = () => {
  mtPhotoStore.close()
  showPreview.value = false
  previewUrl.value = ''
  previewMediaList.value = []
  previewMD5.value = ''
}

const handleScroll = async () => {
  const el = scrollContainer.value
  if (!el) return
  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 120
  if (!nearBottom) return
  await mtPhotoStore.loadMore()
}

const handleMediaClick = async (item: MtPhotoMediaItem) => {
  previewMD5.value = item.md5
  previewType.value = item.type
  previewCanUpload.value = true

  // 图片直接用网关缩略图预览；视频则解析本地路径以便播放
  previewUrl.value = getThumbUrl('h220', item.md5)
  previewMediaList.value = []
  if (item.type === 'image') {
    // 仅在“点图片”时启用画廊模式：左右切换浏览当前已加载的相册图片列表。
    const list: UploadedMedia[] = mtPhotoStore.mediaItems
      .filter(m => m.type === 'image')
      .map(m => ({ url: getThumbUrl('h220', m.md5), type: 'image', md5: m.md5 }))
    previewMediaList.value = list
  }
  if (item.type === 'video') {
    try {
      const res = await mtphotoApi.resolveMtPhotoFilePath(item.md5)
      if (res?.filePath) {
        previewUrl.value = res.filePath
      }
    } catch {
      // ignore
    }
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
  if (!userStore.currentUser) return
  if (!previewMD5.value) return

  if (!mediaStore.imgServer) {
    await mediaStore.loadImgServer()
  }
  if (!mediaStore.imgServer) {
    show('图片服务器地址未获取')
    return
  }

  const cookieData = generateCookie(userStore.currentUser.id, userStore.currentUser.name)
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
  const userAgent = navigator.userAgent

  try {
    const res = await mtphotoApi.importMtPhotoMedia({
      userid: userStore.currentUser.id,
      md5: previewMD5.value,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.msg) {
      const port = String(res.port || await systemConfigStore.resolveImagePort(res.msg, mediaStore.imgServer))
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`

      mediaStore.addUploadedMedia({
        url: remoteUrl,
        type: previewType.value === 'video' ? 'video' : 'image',
        localFilename: res.localFilename
      })

      show('图片已加载，点击可发送')
      showPreview.value = false
      mtPhotoStore.close()
      mediaStore.requestOpenUploadMenu()
      return
    }

    show(`导入失败: ${res?.msg || res?.error || '未知错误'}`)
  } catch (e: any) {
    console.error('导入失败:', e)
    show('导入失败')
  }
}
</script>
