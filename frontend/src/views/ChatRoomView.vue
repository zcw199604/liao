<template>
  <div
    ref="pageRef"
    class="page-container bg-canvas relative overflow-hidden"
    :style="{
      transform: `translateX(${pageTranslateX}px)`,
      transition: isPageAnimating ? swipeTransition : 'none'
    }"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <!-- 侧边栏抽屉 -->
    <Transition name="slide-right">
      <div
        v-if="showSidebar"
        ref="sidebarRef"
        class="absolute inset-y-0 left-0 w-[80%] max-w-sm z-50 shadow-2xl bg-canvas border-r border-line"
        :style="{
          transform: `translateX(${sidebarTranslateX}px)`,
          transition: isSidebarAnimating ? swipeTransition : 'none'
        }"
      >
        <ChatSidebar
          :current-user-id="chatStore.currentChatUser?.id"
          @select="handleSidebarSelect"
          @match-success="handleSidebarMatchSuccess"
        />
      </div>
    </Transition>

    <!-- 侧边栏遮罩 -->
    <Transition name="fade">
      <div
        v-if="showSidebar"
        class="absolute inset-0 z-40 bg-black/50 backdrop-blur-sm"
        @click="showSidebar = false"
      ></div>
    </Transition>

    <div ref="chatHeaderWrapRef" class="absolute top-0 left-0 right-0 z-20">
      <ChatHeader
        :user="chatStore.currentChatUser"
        :connected="chatStore.wsConnected"
        :connection-state="connectionState"
        :status-text="connectionText"
        @back="handleBack"
        @toggle-favorite="handleToggleFavorite"
        @blacklist="handleBlacklist"
        @clear-and-reload="handleClearAndReload"
        @toggle-sidebar="showSidebar = !showSidebar"
        @reconnect="handleReconnect"
      />
      <div
        v-if="isTemporaryConversation"
        class="mx-4 mt-2 rounded-xl border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-xs text-blue-200 flex items-center gap-2 shadow-sm"
      >
        <i class="fas fa-random"></i>
        <span class="truncate">临时接入会话，发送首条消息后会刷新当前身份历史</span>
      </div>
    </div>

    <MessageList
      class="h-full"
      :style="messageListStyle"
      :floating-bottom-offset-px="floatingButtonBottomPx"
      :messages="messages"
      :is-typing="messageStore.isTyping"
      :loading-more="messageStore.loadingMore"
      :can-load-more="canLoadMore"
      @load-more="handleLoadMore"
      @close-all-panels="handleCloseAllPanels"
      @retry="handleRetry"
      @preview="handlePreview"
      ref="messageListRef"
    />

    <!-- 底部悬浮控制区 -->
    <div ref="bottomDockRef" class="absolute bottom-0 left-0 right-0 z-30 flex flex-col">
      <!-- 上传进度 -->
      <div
        v-if="uploadProgress"
        data-testid="upload-progress"
        class="mx-4 mb-2 rounded-xl border border-line bg-surface/90 backdrop-blur-md px-3 py-2 text-xs text-fg-muted shadow-sm"
      >
        <div class="flex items-center justify-between mb-1.5">
          <span class="flex items-center gap-2">
            <span class="radar-spinner !w-3 !h-3"></span>
            上传中{{ uploadProgress.total > 1 ? ` ${uploadProgress.current}/${uploadProgress.total}` : '' }}
          </span>
          <span>{{ uploadProgress.percent }}%</span>
        </div>
        <div class="h-1 rounded-full bg-surface-3 overflow-hidden">
          <div class="h-full bg-indigo-500 transition-[width] duration-200" :style="{ width: `${uploadProgress.percent}%` }"></div>
        </div>
      </div>

      <!-- 上传菜单 -->
      <UploadMenu
        v-model:visible="showUploadMenu"
        :uploaded-media="mediaStore.uploadedMedia"
        :can-open-chat-history="!!chatStore.currentChatUser"
        @send="handleSendMedia"
        @upload-file="handleUploadFile"
        @open-chat-history="handleOpenChatHistory"
        @open-all-uploads="handleOpenAllUploads"
        @open-mt-photo="handleOpenMtPhoto"
        @open-douyin-favorite-authors="handleOpenDouyinFavoriteAuthors"
      />

      <!-- 表情面板 -->
      <EmojiPanel
        v-model:visible="showEmojiPanel"
        @select="handleEmojiSelect"
      />

      <!-- 输入框 -->
      <ChatInput
        v-model="inputText"
        :disabled="!chatStore.wsConnected"
        :ws-connected="chatStore.wsConnected"
        @send="handleSend"
        @show-upload="handleToggleUpload"
        @show-emoji="handleToggleEmoji"
        @typing-start="handleTypingStart"
        @typing-end="handleTypingEnd"
        @start-match="handleStartMatch"
        @paste-files="uploadAndSendFiles"
      />
    </div>

    <!-- 拖拽上传提示 -->
    <div
      v-if="isDraggingFiles"
      data-testid="drop-overlay"
      class="absolute inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm pointer-events-none"
    >
      <div class="rounded-2xl border-2 border-dashed border-white/60 px-8 py-6 text-white text-center">
        <i class="fas fa-cloud-upload-alt text-4xl mb-3"></i>
        <p class="text-sm">松开即可发送给 {{ chatStore.currentChatUser?.nickname || '对方' }}</p>
      </div>
    </div>

    <!-- 媒体预览 -->
    <MediaPreview
      v-model:visible="showMediaPreview"
      :url="previewUrl"
      :type="previewType"
      :can-upload="previewCanUpload"
      :media-list="previewMediaList"
      @upload="confirmPreviewUpload"
    />

    <!-- 隐藏的文件输入（支持多选） -->
    <input
      ref="fileInput"
      type="file"
      multiple
      @change="handleFileChange"
      style="display: none"
    />

    <Toast />

    <!-- 清空聊天记录确认弹窗 -->
    <Dialog
      v-model:visible="showClearDialog"
      title="清空聊天记录"
      confirm-text="清空"
      :show-warning="true"
      @confirm="executeClearAndReload"
    >
      确定要清空并重新加载聊天记录吗？<br/>本地缓存的消息将被清除。
    </Dialog>

    <!-- 拉黑确认弹窗 -->
    <Dialog
      v-model:visible="showBlacklistDialog"
      title="拉黑用户"
      confirm-text="确认拉黑"
      :show-warning="true"
      @confirm="executeBlacklist"
    >
      确定要拉黑 <strong>{{ chatStore.currentChatUser?.nickname || '该用户' }}</strong> 吗？
    </Dialog>

    <!-- 聊天历史图片/视频弹窗 -->
    <teleport to="body">
      <div
        v-if="showHistoryMediaModal"
        data-testid="history-media-modal"
        class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center overscroll-contain"
        @click="closeHistoryMediaModal"
      >
        <div class="w-[90%] max-w-2xl h-[70vh] bg-surface rounded-2xl shadow-2xl flex flex-col" @click.stop>
          <div class="flex items-center justify-between px-6 py-4 border-b border-line">
            <div class="flex items-center gap-2">
              <i class="fas fa-history text-green-500"></i>
              <h3 class="text-lg font-bold text-fg">
                与 {{ chatStore.currentChatUser?.nickname }} 的聊天历史图片
              </h3>
            </div>
            <button
              @click="closeHistoryMediaModal"
              class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3"
              aria-label="关闭"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>

          <div v-if="historyMediaLoading" class="flex-1 flex items-center justify-center">
            <div class="text-center">
              <div class="radar-spinner mx-auto mb-3"></div>
              <p class="text-fg-subtle text-sm">加载中...</p>
            </div>
          </div>

          <div v-else-if="historyMedia && historyMedia.length > 0" class="flex-1 overflow-y-auto overscroll-contain p-6 no-scrollbar">
            <div class="grid grid-cols-3 sm:grid-cols-4 gap-4">
              <MediaTile
                v-for="(media, idx) in historyMedia"
                :key="'history-media-' + idx"
                :src="media.url"
                :type="media.type"
                class="aspect-square rounded-xl overflow-hidden cursor-pointer border-2 border-line-strong hover:border-green-500 hover:scale-105 transition-all"
                :show-skeleton="false"
                :indicator-size="'md'"
                :muted="true"
                @click="openPreviewUpload(media)"
              >
                <template #file>
                  <i class="fas fa-file text-3xl"></i>
                </template>
              </MediaTile>
            </div>
          </div>

          <div v-else class="flex-1 flex items-center justify-center">
            <div class="text-center text-fg-subtle">
              <i class="fas fa-image text-5xl mb-4 opacity-30"></i>
              <p>暂无聊天历史图片</p>
            </div>
          </div>

          <div class="px-6 py-4 border-t border-line text-center text-xs text-fg-subtle">
            点击图片/视频预览，在预览中可上传/重新上传，再在上方"已上传的文件"中点击发送
          </div>
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useDouyinStore } from '@/stores/douyin'
import { useUserStore } from '@/stores/user'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useMessage } from '@/composables/useMessage'
import { useUpload } from '@/composables/useUpload'
import { useWebSocket } from '@/composables/useWebSocket'
import { useToast } from '@/composables/useToast'
import { useChat } from '@/composables/useChat'
import { useSwipeAction } from '@/composables/useInteraction'
import { useConnectionStatus } from '@/composables/useConnectionStatus'
import { extractUploadLocalPath, inferMediaTypeFromUrl } from '@/utils/media'
import { generateCookie } from '@/utils/cookie'
import { hasNativeEdgeSwipeBack } from '@/utils/device'
import { UPSTREAM_REFERER, WS_DISCONNECTED_TIP } from '@/constants/config'
import {
  EDGE_BACK_START_PX,
  DRAWER_CLOSE_EDGE_PX,
  SWIPE_RESET_DURATION_MS
} from '@/constants/interaction'
import * as mediaApi from '@/api/media'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import MessageList from '@/components/chat/MessageList.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import UploadMenu from '@/components/chat/UploadMenu.vue'
import EmojiPanel from '@/components/chat/EmojiPanel.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import MediaTile from '@/components/common/MediaTile.vue'
import Toast from '@/components/common/Toast.vue'
import Dialog from '@/components/common/Dialog.vue'
import type { ChatMessage, UploadedMedia, User } from '@/types'

