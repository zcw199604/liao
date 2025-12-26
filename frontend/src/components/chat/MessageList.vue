<template>
  <div class="chat-area flex flex-col no-scrollbar" ref="chatBox" @click="$emit('closeAllPanels')" @scroll="handleScroll">
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

      <div class="msg-bubble shadow-sm" :class="msg.isSelf ? 'msg-right' : 'msg-left'">
        <!-- 文本（支持表情解析，双击复制） -->
        <span 
          v-if="!msg.isImage && !msg.isVideo" 
          v-html="parseEmoji(msg.content, emojiMap)"
          @dblclick="copyToClipboard(msg.content)"
          class="cursor-text select-text"
          title="双击复制"
        ></span>

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

    <!-- 底部空间，防止最新消息被遮挡 -->
    <div class="h-4"></div>

    <!-- 回到底部/新消息悬浮按钮 -->
    <transition name="fade">
      <button
        v-if="!isAtBottom || hasNewMessages"
        @click="scrollToBottom(true)"
        class="fixed bottom-24 right-6 rounded-full shadow-xl flex items-center justify-center text-white transition-all z-10 overflow-hidden group"
        :class="hasNewMessages ? 'bg-indigo-600 hover:bg-indigo-700 px-4 py-2 gap-2 h-10 w-auto' : 'bg-[#27272a] hover:bg-[#3f3f46] w-10 h-10'"
        :title="hasNewMessages ? '有新消息' : '回到底部'"
      >
        <i class="fas fa-arrow-down text-sm transition-transform group-hover:translate-y-0.5"></i>
        <span v-if="hasNewMessages" class="text-xs font-bold whitespace-nowrap">新消息</span>
        <span v-if="hasNewMessages" class="absolute -top-1 -right-1 w-3 h-3 bg-red-500 rounded-full animate-pulse"></span>
      </button>
    </transition>
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
import { useToast } from '@/composables/useToast'

const messageStore = useMessageStore()
const { show } = useToast()

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
const isAtBottom = ref(true)
const hasNewMessages = ref(false)

// 检测滚动位置
let scrollTimer: ReturnType<typeof setTimeout> | null = null
const handleScroll = () => {
  if (scrollTimer) clearTimeout(scrollTimer)

  scrollTimer = setTimeout(() => {
    if (!chatBox.value) return
    const { scrollTop, scrollHeight, clientHeight } = chatBox.value
    // 距离底部小于100px认为在底部
    const distanceToBottom = scrollHeight - scrollTop - clientHeight
    const isBottom = distanceToBottom < 100
    
    isAtBottom.value = isBottom
    if (isBottom) {
      hasNewMessages.value = false
    }
  }, 100)
}

const getMessageKey = (msg: ChatMessage): string => {
  const tid = String(msg.tid || '').trim()
  if (tid) return `tid:${tid}`
  const fromUserId = String(msg.fromuser?.id || '')
  const type = String(msg.type || '')
  const time = String(msg.time || '')
  const content = String(msg.content || '')
  return `fallback:${fromUserId}|${type}|${time}|${content}`
}

const scrollToBottom = (force = false) => {
  nextTick(() => {
    if (chatBox.value) {
      chatBox.value.scrollTo({
        top: chatBox.value.scrollHeight,
        behavior: force ? 'smooth' : 'auto'
      })
      hasNewMessages.value = false
    }
  })
}

// 滚动到顶部（查看历史消息）
const scrollToTop = () => {
  nextTick(() => {
    if (chatBox.value) {
      chatBox.value.scrollTop = 0
    }
  })
}

const previewMedia = (url: string, type: 'image' | 'video') => {
  window.dispatchEvent(new CustomEvent('preview-media', {
    detail: { url, type }
  }))
}

const copyToClipboard = async (text: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    show('已复制')
  } catch (err) {
    console.error('复制失败:', err)
    show('复制失败')
  }
}

watch(() => props.messages.length, (newVal, oldVal) => {
  // 忽略加载历史消息时的长度变化
  if (messageStore.isLoadingHistory) return

  // 如果是收到新消息（数量增加且不是历史记录加载）
  if (newVal > oldVal) {
    if (isAtBottom.value) {
      // 如果已经在底部，直接滚动
      scrollToBottom(true)
    } else {
      // 否则显示新消息提示
      hasNewMessages.value = true
    }
  }
}, { flush: 'post' })

onMounted(() => {
  scrollToBottom()
  isAtBottom.value = true
})

defineExpose({
  scrollToBottom,
  scrollToTop
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
