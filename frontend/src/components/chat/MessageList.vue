<template>
  <div class="relative h-full">
    <DynamicScroller
      ref="scrollerRef"
      v-bind="attrs"
      class="chat-area flex flex-col no-scrollbar"
      :items="renderItems"
      :min-item-size="80"
      :prerender="20"
      key-field="key"
      @click="$emit('closeAllPanels')"
      @scroll.passive="handleScroll"
    >
      <template #default="{ item: row, index, active }">
        <DynamicScrollerItem
          :item="row"
          :active="active"
          :data-index="index"
          :size-dependencies="getSizeDependencies(row)"
        >
          <!-- 加载更多历史消息按钮 -->
          <div v-if="row.kind === 'loadMore'" class="flex justify-center py-3">
            <button
              @click="$emit('loadMore')"
              :disabled="loadingMore || !canLoadMore"
              class="px-4 py-2 bg-surface-3 text-fg-muted text-sm rounded-full active:bg-surface-hover disabled:opacity-50 border border-line"
            >
              <span v-if="loadingMore">加载中...</span>
              <span v-else>{{ canLoadMore ? '查看历史消息' : '暂无更多历史消息' }}</span>
            </button>
          </div>

          <!-- 首次加载历史记录：骨架屏占位，减少布局跳动 -->
          <div
            v-else-if="row.kind === 'skeleton'"
            class="flex flex-col w-full mb-3"
            :class="row.align === 'left' ? 'items-start' : 'items-end'"
          >
            <div class="msg-bubble shadow-sm" :class="row.align === 'left' ? 'msg-left' : 'msg-right'">
              <div class="flex flex-col gap-2">
                <Skeleton class="h-3 w-40 rounded" />
                <Skeleton class="h-3 w-28 rounded" />
              </div>
            </div>
          </div>

          <div
            v-else-if="row.kind === 'message'"
            class="flex flex-col w-full mb-3"
            :class="row.message.isSelf ? 'items-end' : 'items-start'"
          >
            <!-- 昵称 + 时间 -->
            <div
              class="text-xs text-fg-subtle mb-1 flex items-center gap-2"
              :class="row.message.isSelf ? 'mr-1 justify-end' : 'ml-1'"
            >
              <span v-if="row.message.fromuser?.nickname" class="font-medium text-fg-muted">{{ row.message.fromuser.nickname }}</span>
              <span v-if="row.message.time">{{ formatTime(row.message.time) }}</span>
            </div>

            <div class="msg-bubble shadow-sm" :class="row.message.isSelf ? 'msg-right' : 'msg-left'">
              <template v-if="row.message.segments && row.message.segments.length">
                <div class="flex flex-col gap-2">
                  <template v-for="(seg, idx) in row.message.segments" :key="idx">
                    <span
                      v-if="seg.kind === 'text'"
                      v-html="parseEmoji(seg.text, emojiMap)"
                      @dblclick="copyToClipboard(row.message.content)"
                      class="cursor-text select-text"
                      title="双击复制"
                    ></span>

                    <ChatMedia
                      v-else-if="seg.kind === 'image'"
                      type="image"
                      :src="getMediaUrl(seg.url)"
                      @layout="handleMediaLayout"
                      @preview="(url) => previewMedia(url, 'image')"
                    />

                    <ChatMedia
                      v-else-if="seg.kind === 'video'"
                      type="video"
                      :src="getMediaUrl(seg.url)"
                      :previewable="false"
                      @layout="handleMediaLayout"
                    />

                    <div
                      v-else-if="seg.kind === 'file'"
                      class="p-3 rounded-lg flex items-center gap-3 min-w-[200px] max-w-sm cursor-pointer transition border group"
                      :class="row.message.isSelf ? 'bg-white/10 hover:bg-white/20 border-white/10 text-white' : 'bg-surface-3/70 hover:bg-surface-hover border-line text-fg'"
                      @click="downloadFile(getMediaUrl(seg.url))"
                    >
                      <div
                        class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
                        :class="row.message.isSelf ? 'bg-gray-800 text-indigo-400' : 'bg-surface text-indigo-600 dark:text-indigo-400 border border-line'"
                      >
                        <i class="fas fa-file text-2xl"></i>
                      </div>
                      <div class="flex-1 overflow-hidden min-w-0">
                        <div
                          class="text-sm truncate font-medium"
                          :class="row.message.isSelf ? 'text-white/90' : 'text-fg'"
                          :title="getDownloadFileName(getMediaUrl(seg.url))"
                        >
                          {{ getDownloadFileName(getMediaUrl(seg.url)) }}
                        </div>
                        <div class="text-xs mt-0.5" :class="row.message.isSelf ? 'text-white/50' : 'text-fg-muted'">点击下载</div>
                      </div>
                      <div
                        class="w-8 h-8 rounded-full flex items-center justify-center transition"
                        :class="row.message.isSelf ? 'bg-white/5 text-gray-300 group-hover:bg-white/10 group-hover:text-white' : 'bg-surface/60 text-fg-muted group-hover:bg-surface-hover group-hover:text-fg border border-line'"
                      >
                        <i class="fas fa-download text-sm"></i>
                      </div>
                    </div>
                  </template>
                </div>
              </template>

              <template v-else>
                <!-- 文本（支持表情解析，双击复制） -->
                <span
                  v-if="!row.message.isImage && !row.message.isVideo && !row.message.isFile"
                  v-html="parseEmoji(row.message.content, emojiMap)"
                  @dblclick="copyToClipboard(row.message.content)"
                  class="cursor-text select-text"
                  title="双击复制"
                ></span>

                <!-- 图片 -->
                <template v-else-if="row.message.isImage">
                  <ChatMedia
                    type="image"
                    :src="getMediaUrl(row.message.imageUrl || row.message.content || '')"
                    @layout="handleMediaLayout"
                    @preview="(url) => previewMedia(url, 'image')"
                  />
                </template>

                <!-- 视频 -->
                <ChatMedia
                  v-else-if="row.message.isVideo"
                  type="video"
                  :src="getMediaUrl(row.message.videoUrl || row.message.content || '')"
                  :previewable="false"
                  @layout="handleMediaLayout"
                />

                <!-- 文件 -->
                <div
                  v-else-if="row.message.isFile"
                  class="p-3 rounded-lg flex items-center gap-3 min-w-[200px] max-w-sm cursor-pointer transition border group"
                  :class="row.message.isSelf ? 'bg-white/10 hover:bg-white/20 border-white/10 text-white' : 'bg-surface-3/70 hover:bg-surface-hover border-line text-fg'"
                  @click="downloadFile(getMediaUrl(row.message.fileUrl || row.message.content || ''))"
                >
                  <div
                    class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
                    :class="row.message.isSelf ? 'bg-gray-800 text-indigo-400' : 'bg-surface text-indigo-600 dark:text-indigo-400 border border-line'"
                  >
                    <i class="fas fa-file text-2xl"></i>
                  </div>
                  <div class="flex-1 overflow-hidden min-w-0">
                    <div
                      class="text-sm truncate font-medium"
                      :class="row.message.isSelf ? 'text-white/90' : 'text-fg'"
                      :title="getDownloadFileName(getMediaUrl(row.message.fileUrl || row.message.content || ''))"
                    >
                      {{ getDownloadFileName(getMediaUrl(row.message.fileUrl || row.message.content || '')) }}
                    </div>
                    <div class="text-xs mt-0.5" :class="row.message.isSelf ? 'text-white/50' : 'text-fg-muted'">点击下载</div>
                  </div>
                  <div
                    class="w-8 h-8 rounded-full flex items-center justify-center transition"
                    :class="row.message.isSelf ? 'bg-white/5 text-gray-300 group-hover:bg-white/10 group-hover:text-white' : 'bg-surface/60 text-fg-muted group-hover:bg-surface-hover group-hover:text-fg border border-line'"
                  >
                    <i class="fas fa-download text-sm"></i>
                  </div>
                </div>
              </template>
            </div>

            <!-- 发送状态（仅自己发送的乐观消息） -->
            <div
              v-if="row.message.isSelf && (row.message.sendStatus === 'sending' || row.message.sendStatus === 'failed')"
              class="mt-1 text-[10px] flex items-center gap-2 select-none"
              :class="row.message.isSelf ? 'justify-end mr-1' : 'ml-1'"
            >
              <template v-if="row.message.sendStatus === 'sending'">
                <span class="text-fg/50">发送中…</span>
              </template>
              <template v-else>
                <span class="text-red-400/90">发送失败</span>
                <button
                  class="text-red-300 hover:text-red-200 underline underline-offset-2"
                  @click.stop="$emit('retry', row.message)"
                >
                  重试
                </button>
              </template>
            </div>
          </div>

          <!-- 正在输入提示 -->
          <div v-else-if="row.kind === 'typing'" class="flex w-full justify-start mb-3">
            <div class="msg-bubble msg-left flex items-center gap-2">
              <span class="text-fg-muted">正在输入</span>
              <div class="flex gap-1">
                <span class="w-2 h-2 bg-fg-muted/70 rounded-full animate-bounce" style="animation-delay: 0s"></span>
                <span class="w-2 h-2 bg-fg-muted/70 rounded-full animate-bounce" style="animation-delay: 0.2s"></span>
                <span class="w-2 h-2 bg-fg-muted/70 rounded-full animate-bounce" style="animation-delay: 0.4s"></span>
              </div>
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>

    <!-- 回到底部/新消息悬浮按钮 -->
    <transition name="fade">
      <button
        v-if="!isAtBottom || hasNewMessages"
        @click="scrollToBottom(true)"
        class="fixed right-6 rounded-full shadow-xl flex items-center justify-center transition-all z-10 overflow-hidden group"
        :style="{ bottom: `${props.floatingBottomOffsetPx}px` }"
        :class="hasNewMessages ? 'bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 gap-2 h-10 w-auto' : 'bg-surface-3 hover:bg-surface-hover text-fg w-10 h-10 border border-line'"
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
// 消息列表：支持骨架屏、乐观发送状态、媒体渲染与虚拟滚动，保持长对话滚动流畅。
import { computed, nextTick, onMounted, onUnmounted, ref, useAttrs, watch } from 'vue'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import type { ChatMessage } from '@/types'
import { formatTime } from '@/utils/time'
import { parseEmoji } from '@/utils/string'
import { copyToClipboard as copyTextToClipboard } from '@/utils/clipboard'
import { emojiMap } from '@/constants/emoji'
import { useUpload } from '@/composables/useUpload'
import { useMessageStore } from '@/stores/message'
import { useToast } from '@/composables/useToast'
import Skeleton from '@/components/common/Skeleton.vue'
import ChatMedia from '@/components/chat/ChatMedia.vue'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()
const messageStore = useMessageStore()
const { show } = useToast()

