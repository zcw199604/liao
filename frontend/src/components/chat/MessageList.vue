<template>
  <div class="chat-area flex flex-col no-scrollbar" ref="chatBox" @click="$emit('closeAllPanels')">
    <!-- 加载更多历史消息按钮 -->
    <div class="flex justify-center py-3">
      <button
        @click="$emit('loadMore')"
        :disabled="loadingMore || !canLoadMore"
        class="px-4 py-2 bg-[#27272a] text-gray-400 text-sm rounded-full active:bg-[#3a3a3f] disabled:opacity-50"
      >
        <span v-if="loadingMore">加载中...</span>
        <span v-else>{{ canLoadMore ? '查看历史消息' : '暂无更多历史消息' }}</span>
      </button>
    </div>

    <div
      v-for="msg in messages"
      :key="getMessageKey(msg)"
      class="flex flex-col w-full mb-3"
      :class="msg.isSelf ? 'items-end' : 'items-start'"
    >
      <!-- 昵称 + 时间 -->
      <div
        class="text-xs text-gray-500 mb-1 flex items-center gap-2"
        :class="msg.isSelf ? 'mr-1 justify-end' : 'ml-1'"
      >
        <span v-if="msg.fromuser?.nickname" class="font-medium">{{ msg.fromuser.nickname }}</span>
        <span v-if="msg.time">{{ formatTime(msg.time) }}</span>
      </div>

      <div class="msg-bubble" :class="msg.isSelf ? 'msg-right' : 'msg-left'">
        <!-- 文本（支持表情解析） -->
        <span v-if="!msg.isImage && !msg.isVideo" v-html="parseEmoji(msg.content, emojiMap)"></span>

        <!-- 图片 -->
        <img
          v-if="msg.isImage"
          :src="getMediaUrl(msg.imageUrl || msg.content || '')"
          class="rounded-lg max-w-full block cursor-pointer"
          @click="previewMedia(getMediaUrl(msg.imageUrl || msg.content || ''), 'image')"
        />

        <!-- 视频 -->
        <video
          v-if="msg.isVideo"
          :src="getMediaUrl(msg.videoUrl || msg.content || '')"
          controls
          class="rounded-lg max-w-full block"
        ></video>
      </div>
    </div>

    <!-- 正在输入提示 -->
    <div v-if="isTyping" class="flex w-full justify-start mb-3">
      <div class="msg-bubble msg-left flex items-center gap-2">
        <span class="text-gray-400">正在输入</span>
        <div class="flex gap-1">
          <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0s"></span>
          <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.2s"></span>
          <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.4s"></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue'
import type { ChatMessage } from '@/types'
import { formatTime } from '@/utils/time'
import { parseEmoji } from '@/utils/string'
import { emojiMap } from '@/constants/emoji'
import { useUpload } from '@/composables/useUpload'
import { useMessageStore } from '@/stores/message'

const messageStore = useMessageStore()

interface Props {
  messages: ChatMessage[]
  isTyping: boolean
  loadingMore: boolean
  canLoadMore: boolean
}

const props = defineProps<Props>()
defineEmits<{
  'loadMore': []
  'closeAllPanels': []
}>()

const chatBox = ref<HTMLElement | null>(null)
const { getMediaUrl } = useUpload()

const getMessageKey = (msg: ChatMessage): string => {
  const tid = String(msg.tid || '').trim()
  if (tid) return `tid:${tid}`

  const fromUserId = String(msg.fromuser?.id || '')
  const type = String(msg.type || '')
  const time = String(msg.time || '')
  const content = String(msg.content || '')
  return `fallback:${fromUserId}|${type}|${time}|${content}`
}

const scrollToBottom = () => {
  nextTick(() => {
    if (chatBox.value) {
      chatBox.value.scrollTop = chatBox.value.scrollHeight
    }
  })
}

const previewMedia = (url: string, type: 'image' | 'video') => {
  window.dispatchEvent(new CustomEvent('preview-media', {
    detail: { url, type }
  }))
}

watch(() => props.messages.length, () => {
  // 如果正在加载历史消息，不自动滚动
  if (messageStore.isLoadingHistory) {
    return
  }
  scrollToBottom()
}, { flush: 'post' })

onMounted(() => {
  scrollToBottom()
})

defineExpose({
  scrollToBottom
})
</script>
