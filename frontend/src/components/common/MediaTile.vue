<template>
  <div
    ref="rootRef"
    class="relative overflow-hidden select-none group"
    :style="aspectStyle"
    @click="handleClick"
  >
    <template v-if="hasError">
      <slot name="error" :type="resolvedType" :src="src">
        <div class="w-full h-full flex items-center justify-center text-gray-500 flex-col gap-2 p-4 select-none">
          <i :class="errorIconClass"></i>
          <span class="text-xs">{{ errorText }}</span>
        </div>
      </slot>
    </template>

    <template v-else>
      <Skeleton
        v-if="showSkeleton && shouldLoad && !isLoaded && resolvedType !== 'file'"
        class="absolute inset-0"
      />

      <img
        v-if="resolvedType === 'image' && shouldLoad"
        :src="src"
        :alt="alt"
        :loading="imgLoading"
        :decoding="imgDecoding"
        :referrerpolicy="imgReferrerPolicy"
        class="block"
        :class="[mediaSizeClass, fitClass, mediaClass]"
        @load="handleLoaded"
        @error="handleError"
      />

      <video
        v-else-if="resolvedType === 'video' && shouldLoad"
        :src="src"
        :poster="poster"
        :controls="controls"
        :autoplay="autoplay"
        :muted="muted"
        :loop="loop"
        :playsinline="playsinline"
        :preload="preload"
        class="block bg-black"
        :class="[mediaSizeClass, fitClass, mediaClass]"
        @loadeddata="handleLoaded"
        @error="handleError"
      ></video>

      <div v-else-if="resolvedType === 'file'" class="w-full h-full flex items-center justify-center bg-gray-800 text-gray-400">
        <slot name="file">
          <i class="fas fa-file text-2xl"></i>
        </slot>
      </div>

      <Skeleton v-else-if="showSkeleton" class="w-full h-full" />

      <div
        v-if="hoverOverlay && resolvedType !== 'file'"
        class="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors pointer-events-none"
      ></div>

      <div v-if="$slots['top-left']" class="absolute top-2 left-2 z-20 flex items-center gap-2" :class="topLeftSlotClass">
        <slot name="top-left"></slot>
      </div>

      <div v-if="$slots['top-right']" class="absolute top-2 right-2 z-20 flex items-center gap-2" :class="topRightSlotClass">
        <slot name="top-right"></slot>
      </div>

      <div v-if="$slots['bottom-left']" class="absolute bottom-2 left-2 z-20 flex items-center gap-2" :class="bottomLeftSlotClass">
        <slot name="bottom-left"></slot>
      </div>

      <div v-if="$slots['bottom-right']" class="absolute bottom-2 right-2 z-20 flex items-center gap-2" :class="bottomRightSlotClass">
        <slot name="bottom-right"></slot>
      </div>

      <div
        v-if="shouldShowCenter"
        class="absolute inset-0 z-10 flex items-center justify-center pointer-events-none"
      >
        <slot name="center">
          <div
            v-if="resolvedType === 'video' && showVideoIndicator && !controls"
            class="rounded-full bg-black/40 backdrop-blur-sm flex items-center justify-center border border-white/15 shadow-lg"
            :class="indicatorBoxClass"
          >
            <i class="fas fa-play text-white" :class="indicatorIconClass"></i>
          </div>
        </slot>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, useSlots, watch } from 'vue'
import Skeleton from '@/components/common/Skeleton.vue'
import { inferMediaTypeFromUrl } from '@/utils/media'

type MediaType = 'image' | 'video' | 'file'

// 全局图片尺寸缓存：URL -> aspectRatio
// 用于虚拟滚动回滚时，避免图片再次加载导致的高度抖动
const mediaSizeCache = new Map<string, number>()

interface Props {
  src: string
  type?: MediaType | 'auto'
  alt?: string
  fit?: 'cover' | 'contain' | 'fill'
  fill?: boolean
  lazy?: boolean
  aspectRatio?: number
  showSkeleton?: boolean
  hoverOverlay?: boolean
  mediaClass?: string
  indicatorSize?: 'sm' | 'md' | 'lg'
  showVideoIndicator?: boolean
  revealTopLeft?: boolean
  revealTopRight?: boolean
  revealBottomLeft?: boolean
  revealBottomRight?: boolean

  // video attrs
  controls?: boolean
  autoplay?: boolean
  muted?: boolean
  loop?: boolean
  playsinline?: boolean
  preload?: 'auto' | 'metadata' | 'none'
  poster?: string

  // img attrs
  imgReferrerPolicy?: ReferrerPolicy
  imgLoading?: 'lazy' | 'eager'
  imgDecoding?: 'async' | 'auto' | 'sync'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'auto',
  alt: '',
  fit: 'cover',
  fill: true,
  lazy: true,
  aspectRatio: undefined,
  showSkeleton: false,
  hoverOverlay: false,
  mediaClass: '',
  indicatorSize: 'md',
  showVideoIndicator: true,
  revealTopLeft: false,
  revealTopRight: false,
  revealBottomLeft: false,
  revealBottomRight: false,
  controls: false,
  autoplay: false,
  muted: false,
  loop: false,
  playsinline: true,
  preload: 'metadata',
  poster: undefined,
  imgReferrerPolicy: undefined,
  imgLoading: 'lazy',
  imgDecoding: 'async'
})

