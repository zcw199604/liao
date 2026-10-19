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

            <div
              class="msg-bubble shadow-sm"
              :class="row.message.isSelf ? 'msg-right' : 'msg-left'"
              @contextmenu="handleMessageContextMenu(row.message, $event)"
              @touchstart.passive="startMessageLongPress(row.message, $event)"
              @touchmove.passive="cancelMessageLongPress"
              @touchend="cancelMessageLongPress"
              @touchcancel="cancelMessageLongPress"
            >
              <template v-if="row.message.segments && row.message.segments.length">
                <div class="flex flex-col gap-2">
                  <template v-for="(seg, idx) in row.message.segments" :key="idx">
                    <span
                      v-if="seg.kind === 'text'"
                      v-html="parseEmoji(seg.text, emojiMap)"
                      @dblclick="copyToClipboard(row.message.content)"
                      class="msg-text cursor-text select-text"
                      title="双击或长按复制"
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
                      @layout="handleMediaLayout"
                      @preview="(url) => previewMedia(url, 'video')"
                    />

                    <ChatFileCard
                      v-else-if="seg.kind === 'file'"
                      :url="getMediaUrl(seg.url)"
                      :is-self="row.message.isSelf"
                    />
                  </template>
                </div>
              </template>

              <template v-else>
                <!-- 文本（支持表情解析，双击/长按复制） -->
                <span
                  v-if="!row.message.isImage && !row.message.isVideo && !row.message.isFile"
                  v-html="parseEmoji(row.message.content, emojiMap)"
                  @dblclick="copyToClipboard(row.message.content)"
                  class="msg-text cursor-text select-text"
                  title="双击或长按复制"
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
                  @layout="handleMediaLayout"
                  @preview="(url) => previewMedia(url, 'video')"
                />

                <!-- 文件 -->
                <ChatFileCard
                  v-else-if="row.message.isFile"
                  :url="getMediaUrl(row.message.fileUrl || row.message.content || '')"
                  :is-self="row.message.isSelf"
                />
              </template>
            </div>

            <!-- 发送状态（仅自己发送的乐观消息） -->
            <div
              v-if="row.message.isSelf && (row.message.sendStatus === 'sending' || row.message.sendStatus === 'failed')"
              class="mt-1 text-[10px] flex items-center gap-2 select-none justify-end mr-1"
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

    <!-- 消息操作菜单（长按/右键） -->
    <Teleport to="body">
      <div
        v-if="messageMenu"
        class="fixed inset-0 z-[80]"
        data-testid="message-menu-backdrop"
        @click="closeMessageMenu"
        @contextmenu.prevent="closeMessageMenu"
      >
        <div
          class="fixed ui-card-sm py-1 min-w-[120px]"
          data-testid="message-menu"
          :style="{ left: `${messageMenu.x}px`, top: `${messageMenu.y}px` }"
          @click.stop
        >
          <button
            v-if="messageMenu.copyText"
            data-testid="message-menu-copy"
            class="w-full px-4 py-2.5 text-left text-sm hover:bg-surface-3 text-fg flex items-center gap-3"
            @click="handleMenuCopy"
          >
            <i class="far fa-copy w-4 text-fg-muted"></i>
            <span>复制</span>
          </button>
          <button
            v-if="messageMenu.canRetry"
            data-testid="message-menu-retry"
            class="w-full px-4 py-2.5 text-left text-sm hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line"
            @click="handleMenuRetry"
          >
            <i class="fas fa-redo-alt w-4 text-fg-muted"></i>
            <span>重新发送</span>
          </button>
        </div>
      </div>
    </Teleport>
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
import ChatFileCard from '@/components/chat/ChatFileCard.vue'

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

const emit = defineEmits<{
  loadMore: []
  closeAllPanels: []
  retry: [ChatMessage]
  preview: [url: string, type: 'image' | 'video']
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

// 加载更早的历史消息前记录滚动位置，加载后按高度差恢复，保持用户正在看的内容不跳动
type ScrollAnchor = { scrollHeight: number; scrollTop: number }

const captureScrollAnchor = (): ScrollAnchor | null => {
  const el = getScrollerEl()
  return el ? { scrollHeight: el.scrollHeight, scrollTop: el.scrollTop } : null
}

const restoreScrollAnchor = (anchor: ScrollAnchor | null) => {
  if (!anchor) return
  const apply = () => {
    const el = getScrollerEl()
    if (!el) return
    el.scrollTop = Math.max(0, el.scrollHeight - anchor.scrollHeight + anchor.scrollTop)
  }
  nextTick(() => {
    apply()
    // 虚拟列表测量新条目高度后 scrollHeight 会再变化，下一帧再校正一次
    if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(apply)
    }
  })
}