const router = useRouter()
const route = useRoute()
const chatStore = useChatStore()
const messageStore = useMessageStore()
const mediaStore = useMediaStore()
const mtPhotoStore = useMtPhotoStore()
const douyinStore = useDouyinStore()
const systemConfigStore = useSystemConfigStore()
const userStore = useUserStore()
const { sendText, sendImage, sendVideo, retryMessage, sendTypingStatus } = useMessage()
const { uploadFile, getMediaUrl } = useUpload()
const { connect, setScrollToBottom, send, reconnectNow } = useWebSocket()
const { show } = useToast()
const { toggleFavorite, enterChat, startMatch, loadUsers } = useChat()
const { state: connectionState, text: connectionText } = useConnectionStatus()

const inputText = ref('')
const showUploadMenu = ref(false)
const showEmojiPanel = ref(false)
const showMediaPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewCanUpload = ref(false)
const previewTarget = ref<UploadedMedia | null>(null)
const previewMediaList = ref<UploadedMedia[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const messageListRef = ref<any>(null)

const showClearDialog = ref(false)
const showBlacklistDialog = ref(false)
const pageRef = ref<HTMLElement | null>(null)
const sidebarRef = ref<HTMLElement | null>(null)
const showSidebar = ref(false)

const showHistoryMediaModal = ref(false)
const historyMediaLoading = ref(false)
const historyMedia = ref<UploadedMedia[]>([])

// 上传进度：current/total 为第几个文件，percent 为当前文件进度
const uploadProgress = ref<{ current: number; total: number; percent: number } | null>(null)
const isDraggingFiles = ref(false)
let dragDepth = 0

// iOS 浏览器自带边缘返回手势，此时不启用页面自定义的边缘右滑返回
const edgeSwipeBackEnabled = !hasNativeEdgeSwipeBack()

// 动态测量：顶部 Header 与底部悬浮区高度，避免固定 padding 造成留白过大
const chatHeaderWrapRef = ref<HTMLElement | null>(null)
const bottomDockRef = ref<HTMLElement | null>(null)

const headerHeightPx = ref(0)
const bottomDockHeightPx = ref(0)

const chatAreaBasePaddingPx = 15
// 让最后一条消息更贴近输入框：底部只留一个小间隙，避免出现类似 pb-32 的过大留白
const messageListBottomGapPx = 8

const resolveReuploadErrorMessage = (payload: unknown): string | undefined => {
  if (payload == null) return undefined
  if (typeof payload === 'string') {
    const trimmed = payload.trim()
    if (!trimmed) return undefined
    const looksLikeJson =
      (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
      (trimmed.startsWith('[') && trimmed.endsWith(']'))
    if (looksLikeJson) {
      try {
        return resolveReuploadErrorMessage(JSON.parse(trimmed)) || trimmed
      } catch {
        return trimmed
      }
    }
    return trimmed
  }
  if (typeof payload === 'object') {
    const data = payload as Record<string, unknown>
    if (typeof data.msg === 'string' && data.msg.trim()) return data.msg.trim()
    if (typeof data.error === 'string' && data.error.trim()) return data.error.trim()
    if (typeof data.message === 'string' && data.message.trim()) return data.message.trim()
  }
  return undefined
}

const refreshLayoutMeasurements = () => {
  // getBoundingClientRect 在部分环境可能为 0；这里用 offsetHeight 做兜底
  const headerRectH = chatHeaderWrapRef.value?.getBoundingClientRect().height ?? 0
  const dockRectH = bottomDockRef.value?.getBoundingClientRect().height ?? 0

  const headerH = headerRectH || chatHeaderWrapRef.value?.offsetHeight || 0
  const dockH = dockRectH || bottomDockRef.value?.offsetHeight || 0

  headerHeightPx.value = Math.max(0, headerH)
  bottomDockHeightPx.value = Math.max(0, dockH)
}

// 面板开合等布局变化后：重新测量，并在用户本来就在底部时保持贴底
const relayoutKeepBottom = async () => {
  await nextTick()
  refreshLayoutMeasurements()

  if (messageListRef.value?.getIsAtBottom?.()) {
    messageListRef.value?.scrollToBottom()
  }
}

let layoutResizeObserver: ResizeObserver | null = null
const initLayoutResizeObserver = () => {
  // 极少数环境下可能缺少 ResizeObserver（例如部分测试环境）
  if (typeof ResizeObserver === 'undefined') return

  layoutResizeObserver = new ResizeObserver(() => {
    refreshLayoutMeasurements()

    // 只有在用户已经处于底部时才保持贴底，避免打断用户阅读历史消息
    if (messageListRef.value?.getIsAtBottom?.()) {
      messageListRef.value?.scrollToBottom()
    }
  })

  if (chatHeaderWrapRef.value) layoutResizeObserver.observe(chatHeaderWrapRef.value)
  if (bottomDockRef.value) layoutResizeObserver.observe(bottomDockRef.value)
}

const messageListStyle = computed(() => {
  const headerHeight = Math.max(0, Math.round(headerHeightPx.value || 0))
  const dockHeight = Math.max(0, Math.round(bottomDockHeightPx.value || 0))

  return {
    paddingTop: `${headerHeight + chatAreaBasePaddingPx}px`,
    // dockHeight 已包含输入框(含安全区)高度，这里只额外留少量间隙即可
    paddingBottom: `${dockHeight + messageListBottomGapPx}px`
  }
})

const floatingButtonBottomMarginPx = 16
const floatingButtonBottomPx = computed(() => {
  const dockHeight = Math.max(0, Math.round(bottomDockHeightPx.value || 0))
  return dockHeight + floatingButtonBottomMarginPx
})

// 手势动画状态
const pageTranslateX = ref(0)
const sidebarTranslateX = ref(0)
const isPageAnimating = ref(false)
const isSidebarAnimating = ref(false)
const swipeTransition = `transform ${SWIPE_RESET_DURATION_MS}ms cubic-bezier(0.25, 0.46, 0.45, 0.94)`

const messages = computed(() => {
  if (!chatStore.currentChatUser) return []
  if (!userStore.currentUser) return []
  return messageStore.getMessages(userStore.currentUser.id, chatStore.currentChatUser.id)
})

const currentConversationKey = computed(() => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return ''
  return messageStore.conversationKey(userStore.currentUser.id, chatStore.currentChatUser.id)
})

const isTemporaryConversation = computed(() => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return false
  return chatStore.isTemporaryConversation(userStore.currentUser.id, chatStore.currentChatUser.id)
})

