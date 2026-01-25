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
	           <div class="flex items-center gap-3 px-2 pointer-events-auto min-w-0">
	              <span v-if="realMediaList.length > 1" class="text-white/90 font-medium text-sm drop-shadow-md">
	                {{ currentIndex + 1 }} / {{ realMediaList.length }}
	              </span>
	              <span
	                v-if="currentMedia.title"
	                class="text-white/90 font-medium text-sm drop-shadow-md truncate max-w-[56vw] sm:max-w-[68vw]"
	                :title="currentMedia.title"
	              >
	                {{ currentMedia.title }}
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
              <div v-if="currentMedia.type === 'video' && !isVideoFullscreen" ref="speedMenuRef" class="relative">
                <button
                  class="h-10 px-3 rounded-full bg-white/10 hover:bg-white/20 flex items-center gap-2 text-white transition backdrop-blur-sm select-none"
                  :title="isTempSpeedBoosting ? '临时 2x 播放中，松开恢复' : '播放倍速（长按临时 x2）'"
                  @click.stop="handleToggleSpeedMenu"
                  @pointerdown="handleSpeedPressStart"
                  @pointerup="handleSpeedPressEnd"
                  @pointercancel="handleSpeedPressCancel"
                  @pointerleave="handleSpeedPressCancel"
                  @contextmenu.prevent
                >
                  <i class="fas fa-tachometer-alt text-xs text-white/80"></i>
                  <span class="text-xs font-medium">x{{ playbackRate }}</span>
                  <i class="fas fa-chevron-down text-[10px] text-white/70"></i>
                  <span
                    v-if="isTempSpeedBoosting"
                    class="ml-0.5 px-1.5 py-0.5 rounded-full bg-indigo-500/30 text-indigo-200 text-[10px] font-semibold"
                  >2X</span>
                </button>

                <div
                  v-if="showSpeedMenu"
                  class="absolute right-0 top-12 min-w-[120px] bg-[#111113]/95 backdrop-blur-md border border-white/10 rounded-xl shadow-2xl overflow-hidden z-50"
                  @click.stop
                >
                  <button
                    v-for="r in playbackRateOptions"
                    :key="r"
                    class="w-full px-4 py-2 text-left text-sm text-white/90 hover:bg-white/10 transition flex items-center justify-between"
                    @click.stop="selectPlaybackRate(r)"
                  >
                    <span>x{{ r }}</span>
                    <i v-if="playbackRate === r" class="fas fa-check text-indigo-400 text-xs"></i>
                  </button>
                </div>
              </div>

              <!-- 下载按钮 -->
              <button
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
                title="下载"
                @click.stop="handleDownload"
              >
                <i class="fas fa-download text-sm"></i>
              </button>

              <!-- 实况下载（Douyin Live Photo） -->
              <button
                v-if="canDownloadLivePhoto"
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
                title="下载实况：点击导出 JPG；右键导出 ZIP（iOS Live Photo）"
                @click.stop="handleDownloadLivePhoto('jpg')"
                @contextmenu.prevent.stop="handleDownloadLivePhoto('zip')"
              >
                <span class="text-[10px] font-bold tracking-wide">LIVE</span>
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

          <video
            v-if="livePhotoMotionUrl"
            ref="livePhotoVideoRef"
            :src="livePhotoMotionUrl"
            playsinline
            webkit-playsinline
            muted
            loop
            preload="auto"
            class="absolute inset-0 max-w-full max-h-full object-contain select-none pointer-events-none transition-opacity duration-150"
            :class="livePhotoVisible ? 'opacity-100' : 'opacity-0'"
            :style="imageStyle"
            @error="handleMediaError"
          ></video>
        </div>

	       <!-- 视频预览 -->
	        <div v-else-if="currentMedia.type === 'video'" class="relative w-full h-full flex items-center justify-center pb-20">
			          <div
			            ref="videoWrapperRef"
			            :key="currentMedia.url + '-video-wrapper'"
			            class="media-preview-video-wrapper relative inline-flex touch-none"
			            @pointerdown="handleVideoPointerDown"
			            @pointermove="handleVideoPointerMove"
			            @pointerup="handleVideoPointerUp"
			            @pointercancel="handleVideoPointerCancel"
			            @contextmenu.prevent
			          >
		             <video
		              ref="videoRef"
		              :src="currentMediaDisplayUrl"
		              playsinline
		              webkit-playsinline
	              controls
	              autoplay
	              class="media-preview-video shadow-2xl rounded-lg bg-black"
	              @loadedmetadata="handleVideoLoadedMetadata"
	              @error="handleMediaError"
		            ></video>

		            <!-- 全屏左上：倍速/慢放（避免与右侧抓帧/抽帧重叠；同时保证真全屏时仍可调速） -->
		            <div
		              v-if="isVideoFullscreen"
		              ref="speedMenuRef"
		              class="absolute left-3 sm:left-4 top-3 sm:top-4 z-50 pointer-events-none"
		            >
		              <div class="relative pointer-events-auto" @pointerdown.stop>
		                <button
		                  class="h-10 px-3 rounded-full bg-black/35 hover:bg-black/45 flex items-center gap-2 text-white transition backdrop-blur-md border border-white/15 shadow-xl select-none"
		                  :title="isTempSpeedBoosting ? '临时 2x 播放中，松开恢复' : '播放倍速（长按临时 x2）'"
		                  @click.stop="handleToggleSpeedMenu"
		                  @pointerdown="handleSpeedPressStart"
		                  @pointerup="handleSpeedPressEnd"
		                  @pointercancel="handleSpeedPressCancel"
		                  @pointerleave="handleSpeedPressCancel"
		                  @contextmenu.prevent
		                >
		                  <i class="fas fa-tachometer-alt text-xs text-white/80"></i>
		                  <span class="text-xs font-medium">x{{ playbackRate }}</span>
		                  <i class="fas fa-chevron-down text-[10px] text-white/70"></i>
		                  <span
		                    v-if="isTempSpeedBoosting"
		                    class="ml-0.5 px-1.5 py-0.5 rounded-full bg-indigo-500/30 text-indigo-200 text-[10px] font-semibold"
		                  >2X</span>
		                </button>

		                <div
		                  v-if="showSpeedMenu"
		                  class="absolute left-0 top-12 min-w-[120px] bg-[#111113]/95 backdrop-blur-md border border-white/10 rounded-xl shadow-2xl overflow-hidden z-50"
		                  @click.stop
		                >
		                  <button
		                    v-for="r in playbackRateOptions"
		                    :key="r"
		                    class="w-full px-4 py-2 text-left text-sm text-white/90 hover:bg-white/10 transition flex items-center justify-between"
		                    @click.stop="selectPlaybackRate(r)"
		                  >
		                    <span>x{{ r }}</span>
		                    <i v-if="playbackRate === r" class="fas fa-check text-indigo-400 text-xs"></i>
		                  </button>
		                </div>
		              </div>
		            </div>

		            <!-- 点击浮现三按钮（倒退/播放暂停/快进），1秒后自动隐藏 -->
		            <div
		              v-show="showVideoOverlayControls"
		              class="absolute inset-0 z-40 flex items-center justify-center pointer-events-none"
	            >
	              <div class="pointer-events-auto flex items-center gap-6 sm:gap-8">
	                <button
	                  class="w-14 h-14 sm:w-16 sm:h-16 rounded-full bg-black/35 hover:bg-black/45 text-white border border-white/15 backdrop-blur-md shadow-xl transition active:scale-95 flex items-center justify-center"
		                  title="倒退 1 秒"
		                  @click.stop="handleOverlaySeek(-1)"
		                  @pointerdown.stop
		                >
		                  <i class="fas fa-backward text-lg"></i>
		                </button>
	                <button
	                  class="w-16 h-16 sm:w-20 sm:h-20 rounded-full bg-white/15 hover:bg-white/25 text-white border border-white/20 backdrop-blur-md shadow-2xl transition active:scale-95 flex items-center justify-center"
	                  :title="isVideoPlaying ? '暂停' : '播放'"
	                  @click.stop="handleOverlayTogglePlay"
	                  @pointerdown.stop
	                >
	                  <i v-if="isVideoPlaying" class="fas fa-pause text-2xl"></i>
	                  <i v-else class="fas fa-play text-2xl ml-1"></i>
	                </button>
	                <button
	                  class="w-14 h-14 sm:w-16 sm:h-16 rounded-full bg-black/35 hover:bg-black/45 text-white border border-white/15 backdrop-blur-md shadow-xl transition active:scale-95 flex items-center justify-center"
		                  title="快进 1 秒"
		                  @click.stop="handleOverlaySeek(1)"
		                  @pointerdown.stop
		                >
		                  <i class="fas fa-forward text-lg"></i>
		                </button>
		              </div>
		            </div>

		            <!-- 全屏右侧：抓帧/抽帧快捷按钮（随控制浮层显示/隐藏） -->
		            <div
		              v-if="isVideoFullscreen && showVideoOverlayControls"
		              class="absolute right-3 sm:right-4 top-1/2 -translate-y-1/2 z-50 pointer-events-none"
		            >
		              <div class="pointer-events-auto flex flex-col gap-3">
		                <button
		                  class="w-11 h-11 rounded-full bg-black/35 hover:bg-black/45 text-white border border-white/15 backdrop-blur-md shadow-xl transition active:scale-95 flex items-center justify-center disabled:opacity-60 disabled:cursor-not-allowed"
		                  title="抓帧（暂停并抓取当前帧）"
		                  :disabled="captureFrameLoading"
		                  @click.stop="handleCaptureFrame"
		                  @pointerdown.stop
		                >
		                  <span
		                    v-if="captureFrameLoading"
		                    class="w-4 h-4 border-2 border-white/90 border-t-transparent rounded-full animate-spin"
		                  ></span>
		                  <i v-else class="fas fa-camera"></i>
		                </button>
		                <button
		                  v-if="canExtractFrames"
		                  class="w-11 h-11 rounded-full bg-black/35 hover:bg-black/45 text-white border border-white/15 backdrop-blur-md shadow-xl transition active:scale-95 flex items-center justify-center"
		                  title="抽帧（进入抽帧任务）"
		                  @click.stop="handleExtractFrames"
		                  @pointerdown.stop
		                >
		                  <i class="fas fa-film"></i>
		                </button>
		              </div>
		            </div>
		          </div>
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
	                <MediaTile
	                  :src="item.url"
	                  :type="item.type"
	                  class="w-12 h-12 flex-shrink-0 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-200 shadow-lg"
	                  :class="index === currentIndex ? 'border-indigo-500 scale-110 opacity-100 ring-2 ring-indigo-500/30' : 'border-transparent opacity-40 hover:opacity-80 hover:scale-105'"
	                  :show-skeleton="false"
	                  :indicator-size="'sm'"
	                  :muted="true"
	                  @click="jumpTo(index)"
	                >
	                  <template #file>
	                    <i class="fas fa-file text-sm"></i>
	                  </template>
	                </MediaTile>
	              </div>
	            </template>
	          </RecycleScroller>

          <div
            v-else
            ref="thumbnailContainer"
            class="flex gap-3 px-4 overflow-x-auto no-scrollbar max-w-full items-center h-16 w-full sm:w-auto sm:max-w-[80vw]"
          >
	            <MediaTile
	              v-for="(item, idx) in realMediaList"
	              :key="'thumb-' + idx"
	              :src="item.url"
	              :type="item.type"
	              class="w-12 h-12 flex-shrink-0 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-200 shadow-lg"
	              :class="idx === currentIndex ? 'border-indigo-500 scale-110 opacity-100 ring-2 ring-indigo-500/30' : 'border-transparent opacity-40 hover:opacity-80 hover:scale-105'"
	              :show-skeleton="false"
	              :indicator-size="'sm'"
	              :muted="true"
	              @click="jumpTo(idx)"
	            >
	              <template #file>
	                <i class="fas fa-file text-sm"></i>
	              </template>
	            </MediaTile>
	          </div>
	        </div>

	        <!-- 上传按钮（如果允许上传） -->
	        <div class="absolute bottom-28 left-1/2 transform -translate-x-1/2 flex items-center gap-3 z-50">
		          <button
		            v-if="currentMedia.type === 'video' && !isVideoFullscreen"
		            class="h-11 px-4 bg-black/35 hover:bg-black/45 text-white rounded-full font-medium transition shadow-xl backdrop-blur-md border border-white/15 flex items-center gap-2 active:scale-95 disabled:opacity-60 disabled:cursor-not-allowed"
		            title="暂停并抓取当前帧（下载+上传）"
		            :disabled="captureFrameLoading"
		            @click.stop="handleCaptureFrame"
		          >
            <span v-if="captureFrameLoading" class="w-4 h-4 border-2 border-white/90 border-t-transparent rounded-full animate-spin"></span>
            <i v-else class="fas fa-camera"></i>
            <span>抓帧</span>
          </button>

		          <button
		            v-if="canExtractFrames && !isVideoFullscreen"
		            @click.stop="handleExtractFrames"
		            class="h-11 px-4 bg-black/35 hover:bg-black/45 text-white rounded-full font-medium transition shadow-xl backdrop-blur-md border border-white/15 flex items-center gap-2 active:scale-95"
		            title="从该视频抽取图片"
		          >
            <i class="fas fa-film"></i>
            <span>抽帧</span>
          </button>

          <button
            v-if="canUpload"
            :disabled="uploadDisabled || uploadLoading"
            @click="$emit('upload')"
            class="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-full font-medium transition shadow-lg shadow-indigo-600/30 flex items-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:bg-indigo-600"
          >
            <span v-if="uploadLoading" class="w-4 h-4 border-2 border-white/90 border-t-transparent rounded-full animate-spin"></span>
            <i v-else class="fas fa-cloud-upload-alt"></i>
            <span>{{ uploadText || ('上传此' + (currentMedia.type === 'image' ? '图片' : (currentMedia.type === 'video' ? '视频' : '文件'))) }}</span>
          </button>
        </div>
      </div>
    </transition>

    <!-- 详情面板 -->
    <MediaDetailPanel v-model:visible="showDetails" :media="currentMedia" />
  </teleport>
