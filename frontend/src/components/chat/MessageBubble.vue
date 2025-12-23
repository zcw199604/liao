<template>
  <div
    class="flex mb-3"
    :class="message.isSelf ? 'justify-end' : 'justify-start'"
  >
    <div
      class="msg-bubble"
      :class="message.isSelf ? 'msg-right' : 'msg-left'"
    >
      <!-- 文本消息 -->
      <div v-if="!message.isImage && !message.isVideo" v-html="parsedContent"></div>

      <!-- 图片消息 -->
      <div v-else-if="message.isImage" class="cursor-pointer" @click="previewImage">
        <img
          :src="imageUrl"
          alt="图片"
          class="max-w-full rounded"
          style="max-height: 300px;"
          @error="handleImageError"
        />
      </div>

      <!-- 视频消息 -->
      <div v-else-if="message.isVideo" class="cursor-pointer" @click="previewVideo">
        <video
          :src="videoUrl"
          class="max-w-full rounded"
          style="max-height: 300px;"
          controls
        ></video>
      </div>

      <!-- 时间戳 -->
      <div
        class="text-xs mt-1 opacity-70"
        :class="message.isSelf ? 'text-right' : 'text-left'"
      >
        {{ formatTime(message.time) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ChatMessage } from '@/types'
import { formatTime } from '@/utils/time'
import { parseEmoji } from '@/utils/string'
import { emojiMap } from '@/constants/emoji'
import { useUpload } from '@/composables/useUpload'

interface Props {
  message: ChatMessage
}

const props = defineProps<Props>()
const { getMediaUrl } = useUpload()

const parsedContent = computed(() => {
  if (!props.message.content) return ''
  return parseEmoji(props.message.content, emojiMap)
})

const imageUrl = computed(() => {
  return getMediaUrl(props.message.imageUrl || props.message.content || '')
})

const videoUrl = computed(() => {
  return getMediaUrl(props.message.videoUrl || props.message.content || '')
})

const previewImage = () => {
  window.dispatchEvent(new CustomEvent('preview-media', {
    detail: { url: imageUrl.value, type: 'image' }
  }))
}

const previewVideo = () => {
  window.dispatchEvent(new CustomEvent('preview-media', {
    detail: { url: videoUrl.value, type: 'video' }
  }))
}

const handleImageError = (e: Event) => {
  console.error('图片加载失败:', imageUrl.value)
}
</script>
