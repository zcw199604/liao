<template>
  <teleport to="body">
    <transition name="fade">
      <div
        v-if="visible"
        class="fixed inset-0 bg-black/60 backdrop-blur-xl flex items-center justify-center z-[100] overflow-hidden select-none"
        @click.self="handleClose"
      >
        <!-- 顶部工具栏 -->
        <div class="absolute top-0 left-0 right-0 p-4 flex justify-between items-center z-20 bg-gradient-to-b from-black/50 to-transparent pointer-events-none">
           <!-- 缩放提示/状态/计数 -->
           <div class="flex items-center gap-3 px-2 pointer-events-auto">
              <span v-if="realMediaList.length > 1" class="text-white/90 font-medium text-sm drop-shadow-md">
                {{ currentIndex + 1 }} / {{ realMediaList.length }}
              </span>
              <span v-if="currentMedia.type === 'image'" class="text-white/50 text-xs shadow-black/50 drop-shadow-md">
                {{ scale > 1 ? '拖动查看 · 点击还原' : '点击放大' }}
              </span>
           </div>

           <div class="flex items-center gap-4 pointer-events-auto">
              <!-- 信息按钮 -->
              <button
                v-if="hasMediaDetails"
                @click.stop="handleShowDetails"
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
                title="查看详细信息"
              >
                <i class="fas fa-info-circle text-sm"></i>
              </button>

              <!-- 倍速/慢放（仅视频） -->
              <div
                v-if="currentMedia.type === 'video'"
                class="h-10 px-3 rounded-full bg-white/10 hover:bg-white/20 flex items-center gap-2 text-white transition backdrop-blur-sm"
                title="播放倍速"
              >
                <i class="fas fa-tachometer-alt text-xs text-white/80"></i>
                <select v-model.number="playbackRate" class="bg-transparent text-xs text-white outline-none cursor-pointer">
                  <option v-for="r in playbackRateOptions" :key="r" :value="r">
                    x{{ r }}
                  </option>
                </select>
              </div>

              <!-- 下载按钮 -->
              <button
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
                title="下载"
                @click.stop="handleDownload"
              >
                <i class="fas fa-download text-sm"></i>
              </button>

              <!-- 关闭按钮 -->
              <button
                @click="handleClose"
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
              >
                <i class="fas fa-times text-lg"></i>
              </button>
           </div>
        </div>

        <!-- 左右切换按钮 -->
        <template v-if="realMediaList.length > 1">
          <button 
            class="absolute left-2 sm:left-6 top-1/2 -translate-y-1/2 w-10 h-10 sm:w-12 sm:h-12 rounded-full bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center backdrop-blur-md transition z-30 focus:outline-none"
            @click.stop="prev"
            title="上一张 (←)"
          >
            <i class="fas fa-chevron-left text-lg sm:text-xl"></i>
          </button>
          
          <button 
            class="absolute right-2 sm:right-6 top-1/2 -translate-y-1/2 w-10 h-10 sm:w-12 sm:h-12 rounded-full bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center backdrop-blur-md transition z-30 focus:outline-none"
            @click.stop="next"
            title="下一张 (→)"
          >
            <i class="fas fa-chevron-right text-lg sm:text-xl"></i>
          </button>
        </template>

        <!-- 图片预览 (支持点击放大和拖动) -->
        <div 
          v-if="currentMedia.type === 'image'" 
          class="relative w-full h-full flex items-center justify-center p-0 transition-opacity duration-200 pb-20"
          @click.self="handleClose"
        >
          <img
            :key="currentMedia.url"
            :src="currentMediaDisplayUrl"
            class="max-w-full max-h-full object-contain cursor-grab active:cursor-grabbing select-none"
            :class="{ 'transition-transform duration-300 ease-out': !isDragging }"
            :style="imageStyle"
            alt="预览"
            draggable="false"
            @mousedown="startDrag"
            @touchstart="startDrag"
            @click.stop="handleClick"
            @error="handleMediaError"
          />
        </div>

        <!-- 视频预览 -->
        <div v-else-if="currentMedia.type === 'video'" class="relative w-full h-full flex items-center justify-center pb-20">
             <video
              :key="currentMedia.url + '-video'"
              ref="videoRef"
              :src="currentMediaDisplayUrl"
              controls
              autoplay
              class="max-w-[95%] max-h-[95%] shadow-2xl rounded-lg bg-black"
              @loadedmetadata="handleVideoLoadedMetadata"
              @error="handleMediaError"
            ></video>
        </div>

        <!-- 文件预览 -->
        <div v-else-if="currentMedia.type === 'file'" class="relative w-full h-full flex flex-col items-center justify-center pb-20 text-white">
            <i class="fas fa-file text-9xl mb-4 text-gray-400"></i>
            <p class="text-xl max-w-md text-center break-all px-4">{{ currentMedia.url.split('/').pop() }}</p>
            <p class="text-sm text-gray-400 mt-2">暂不支持预览此文件类型</p>
        </div>

        <!-- 底部缩略图栏 -->
        <div
          v-if="realMediaList.length > 1"
          class="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-black/90 via-black/50 to-transparent flex items-end justify-center z-40 pb-6 pointer-events-auto"
          @click.stop
        >
          <RecycleScroller
            v-if="useVirtualThumbnails"
            ref="thumbnailScrollerRef"
            class="no-scrollbar max-w-full items-center h-16 w-full sm:w-auto sm:max-w-[80vw]"
            :items="realMediaList"
            :item-size="thumbItemSize"
            direction="horizontal"
            key-field="url"
          >
            <template #default="{ item, index }">
              <div class="w-[60px] h-full flex items-center justify-center">
                <div
                  class="relative w-12 h-12 flex-shrink-0 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-200 shadow-lg"
                  :class="index === currentIndex ? 'border-indigo-500 scale-110 opacity-100 ring-2 ring-indigo-500/30' : 'border-transparent opacity-40 hover:opacity-80 hover:scale-105'"
                  @click="jumpTo(index)"
                >
                  <img v-if="item.type === 'image'" :src="item.url" class="w-full h-full object-cover" loading="lazy" />
                  <video v-else-if="item.type === 'video'" :src="item.url" class="w-full h-full object-cover"></video>
                  <div v-else class="w-full h-full flex items-center justify-center bg-gray-800 text-gray-400">
                    <i class="fas fa-file text-sm"></i>
                  </div>
                  <div v-if="item.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/40">
                    <i class="fas fa-play text-[8px] text-white/90"></i>
                  </div>
                </div>
              </div>
            </template>
          </RecycleScroller>

          <div
            v-else
            ref="thumbnailContainer"
            class="flex gap-3 px-4 overflow-x-auto no-scrollbar max-w-full items-center h-16 w-full sm:w-auto sm:max-w-[80vw]"
          >
            <div
              v-for="(item, idx) in realMediaList"
              :key="'thumb-' + idx"
              class="relative w-12 h-12 flex-shrink-0 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-200 shadow-lg"
              :class="idx === currentIndex ? 'border-indigo-500 scale-110 opacity-100 ring-2 ring-indigo-500/30' : 'border-transparent opacity-40 hover:opacity-80 hover:scale-105'"
              @click="jumpTo(idx)"
            >
              <img v-if="item.type === 'image'" :src="item.url" class="w-full h-full object-cover" loading="lazy" />
              <video v-else-if="item.type === 'video'" :src="item.url" class="w-full h-full object-cover"></video>
              <div v-else class="w-full h-full flex items-center justify-center bg-gray-800 text-gray-400">
                <i class="fas fa-file text-sm"></i>
              </div>
              <div v-if="item.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/40">
                <i class="fas fa-play text-[8px] text-white/90"></i>
              </div>
            </div>
          </div>
        </div>

        <!-- 上传按钮（如果允许上传） -->
        <div class="absolute bottom-28 left-1/2 transform -translate-x-1/2 flex items-center gap-3 z-50">
          <button
            v-if="currentMedia.type === 'video'"
            class="px-6 py-3 bg-sky-600 hover:bg-sky-700 text-white rounded-full font-medium transition shadow-lg shadow-sky-600/30 flex items-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed"
            title="暂停并抓取当前帧（下载+上传）"
            :disabled="captureFrameLoading"
            @click.stop="handleCaptureFrame"
          >
            <span v-if="captureFrameLoading" class="w-4 h-4 border-2 border-white/90 border-t-transparent rounded-full animate-spin"></span>
            <i v-else class="fas fa-camera"></i>
            <span>抓帧</span>
          </button>

          <button
            v-if="canExtractFrames"
            @click.stop="handleExtractFrames"
            class="px-6 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-full font-medium transition shadow-lg shadow-emerald-600/30 flex items-center gap-2"
            title="从该视频抽取图片"
          >
            <i class="fas fa-film"></i>
            <span>抽帧</span>
          </button>

          <button
            v-if="canUpload"
            @click="$emit('upload')"
            class="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-full font-medium transition shadow-lg shadow-indigo-600/30 flex items-center gap-2"
          >
            <i class="fas fa-cloud-upload-alt"></i>
            <span>上传此{{ currentMedia.type === 'image' ? '图片' : (currentMedia.type === 'video' ? '视频' : '文件') }}</span>
          </button>
        </div>
      </div>
    </transition>

    <!-- 详情面板 -->
    <MediaDetailPanel v-model:visible="showDetails" :media="currentMedia" />
  </teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import type { UploadedMedia } from '@/types'
