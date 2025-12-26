<template>
  <div
    class="flex mb-3"
    :class="message.isSelf ? 'justify-end' : 'justify-start'"
  >
    <div
      class="msg-bubble shadow-sm"
      :class="message.isSelf ? 'msg-right' : 'msg-left'"
    >
      <!-- 文本消息 -->
      <div v-if="!message.isImage && !message.isVideo" v-html="parsedContent"></div>

      <!-- 图片消息 -->
      <div v-else-if="message.isImage" class="cursor-pointer group relative" @click="previewImage">
        <img
          :src="imageUrl"
          alt="图片"
          class="rounded-lg object-cover bg-gray-900/50"
          :class="[
             'max-w-full sm:max-w-sm',
             'max-h-[40vh] min-h-[100px] min-w-[100px]'
          ]"
          @error="handleImageError"
        />
        <!-- 放大图标提示 -->
        <div class="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition flex items-center justify-center opacity-0 group-hover:opacity-100 rounded-lg">
          <i class="fas fa-search-plus text-white/80 drop-shadow-md"></i>
        </div>
      </div>

      <!-- 视频消息 -->
      <div v-else-if="message.isVideo" class="cursor-pointer relative group" @click="previewVideo">
        <video
          :src="videoUrl"
          class="rounded-lg bg-black max-w-full sm:max-w-sm max-h-[40vh]"
        ></video>
        <!-- 播放覆盖层，点击预览 -->
        <div class="absolute inset-0 flex items-center justify-center bg-black/20 group-hover:bg-black/30 transition rounded-lg">
          <div class="w-10 h-10 rounded-full bg-black/50 backdrop-blur-sm flex items-center justify-center text-white border border-white/20 shadow-lg group-hover:scale-110 transition">
            <i class="fas fa-play text-xs ml-0.5"></i>
          </div>
        </div>
      </div>

      <!-- 时间戳 -->
      <div
        class="text-[10px] mt-1.5 opacity-50 select-none"
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