const emit = defineEmits<{
  click: [event: MouseEvent]
  load: [event: Event]
  error: [event: Event]
  layout: []
}>()

const slots = useSlots()

const rootRef = ref<HTMLElement | null>(null)
const shouldLoad = ref(false)
const isLoaded = ref(false)
const hasError = ref(false)
let observer: IntersectionObserver | null = null

const resolvedType = computed<MediaType>(() => {
  if (props.type !== 'auto') return props.type
  return inferMediaTypeFromUrl(props.src)
})

const topLeftSlotClass = computed(() => (props.revealTopLeft ? 'media-tile-reveal' : ''))
const topRightSlotClass = computed(() => (props.revealTopRight ? 'media-tile-reveal' : ''))
const bottomLeftSlotClass = computed(() => (props.revealBottomLeft ? 'media-tile-reveal' : ''))
const bottomRightSlotClass = computed(() => (props.revealBottomRight ? 'media-tile-reveal' : ''))

// 获取有效的 aspectRatio：优先使用 props，其次使用缓存
const effectiveAspectRatio = computed(() => {
  // 1. 优先使用传入的 aspectRatio
  const r = props.aspectRatio
  if (r && Number.isFinite(r) && r > 0) return r

  // 2. 其次使用缓存的尺寸（虚拟滚动回滚时有效）
  const cached = mediaSizeCache.get(props.src)
  if (cached && Number.isFinite(cached) && cached > 0) return cached

  // 3. 没有缓存则返回 undefined，由外层决定默认占位
  return undefined
})

const aspectStyle = computed(() => {
  const r = effectiveAspectRatio.value
  if (!r || !Number.isFinite(r) || r <= 0) return undefined
  return { aspectRatio: String(r) }
})

const fitClass = computed(() => {
  switch (props.fit) {
    case 'contain':
      return 'object-contain'
    case 'fill':
      return 'object-fill'
    default:
      return 'object-cover'
  }
})

const mediaSizeClass = computed(() => {
  return props.fill ? 'w-full h-full' : 'max-w-full max-h-full'
})

const errorIconClass = computed(() => {
  switch (resolvedType.value) {
    case 'video':
      return 'fas fa-video-slash text-2xl'
    case 'file':
      return 'fas fa-file-excel text-2xl'
    default:
      return 'fas fa-image-slash text-2xl'
  }
})

const errorText = computed(() => {
  switch (resolvedType.value) {
    case 'video':
      return '视频加载失败'
    case 'file':
      return '文件不可用'
    default:
      return '图片加载失败'
  }
})

const indicatorBoxClass = computed(() => {
  switch (props.indicatorSize) {
    case 'sm':
      return 'w-6 h-6'
    case 'lg':
      return 'w-10 h-10'
    default:
      return 'w-8 h-8'
  }
})

const indicatorIconClass = computed(() => {
  switch (props.indicatorSize) {
    case 'sm':
      return 'text-[10px] ml-0.5'
    case 'lg':
      return 'text-sm ml-0.5'
    default:
      return 'text-xs ml-0.5'
  }
})

const shouldShowCenter = computed(() => {
  if (slots.center) return true
  return resolvedType.value === 'video' && props.showVideoIndicator && !props.controls
})

const handleError = (e: Event) => {
  hasError.value = true
  isLoaded.value = false
  emit('error', e)
  emit('layout')
}

const handleLoaded = (e: Event) => {
  isLoaded.value = true
  hasError.value = false

  // 缓存媒体尺寸，用于虚拟滚动回滚时避免再次抖动
  const target = e.target as HTMLImageElement | HTMLVideoElement
  if (target && props.src) {
    let width = 0
    let height = 0

    if (target instanceof HTMLImageElement) {
      width = target.naturalWidth
      height = target.naturalHeight
    } else if (target instanceof HTMLVideoElement) {
      width = target.videoWidth
      height = target.videoHeight
    }

    if (width > 0 && height > 0) {
      mediaSizeCache.set(props.src, width / height)
    }
  }

  emit('load', e)
  emit('layout')
}

const handleClick = (e: MouseEvent) => {
  emit('click', e)
}

watch(() => props.src, () => {
  isLoaded.value = false
  hasError.value = false
})

onMounted(() => {
  if (!props.lazy) {
    shouldLoad.value = true
    return
  }

  if (!rootRef.value) {
    shouldLoad.value = true
    return
  }

  if (typeof IntersectionObserver === 'undefined') {
    shouldLoad.value = true
    return
  }

  observer = new IntersectionObserver(entries => {
    const entry = entries[0]
    if (!entry) return
    if (entry.isIntersecting) {
      shouldLoad.value = true
      observer?.disconnect()
      observer = null
    }
  }, { root: null, threshold: 0.1 })

  observer.observe(rootRef.value)
})

onUnmounted(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style scoped>
@media (hover: hover) and (pointer: fine) {
  .media-tile-reveal {
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.2s ease-in-out;
  }

  .group:hover .media-tile-reveal {
    opacity: 1;
    pointer-events: auto;
  }
}

@media (hover: none) {
  .media-tile-reveal {
    opacity: 1;
    pointer-events: auto;
  }
}
</style>