const canLoadMore = computed(() => {
  return !!chatStore.currentChatUser
})

// 草稿按会话保存：切换会话时先存旧会话草稿，再恢复新会话草稿，避免把半截消息发给别人
watch(
  currentConversationKey,
  (newKey, oldKey) => {
    if (oldKey) messageStore.setDraft(oldKey, inputText.value)
    inputText.value = messageStore.getDraft(newKey)
  },
  { immediate: true }
)

const handleSidebarSelect = (user: User) => {
  if (user.id === chatStore.currentChatUser?.id) {
    showSidebar.value = false
    return
  }

  // 切换聊天对象
  showSidebar.value = false
  enterChat(user, true)
  // 更新路由参数但不刷新页面
  router.replace(`/chat/${user.id}`)
}

const handleSidebarMatchSuccess = (user: User) => {
  showSidebar.value = false
  enterChat(user, false)
  router.replace(`/chat/${user.id}`)
}

const handleReconnect = () => {
  if (reconnectNow()) show('正在重新连接…')
}

const handleSend = async () => {
  if (!inputText.value.trim() || !chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show(WS_DISCONNECTED_TIP)
    return
  }

  sendText(inputText.value, chatStore.currentChatUser)
  inputText.value = ''

  // 自己发送的消息总是回到底部（MessageList 也会处理，这里确保布局测量后再贴底）
  await nextTick()
  refreshLayoutMeasurements()
  messageListRef.value?.scrollToBottom(true)
}

