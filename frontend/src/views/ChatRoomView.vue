<template>
  <div class="page-container bg-[#0f0f13]">
    <ChatHeader
      :user="chatStore.currentChatUser"
      :connected="chatStore.wsConnected"
      @back="handleBack"
      @toggle-favorite="handleToggleFavorite"
      @clear-and-reload="handleClearAndReload"
    />

    <MessageList
      :messages="messages"
      :is-typing="messageStore.isTyping"
      :loading-more="messageStore.loadingMore"
      :can-load-more="canLoadMore"
      @load-more="handleLoadMore"
      @close-all-panels="handleCloseAllPanels"
      ref="messageListRef"
    />

    <!-- 上传菜单 -->
    <UploadMenu
      v-model:visible="showUploadMenu"
      :uploaded-media="mediaStore.uploadedMedia"
      :can-open-chat-history="!!chatStore.currentChatUser"
      @send="handleSendMedia"
      @upload-file="handleUploadFile"
      @open-chat-history="handleOpenChatHistory"
      @open-all-uploads="handleOpenAllUploads"
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
    />

    <!-- 媒体预览 -->
    <MediaPreview
      v-model:visible="showMediaPreview"
      :url="previewUrl"
      :type="previewType"
      :can-upload="previewCanUpload"
      :media-list="previewMediaList"
      @upload="confirmPreviewUpload"
    />

    <!-- 隐藏的文件输入 -->
    <input
      ref="fileInput"
      type="file"
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

    <!-- 聊天历史图片/视频弹窗 -->
    <teleport to="body">
      <div
        v-if="showHistoryMediaModal"
        class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
        @click="closeHistoryMediaModal"
      >
        <div class="w-[90%] max-w-2xl h-[70vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col" @click.stop>
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
            <div class="flex items-center gap-2">
              <i class="fas fa-history text-green-500"></i>
              <h3 class="text-lg font-bold text-white">
                与 {{ chatStore.currentChatUser?.nickname }} 的聊天历史图片
              </h3>
            </div>
            <button
              @click="closeHistoryMediaModal"
              class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>

          <div v-if="historyMediaLoading" class="flex-1 flex items-center justify-center">
            <div class="text-center">
              <div class="radar-spinner mx-auto mb-3"></div>
              <p class="text-gray-500 text-sm">加载中...</p>
            </div>
          </div>

          <div v-else-if="historyMedia && historyMedia.length > 0" class="flex-1 overflow-y-auto p-6 no-scrollbar">
            <div class="grid grid-cols-3 sm:grid-cols-4 gap-4">
              <div
                v-for="(media, idx) in historyMedia"
                :key="'history-media-' + idx"
                class="aspect-square rounded-xl overflow-hidden cursor-pointer border-2 border-gray-700 hover:border-green-500 hover:scale-105 transition-all relative"
                @click="openPreviewUpload(media)"
              >
                <img v-if="media.type === 'image'" :src="media.url" class="w-full h-full object-cover" />
                <video v-else :src="media.url" class="w-full h-full object-cover"></video>
                <div v-if="media.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/30">
                  <i class="fas fa-play-circle text-white text-3xl"></i>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="flex-1 flex items-center justify-center">
            <div class="text-center text-gray-500">
              <i class="fas fa-image text-5xl mb-4 opacity-30"></i>
              <p>暂无聊天历史图片</p>
            </div>
          </div>

          <div class="px-6 py-4 border-t border-gray-800 text-center text-xs text-gray-500">
            点击图片/视频预览，在预览中可上传/重新上传，再在上方"已上传的文件"中点击发送
          </div>
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'
import { useMessage } from '@/composables/useMessage'
import { useUpload } from '@/composables/useUpload'
import { useWebSocket } from '@/composables/useWebSocket'
import { useToast } from '@/composables/useToast'
import { useChat } from '@/composables/useChat'
import { extractUploadLocalPath } from '@/utils/media'
import { generateCookie } from '@/utils/cookie'
import { IMG_SERVER_IMAGE_PORT, IMG_SERVER_VIDEO_PORT } from '@/constants/config'
import * as mediaApi from '@/api/media'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import MessageList from '@/components/chat/MessageList.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import UploadMenu from '@/components/chat/UploadMenu.vue'
import EmojiPanel from '@/components/chat/EmojiPanel.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import Toast from '@/components/common/Toast.vue'
import Dialog from '@/components/common/Dialog.vue'
import type { UploadedMedia } from '@/types'