import { useToast } from '@/composables/useToast'
import { useUpload } from '@/composables/useUpload'
import { useUserStore } from '@/stores/user'
import { useVideoExtractStore } from '@/stores/videoExtract'
import MediaDetailPanel from './MediaDetailPanel.vue'

interface Props {
  visible: boolean
  url: string
  type: 'image' | 'video' | 'file'
  canUpload?: boolean
  mediaList?: UploadedMedia[]
  resolveOriginalFilename?: (media: UploadedMedia) => Promise<string | undefined | null>
}

const props = withDefaults(defineProps<Props>(), {
  canUpload: false,
  mediaList: () => []
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'upload': []
  'media-change': [media: UploadedMedia]
}>()

const { show } = useToast()
const userStore = useUserStore()
const videoExtractStore = useVideoExtractStore()
const { uploadFile } = useUpload()

// 状态管理
const scale = ref(1)
const translateX = ref(0)
const translateY = ref(0)
const isDragging = ref(false)
const currentIndex = ref(0)
const thumbnailContainer = ref<HTMLElement | null>(null)
const thumbnailScrollerRef = ref<any>(null)

// 详情面板状态
const showDetails = ref(false)

const videoRef = ref<HTMLVideoElement | null>(null)

const playbackRateOptions = [0.1, 0.25, 0.5, 1, 1.5, 2, 5]
const playbackRate = ref<number>(1)
try {
  const raw = localStorage.getItem('media_preview_playback_rate')
  const v = raw === null ? NaN : Number(raw)
  if (playbackRateOptions.includes(v)) playbackRate.value = v
} catch {
  // ignore
}

const applyVideoPlaybackRate = () => {
  const video = videoRef.value
  if (!video) return
  const r = Number(playbackRate.value || 1)
  if (!Number.isFinite(r) || r <= 0) return
  // 同时设置 defaultPlaybackRate，避免部分浏览器在重新加载后回退到 1
  video.playbackRate = r
  video.defaultPlaybackRate = r
}

const handleVideoLoadedMetadata = () => {
  applyVideoPlaybackRate()
}

watch(playbackRate, () => {
  try {
    localStorage.setItem('media_preview_playback_rate', String(playbackRate.value))
  } catch {
    // ignore
  }
  applyVideoPlaybackRate()
})

const captureFrameLoading = ref(false)

const handleCaptureFrame = async () => {
  if (captureFrameLoading.value) return
  const video = videoRef.value
  if (!video) {
    show('视频未就绪')
    return
  }

  captureFrameLoading.value = true
  try {
    // 用户希望“暂停后抓帧”，这里若仍在播放则先暂停，确保画面稳定。
    if (!video.paused) {
      video.pause()
      if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
        await new Promise<void>(resolve => window.requestAnimationFrame(() => resolve()))
      }
    }

    // HAVE_CURRENT_DATA=2，保证当前帧可用
    if (video.readyState < 2 || video.videoWidth <= 0 || video.videoHeight <= 0) {
      show('视频未加载完成，无法抓帧')
      return
    }

    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      show('Canvas 不可用，无法抓帧')
      return
    }

    try {
      ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
    } catch (e: any) {
      const name = String(e?.name || '')
      if (name === 'SecurityError') {
        show('跨域视频受浏览器安全限制，无法抓帧；建议先上传到本地库或使用“抽帧”功能')
        return
      }
      show('抓帧失败')
      return
    }

    const blob = await new Promise<Blob | null>(resolve => {
      canvas.toBlob(b => resolve(b), 'image/png')
    })
    if (!blob) {
      show('抓帧失败')
      return
    }

    const rawBase =
      sanitizeFilename(String(currentMedia.value.originalFilename || '')) ||
      sanitizeFilename(String(currentMedia.value.localFilename || '')) ||
      'video'
    const base = rawBase.replace(/\.[a-zA-Z0-9]+$/, '') || 'video'
    const tMs = Math.max(0, Math.round((video.currentTime || 0) * 1000))
    const filename = sanitizeFilename(`${base}_frame_${tMs}ms.png`) || `frame_${tMs}ms.png`

    // 1) 直接下载
    triggerBlobDownload(blob, filename)

    // 2) 上传到图片库（需要身份）
    const u = userStore.currentUser
    if (!u?.id || !u?.name) {
      show('已下载抓帧图片；选择身份后可自动上传到图片库')
      return
    }

    const file = new File([blob], filename, { type: blob.type || 'image/png' })
    const uploaded = await uploadFile(file, u.id, u.name)
    if (uploaded) {
      show('抓帧已下载并上传（可在上传列表中使用）')
    } else {
      show('抓帧已下载，但上传失败')
    }
  } finally {
    captureFrameLoading.value = false
  }
}

