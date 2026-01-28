<template>
  <MediaTile
    :src="src"
    :type="type"
    :alt="alt"
    :aspect-ratio="effectiveAspectRatio"
    :show-skeleton="true"
    class="rounded-lg bg-surface-3/50"
    :class="containerClass"
    :controls="type === 'video'"
    @layout="emit('layout')"
    @click="handleClick"
  />
</template>

<script setup lang="ts">
// 聊天媒体渲染组件：兼容旧用法（preview/layout），底层统一使用 MediaTile。
import { computed } from 'vue'
import MediaTile from '@/components/common/MediaTile.vue'

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
  // 增大 min-height 预留空间，减少图片加载时的高度跳变（从 0->300px 变成 150->300px）
  containerClass: 'min-h-[150px] min-w-[100px] max-w-full'
})

const emit = defineEmits<{
  preview: [url: string, type: 'image' | 'video']
  layout: []
}>()

const effectiveAspectRatio = computed(() => {
  const r = props.aspectRatio
  if (r && Number.isFinite(r) && r > 0) return r
  return props.type === 'video' ? 16 / 9 : 4 / 3
})

const handleClick = () => {
  if (!props.previewable) return
  emit('preview', props.src, props.type)
}
</script>
