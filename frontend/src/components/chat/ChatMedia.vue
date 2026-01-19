<template>
  <div
    ref="rootRef"
    class="relative overflow-hidden"
    :class="containerClass"
    :style="aspectStyle"
    @click="handleClick"
  >
    <template v-if="hasError">
      <div class="w-full h-full flex items-center justify-center text-gray-500 flex-col gap-2 p-4 select-none">
        <i :class="errorIconClass"></i>
        <span class="text-xs">{{ errorText }}</span>
      </div>
    </template>

    <template v-else>
      <Skeleton v-if="shouldLoad && !isLoaded" class="absolute inset-0 rounded-lg" />

      <img
        v-if="type === 'image' && shouldLoad"
        :src="src"
        :alt="alt"
        class="w-full h-full object-cover block"
        loading="lazy"
        decoding="async"
        @load="handleLoaded"
        @error="onError"
      />

      <video
        v-else-if="type === 'video' && shouldLoad"
        :src="src"
        class="w-full h-full object-cover block"
        controls
        @loadeddata="handleLoaded"
        @error="onError"
      ></video>

      <Skeleton v-else class="w-full h-full rounded-lg" />
    </template>
  </div>
</template>

<script setup lang="ts">
// 聊天媒体渲染组件：提供加载占位、懒加载与错误兜底，避免图片/视频导致布局抖动。
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Skeleton from '@/components/common/Skeleton.vue'

interface Props {
  type: 'image' | 'video'
  src: string
  alt?: string
  previewable?: boolean
  // 宽高比（width / height），用于预占位减少布局抖动；未知时将使用默认占位比例（image=4/3, video=16/9）。
  aspectRatio?: number
  containerClass?: string
}

const props = withDefaults(defineProps<Props>(), {
  alt: '',
  previewable: true,
  aspectRatio: undefined,
  containerClass: 'rounded-lg bg-gray-900/50 min-h-[100px] min-w-[100px] max-w-full'
})

const emit = defineEmits<{
  preview: [url: string, type: 'image' | 'video']
  layout: []
}>()

const rootRef = ref<HTMLElement | null>(null)
const shouldLoad = ref(false)
const isLoaded = ref(false)
const hasError = ref(false)

let observer: IntersectionObserver | null = null

const effectiveAspectRatio = computed(() => {
  const r = props.aspectRatio
  if (r && Number.isFinite(r) && r > 0) return r
  return props.type === 'video' ? 16 / 9 : 4 / 3
})

const aspectStyle = computed(() => {
  const r = effectiveAspectRatio.value
  if (!r || !Number.isFinite(r) || r <= 0) return undefined
  return { aspectRatio: String(r) }
})

const errorIconClass = computed(() => {
  return props.type === 'image' ? 'fas fa-image-slash text-2xl' : 'fas fa-video-slash text-2xl'
})

const errorText = computed(() => {
  return props.type === 'image' ? '图片加载失败' : '视频加载失败'
})

const onError = () => {
  hasError.value = true
  isLoaded.value = false
  emit('layout')
}

const handleLoaded = () => {
  isLoaded.value = true
  hasError.value = false
  emit('layout')
}

const handleClick = () => {
  if (!props.previewable) return
  if (hasError.value) return
  if (!shouldLoad.value) return
  emit('preview', props.src, props.type)
}

onMounted(() => {
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