const handleRetry = async (msg: ChatMessage) => {
  if (!chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show(WS_DISCONNECTED_TIP)
    return
  }

  retryMessage(msg)
  await relayoutKeepBottom()
}

const handleToggleUpload = async () => {
  showUploadMenu.value = !showUploadMenu.value
  if (showUploadMenu.value) {
    showEmojiPanel.value = false
    if (userStore.currentUser) {
      await mediaStore.loadImgServer()
      await mediaStore.loadCachedImages(userStore.currentUser.id)
    }
  }

  await relayoutKeepBottom()
}

const handleToggleEmoji = async () => {
  showEmojiPanel.value = !showEmojiPanel.value
  if (showEmojiPanel.value) {
    showUploadMenu.value = false
  }

  await relayoutKeepBottom()
}

const handleCloseAllPanels = async () => {
  showUploadMenu.value = false
  showEmojiPanel.value = false

  await relayoutKeepBottom()
}

const handleTypingStart = () => {
  if (chatStore.currentChatUser && chatStore.wsConnected) {
    sendTypingStatus(true, chatStore.currentChatUser)
  }
}

const handleTypingEnd = () => {
  if (chatStore.currentChatUser && chatStore.wsConnected) {
    sendTypingStatus(false, chatStore.currentChatUser)
  }
}