const sanitizeFilename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  const normalized = raw.replace(/\\/g, '/')
  const withoutQuery = normalized.split('?')[0] || ''
  const withoutHash = withoutQuery.split('#')[0] || ''
  const parts = withoutHash.split('/').filter(Boolean)
  return parts[parts.length - 1] || ''
}

const resolveCurrentOriginalFilename = async () => {
  const resolver = props.resolveOriginalFilename
  if (!resolver) return

  const media = currentMedia.value
  if (!media || media.originalFilename) return
  if (!media.md5) return

  try {
    const resolved = await resolver(media)
    const safeName = sanitizeFilename(String(resolved || ''))
    if (!safeName) return

    // 避免将 filePath 等路径信息写入或展示：仅保存 basename。
    media.originalFilename = safeName
  } catch (e) {
    console.warn('resolveOriginalFilename failed:', e)
  }
}

const handleShowDetails = async () => {
  await resolveCurrentOriginalFilename()
  showDetails.value = true
}

const handleExtractFrames = async () => {
  const ok = await videoExtractStore.openCreateFromMedia(currentMedia.value, userStore.currentUser?.id)
  if (!ok) {
    show('当前媒体不支持抽帧')
    return
  }
  handleClose()
}

// 判断是否有详细信息
const hasMediaDetails = computed(() => {
  const media = currentMedia.value
  if (!media) return false

  return (
    media.fileSize !== undefined ||
    !!media.originalFilename ||
    !!media.localFilename ||
    !!media.fileExtension ||
    !!media.fileType ||
    !!media.uploadTime ||
    !!media.updateTime ||
    !!media.md5 ||
    !!media.pHash ||
    media.similarity !== undefined ||
    media.width !== undefined ||
    media.height !== undefined ||
    media.duration !== undefined ||
    !!media.day
  )
})