</template>

<script setup lang="ts">
// 媒体预览弹窗：支持图片/视频/文件的全屏预览与画廊切换，并增强视频播放交互体验。
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
	import type { UploadedMedia } from '@/types'
	import { useToast } from '@/composables/useToast'
	import { useUpload } from '@/composables/useUpload'
	import { useUserStore } from '@/stores/user'
	import MediaTile from '@/components/common/MediaTile.vue'
import { useVideoExtractStore } from '@/stores/videoExtract'
import Plyr from 'plyr'
import 'plyr/dist/plyr.css'
import MediaDetailPanel from './MediaDetailPanel.vue'

interface Props {
  visible: boolean
  url: string
  type: 'image' | 'video' | 'file'
  canUpload?: boolean
  uploadDisabled?: boolean
  uploadLoading?: boolean
  uploadText?: string
  mediaList?: UploadedMedia[]
  resolveOriginalFilename?: (media: UploadedMedia) => Promise<string | undefined | null>
}

const props = withDefaults(defineProps<Props>(), {
  canUpload: false,
  uploadDisabled: false,
  uploadLoading: false,
  uploadText: '',
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

const videoWrapperRef = ref<HTMLElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const livePhotoVideoRef = ref<HTMLVideoElement | null>(null)
let plyrInstance: Plyr | null = null

const speedMenuRef = ref<HTMLElement | null>(null)
const showSpeedMenu = ref(false)
const isTempSpeedBoosting = ref(false)
let speedLongPressTimer: ReturnType<typeof setTimeout> | null = null
let suppressSpeedClick = false
let speedBoostWasPaused = false

const showVideoOverlayControls = ref(false)
const isVideoPlaying = ref(false)
const isVideoFullscreen = ref(false)
let overlayHideTimer: ReturnType<typeof setTimeout> | null = null

const VIDEO_GESTURE_THRESHOLD_PX = 22
const VIDEO_SWIPE_SEEK_STEP_SEC = 1
const DOUBLE_TAP_WINDOW_MS = 300
const DOUBLE_TAP_MAX_DIST_PX = 24
let tapTimer: ReturnType<typeof setTimeout> | null = null
let tapCount = 0
let lastTapAt = 0
let lastTapX = 0
let lastTapY = 0

type VideoGestureDirection = 'H' | 'V'
type VideoGestureState = {
  pointerId: number
  startX: number
  startY: number
  startAtMs: number
  direction: VideoGestureDirection | null
  started: boolean
  startTimeSec: number
  startVolume: number
  durationSec: number
  rectWidth: number
  rectHeight: number
  rafId: number | null
  latestDx: number
  latestDy: number
}
let videoGesture: VideoGestureState | null = null
let volumeGestureSupported: boolean | null = null
let cleanupVideoStateListeners: (() => void) | null = null

const destroyPlyr = () => {
  if (cleanupVideoStateListeners) {
    cleanupVideoStateListeners()
    cleanupVideoStateListeners = null
  }
  if (!plyrInstance) return
  try {
    plyrInstance.destroy()
  } catch (e) {
    console.warn('Plyr destroy failed:', e)
  } finally {
    plyrInstance = null
  }
}

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
  if (isTempSpeedBoosting.value) return

  // 同时设置 defaultPlaybackRate，避免部分浏览器在重新加载后回退到 1
  if (plyrInstance) {
    try {
      plyrInstance.speed = r
    } catch {
      // ignore
    }
  } else {
    video.playbackRate = r
  }
  video.defaultPlaybackRate = r
}

const handleVideoLoadedMetadata = () => {
  applyVideoPlaybackRate()
  syncVideoPlayState()
}

const handleDocumentPointerDown = (e: PointerEvent) => {
  if (!showSpeedMenu.value) return
  const root = speedMenuRef.value
  if (!root) {
    showSpeedMenu.value = false
    return
  }
  const t = e.target
  if (t instanceof Node && root.contains(t)) return
  showSpeedMenu.value = false
}

const closeSpeedMenu = () => {
  showSpeedMenu.value = false
}

const handleToggleSpeedMenu = () => {
  if (suppressSpeedClick) {
    suppressSpeedClick = false
    return
  }
  showSpeedMenu.value = !showSpeedMenu.value
}

const selectPlaybackRate = (r: number) => {
  playbackRate.value = r
  closeSpeedMenu()
}

const syncVideoPlayState = () => {
  const video = videoRef.value
  isVideoPlaying.value = !!video && !video.paused && !video.ended
}

const attachVideoStateListeners = () => {
  if (cleanupVideoStateListeners) {
    cleanupVideoStateListeners()
    cleanupVideoStateListeners = null
  }

  const video = videoRef.value
  if (!video) return
  const onPlay = () => syncVideoPlayState()
  const onPause = () => syncVideoPlayState()
  const onEnded = () => syncVideoPlayState()
  video.addEventListener('play', onPlay)
  video.addEventListener('pause', onPause)
  video.addEventListener('ended', onEnded)
  syncVideoPlayState()

  cleanupVideoStateListeners = () => {
    video.removeEventListener('play', onPlay)
    video.removeEventListener('pause', onPause)
    video.removeEventListener('ended', onEnded)
  }
}

const setVideoSpeed = (r: number, opts?: { setDefault?: boolean }) => {
  const video = videoRef.value
  if (!video) return
  const rate = Number(r || 1)
  if (!Number.isFinite(rate) || rate <= 0) return

  if (plyrInstance) {
    try {
      plyrInstance.speed = rate
    } catch {
      // ignore
    }
  } else {
    video.playbackRate = rate
  }

  if (opts?.setDefault) {
    video.defaultPlaybackRate = rate
  }
}

const setVideoCurrentTime = (t: number) => {
  const video = videoRef.value
  if (!video) return
  const current = Number(video.currentTime || 0)
  const duration = Number(video.duration)
  const hasDuration = Number.isFinite(duration) && duration > 0
  const next = Number.isFinite(t) ? t : current
  const clamped = Math.max(0, hasDuration ? Math.min(duration, next) : next)

  if (plyrInstance) {
    try {
      plyrInstance.currentTime = clamped
      return
    } catch {
      // ignore
    }
  }
  video.currentTime = clamped
}

const setVideoVolume = (v: number) => {
  const video = videoRef.value
  if (!video) return
  const next = Math.max(0, Math.min(1, Number.isFinite(v) ? v : video.volume))

  if (plyrInstance) {
    try {
      plyrInstance.volume = next
    } catch {
      video.volume = next
    }
  } else {
    video.volume = next
  }

  if (volumeGestureSupported === null) {
    // iOS Safari（移动端）通常禁止网页脚本调节音量，这里做一次性能力探测。
    const after = video.volume
    volumeGestureSupported = Math.abs(after - next) < 0.001
    if (volumeGestureSupported === false) {
      show('当前浏览器限制网页调节音量，请使用实体音量键')
    }
  }
}

const clearOverlayHideTimer = () => {
  if (!overlayHideTimer) return
  clearTimeout(overlayHideTimer)
  overlayHideTimer = null
}

const showOverlayWithAutoHide = () => {
  showVideoOverlayControls.value = true
  clearOverlayHideTimer()
  overlayHideTimer = setTimeout(() => {
    showVideoOverlayControls.value = false
    overlayHideTimer = null
  }, 1000)
}

const showOverlayDuringGesture = () => {
  showVideoOverlayControls.value = true
  clearOverlayHideTimer()
}

const toggleVideoPlay = async () => {
  const video = videoRef.value
  if (!video) return

  try {
    if (video.paused || video.ended) {
      if (plyrInstance) await (plyrInstance.play() as any)
      else await video.play()
    } else {
      if (plyrInstance) plyrInstance.pause()
      else video.pause()
    }
  } catch (e) {
    console.warn('toggleVideoPlay failed:', e)
    show('播放失败，请点击控制栏播放')
  } finally {
    syncVideoPlayState()
  }
}

const handleOverlayTogglePlay = () => {
  showOverlayWithAutoHide()
  void toggleVideoPlay()
}

const handleOverlaySeek = (deltaSec: number) => {
  showOverlayWithAutoHide()
  const video = videoRef.value
  if (!video) return
  const cur = Number.isFinite(video.currentTime) ? video.currentTime : 0
  setVideoCurrentTime(cur + deltaSec)
}

const syncFullscreenState = () => {
  const plyrActive = Boolean((plyrInstance as any)?.fullscreen?.active)
  const doc: any = typeof document !== 'undefined' ? document : null
  const el = videoWrapperRef.value
  const docEl = doc ? doc.fullscreenElement || doc.webkitFullscreenElement : null
  isVideoFullscreen.value = plyrActive || (!!el && docEl === el)
}

const handleDocumentFullscreenChange = () => {
  syncFullscreenState()
}

const toggleVideoFullscreen = () => {
  if (plyrInstance && (plyrInstance as any)?.fullscreen?.toggle) {
    try {
      ;(plyrInstance as any).fullscreen.toggle()
      syncFullscreenState()
      return
    } catch {
      // ignore
    }
  }

  const el = videoWrapperRef.value as any
  const doc: any = typeof document !== 'undefined' ? document : null
  if (!el || !doc) return

  const docEl = doc.fullscreenElement || doc.webkitFullscreenElement
  try {
    if (docEl) {
      ;(doc.exitFullscreen || doc.webkitExitFullscreen)?.call(doc)
    } else {
      ;(el.requestFullscreen || el.webkitRequestFullscreen)?.call(el)
    }
  } catch {
    // ignore
  } finally {
    syncFullscreenState()
  }
}

const exitVideoFullscreen = () => {
  try {
    ;(plyrInstance as any)?.fullscreen?.exit?.()
  } catch {
    // ignore
  }

  const doc: any = typeof document !== 'undefined' ? document : null
  try {
    if (doc?.fullscreenElement || doc?.webkitFullscreenElement) {
      ;(doc.exitFullscreen || doc.webkitExitFullscreen)?.call(doc)
    }
  } catch {
    // ignore
  } finally {
    isVideoFullscreen.value = false
  }
}

const clearTapState = () => {
  if (tapTimer) {
    clearTimeout(tapTimer)
    tapTimer = null
  }
  tapCount = 0
  lastTapAt = 0
}

const handleVideoTap = (e: PointerEvent) => {
  const now = Date.now()
  const x = e.clientX
  const y = e.clientY
  const dt = now - lastTapAt
  const dx = x - lastTapX
  const dy = y - lastTapY
  const maxDist = DOUBLE_TAP_MAX_DIST_PX
  const isSecondTap =
    tapCount === 1 &&
    dt > 0 &&
    dt <= DOUBLE_TAP_WINDOW_MS &&
    dx * dx + dy * dy <= maxDist * maxDist

  if (isSecondTap) {
    clearTapState()
    toggleVideoFullscreen()
    showOverlayWithAutoHide()
    return
  }

  // 延迟单击：留出时间窗口识别双击
  lastTapAt = now
  lastTapX = x
  lastTapY = y
  tapCount = 1
  // 立即给出可见反馈（浮层），播放/暂停动作延迟到窗口结束
  showOverlayWithAutoHide()
  if (tapTimer) clearTimeout(tapTimer)
  tapTimer = setTimeout(() => {
    tapTimer = null
    tapCount = 0
    void toggleVideoPlay()
  }, DOUBLE_TAP_WINDOW_MS)
}

const clearSpeedLongPressTimer = () => {
  if (!speedLongPressTimer) return
  clearTimeout(speedLongPressTimer)
  speedLongPressTimer = null
}

const stopTempSpeedBoost = () => {
  if (!isTempSpeedBoosting.value) return
  isTempSpeedBoosting.value = false
  setVideoSpeed(playbackRate.value, { setDefault: false })
  if (speedBoostWasPaused) {
    try {
      plyrInstance?.pause()
      videoRef.value?.pause()
    } catch {
      // ignore
    }
  }
  speedBoostWasPaused = false
}

const handleSpeedPressStart = (e: PointerEvent) => {
  if (e.button !== undefined && e.button !== 0) return
  suppressSpeedClick = false
  clearSpeedLongPressTimer()

  speedLongPressTimer = setTimeout(() => {
    const video = videoRef.value
    if (!video) return
    suppressSpeedClick = true
    closeSpeedMenu()
    speedBoostWasPaused = video.paused || video.ended
    isTempSpeedBoosting.value = true
    setVideoSpeed(2, { setDefault: false })
    if (speedBoostWasPaused) void toggleVideoPlay()
  }, 320)
}

const handleSpeedPressEnd = () => {
  const wasBoosting = isTempSpeedBoosting.value
  clearSpeedLongPressTimer()
  if (wasBoosting) {
    suppressSpeedClick = true
    stopTempSpeedBoost()
  }
}

const handleSpeedPressCancel = () => {
  const wasBoosting = isTempSpeedBoosting.value
  clearSpeedLongPressTimer()
  if (wasBoosting) stopTempSpeedBoost()
}

const isTargetInPlyrControls = (target: EventTarget | null): boolean => {
  const el = target instanceof Element ? target : null
  if (!el) return false
  return !!el.closest('.plyr__controls, .plyr__control, .plyr__progress, .plyr__volume')
}

const cancelVideoGestureRaf = () => {
  if (!videoGesture?.rafId) return
  cancelAnimationFrame(videoGesture.rafId)
  videoGesture.rafId = null
}

const applyVideoGestureFrame = () => {
  if (!videoGesture || !videoGesture.started || !videoGesture.direction) return
  const video = videoRef.value
  if (!video) return

  if (videoGesture.direction === 'H') {
    // 短视频场景：左右滑动采用“步进”而非按时长线性映射，避免轻微滑动导致跳跃过大
    const width = videoGesture.rectWidth
    const stepPx =
      Number.isFinite(width) && width > 0 ? Math.max(40, Math.min(80, width * 0.08)) : 48
    const steps = Math.trunc(videoGesture.latestDx / stepPx)
    const target = videoGesture.startTimeSec + steps * VIDEO_SWIPE_SEEK_STEP_SEC
    setVideoCurrentTime(target)
  } else {
    if (volumeGestureSupported === false) return
    const height = videoGesture.rectHeight
    if (Number.isFinite(height) && height > 0) {
      // 上滑增加音量：dy 为负 -> -dy/height 为正
      const target = videoGesture.startVolume + (-videoGesture.latestDy / height)
      setVideoVolume(target)
      return
    }
    // 兜底：按固定像素映射
    const volPerPxFallback = 1 / 220
    setVideoVolume(videoGesture.startVolume - videoGesture.latestDy * volPerPxFallback)
  }
}

const scheduleVideoGestureApply = () => {
  if (!videoGesture) return
  if (videoGesture.rafId !== null) return
  videoGesture.rafId = requestAnimationFrame(() => {
    if (!videoGesture) return
    videoGesture.rafId = null
    applyVideoGestureFrame()
  })
}

const handleVideoPointerDown = (e: PointerEvent) => {
  if (currentMedia.value.type !== 'video') return
  if (isTargetInPlyrControls(e.target)) return
  if (e.button !== undefined && e.button !== 0) return
  if (videoGesture) return
  const video = videoRef.value
  if (!video) return

  syncVideoPlayState()
  closeSpeedMenu()

  const startTimeSec = Number.isFinite(video.currentTime) ? video.currentTime : 0
  const startVolume = typeof video.volume === 'number' ? video.volume : 1
  const durationSec = Number(video.duration)
  const rect = (e.currentTarget as HTMLElement | null)?.getBoundingClientRect()
  const rectWidth = rect?.width || 0
  const rectHeight = rect?.height || 0
  videoGesture = {
    pointerId: e.pointerId,
    startX: e.clientX,
    startY: e.clientY,
    startAtMs: Date.now(),
    direction: null,
    started: false,
    startTimeSec,
    startVolume,
    durationSec,
    rectWidth,
    rectHeight,
    rafId: null,
    latestDx: 0,
    latestDy: 0
  }

  try {
    ;(e.currentTarget as HTMLElement | null)?.setPointerCapture?.(e.pointerId)
  } catch {
    // ignore
  }
}

const handleVideoPointerMove = (e: PointerEvent) => {
  if (!videoGesture || videoGesture.pointerId !== e.pointerId) return
  const dx = e.clientX - videoGesture.startX
  const dy = e.clientY - videoGesture.startY
  videoGesture.latestDx = dx
  videoGesture.latestDy = dy

  const threshold = VIDEO_GESTURE_THRESHOLD_PX
  if (!videoGesture.started) {
    if (Math.abs(dx) < threshold && Math.abs(dy) < threshold) return
    videoGesture.started = true
    const absDx = Math.abs(dx)
    const absDy = Math.abs(dy)
    const lockRatio = 1.5
    if (absDx >= absDy * lockRatio) videoGesture.direction = 'H'
    else if (absDy >= absDx * lockRatio) videoGesture.direction = 'V'
    else videoGesture.direction = absDx >= absDy ? 'H' : 'V'
    showOverlayDuringGesture()
    // 一旦进入滑动模式，取消可能存在的“单击/双击”判定，避免误触发播放/暂停
    if (tapTimer) {
      clearTimeout(tapTimer)
      tapTimer = null
    }
    tapCount = 0
    lastTapAt = 0
  }

  if (videoGesture.started && e.cancelable) e.preventDefault()
  scheduleVideoGestureApply()
}

const handleVideoPointerUp = (e: PointerEvent) => {
  if (!videoGesture || videoGesture.pointerId !== e.pointerId) return
  const elapsed = Date.now() - videoGesture.startAtMs
  const moved = Math.max(Math.abs(videoGesture.latestDx), Math.abs(videoGesture.latestDy))
  const asTap = elapsed < 500 && moved < VIDEO_GESTURE_THRESHOLD_PX
  const wasStarted = videoGesture.started
  cancelVideoGestureRaf()
  videoGesture = null

  if (asTap && !wasStarted) {
    handleVideoTap(e)
    return
  }

  if (wasStarted) {
    showOverlayWithAutoHide()
  }
}

const handleVideoPointerCancel = (e: PointerEvent) => {
  if (!videoGesture || videoGesture.pointerId !== e.pointerId) return
  cancelVideoGestureRaf()
  videoGesture = null
}

const initPlyr = () => {
  if (typeof window === 'undefined') return
  if (!props.visible) return
  if (currentMedia.value.type !== 'video') return
  const video = videoRef.value
  if (!video) return

  // 同一元素不重复初始化
  if (plyrInstance && (plyrInstance as any).media === video) {
    applyVideoPlaybackRate()
    attachVideoStateListeners()
    syncFullscreenState()
    return
  }

  destroyPlyr()

  try {
    plyrInstance = new Plyr(video, {
      // 去掉中央大按钮（play-large），避免遮挡暂停画面；其余能力与原生 controls 等价。
      controls: ['play', 'progress', 'current-time', 'mute', 'volume', 'fullscreen'],
      // 由组件自己处理“点击播放/暂停”，避免与自定义手势/浮层冲突
      clickToPlay: false,
      // 禁用 iOS 原生全屏接管，确保自定义浮层/按钮在“全屏模式”可见（使用 Plyr fallback/CSS 全屏）
      fullscreen: {
        enabled: true,
        fallback: true,
        iosNative: false,
        container: '.media-preview-video-wrapper'
      },
      // 避免与 MediaPreview 的全局快捷键（←/→/Esc）冲突
      keyboard: { focused: false, global: false }
    })
  } catch (e) {
    console.warn('Plyr init failed:', e)
    plyrInstance = null
    return
  }

  applyVideoPlaybackRate()
  attachVideoStateListeners()
  syncFullscreenState()
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

const isLivePhotoStill = computed(() => {
  const media = currentMedia.value
  if (!media || media.type !== 'image') return false
  const ctx = media.context
  if (!ctx || ctx.provider !== 'douyin') return false
  const key = String(ctx.key || '').trim()
  const idx = Number(ctx.index)
  const vidIdx = Number(ctx.liveVideoIndex)
  if (!key) return false
  if (!Number.isFinite(idx) || idx < 0) return false
  if (!Number.isFinite(vidIdx) || vidIdx < 0) return false
  return true
})

const livePhotoMotionUrl = computed(() => {
  if (!isLivePhotoStill.value) return ''
  const ctx = currentMedia.value.context
  const key = String(ctx?.key || '').trim()
  const vidIdx = Number(ctx?.liveVideoIndex)
  if (!key || !Number.isFinite(vidIdx) || vidIdx < 0) return ''

  const found = realMediaList.value.find((m) => {
    if (!m || m.type !== 'video') return false
    const mctx = m.context
    if (!mctx || mctx.provider !== 'douyin') return false
    if (String(mctx.key || '').trim() !== key) return false
    return Number(mctx.index) === vidIdx
  })
  if (found?.url) return String(found.url || '').trim()

  return `/api/douyin/download?key=${encodeURIComponent(key)}&index=${encodeURIComponent(String(vidIdx))}`
})

const canDownloadLivePhoto = computed(() => !!livePhotoMotionUrl.value && isLivePhotoStill.value)

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

const extractImgUploadPath = (href: string) => {
  const raw = String(href || '').trim()
  if (!raw) return ''

  const marker = '/img/Upload/'
  try {
    const u = new URL(raw, window.location.origin)
    const idx = (u.pathname || '').indexOf(marker)
    if (idx < 0) return ''
    const p = (u.pathname || '').slice(idx + marker.length).replace(/^\/+/, '')
    if (!p || p.includes('..') || p.length > 1024) return ''
    return p
  } catch {
    const idx = raw.indexOf(marker)
    if (idx < 0) return ''
    const p = raw.slice(idx + marker.length).split(/[?#]/)[0]?.replace(/^\/+/, '') || ''
    if (!p || p.includes('..') || p.length > 1024) return ''
    return p
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

const handleDownloadLivePhoto = async (format: 'jpg' | 'zip' = 'jpg') => {
  if (!isLivePhotoStill.value) return
  const ctx = currentMedia.value.context
  const key = String(ctx?.key || '').trim()
  const imageIndex = Number(ctx?.index)
  const videoIndex = Number(ctx?.liveVideoIndex)
  if (!key || !Number.isFinite(imageIndex) || !Number.isFinite(videoIndex)) return

  const apiHref = `/api/douyin/livePhoto?format=${encodeURIComponent(format)}&key=${encodeURIComponent(key)}&imageIndex=${encodeURIComponent(String(imageIndex))}&videoIndex=${encodeURIComponent(String(videoIndex))}`
  const token = localStorage.getItem('authToken')
  if (!token) {
    show('未登录或Token缺失')
    return
  }

  try {
    const resp = await fetch(apiHref, {
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
    const fallbackName = format === 'zip' ? 'livephoto.zip' : 'live.jpg'
    const filename = filenameFromHeader || fallbackName
    triggerBlobDownload(blob, sanitizeFilename(filename) || fallbackName)
  } catch (e) {
    console.error('download live photo failed:', e)
    show('下载失败')
  }
}

const handleDownload = async () => {
  const href = String(currentMedia.value.downloadUrl || currentMedia.value.url || '').trim()
  if (!href) return

  // 上游 imgServer（/img/Upload/*）属于跨域资源：浏览器会忽略 <a download> 导致新开页预览。
  // 这里将其转换为同源代理下载，避免跳出当前页面，并确保文件名可用。
  let apiHref = href
  if (!isSameOriginApiUrl(apiHref)) {
    const uploadPath = extractImgUploadPath(apiHref)
    if (uploadPath) {
      apiHref = `/api/downloadImgUpload?path=${encodeURIComponent(uploadPath)}`
    }
  }

  // /api 资源需要带 Authorization；其他 URL 保持直链下载行为。
  if (!isSameOriginApiUrl(apiHref)) {
    triggerDirectDownload(apiHref)
    return
  }

  const token = localStorage.getItem('authToken')
  if (!token) {
    show('未登录或Token缺失')
    return
  }

  try {
    const resp = await fetch(apiHref, {
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
    clearTapState()
    exitVideoFullscreen()
    clearOverlayHideTimer()
    showVideoOverlayControls.value = false
    cancelVideoGestureRaf()
    videoGesture = null
    closeSpeedMenu()
    handleSpeedPressCancel()
    stopLivePhotoHold()
    resetMediaLoadState()
    try {
      videoRef.value?.pause()
    } catch {
      // ignore
    }
    destroyPlyr()
    nextTick(() => {
      const run = () => {
        applyVideoPlaybackRate()
        initPlyr()
      }
      if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
        window.requestAnimationFrame(() => run())
      } else {
        run()
      }
    })
  }
)

// 拖动辅助变量
let startX = 0
let startY = 0
let initialTranslateX = 0
let initialTranslateY = 0
let hasMoved = false
let livePhotoHoldTimer: ReturnType<typeof setTimeout> | null = null
const livePhotoVisible = ref(false)
let livePhotoSuppressClick = false

const stopLivePhotoHold = () => {
  if (livePhotoHoldTimer) {
    clearTimeout(livePhotoHoldTimer)
    livePhotoHoldTimer = null
  }
  if (livePhotoVisible.value) {
    livePhotoSuppressClick = true
  }
  livePhotoVisible.value = false
  try {
    const v = livePhotoVideoRef.value
    if (v) {
      v.pause()
      v.currentTime = 0
    }
  } catch {
    // ignore
  }
}

const imageStyle = computed(() => {
  return {
    transform: `translate3d(${translateX.value}px, ${translateY.value}px, 0) scale(${scale.value})`
  }
})

const handleClose = () => {
  clearTapState()
  exitVideoFullscreen()
  clearOverlayHideTimer()
  showVideoOverlayControls.value = false
  cancelVideoGestureRaf()
  videoGesture = null
  closeSpeedMenu()
  handleSpeedPressCancel()
  stopLivePhotoHold()
  resetZoom()
  showDetails.value = false
  emit('update:visible', false)
}

const resetZoom = () => {
  scale.value = 1
  translateX.value = 0
  translateY.value = 0
  isDragging.value = false
  stopLivePhotoHold()
}

const handleClick = () => {
  if (livePhotoSuppressClick) {
    livePhotoSuppressClick = false
    return
  }
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
  
  stopLivePhotoHold()
  if (isLivePhotoStill.value && livePhotoMotionUrl.value) {
    // iOS 可能要求播放必须发生在“用户手势”同步上下文：这里先尝试播放（静音+隐藏），
    // 再在长按阈值后显示，避免 setTimeout 触发被判定为非手势播放。
    try {
      const v = livePhotoVideoRef.value
      if (v) {
        v.currentTime = 0
        const p = v.play()
        if (p && typeof (p as any).catch === 'function') {
          ;(p as any).catch(() => {})
        }
      }
    } catch {
      // ignore
    }

    livePhotoHoldTimer = setTimeout(() => {
      if (!isDragging.value) return
      if (hasMoved) return
      livePhotoVisible.value = true
      livePhotoSuppressClick = true
    }, 280)
  }

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
      stopLivePhotoHold()
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
  stopLivePhotoHold()
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
    window.addEventListener('pointerdown', handleDocumentPointerDown, true)
    document.addEventListener('fullscreenchange', handleDocumentFullscreenChange)
    document.addEventListener('webkitfullscreenchange' as any, handleDocumentFullscreenChange)
    syncFullscreenState()
    
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
    nextTick(() => initPlyr())
  } else {
    showDetails.value = false
    clearTapState()
    exitVideoFullscreen()
    clearOverlayHideTimer()
    showVideoOverlayControls.value = false
    cancelVideoGestureRaf()
    videoGesture = null
    closeSpeedMenu()
    handleSpeedPressCancel()
    destroyPlyr()
    window.removeEventListener('keydown', handleKeydown)
    window.removeEventListener('pointerdown', handleDocumentPointerDown, true)
    document.removeEventListener('fullscreenchange', handleDocumentFullscreenChange)
    document.removeEventListener('webkitfullscreenchange' as any, handleDocumentFullscreenChange)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('pointerdown', handleDocumentPointerDown, true)
  document.removeEventListener('fullscreenchange', handleDocumentFullscreenChange)
  document.removeEventListener('webkitfullscreenchange' as any, handleDocumentFullscreenChange)
  destroyPlyr()
  clearTapState()
  exitVideoFullscreen()
  clearOverlayHideTimer()
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

/* Plyr 主题与布局适配（仅作用于 MediaPreview 内的播放器） */
:deep(.plyr) {
  --plyr-color-main: rgb(99 102 241); /* indigo-500 */
  --plyr-tooltip-background: rgba(17, 17, 19, 0.95);
  --plyr-menu-background: rgba(17, 17, 19, 0.95);
  --plyr-menu-color: rgba(255, 255, 255, 0.92);

  margin: auto;
  max-width: 95%;
  max-height: 95%;
  border-radius: 0.5rem;
  overflow: hidden;
  background: #000;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.55);
}

/* 非全屏：Plyr 初始化前的 video 约束（避免视频过大）；Plyr 接管后交给容器约束 */
.media-preview-video {
  display: block;
  margin: 0 auto;
  max-width: 95%;
  max-height: 95%;
}

:deep(.plyr) .media-preview-video {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

/* 全屏模式：让全屏容器占满视口并居中内容 */
.media-preview-video-wrapper:fullscreen,
.media-preview-video-wrapper:-webkit-full-screen,
.media-preview-video-wrapper:-moz-full-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100vw;
  height: 100vh;
  margin: 0;
  padding: 0;
  background: #000;
}

/* 全屏模式：取消 95% 限制与圆角阴影，避免偏移与“黑边”视觉 */
.media-preview-video-wrapper:fullscreen :deep(.plyr),
.media-preview-video-wrapper:-webkit-full-screen :deep(.plyr),
.media-preview-video-wrapper:-moz-full-screen :deep(.plyr) {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  border-radius: 0;
  box-shadow: none;
}

.media-preview-video-wrapper:fullscreen .media-preview-video,
.media-preview-video-wrapper:-webkit-full-screen .media-preview-video,
.media-preview-video-wrapper:-moz-full-screen .media-preview-video {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  border-radius: 0;
  box-shadow: none;
  object-fit: contain;
}

/* 保留原生 controls 作为降级时，隐藏 WebKit/Blink 的中央遮罩按钮（渐进增强） */
.media-preview-video::-webkit-media-controls-overlay-play-button {
  display: none !important;
}
.media-preview-video::-webkit-media-controls-start-playback-button {
  display: none !important;
}
</style>