const handleEmojiSelect = async (text: string) => {
  inputText.value += text
  showEmojiPanel.value = false

  await relayoutKeepBottom()
}

const handleUploadFile = () => {
  if (fileInput.value) {
    fileInput.value.click()
  }
}

// 发送已上传的媒体给指定用户
const sendMediaTo = (media: UploadedMedia, targetUser: User) => {
  if (media.type === 'video') {
    void sendVideo(media.url, targetUser, media.localFilename)
  } else {
    // 图片和普通文件都走这个通道
    void sendImage(media.url, targetUser, media.localFilename)
  }
}

// 上传并直接发送：逐个上传，上传完成时若仍停留在同一会话且连接正常则自动发送，
// 否则保留在“已上传的文件”里，由用户手动发送，避免上传期间切换会话后发错人
const uploadAndSendFiles = async (files: File[]) => {
  if (!files.length || !userStore.currentUser) return
  if (uploadProgress.value) {
    show('正在上传，请稍候')
    return
  }

  showUploadMenu.value = false
  const targetUser = chatStore.currentChatUser
  const { id: myUserId, name: myUserName } = userStore.currentUser
  let sentCount = 0
  let keptCount = 0
  let failedCount = 0

  try {
    for (let i = 0; i < files.length; i++) {
      uploadProgress.value = { current: i + 1, total: files.length, percent: 0 }
      const media = await uploadFile(files[i]!, myUserId, myUserName, {
        onProgress: (percent) => {
          if (uploadProgress.value) uploadProgress.value.percent = percent
        }
      })
      if (!media) {
        failedCount++
        continue
      }

      if (targetUser && chatStore.currentChatUser?.id === targetUser.id && chatStore.wsConnected) {
        sendMediaTo(media, targetUser)
        sentCount++
      } else {
        keptCount++
      }
    }
  } finally {
    uploadProgress.value = null
  }

  if (failedCount > 0) {
    show(files.length === 1 ? '文件上传失败' : `${failedCount} 个文件上传失败`)
  } else if (keptCount > 0) {
    show('已上传，可在“已上传的文件”中点击发送')
  }
  if (sentCount > 0) {
    await nextTick()
    messageListRef.value?.scrollToBottom(true)
  }
}

const handleFileChange = async (e: Event) => {
  const target = e.target as HTMLInputElement
  const files = Array.from(target.files || [])
  target.value = ''
  await uploadAndSendFiles(files)
}

// === 拖拽上传（桌面端）===
const isFileDrag = (e: DragEvent) => Array.from(e.dataTransfer?.types || []).includes('Files')

const handleDragEnter = (e: DragEvent) => {
  if (!isFileDrag(e) || !chatStore.currentChatUser) return
  e.preventDefault()
  dragDepth++
  isDraggingFiles.value = true
}

const handleDragOver = (e: DragEvent) => {
  if (!isFileDrag(e) || !chatStore.currentChatUser) return
  e.preventDefault()
}

const handleDragLeave = (e: DragEvent) => {
  if (!isFileDrag(e)) return
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) isDraggingFiles.value = false
}

const handleDrop = (e: DragEvent) => {
  if (!isFileDrag(e)) return
  e.preventDefault()
  dragDepth = 0
  isDraggingFiles.value = false
  if (!chatStore.currentChatUser) return
  void uploadAndSendFiles(Array.from(e.dataTransfer?.files || []))
}

const handleSendMedia = (media: UploadedMedia) => {
  if (!chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show(WS_DISCONNECTED_TIP)
    return
  }

  sendMediaTo(media, chatStore.currentChatUser)
  showUploadMenu.value = false
}

const handleBack = () => {
  messageStore.isTyping = false
  chatStore.exitChat()
  // 从列表页进入时直接后退，避免历史栈里堆积 列表→聊天→列表
  const backPath = (typeof window !== 'undefined' ? window.history.state?.back : undefined) as unknown
  if (typeof backPath === 'string' && backPath.startsWith('/list')) {
    router.back()
  } else {
    router.replace('/list')
  }
}