interface Props {
  messages: ChatMessage[]
  isTyping: boolean
  loadingMore: boolean
  canLoadMore: boolean
  floatingBottomOffsetPx?: number
}

const props = withDefaults(defineProps<Props>(), {
  floatingBottomOffsetPx: 96
})

defineEmits<{
  loadMore: []
  closeAllPanels: []
  retry: [ChatMessage]
}>()

type RenderItem =
  | { kind: 'loadMore'; key: 'loadMore' }
  | { kind: 'skeleton'; key: string; align: 'left' | 'right' }
  | { kind: 'message'; key: string; message: ChatMessage }
  | { kind: 'typing'; key: 'typing' }

const scrollerRef = ref<any>(null)
const { getMediaUrl } = useUpload()
const isAtBottom = ref(true)
const hasNewMessages = ref(false)

let pendingScrollBehavior: ScrollBehavior = 'auto'
let scrollScheduled = false

const scheduleScrollToBottom = (behavior: ScrollBehavior) => {
  if (behavior === 'smooth') pendingScrollBehavior = 'smooth'
  if (scrollScheduled) return
  scrollScheduled = true

  const run = async () => {
    scrollScheduled = false
    const b = pendingScrollBehavior
    pendingScrollBehavior = 'auto'

    await nextTick()

    const scroller = scrollerRef.value
    if (scroller?.scrollToBottom) {
      scroller.scrollToBottom()
    } else if (scroller?.scrollToItem) {
      scroller.scrollToItem(renderItems.value.length - 1)
    }

    const el = getScrollerEl()
    el?.scrollTo({ top: el.scrollHeight, behavior: b })
  }

  if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
    window.requestAnimationFrame(() => void run())
  } else {
    void run()
  }
}

