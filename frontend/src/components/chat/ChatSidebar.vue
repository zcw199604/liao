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
            @click="handleOpenFavorites"
            class="w-full px-4 py-3 text-left hover:bg-[#27272a] text-white flex items-center gap-3 border-t border-gray-700 transition"
          >
            <i class="fas fa-star text-yellow-500"></i>
            <span>全局收藏</span>
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

        <!-- 右侧操作栏 -->
        <div class="flex flex-col items-end justify-center ml-2 gap-2 relative">
            <!-- 未读消息数气泡 -->
            <span
              v-if="user.unreadCount && user.unreadCount > 0"
              class="px-2 py-0.5 bg-red-500 text-white text-xs rounded-full min-w-[20px] text-center font-medium"
            >
              {{ user.unreadCount }}
            </span>
            <!-- 更多操作按钮 -->
            <div class="relative">
              <button
                @click.stop="toggleUserMenu(user.id)"
                class="w-6 h-6 flex items-center justify-center text-gray-500 hover:text-white transition rounded-full hover:bg-white/10"
                :class="{'text-white': activeMenuUserId === user.id}"
                title="更多操作"
              >
                <i class="fas fa-ellipsis-v text-xs"></i>
              </button>
              
              <!-- 下拉菜单 -->
              <div
                v-if="activeMenuUserId === user.id"
                @click.stop
                class="absolute right-0 top-8 w-32 bg-[#27272a] rounded-lg shadow-xl border border-gray-700 z-50 overflow-hidden"
              >
                <button
                  @click="handleCheckOnlineStatus(user)"
                  class="w-full px-4 py-2 text-left text-sm text-gray-300 hover:bg-[#3f3f46] hover:text-white flex items-center gap-2 transition border-b border-gray-700"
                >
                  <i class="fas fa-signal text-xs text-green-500"></i>
                  <span>在线记录</span>
                </button>
                <button
                  @click="handleToggleGlobalFavorite(user)"
                  class="w-full px-4 py-2 text-left text-sm hover:bg-[#3f3f46] hover:text-white flex items-center gap-2 transition border-b border-gray-700"
                  :class="isGlobalFavorite(user) ? 'text-yellow-500' : 'text-gray-300'"
                >
                  <i class="fas fa-star text-xs"></i>
                  <span>{{ isGlobalFavorite(user) ? '取消全局收藏' : '全局收藏' }}</span>
                </button>
                <button
                  @click="confirmDeleteUser(user)"
                  class="w-full px-4 py-2 text-left text-sm text-red-400 hover:bg-[#3f3f46] hover:text-red-300 flex items-center gap-2 transition"
                >
                  <i class="fas fa-trash-alt text-xs"></i>
                  <span>删除会话</span>
                </button>
              </div>
            </div>
        </div>
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

    <Dialog
      v-model:visible="showDeleteUserDialog"
      title="删除会话"
      content="确定要删除与该用户的会话吗？"
      @confirm="executeDeleteUser"
    />

    <!-- 在线状态弹窗 -->
    <Dialog
      v-model:visible="showOnlineStatusDialog"
      title="用户在线状态"
      :show-cancel="false"
      confirm-text="关闭"
      @confirm="showOnlineStatusDialog = false"
    >
      <div class="text-center py-4 space-y-3">
        <div class="flex items-center justify-center gap-2">
           <div class="text-gray-400">当前状态:</div>
           <div :class="onlineStatusData.isOnline ? 'text-green-500 font-bold' : 'text-gray-500 font-bold'">
             {{ onlineStatusData.isOnline ? '在线' : '离线' }}
           </div>
        </div>
        <div class="text-sm text-gray-500">
          最近登录: {{ onlineStatusData.lastTime || '未知' }}
        </div>
      </div>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useFavoriteStore } from '@/stores/favorite'
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
import { deleteUser } from '@/api/chat'

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
const favoriteStore = useFavoriteStore()
const { loadUsers } = useChat()
const { connect, disconnect, checkUserOnlineStatus } = useWebSocket()
const { show } = useToast()

const showTopMenu = ref(false)
const showSettings = ref(false)
const settingsMode = ref<'identity' | 'system' | 'favorites'>('identity')
const showSwitchIdentityDialog = ref(false)
const showDeleteUserDialog = ref(false)
const userToDelete = ref<User | null>(null)
const activeMenuUserId = ref<string | null>(null)
const listAreaRef = ref<HTMLElement | null>(null)
const isRefreshing = ref(false)

// 在线状态弹窗数据
const showOnlineStatusDialog = ref(false)
const onlineStatusData = ref({
  isOnline: false,
  lastTime: ''
})

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

const toggleUserMenu = (userId: string) => {
  if (activeMenuUserId.value === userId) {
    activeMenuUserId.value = null
  } else {
    activeMenuUserId.value = userId
  }
}

const closeUserMenu = () => {
  activeMenuUserId.value = null
}

const handleCheckOnlineStatus = (user: User) => {
  closeUserMenu()
  checkUserOnlineStatus(user.id)
  show('正在查询...')
}

const isGlobalFavorite = (user: User) => {
  if (!userStore.currentUser) return false
  return favoriteStore.isFavorite(userStore.currentUser.id, user.id)
}

const handleToggleGlobalFavorite = async (user: User) => {
  closeUserMenu()
  if (!userStore.currentUser) return

  const isFav = isGlobalFavorite(user)
  if (isFav) {
    await favoriteStore.removeFavorite(userStore.currentUser.id, user.id)
    show('已取消全局收藏')
  } else {
    await favoriteStore.addFavorite(userStore.currentUser.id, user.id, user.nickname || user.name)
    show('已加入全局收藏')
  }
}

const onCheckOnlineResult = (e: any) => {
  const data = e.detail
  if (data && data.data) {
    onlineStatusData.value = {
      isOnline: data.data.IF_Online === '1',
      lastTime: data.data.TimeAll
    }
    showOnlineStatusDialog.value = true
  }
}

const confirmDeleteUser = (user: User) => {
  closeUserMenu()
  userToDelete.value = user
  showDeleteUserDialog.value = true
}

const executeDeleteUser = async () => {
  if (!userToDelete.value || !userStore.currentUser) return
  
  try {
    await deleteUser(userStore.currentUser.id, userToDelete.value.id)
    chatStore.removeUser(userToDelete.value.id)
    show('删除成功')
  } catch (error) {
    console.error('删除失败', error)
    show('删除失败')
  } finally {
    showDeleteUserDialog.value = false
    userToDelete.value = null
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

const handleOpenFavorites = () => {
  showTopMenu.value = false
  settingsMode.value = 'favorites'
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
  // 加载全局收藏数据
  if (favoriteStore.allFavorites.length === 0) {
    favoriteStore.loadAllFavorites()
  }

  await nextTick()
  if (listAreaRef.value && chatStore.listScrollTop > 0) {
    listAreaRef.value.scrollTop = chatStore.listScrollTop
  }
  window.addEventListener('match-success', handleMatchSuccess)
  window.addEventListener('check-online-result', onCheckOnlineResult)
  document.addEventListener('click', closeUserMenu)
})

onUnmounted(() => {
  window.removeEventListener('match-success', handleMatchSuccess)
  window.removeEventListener('check-online-result', onCheckOnlineResult)
  document.removeEventListener('click', closeUserMenu)
})
</script>