const resetPageTranslate = () => {
  isPageAnimating.value = true
  pageTranslateX.value = 0
  setTimeout(() => {
    isPageAnimating.value = false
  }, SWIPE_RESET_DURATION_MS)
}

// 页面边缘右滑返回
const { coordsStart: pageSwipeStart } = useSwipeAction(pageRef, {
  threshold: 0, // 手动处理
  passive: true,
  onSwipeProgress: (deltaX, deltaY) => {
    if (!edgeSwipeBackEnabled) return
    if (showSidebar.value) return // 抽屉开启时不响应

    // 起手位置判定
    if (pageSwipeStart.x > EDGE_BACK_START_PX) return

    // 方向判定
    if (Math.abs(deltaY) > Math.abs(deltaX)) return // 垂直主导忽略
    if (deltaX <= 0) return // 仅处理向右滑

    // 跟手位移，限制最大视觉位移
    pageTranslateX.value = Math.min(deltaX, 150)
  },
  onSwipeEnd: (direction) => {
    if (!edgeSwipeBackEnabled) return
    if (showSidebar.value) return
    if (pageSwipeStart.x > EDGE_BACK_START_PX) {
      pageTranslateX.value = 0
      return
    }

    if (direction === 'right' && pageTranslateX.value > 80) { // 简单阈值
      // 触发返回
      handleBack()
    } else {
      // 回弹
      resetPageTranslate()
    }
  },
  onSwipeFinish: (_deltaX, _deltaY, isTriggered) => {
    // useSwipeAction 在“方向不明确/未命中阈值”等情况下可能不会触发 onSwipeEnd；
    // 这里做兜底复位，避免残留偏移卡住。
    if (isTriggered) return
    if (showSidebar.value) {
      pageTranslateX.value = 0
      isPageAnimating.value = false
      return
    }
    if (pageTranslateX.value === 0) return
    resetPageTranslate()
  }
})

// 侧边栏抽屉左滑关闭
const { coordsStart: drawerSwipeStart } = useSwipeAction(sidebarRef, {
  threshold: 0,
  passive: true,
  onSwipeProgress: (deltaX, deltaY) => {
    if (!showSidebar.value) return
    const el = sidebarRef.value
    if (!el) return

    // 起手热区判定 (右侧边缘)
    const rect = el.getBoundingClientRect()
    // 注意：抽屉是 w-[80%]，right 是抽屉的右边缘
    // 用户在抽屉内部右侧边缘起手向左滑
    if (drawerSwipeStart.x < rect.right - DRAWER_CLOSE_EDGE_PX) return

    // 方向判定 (左滑 deltaX < 0)
    if (deltaX >= 0) return
    if (Math.abs(deltaY) > Math.abs(deltaX)) return

    // 跟手位移 (负值)
    sidebarTranslateX.value = deltaX
  },
  onSwipeEnd: () => {
    if (!showSidebar.value) return

    // 向左滑且距离足够则关闭
    if (sidebarTranslateX.value < -80) {
      showSidebar.value = false
      // 关闭后需要重置状态，但 Transition 会处理离场，这里重置位移以便下次打开
      setTimeout(() => {
        sidebarTranslateX.value = 0
      }, SWIPE_RESET_DURATION_MS)
    } else {
      // 回弹
      isSidebarAnimating.value = true
      sidebarTranslateX.value = 0
      setTimeout(() => {
        isSidebarAnimating.value = false
      }, SWIPE_RESET_DURATION_MS)
    }
  },
  onSwipeFinish: (_deltaX, _deltaY, isTriggered) => {
    // 兜底复位：避免极端手势结束时 onSwipeEnd 未触发导致的残留偏移
    if (isTriggered) return
    if (!showSidebar.value) {
      sidebarTranslateX.value = 0
      isSidebarAnimating.value = false
      return
    }
    if (sidebarTranslateX.value === 0) return
    isSidebarAnimating.value = true
    sidebarTranslateX.value = 0
    setTimeout(() => {
      isSidebarAnimating.value = false
    }, SWIPE_RESET_DURATION_MS)
  }
})

const handleToggleFavorite = () => {
  if (!chatStore.currentChatUser) return
  toggleFavorite(chatStore.currentChatUser)
}

const generateWarningReportUuid = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  const bytes = new Uint8Array(16)
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < bytes.length; i++) {
      bytes[i] = Math.floor(Math.random() * 256)
    }
  }

  // RFC 4122 v4
  bytes[6] = (bytes[6]! & 0x0f) | 0x40
  bytes[8] = (bytes[8]! & 0x3f) | 0x80

  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