const showHistorySkeleton = computed(() => messageStore.isLoadingHistory && props.messages.length === 0)

const getMessageKey = (msg: ChatMessage): string => {
  const clientId = String(msg.clientId || '').trim()
  if (clientId) return `cid:${clientId}`
  const tid = String(msg.tid || '').trim()
  if (tid) return `tid:${tid}`
  const fromUserId = String(msg.fromuser?.id || '')
  const type = String(msg.type || '')
  const time = String(msg.time || '')
  const content = String(msg.content || '')
  return `fallback:${fromUserId}|${type}|${time}|${content}`
}

const renderItems = computed<RenderItem[]>(() => {
  const rows: RenderItem[] = [{ kind: 'loadMore', key: 'loadMore' }]

  if (showHistorySkeleton.value) {
    for (let i = 0; i < 6; i++) {
      rows.push({
        kind: 'skeleton',
        key: `sk:${i}`,
        align: i % 2 === 0 ? 'left' : 'right'
      })
    }
  } else {
    for (const msg of props.messages) {
      rows.push({ kind: 'message', key: getMessageKey(msg), message: msg })
    }
  }

  if (props.isTyping) rows.push({ kind: 'typing', key: 'typing' })
  return rows
})

// 精简 sizeDependencies：只保留真正影响消息高度的字段
// 移除 sendStatus（只是图标变化）、imageUrl/videoUrl/fileUrl（URL 变化不影响高度，高度变化由 layout 事件处理）
const getSizeDependencies = (row: RenderItem) => {
  if (row.kind === 'message') {
    const m = row.message
    return [
      m.content,           // 文本内容变化影响高度
      m.segments?.length ?? 0  // segments 数量变化影响高度
    ]
  }
  if (row.kind === 'loadMore') return [props.loadingMore, props.canLoadMore]
  if (row.kind === 'typing') return [props.isTyping]
  return [row.kind]
}