const canExtractFrames = computed(() => {
  const media = currentMedia.value
  if (!media || media.type !== 'video') return false
  const url = String(media.url || '').trim()
  const md5 = String(media.md5 || '').trim()
  if (md5 && (url.startsWith('/lsp/') || url.startsWith('/api/'))) return true
  if (url.startsWith('/videos/') || url.startsWith('/upload/')) return true
  if (url.includes('/upload/')) return true
  return false
})

// 整合后的媒体列表
const realMediaList = computed<UploadedMedia[]>(() => {
  if (props.mediaList && props.mediaList.length > 0) {
    return props.mediaList
  }
  // 兼容旧模式：单张图片构造成列表
  return [{ url: props.url, type: props.type }]
})

const thumbItemSize = 60
const useVirtualThumbnails = computed(() => realMediaList.value.length > 200)

const currentMedia = computed<UploadedMedia>(() => {
  if (realMediaList.value.length === 0) {
    return { url: '', type: 'image' }
  }
  const item = realMediaList.value[currentIndex.value]
  if (item) return item
  return realMediaList.value[0] || { url: '', type: 'image' }
})

const isSameOriginApiUrl = (href: string) => {
  const trimmed = (href || '').trim()
  if (!trimmed) return false
  try {
    const u = new URL(trimmed, window.location.origin)
    return u.origin === window.location.origin && u.pathname.startsWith('/api/')
  } catch {
    return trimmed.startsWith('/api/')
  }
}

