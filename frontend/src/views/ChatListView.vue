<template>
  <div class="page-container bg-[#0f0f13]" @click="showTopMenu = false">
    <!-- 顶部切换栏 -->
    <div class="flex items-center justify-between pt-4 pb-2 px-4 bg-[#0f0f13] z-10">
      <!-- 左侧：菜单按钮（下拉） -->
      <div class="relative">
        <button
          @click.stop="showTopMenu = !showTopMenu"
          class="w-10 h-10 flex items-center justify-center text-gray-400 hover:text-white transition"
        >
          <i class="fas fa-bars text-xl"></i>
        </button>

        <!-- 下拉菜单 -->
        <div
          v-if="showTopMenu"
          @click.stop
          class="absolute left-0 top-12 w-48 bg-[#18181b] rounded-xl shadow-2xl border border-gray-700 z-50"
        >
          <button
            @click="handleOpenSettings"
            class="w-full px-4 py-3 text-left hover:bg-[#27272a] text-white flex items-center gap-3 rounded-t-xl transition"
          >
            <i class="fas fa-user-edit text-blue-400"></i>
            <span>身份信息</span>
          </button>
          <button
            @click="handleOpenSystemSettings"
            class="w-full px-4 py-3 text-left hover:bg-[#27272a] text-white flex items-center gap-3 border-t border-gray-700 transition"
          >
            <i class="fas fa-cog text-gray-400"></i>
            <span>系统设置</span>
          </button>
          <button
            @click="handleSwitchIdentity"
            class="w-full px-4 py-3 text-left hover:bg-[#27272a] text-white flex items-center gap-3 border-t border-gray-700 rounded-b-xl transition"
          >
            <i class="fas fa-user-circle text-indigo-400"></i>
            <span>切换身份</span>
          </button>
        </div>
      </div>

      <!-- 中间：切换栏 -->
      <div class="flex bg-[#1f1f22] p-1 rounded-full">
        <button
          @click="chatStore.activeTab = 'history'"
          :class="chatStore.activeTab === 'history' ? 'bg-[#2d2d33] text-white shadow-md' : 'text-gray-500'"
          class="px-6 py-1.5 rounded-full text-sm font-medium transition-all duration-300"
        >
          消息
        </button>
        <button
          @click="chatStore.activeTab = 'favorite'"
          :class="chatStore.activeTab === 'favorite' ? 'bg-[#2d2d33] text-white shadow-md' : 'text-gray-500'"
          class="px-6 py-1.5 rounded-full text-sm font-medium transition-all duration-300"
        >
          收藏
        </button>
      </div>

      <!-- 右侧：连接状态 -->
      <div class="flex items-center gap-1">
        <span class="w-2 h-2 rounded-full" :class="chatStore.wsConnected ? 'bg-green-500' : 'bg-red-500'"></span>
      </div>
    </div>

    <!-- 列表内容 -->
    <div class="list-area no-scrollbar px-4 pt-2" ref="listAreaRef">
      <div
        v-for="user in chatStore.displayList"
        :key="user.id"
        @click="handleEnterChat(user)"
        class="flex items-center p-4 mb-3 bg-[#18181b] rounded-2xl active:scale-[0.98] transition-transform duration-100 cursor-pointer"
      >
        <!-- 纯色块代替头像 -->
        <div
          :class="getColorClass(user.id)"
          class="w-12 h-12 rounded-xl flex items-center justify-center text-white font-bold text-lg shadow-lg shrink-0"
        >
          {{ user.nickname.charAt(0).toUpperCase() }}
        </div>

        <!-- 文本信息 -->
        <div class="ml-4 flex-1 min-w-0">
          <div class="flex justify-between items-baseline mb-1">
            <span class="font-bold text-base text-white truncate">{{ user.nickname }}</span>
            <span class="text-xs text-gray-500">{{ user.lastTime }}</span>
          </div>
          <div class="flex justify-between items-center">
            <p class="text-sm text-gray-400 truncate pr-2">{{ user.lastMsg }}</p>
            <!-- 收藏标识 -->
            <i v-if="user.isFavorite && chatStore.activeTab === 'history'" class="fas fa-star text-xs text-yellow-500"></i>
          </div>
        </div>

        <!-- 未读消息数气泡 -->
        <span
          v-if="user.unreadCount && user.unreadCount > 0"
          class="px-2 py-0.5 bg-red-500 text-white text-xs rounded-full min-w-[20px] text-center font-medium ml-2"
        >
          {{ user.unreadCount }}
        </span>
      </div>

      <!-- 空状态提示 -->
      <div v-if="!chatStore.displayList || chatStore.displayList.length === 0" class="flex flex-col items-center justify-center mt-20 text-gray-600">
        <i class="far fa-comments text-5xl mb-4 opacity-50"></i>
        <p class="text-sm">暂无{{ chatStore.activeTab === 'history' ? '消息' : '收藏' }}</p>
      </div>
    </div>

    <!-- 底部悬浮匹配按钮 -->
    <div class="match-btn-container">
      <button
        v-if="!chatStore.isMatching"
        @click="handleStartMatch"
        class="match-btn flex items-center px-6 py-3 rounded-full text-white font-bold"
      >
        <i class="fas fa-random mr-2"></i> 匹配新用户
      </button>
      <button
        v-else
        @click="handleCancelMatch"
        class="flex items-center px-6 py-3 bg-red-600 rounded-full text-white font-bold shadow-xl animate-pulse"
      >
        <i class="fas fa-stop mr-2"></i> 取消匹配
      </button>
    </div>

    <!-- 匹配蒙层 -->
    <teleport to="body">
      <div
        v-if="chatStore.isMatching"
        class="fixed inset-0 z-[60] bg-black/90 flex flex-col items-center justify-center text-center"
      >
        <div class="relative w-32 h-32 mb-8">
          <div class="absolute inset-0 border-4 border-blue-500/30 rounded-full animate-ping"></div>
          <div class="absolute inset-0 border-4 border-blue-500 rounded-full flex items-center justify-center">
            <i class="fas fa-satellite-dish text-4xl text-blue-400"></i>
          </div>
        </div>
        <h2 class="text-xl font-bold mb-2 text-white">正在寻找有缘人...</h2>
        <p class="text-gray-400 text-sm">匹配完全匿名</p>
        <button
          type="button"
          @click="handleCancelMatch"
          class="mt-8 px-6 py-2 border border-gray-600 rounded-full text-gray-200 text-sm hover:bg-white/5 transition"
        >
          取消
        </button>
      </div>
    </teleport>

    <Toast />

    <SettingsDrawer v-model:visible="showSettings" :mode="settingsMode" />

    <Dialog
      v-model:visible="showSwitchIdentityDialog"
      title="确认切换身份"
      content="切换身份将断开当前连接，当前聊天会话将关闭"
      @confirm="confirmSwitchIdentity"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useChat } from '@/composables/useChat'