const handleBlacklist = () => {
  if (!chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show(WS_DISCONNECTED_TIP)
    return
  }

  showBlacklistDialog.value = true
}

const executeBlacklist = () => {
  if (!chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show(WS_DISCONNECTED_TIP)
    return
  }

  const ok = send({
    act: 'warningreport',
    id: chatStore.currentChatUser.id,
    msg: generateWarningReportUuid()
  })

  if (ok !== false) {
    show('已发送拉黑请求')
    showBlacklistDialog.value = false
  }
}

const handleClearAndReload = () => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return
  showClearDialog.value = true
}

const executeClearAndReload = async () => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return

  const userId = chatStore.currentChatUser.id

  try {
    // 1. 清空本地缓存的聊天记录
    messageStore.clearHistory(userStore.currentUser.id, userId)
    show('正在重新加载聊天记录...')

    // 2. 重新加载聊天记录
    const count = await messageStore.loadHistory(
      userStore.currentUser.id,
      userId,
      {
        isFirst: true,         // 首次加载模式
        firstTid: '0',         // 从头开始
        myUserName: userStore.currentUser.name,
        incremental: false     // 完全替换
      }
    )

    // 3. 滚动到底部
    await nextTick()
    if (messageListRef.value) {
      messageListRef.value.scrollToBottom()
    }

    if (count > 0) {
      show(`已重新加载 ${count} 条消息`)
    } else {
      show('暂无聊天记录')
    }
  } catch (e) {
    console.error('重新加载聊天记录失败:', e)
    show('重新加载失败，请稍后重试')
  }
}

// 加载更早的历史消息：保持当前阅读位置不跳动，成功时不再弹 toast
const handleLoadMore = async () => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return

  const anchor = messageListRef.value?.captureScrollAnchor?.() ?? null
  const lastTid = currentConversationKey.value ? messageStore.firstTidMap[currentConversationKey.value] : undefined
  const count = await messageStore.loadHistory(userStore.currentUser.id, chatStore.currentChatUser.id, {
    isFirst: false,
    firstTid: lastTid,
    myUserName: userStore.currentUser.name
  })

  if (count > 0) {
    messageListRef.value?.restoreScrollAnchor?.(anchor)
  } else if (count === 0) {
    show('没有更多历史消息了')
  } else {
    show('加载失败')
  }
}

const handleOpenChatHistory = async () => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return

  showHistoryMediaModal.value = true
  historyMediaLoading.value = true
  try {
    const res = await mediaApi.getChatImages(userStore.currentUser.id, chatStore.currentChatUser.id, 50)
    const list: string[] = Array.isArray(res) ? res : []
    historyMedia.value = list.map(url => ({
      url,
      type: inferMediaTypeFromUrl(url)
    }))
  } catch (e) {
    console.error('加载聊天历史图片失败:', e)
    historyMedia.value = []
  } finally {
    historyMediaLoading.value = false
  }
}

const closeHistoryMediaModal = () => {
  showHistoryMediaModal.value = false
}

// 历史图片弹窗支持 Esc 关闭（预览打开时交给预览自己处理）
const handleHistoryModalKeydown = (e: KeyboardEvent) => {
  if (e.key !== 'Escape' || showMediaPreview.value) return
  closeHistoryMediaModal()
}

watch(showHistoryMediaModal, (visible) => {
  if (visible) window.addEventListener('keydown', handleHistoryModalKeydown)
  else window.removeEventListener('keydown', handleHistoryModalKeydown)
})

const openPreviewUpload = (media: UploadedMedia) => {
  previewTarget.value = media
  previewUrl.value = media.url
  previewType.value = media.type
  previewCanUpload.value = true
  showMediaPreview.value = true
}