const getScrollerEl = (): HTMLElement | null => {
  const scroller = scrollerRef.value
  const el = scroller?.$el as HTMLElement | undefined
  return el || null
}

// 移除 ResizeObserver 的自动 scrollToBottom 逻辑
// 原因：图片加载导致容器高度变化时会频繁触发，与 handleMediaLayout 竞争造成抖动
// 改为依赖 CSS overflow-anchor 和智能粘滞逻辑处理滚动位置

// 检测滚动位置
let scrollTimer: ReturnType<typeof setTimeout> | null = null
const handleScroll = () => {
  if (scrollTimer) clearTimeout(scrollTimer)

  scrollTimer = setTimeout(() => {
    const el = getScrollerEl()
    if (!el) return

    const { scrollTop, scrollHeight, clientHeight } = el
    const distanceToBottom = scrollHeight - scrollTop - clientHeight
    const isBottom = distanceToBottom < 100

    isAtBottom.value = isBottom
    if (isBottom) {
      hasNewMessages.value = false
    }
  }, 100)
}

const scrollToBottom = (force = false) => {
  hasNewMessages.value = false
  if (force) isAtBottom.value = true
  scheduleScrollToBottom(force ? 'smooth' : 'auto')
}

// 智能粘滞逻辑：实时检测滚动位置，只有用户在底部附近时才自动滚动
// 使用 debounce 减少频繁调用，避免多张图片同时加载时的抖动
let mediaLayoutTimer: ReturnType<typeof setTimeout> | null = null
const MEDIA_LAYOUT_DEBOUNCE_MS = 150
const STICK_TO_BOTTOM_THRESHOLD = 80 // 距离底部小于此值时认为用户在底部

const handleMediaLayout = () => {
  if (mediaLayoutTimer) clearTimeout(mediaLayoutTimer)

  mediaLayoutTimer = setTimeout(() => {
    const el = getScrollerEl()
    if (!el) return

    // 实时计算距离底部的距离，不依赖可能过时的 isAtBottom 状态
    const { scrollTop, scrollHeight, clientHeight } = el
    const distanceToBottom = scrollHeight - scrollTop - clientHeight

    // 只有用户本来就在底部附近时才自动滚动，避免打断用户浏览历史消息
    if (distanceToBottom < STICK_TO_BOTTOM_THRESHOLD) {
      scrollToBottom()
    }
  }, MEDIA_LAYOUT_DEBOUNCE_MS)
}

// 滚动到顶部（查看历史消息）
const scrollToTop = () => {
  nextTick(() => {
    const scroller = scrollerRef.value
    if (scroller?.scrollToItem) {
      scroller.scrollToItem(0)
      return
    }

    const el = getScrollerEl()
    if (el) el.scrollTop = 0
  })
}

const previewMedia = (url: string, type: 'image' | 'video') => {
  window.dispatchEvent(
    new CustomEvent('preview-media', {
      detail: { url, type }
    })
  )
}

const getDownloadFileName = (url: string): string => {
  const raw = String(url || '')
  if (!raw) return '未知文件'
  try {
    const u = new URL(raw)
    return decodeURIComponent(u.pathname.split('/').pop() || '未知文件')
  } catch {
    return raw.split('/').pop() || '未知文件'
  }
}

const downloadFile = (url: string) => {
  const href = String(url || '')
  if (!href) return

  const link = document.createElement('a')
  link.href = href
  link.download = getDownloadFileName(href)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const copyToClipboard = async (text: string) => {
  if (!text) return
  const ok = await copyTextToClipboard(text)
  show(ok ? '已复制' : '复制失败')
}

watch(
  () => props.messages.length,
  (newVal, oldVal) => {
    // 忽略加载历史消息时的长度变化
    if (messageStore.isLoadingHistory) return

    if (newVal > oldVal) {
      if (isAtBottom.value) {
        scrollToBottom()
      } else {
        hasNewMessages.value = true
      }
    }
  },
  { flush: 'post' }
)

onMounted(() => {
  scrollToBottom()
  isAtBottom.value = true
})

onUnmounted(() => {
  // 清理定时器
  if (scrollTimer) clearTimeout(scrollTimer)
  if (mediaLayoutTimer) clearTimeout(mediaLayoutTimer)
})

const getIsAtBottom = () => isAtBottom.value

defineExpose({
  scrollToBottom,
  scrollToTop,
  getIsAtBottom
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

.chat-area {
  overflow-anchor: auto;
}
</style>