import { useWebSocket } from '@/composables/useWebSocket'
import { useToast } from '@/composables/useToast'
import { getColorClass } from '@/constants/colors'
import Toast from '@/components/common/Toast.vue'
import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import Dialog from '@/components/common/Dialog.vue'
import type { User } from '@/types'

const router = useRouter()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageStore = useMessageStore()
const { loadUsers, startMatch, cancelMatch, enterChat } = useChat()
const { connect, disconnect } = useWebSocket()
const { show } = useToast()

const showTopMenu = ref(false)
const showSettings = ref(false)
const settingsMode = ref<'identity' | 'system'>('identity')
const showSwitchIdentityDialog = ref(false)
const listAreaRef = ref<HTMLElement | null>(null)
const handleMatchSuccess = (e: any) => {
  const matchedUser = e.detail as User
  enterChat(matchedUser, false)
  router.push(`/chat/${matchedUser.id}`)
}

const handleEnterChat = (user: User) => {
  chatStore.listScrollTop = listAreaRef.value?.scrollTop || 0
  enterChat(user, true)
  router.push(`/chat/${user.id}`)
}

const handleStartMatch = () => {
  const ok = startMatch()
  if (ok) {
    show('正在匹配...')
  }
}

const handleCancelMatch = () => {
  cancelMatch()
  show('已取消匹配')
}

const handleOpenSettings = () => {
  showTopMenu.value = false
  settingsMode.value = 'identity'
  showSettings.value = true
}

const handleOpenSystemSettings = () => {
  showTopMenu.value = false
  settingsMode.value = 'system'
  showSettings.value = true
}

const handleSwitchIdentity = () => {
  showTopMenu.value = false
  showSwitchIdentityDialog.value = true
}

const confirmSwitchIdentity = () => {
  disconnect(true)
  chatStore.exitChat()
  messageStore.resetAll()
  chatStore.clearAllUsers()
  chatStore.activeTab = 'history'
  chatStore.isMatching = false
  userStore.clearCurrentUser()
  router.push('/identity')
}

onMounted(async () => {
  if (!userStore.currentUser) {
    router.push('/identity')
    return
  }

  // 连接WebSocket
  connect()

  // 加载用户列表
  await loadUsers()
  await nextTick()
  if (listAreaRef.value && chatStore.listScrollTop > 0) {
    listAreaRef.value.scrollTop = chatStore.listScrollTop
  }
  window.addEventListener('match-success', handleMatchSuccess)
})

onUnmounted(() => {
  window.removeEventListener('match-success', handleMatchSuccess)
})
</script>