const getFilenameFromContentDisposition = (value: string): string => {
  const raw = (value || '').trim()
  if (!raw) return ''

  const decodeMaybeEscapedFilename = (input: string): string => {
    const trimmed = (input || '').trim().replace(/^\"|\"$/g, '')
    if (!trimmed) return ''
    if (!/%[0-9A-Fa-f]{2}/.test(trimmed)) return trimmed
    try {
      return decodeURIComponent(trimmed.replace(/\+/g, '%20'))
    } catch {
      return trimmed
    }
  }

  // RFC 5987: filename*=UTF-8''...
  const m5987 = raw.match(/filename\*\s*=\s*UTF-8''([^;]+)/i)
  if (m5987 && m5987[1]) {
    const decoded = decodeMaybeEscapedFilename(m5987[1])
    if (decoded) return decoded
  }

  const m = raw.match(/filename\s*=\s*\"([^\"]+)\"/i) || raw.match(/filename\s*=\s*([^;]+)/i)
  if (m && m[1]) {
    return decodeMaybeEscapedFilename(m[1])
  }
  return ''
}

const guessExtFromMime = (mime: string) => {
  const mt = (mime || '').toLowerCase().split(';')[0]?.trim() || ''
  switch (mt) {
    case 'image/jpeg':
      return '.jpg'
    case 'image/png':
      return '.png'
    case 'image/gif':
      return '.gif'
    case 'image/webp':
      return '.webp'
    case 'video/mp4':
      return '.mp4'
    default:
      return ''
  }
}

const triggerBlobDownload = (blob: Blob, filename: string) => {
  const blobUrl = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = filename || 'download'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(blobUrl), 1000)
}

const triggerDirectDownload = (href: string) => {
  const link = document.createElement('a')
  link.href = href
  link.download = ''
  link.target = '_blank'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

const handleDownload = async () => {
  const href = String(currentMedia.value.downloadUrl || currentMedia.value.url || '').trim()
  if (!href) return

  // /api 资源需要带 Authorization；其他 URL 保持直链下载行为。
  if (!isSameOriginApiUrl(href)) {
    triggerDirectDownload(href)
    return
  }

  const token = localStorage.getItem('authToken')
  if (!token) {
    show('未登录或Token缺失')
    return
  }

  try {
    const resp = await fetch(href, {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` }
    })
    if (!resp.ok) {
      let msg = `下载失败: ${resp.status}`
      try {
        const data = await resp.json()
        msg = data?.error || data?.msg || msg
      } catch {
        // ignore
      }
      show(msg)
      return
    }

    const blob = await resp.blob()
    const cd = resp.headers.get('Content-Disposition') || ''
    const filenameFromHeader = getFilenameFromContentDisposition(cd)
    const filename =
      filenameFromHeader ||
      currentMedia.value.originalFilename ||
      currentMedia.value.localFilename ||
      (() => {
        const ext = guessExtFromMime(blob.type)
        const md5 = currentMedia.value.md5 || ''
        if (md5) return `mtphoto_${md5}${ext}`
        return `download${ext}`
      })()

    triggerBlobDownload(blob, sanitizeFilename(filename) || 'download')
  } catch (e) {
    console.error('download failed:', e)
    show('下载失败')
  }
}

const mediaReloadSeq = ref(0)
const mediaRetryCount = ref(0)
let mediaRetryTimer: ReturnType<typeof setTimeout> | null = null

const currentMediaDisplayUrl = computed(() => {
  const baseUrl = currentMedia.value.url || ''
  if (!baseUrl) return ''
  if (mediaReloadSeq.value === 0) return baseUrl
  const sep = baseUrl.includes('?') ? '&' : '?'
  return `${baseUrl}${sep}_=${mediaReloadSeq.value}`
})

const emitMediaChange = () => {
  // 预览面板作为“受控 + 内部索引”混合模式：父组件只负责初始 url，
  // 但上传/导入等动作往往需要跟随当前预览项变化，因此对外同步一次当前媒体。
  emit('media-change', currentMedia.value)
}

const resetMediaLoadState = () => {
  if (mediaRetryTimer) {
    clearTimeout(mediaRetryTimer)
    mediaRetryTimer = null
  }
  mediaReloadSeq.value = 0
  mediaRetryCount.value = 0
}

const handleMediaError = () => {
  // 图片/视频刚上传到上游时，短时间内可能 404；这里做轻量重试，避免用户需要重复“重传”
  if (mediaRetryCount.value >= 2) return
  mediaRetryCount.value += 1

  if (mediaRetryTimer) {
    clearTimeout(mediaRetryTimer)
    mediaRetryTimer = null
  }

  const delay = 600 * mediaRetryCount.value
  mediaRetryTimer = setTimeout(() => {
    mediaReloadSeq.value = Date.now()
  }, delay)
}

// 导航逻辑
const next = () => {
  resetZoom()
  if (currentIndex.value < realMediaList.value.length - 1) {
    currentIndex.value++
  } else {
    currentIndex.value = 0 // 循环
  }
  emitMediaChange()
}

const prev = () => {
  resetZoom()
  if (currentIndex.value > 0) {
    currentIndex.value--
  } else {
    currentIndex.value = realMediaList.value.length - 1 // 循环
  }
  emitMediaChange()
}

const jumpTo = (index: number) => {
  if (index === currentIndex.value) return
  resetZoom()
  currentIndex.value = index
  emitMediaChange()
}

const handleKeydown = (e: KeyboardEvent) => {
  if (!props.visible) return
  
  if (e.key === 'ArrowRight') next()
  if (e.key === 'ArrowLeft') prev()
  if (e.key === 'Escape') handleClose()
}

const scrollThumbnailToIndex = (index: number, behavior: ScrollBehavior) => {
  if (realMediaList.value.length <= 1) return

  if (useVirtualThumbnails.value) {
    const el = thumbnailScrollerRef.value?.$el as HTMLElement | undefined
    if (!el) return
    const containerWidth = el.clientWidth
    const left = Math.max(0, index * thumbItemSize - containerWidth / 2 + thumbItemSize / 2)
    el.scrollTo({ left, behavior })
    return
  }

  if (!thumbnailContainer.value) return
  const container = thumbnailContainer.value
  const children = container.children
  if (!children[index]) return

  const target = children[index] as HTMLElement
  const containerWidth = container.clientWidth
  const targetLeft = target.offsetLeft
  const targetWidth = target.clientWidth

  container.scrollTo({
    left: targetLeft - containerWidth / 2 + targetWidth / 2,
    behavior
  })
}

// 自动滚动缩略图
watch(currentIndex, (newIndex) => {
  if (!props.visible) return
  nextTick(() => {
    scrollThumbnailToIndex(newIndex, 'smooth')
  })
})

watch(
  () => currentMedia.value.url,
  () => {
    resetMediaLoadState()
    nextTick(() => applyVideoPlaybackRate())
  }
)

// 拖动辅助变量
let startX = 0
let startY = 0
let initialTranslateX = 0
let initialTranslateY = 0
let hasMoved = false

const imageStyle = computed(() => {
  return {
    transform: `translate3d(${translateX.value}px, ${translateY.value}px, 0) scale(${scale.value})`
  }
})

const handleClose = () => {
  resetZoom()
  showDetails.value = false
  emit('update:visible', false)
}

const resetZoom = () => {
  scale.value = 1
  translateX.value = 0
  translateY.value = 0
  isDragging.value = false
}

const handleClick = () => {
  if (scale.value === 1) {
    scale.value = 3 // 放大倍数
  } else {
    // 再次点击还原
    resetZoom()
  }
}

const startDrag = (e: MouseEvent | TouchEvent) => {
  // 允许 scale=1 时进行拖动以支持滑动切换
  // if (scale.value <= 1) return 
  
  // 对于触摸事件，不立即阻止默认行为，以便允许点击
  // 但在移动时会阻止默认行为
  
  isDragging.value = true
  hasMoved = false
  
  const clientX = e instanceof MouseEvent ? e.clientX : (e.touches?.[0]?.clientX || 0)
  const clientY = e instanceof MouseEvent ? e.clientY : (e.touches?.[0]?.clientY || 0)
  
  startX = clientX
  startY = clientY
  initialTranslateX = translateX.value
  initialTranslateY = translateY.value
  
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', stopDrag)
  window.addEventListener('touchmove', onDrag, { passive: false })
  window.addEventListener('touchend', stopDrag)
}

const onDrag = (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return
  
  const clientX = e instanceof MouseEvent ? e.clientX : (e.touches?.[0]?.clientX || 0)
  const clientY = e instanceof MouseEvent ? e.clientY : (e.touches?.[0]?.clientY || 0)
  
  const deltaX = clientX - startX
  const deltaY = clientY - startY
  
  // 防抖阈值
  if (Math.abs(deltaX) > 5 || Math.abs(deltaY) > 5) {
      hasMoved = true
      // 移动时阻止默认行为（如滚动）
      if (e.cancelable) e.preventDefault()
  }

  if (scale.value > 1) {
    // 放大模式：自由拖拽
    translateX.value = initialTranslateX + deltaX
    translateY.value = initialTranslateY + deltaY
  } else {
    // 未放大模式：仅水平滑动（Swipe）
    // 增加阻尼感，除以 1.5 还是 1.0 看手感，这里用 1:1 跟随更自然
    translateX.value = deltaX 
    // Y轴保持不动
  }
}

const stopDrag = () => {
  isDragging.value = false
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', stopDrag)
  window.removeEventListener('touchmove', onDrag)
  window.removeEventListener('touchend', stopDrag)
  
  if (!hasMoved) {
      // 如果没有移动，视为点击
      // 如果是放大状态下的点击，应该不需要在这里处理，handleClick 会处理
      // 但如果是在 scale=1 下的点击，handleClick 也会处理
      return
  }
  
  if (scale.value === 1) {
    // 滑动切换判定
    const threshold = 80 // 滑动阈值
    if (translateX.value > threshold) {
      // 向右滑 -> 上一张
      prev()
    } else if (translateX.value < -threshold) {
      // 向左滑 -> 下一张
      next()
    } else {
      // 未达到阈值，回弹
      translateX.value = 0
    }
  } else {
    // 放大状态下的松手，不需要额外逻辑，保持当前位置
    // (后续可以加边缘回弹逻辑，这里暂不处理)
  }
}

// 监听visible变化
watch(() => props.visible, (val) => {
  if (val) {
    resetZoom()
    resetMediaLoadState()
    showDetails.value = false
    window.addEventListener('keydown', handleKeydown)
    
    // 初始化 currentIndex
    // 如果有传入 mediaList，尝试找到 url 对应的 index
    if (props.mediaList && props.mediaList.length > 0 && props.url) {
      const idx = props.mediaList.findIndex(m => m.url === props.url)
      currentIndex.value = idx >= 0 ? idx : 0
      
      // Initial scroll to active thumbnail
      nextTick(() => {
        scrollThumbnailToIndex(currentIndex.value, 'auto')
      })
    } else {
      currentIndex.value = 0
    }
    emitMediaChange()
  } else {
    showDetails.value = false
    window.removeEventListener('keydown', handleKeydown)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  resetMediaLoadState()
})
</script>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