const confirmPreviewUpload = async () => {
  if (!previewTarget.value || !userStore.currentUser) return

  if (!mediaStore.imgServer) {
    await mediaStore.loadImgServer()
  }
  if (!mediaStore.imgServer) {
    show('图片服务器地址未获取')
    return
  }

  const localPath = extractUploadLocalPath(previewTarget.value.url)
  const cookieData = generateCookie(userStore.currentUser.id, userStore.currentUser.name)
  const referer = UPSTREAM_REFERER
  const userAgent = navigator.userAgent

  try {
    const res = await mediaApi.reuploadHistoryImage({
      userId: userStore.currentUser.id,
      localPath,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.msg) {
      const port = await systemConfigStore.resolveImagePort(res.msg, mediaStore.imgServer)
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`

      const filename = localPath.substring(localPath.lastIndexOf('/') + 1)
      mediaStore.addUploadedMedia({
        url: remoteUrl,
        type: previewTarget.value.type,
        localFilename: filename
      })

      show('图片已加载，点击可发送')
      showMediaPreview.value = false
      previewCanUpload.value = false
      previewTarget.value = null
      showHistoryMediaModal.value = false
      showUploadMenu.value = true
      return
    }

    show(`重新上传失败: ${resolveReuploadErrorMessage(res) || '未知错误'}`)
  } catch (e: any) {
    console.error('重新上传失败:', e)
    const responseText = resolveReuploadErrorMessage(e?.response?.data)
    const fallbackText = resolveReuploadErrorMessage(e)
    show(`重新上传失败: ${responseText || fallbackText || '未知错误'}`)
  }
}

const handleOpenAllUploads = async () => {
  if (!userStore.currentUser) return
  mediaStore.managementMode = false
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  mediaStore.showAllUploadImageModal = true
  await mediaStore.loadAllUploadImages(1)
}

const handleOpenMtPhoto = async () => {
  if (!userStore.currentUser) return
  await mtPhotoStore.open()
}

const handleOpenDouyinFavoriteAuthors = () => {
  showUploadMenu.value = false
  douyinStore.open({
    entryMode: 'favorites',
    favoritesTab: 'users'
  })
}

const handleStartMatch = () => {
  if (!chatStore.wsConnected) {
    show('WebSocket 未连接，无法匹配')
    return
  }

  startMatch()
}

// 聊天消息中的图片/视频预览：点击的媒体在当前会话里时启用画廊模式
const handlePreview = (clickedUrl: string, type: 'image' | 'video') => {
  previewUrl.value = clickedUrl
  previewType.value = type
  previewCanUpload.value = false
  previewTarget.value = null

  const list: UploadedMedia[] = messages.value
    .filter(m => m.isImage || m.isVideo)
    .map(m => ({
      url: getMediaUrl(m.imageUrl || m.videoUrl || m.content || ''), // 确保是完整URL
      type: m.isVideo ? 'video' : 'image'
    }))

  // 只有点击的 URL 在列表中时才启用画廊模式，否则 MediaPreview 回退到单图模式
  previewMediaList.value = list.some(m => m.url === clickedUrl) ? list : []
  showMediaPreview.value = true
}

// 刷新页面后内存中的会话列表为空：先拉取列表再恢复路由里的会话
const restoreChatFromRoute = async (): Promise<boolean> => {
  const userId = String(route.params.userId || '')
  if (!userId || chatStore.currentChatUser) return true

  let user = chatStore.getUser(userId)
  if (!user) {
    try {
      await loadUsers()
    } catch (e) {
      console.warn('恢复会话时加载用户列表失败:', e)
    }
    user = chatStore.getUser(userId)
  }

  if (!user) return false
  enterChat(user, true)
  return true
}

onMounted(async () => {
  if (!userStore.currentUser) {
    router.push('/identity')
    return
  }

  // 首次进入页面时测量并建立监听，避免固定 padding 造成留白过大/过小
  refreshLayoutMeasurements()
  initLayoutResizeObserver()

  // 对齐旧版：进入聊天页时刷新“已上传的文件”列表
  try {
    await mediaStore.loadImgServer()
    await mediaStore.loadCachedImages(userStore.currentUser.id)
  } catch (e) {
    console.warn('初始化已上传文件列表失败:', e)
  }

  if (!(await restoreChatFromRoute())) {
    router.replace('/list')
    return
  }

  if (!chatStore.wsConnected) {
    connect()
  }

  if (messageListRef.value) {
    setScrollToBottom(() => {
      messageListRef.value?.scrollToBottom()
    })
  }
})

onBeforeUnmount(() => {
  // 离开页面前保存当前会话草稿
  if (currentConversationKey.value) {
    messageStore.setDraft(currentConversationKey.value, inputText.value)
  }
})

onUnmounted(() => {
  layoutResizeObserver?.disconnect()
  layoutResizeObserver = null
  window.removeEventListener('keydown', handleHistoryModalKeydown)
  chatStore.exitChat()
})

watch(
  () => mediaStore.openUploadMenuSeq,
  async () => {
    if (!chatStore.currentChatUser) return
    showHistoryMediaModal.value = false
    showEmojiPanel.value = false
    showUploadMenu.value = true

    await relayoutKeepBottom()
  }
)

watch(
  () => chatStore.currentChatUser,
  async (user) => {
    if (!user) return
    messageStore.isTyping = false
    // 进入聊天时清零未读
    if (user.unreadCount && user.unreadCount > 0) {
      user.unreadCount = 0
    }

    // 切换用户后，等待数据渲染完成，然后滚动到底部
    await nextTick()
    refreshLayoutMeasurements()
    if (messageListRef.value) {
      messageListRef.value.scrollToBottom()
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.3s ease;
}

.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(-100%);
}
</style>
