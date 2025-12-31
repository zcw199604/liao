<template>
  <div class="flex flex-col h-full bg-[#0f0f13] relative">
    <!-- 顶部切换栏 -->
    <div class="flex items-center justify-between pt-4 pb-2 px-4 bg-[#0f0f13] z-10 shrink-0">
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
          @click="handleTabSwitch('history')"
          :class="chatStore.activeTab === 'history' ? 'bg-[#2d2d33] text-white shadow-md' : 'text-gray-500'"
          class="px-6 py-1.5 rounded-full text-sm font-medium transition-all duration-300"
        >
          消息
        </button>
        <button
          @click="handleTabSwitch('favorite')"
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

    <!-- 加载状态指示器 -->
    <div v-if="isRefreshing" class="h-1 bg-blue-500 animate-pulse shrink-0"></div>

    <!-- 列表内容 -->
    <div class="flex-1 overflow-y-auto no-scrollbar px-4 pt-2" ref="listAreaRef" @click="showTopMenu = false">
      <div
        v-for="user in chatStore.displayList"
        :key="user.id"
        @click="handleUserClick(user)"
        class="flex items-center p-4 mb-3 bg-[#18181b] rounded-2xl active:scale-[0.98] transition-transform duration-100 cursor-pointer"
        :class="{ 'border border-blue-500/30': currentUserId === user.id }"
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

    <!-- 匹配按钮 -->
    <MatchButton />

    <!-- 匹配蒙层 -->
    <MatchOverlay />

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
import MatchButton from '@/components/chat/MatchButton.vue'
import MatchOverlay from '@/components/chat/MatchOverlay.vue'
import type { User } from '@/types'

const props = defineProps<{
  currentUserId?: string // 当前选中的用户ID（用于高亮）
}>()

const emit = defineEmits<{
  (e: 'select', user: User): void
  (e: 'match-success', user: User): void
}>()

const router = useRouter()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageStore = useMessageStore()
const { loadUsers } = useChat()
const { connect, disconnect } = useWebSocket()
const { show } = useToast()

const showTopMenu = ref(false)
const showSettings = ref(false)
const settingsMode = ref<'identity' | 'system'>('identity')
const showSwitchIdentityDialog = ref(false)
const listAreaRef = ref<HTMLElement | null>(null)
const isRefreshing = ref(false)

// 刷新当前tab的数据
const refreshCurrentTab = async () => {
  if (isRefreshing.value || !userStore.currentUser) return

  isRefreshing.value = true
  try {
    if (chatStore.activeTab === 'history') {
      await chatStore.loadHistoryUsers(
        userStore.currentUser.id,
        userStore.currentUser.name
      )
    } else {
      await chatStore.loadFavoriteUsers(
        userStore.currentUser.id,
        userStore.currentUser.name
      )
    }
  } catch (error) {
    console.error('刷新列表失败:', error)
    show('刷新失败，请稍后重试')
  } finally {
    isRefreshing.value = false
  }
}

// 处理tab切换
const handleTabSwitch = async (tab: 'history' | 'favorite') => {
  if (chatStore.activeTab === tab) {
    await refreshCurrentTab()
    return
  }
  chatStore.activeTab = tab
}

const handleMatchSuccess = (e: any) => {
  const matchedUser = e.detail as User
  emit('match-success', matchedUser)
}

const handleUserClick = (user: User) => {
  chatStore.listScrollTop = listAreaRef.value?.scrollTop || 0
  emit('select', user)
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
  chatStore.cancelContinuousMatch()
  userStore.clearCurrentUser()
  router.push('/identity')
}

onMounted(async () => {
  if (!userStore.currentUser) {
    // 父组件会处理重定向，这里不需要
    return
  }

  // 连接WebSocket
  connect()

  // 加载数据
  if (chatStore.historyUsers.length === 0 && chatStore.favoriteUsers.length === 0) {
    await loadUsers()
  }
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