const previewMedia = (url: string, type: 'image' | 'video') => {
  emit('preview', url, type)
}

const copyToClipboard = async (text: string) => {
  if (!text) return
  const ok = await copyTextToClipboard(text)
  show(ok ? '已复制' : '复制失败')
}

// === 消息操作菜单（长按/右键）===
type MessageMenuState = { x: number; y: number; message: ChatMessage; copyText: string; canRetry: boolean }
const messageMenu = ref<MessageMenuState | null>(null)
const LONG_PRESS_MS = 500
const MENU_WIDTH_PX = 140
const MENU_HEIGHT_PX = 100
let longPressTimer: ReturnType<typeof setTimeout> | null = null

// 可复制的纯文本：分段消息只取文本段，媒体消息无文本
const getCopyText = (msg: ChatMessage): string => {
  if (msg.segments && msg.segments.length) {
    return msg.segments.map(seg => (seg.kind === 'text' ? seg.text : '')).join('').trim()
  }
  if (msg.isImage || msg.isVideo || msg.isFile) return ''
  return String(msg.content || '')
}

// 返回 true 表示菜单已打开（有可用操作）
const openMessageMenu = (msg: ChatMessage, clientX: number, clientY: number): boolean => {
  const copyText = getCopyText(msg)
  const canRetry = Boolean(msg.isSelf && msg.sendStatus === 'failed')
  if (!copyText && !canRetry) return false

  const vw = typeof window !== 'undefined' ? window.innerWidth : MENU_WIDTH_PX
  const vh = typeof window !== 'undefined' ? window.innerHeight : MENU_HEIGHT_PX
  messageMenu.value = {
    x: Math.max(8, Math.min(clientX, vw - MENU_WIDTH_PX - 8)),
    y: Math.max(8, Math.min(clientY, vh - MENU_HEIGHT_PX - 8)),
    message: msg,
    copyText,
    canRetry
  }
  return true
}

const closeMessageMenu = () => {
  messageMenu.value = null
}

const handleMessageContextMenu = (msg: ChatMessage, e: MouseEvent) => {
  // 没有可用操作时保留浏览器默认菜单（例如图片另存为）
  if (openMessageMenu(msg, e.clientX, e.clientY)) e.preventDefault()
}

const cancelMessageLongPress = () => {
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }
}

const startMessageLongPress = (msg: ChatMessage, e: TouchEvent) => {
  cancelMessageLongPress()
  const touch = e.touches[0]
  if (!touch) return
  const { clientX, clientY } = touch
  longPressTimer = setTimeout(() => {
    longPressTimer = null
    openMessageMenu(msg, clientX, clientY)
  }, LONG_PRESS_MS)
}

const handleMenuCopy = () => {
  const text = messageMenu.value?.copyText || ''
  closeMessageMenu()
  void copyToClipboard(text)
}

const handleMenuRetry = () => {
  const msg = messageMenu.value?.message
  closeMessageMenu()
  if (msg) emit('retry', msg)
}

// 新消息检测：只关注“最后一条消息”的变化，向前加载历史（prepend）不会触发
const lastMessageKey = computed(() => {
  const last = props.messages[props.messages.length - 1]
  return last ? getMessageKey(last) : ''
})

watch(
  lastMessageKey,
  (newKey, oldKey) => {
    if (!newKey || newKey === oldKey) return

    // 旧的最后一条已不在列表里：说明切换了会话或整体重载，直接贴底
    const appended = Boolean(oldKey) && props.messages.some(m => getMessageKey(m) === oldKey)
    if (!appended) {
      scrollToBottom()
      return
    }

    // 自己发送的消息总是回到底部
    const last = props.messages[props.messages.length - 1]
    if (last?.isSelf) {
      scrollToBottom(true)
      return
    }

    if (isAtBottom.value) {
      scrollToBottom()
    } else {
      hasNewMessages.value = true
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
  cancelMessageLongPress()
})

const getIsAtBottom = () => isAtBottom.value

defineExpose({
  scrollToBottom,
  captureScrollAnchor,
  restoreScrollAnchor,
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

/* 触屏设备上长按弹出自定义菜单，屏蔽系统的文字选择/呼出菜单，避免两者叠加 */
@media (hover: none) {
  .msg-text {
    -webkit-touch-callout: none;
    -webkit-user-select: none;
    user-select: none;
  }
}
</style>