const router = useRouter()
const chatStore = useChatStore()
const messageStore = useMessageStore()
const mediaStore = useMediaStore()
const userStore = useUserStore()
const { sendText, sendImage, sendVideo, sendTypingStatus } = useMessage()
const { uploadFile, getMediaUrl } = useUpload()
const route = useRoute()
const { connect, setScrollToBottom } = useWebSocket()
const { show } = useToast()
const { toggleFavorite, enterChat, startMatch } = useChat()

const inputText = ref('')
const showUploadMenu = ref(false)
const showEmojiPanel = ref(false)
const showMediaPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video'>('image')
const previewCanUpload = ref(false)
const previewTarget = ref<UploadedMedia | null>(null)
const previewMediaList = ref<UploadedMedia[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const messageListRef = ref<any>(null)

const showClearDialog = ref(false)

const showHistoryMediaModal = ref(false)
const historyMediaLoading = ref(false)
const historyMedia = ref<UploadedMedia[]>([])

const messages = computed(() => {
  if (!chatStore.currentChatUser) return []
  return messageStore.getMessages(chatStore.currentChatUser.id)
})

const canLoadMore = computed(() => {
  return !!chatStore.currentChatUser
})

const handleSend = () => {
  if (!inputText.value.trim() || !chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show('连接已断开，请刷新页面重试')
    return
  }

  sendText(inputText.value, chatStore.currentChatUser)
  inputText.value = ''
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
}

const handleToggleEmoji = () => {
  showEmojiPanel.value = !showEmojiPanel.value
  if (showEmojiPanel.value) {
    showUploadMenu.value = false
  }
}

const handleCloseAllPanels = () => {
  showUploadMenu.value = false
  showEmojiPanel.value = false
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

const handleEmojiSelect = (text: string) => {
  inputText.value += text
  showEmojiPanel.value = false
}

const handleUploadFile = () => {
  if (fileInput.value) {
    fileInput.value.click()
  }
}

const handleFileChange = async (e: Event) => {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file || !userStore.currentUser) return

  showUploadMenu.value = false

  const media = await uploadFile(file, userStore.currentUser.id, userStore.currentUser.name)
  if (media) {
    let msg = '文件上传成功'
    if (media.type === 'video') msg = '视频上传成功'
    else if (media.type === 'image') msg = '图片上传成功'
    show(msg)
  } else {
    show('文件上传失败')
  }

  target.value = ''
}

const handleSendMedia = (media: UploadedMedia) => {
  if (!chatStore.currentChatUser) return
  if (!chatStore.wsConnected) {
    show('连接已断开，请刷新页面重试')
    return
  }

  if (media.type === 'video') {
    void sendVideo(media.url, chatStore.currentChatUser, media.localFilename)
  } else {
    // 图片和普通文件都走这个通道
    void sendImage(media.url, chatStore.currentChatUser, media.localFilename)
  }

  showUploadMenu.value = false
}

const handleBack = () => {
  messageStore.isTyping = false
  chatStore.exitChat()
  router.push('/list')
}

const handleToggleFavorite = () => {
  if (!chatStore.currentChatUser) return
  toggleFavorite(chatStore.currentChatUser)
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
    messageStore.clearHistory(userId)
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

const handleLoadMore = async () => {
  if (!chatStore.currentChatUser || !userStore.currentUser) return

  const lastTid = messageStore.firstTidMap[chatStore.currentChatUser.id]
  const count = await messageStore.loadHistory(userStore.currentUser.id, chatStore.currentChatUser.id, {
    isFirst: false,
    firstTid: lastTid,
    myUserName: userStore.currentUser.name
  })

  if (count > 0) {
    show(`加载了 ${count} 条历史消息`)

    // 滚动到顶部，显示刚加载的历史消息
    await nextTick()
    if (messageListRef.value) {
      messageListRef.value.scrollToTop()
    }
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
      type: url.toLowerCase().includes('.mp4') ? 'video' : 'image'
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
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
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
      const port = previewTarget.value.type === 'video' ? IMG_SERVER_VIDEO_PORT : IMG_SERVER_IMAGE_PORT
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`
      mediaStore.addUploadedMedia({ url: remoteUrl, type: previewTarget.value.type })
      show('图片已加载，点击可发送')
      showMediaPreview.value = false
      previewCanUpload.value = false
      previewTarget.value = null
      showHistoryMediaModal.value = false
      showUploadMenu.value = true
      return
    }

    show(`重新上传失败: ${res?.msg || res?.error || '未知错误'}`)
  } catch (e) {
    console.error('重新上传失败:', e)
    show('重新上传失败')
  }
}

const handleOpenAllUploads = async () => {
  if (!userStore.currentUser) return
  mediaStore.managementMode = false
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  mediaStore.showAllUploadImageModal = true
  await mediaStore.loadAllUploadImages(userStore.currentUser.id, 1)
}

const handleStartMatch = () => {
  if (!chatStore.wsConnected) {
    show('WebSocket 未连接，无法匹配')
    return
  }

  const success = startMatch()
  if (success) {
    console.log('从聊天室发起随机匹配')
  }
}

onMounted(async () => {
  if (!userStore.currentUser) {
    router.push('/identity')
    return
  }

  // 对齐旧版：进入聊天页时刷新“已上传的文件”列表
  try {
    await mediaStore.loadImgServer()
    await mediaStore.loadCachedImages(userStore.currentUser.id)
  } catch (e) {
    console.warn('初始化已上传文件列表失败:', e)
  }

  // 兜底：允许直接通过路由进入聊天室
  const userId = String(route.params.userId || '')
  if (userId && !chatStore.currentChatUser) {
    const user = chatStore.getUser(userId)
    if (user) {
      enterChat(user, true)
    } else {
      router.push('/list')
      return
    }
  }

  if (!chatStore.wsConnected) {
    connect()
  }

  if (messageListRef.value) {
    setScrollToBottom(() => {
      messageListRef.value?.scrollToBottom()
    })
  }

  const handlePreview = (e: any) => {
    const clickedUrl = e.detail.url
    previewUrl.value = clickedUrl
    previewType.value = e.detail.type
    previewCanUpload.value = false
    previewTarget.value = null
    
    // 构造媒体列表（仅在查看聊天消息时有效）
    // 通过检测 url 是否在当前消息列表的媒体中来判断
    // 1. 提取所有媒体消息
    const mediaMsgs = messages.value.filter(m => m.isImage || m.isVideo)
    
    // 2. 转换为 UploadedMedia 列表
    const list: UploadedMedia[] = mediaMsgs.map(m => {
       const rawUrl = m.imageUrl || m.videoUrl || m.content || ''
       return {
         url: getMediaUrl(rawUrl), // 确保是完整URL
         type: m.isVideo ? 'video' : 'image'
       }
    })
    
    // 3. 只有当点击的URL在列表中找到时，才启用画廊模式
    // 这是为了防止在"已上传文件"等其他地方预览时出现奇怪的列表
    const found = list.some(m => m.url === clickedUrl)
    if (found && list.length > 0) {
      previewMediaList.value = list
    } else {
      previewMediaList.value = [] // 为空时 MediaPreview 会回退到单图模式
    }
    
    showMediaPreview.value = true
  }
  window.addEventListener('preview-media', handlePreview)

  onUnmounted(() => {
    window.removeEventListener('preview-media', handlePreview)
  })
})

watch(
  () => mediaStore.openUploadMenuSeq,
  () => {
    if (!chatStore.currentChatUser) return
    showHistoryMediaModal.value = false
    showEmojiPanel.value = false
    showUploadMenu.value = true
  }
)

watch(
  () => chatStore.currentChatUser,
  (user) => {
    if (!user) return
    messageStore.isTyping = false
    // 进入聊天时清零未读
    if (user.unreadCount && user.unreadCount > 0) {
      user.unreadCount = 0
    }
  },
  { immediate: true }
)
</script>
