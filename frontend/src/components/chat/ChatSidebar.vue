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

    <!-- 加载状态指示器 (用于 Tab 切换刷新时的反馈) -->
    <div v-if="isRefreshing" class="h-0.5 w-full bg-[#18181b] overflow-hidden shrink-0">
       <div class="h-full bg-blue-500 animate-progress-indeterminate"></div>
    </div>

    <!-- 列表内容 -->
    <PullToRefresh :on-refresh="refreshCurrentTab" class="flex-1 min-h-0">
      <div class="h-full overflow-y-auto no-scrollbar px-4 pt-2" ref="listAreaRef" @click="closeContextMenu">
        <div
          v-for="user in chatStore.displayList"
          :key="user.id"
          @click="handleClick(user)"
          @touchstart="startLongPress(user, $event)"
          @touchend="endLongPress"
          @touchmove="cancelLongPress"
          @mousedown="startLongPress(user, $event)"
          @mouseup="endLongPress"
          @mouseleave="cancelLongPress"
          @contextmenu.prevent="handleContextMenu(user, $event)"
          class="flex items-center p-4 mb-3 bg-[#18181b] rounded-2xl active:scale-[0.98] transition-transform duration-100 cursor-pointer select-none"
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
              <div class="flex items-center gap-2 truncate">
                <span class="font-bold text-base text-white truncate">{{ user.nickname }}</span>
                <!-- 性别年龄标签 -->
                <span 
                  v-if="user.sex !== '未知' || (user.age && user.age !== '0')"
                  class="px-1.5 py-0.5 rounded text-[10px] flex items-center gap-1 shrink-0"
                  :class="user.sex === '男' ? 'bg-blue-500/20 text-blue-400' : (user.sex === '女' ? 'bg-pink-500/20 text-pink-400' : 'bg-gray-500/20 text-gray-400')"
                >
                  <i v-if="user.sex === '男'" class="fas fa-mars"></i>
                  <i v-else-if="user.sex === '女'" class="fas fa-venus"></i>
                  <span v-if="user.age && user.age !== '0'">{{ user.age }}</span>
                </span>
              </div>
              <span class="text-xs text-gray-500 shrink-0 ml-2">{{ formatTime(user.lastTime || '') }}</span>
            </div>
            
            <div class="flex justify-between items-center">
              <div class="flex items-center gap-2 min-w-0 flex-1">
                 <!-- 地址显示 (如果有) -->
                 <span v-if="user.address && user.address !== '未知' && user.address !== '保密'" class="px-1.5 py-0.5 bg-gray-700/50 rounded text-[10px] text-gray-400 truncate max-w-[80px]">
                   {{ user.address }}
                 </span>
                 <p class="text-sm text-gray-400 truncate flex-1">{{ user.lastMsg }}</p>
              </div>
              <!-- 收藏标识 -->
              <i v-if="user.isFavorite && chatStore.activeTab === 'history'" class="fas fa-star text-xs text-yellow-500 ml-2 shrink-0"></i>
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
          </div>
        </div>
  
        <!-- 空状态提示 -->
        <div v-if="!chatStore.displayList || chatStore.displayList.length === 0" class="flex flex-col items-center justify-center mt-20 text-gray-600">
          <i class="far fa-comments text-5xl mb-4 opacity-50"></i>
          <p class="text-sm">暂无{{ chatStore.activeTab === 'history' ? '消息' : '收藏' }}</p>
        </div>
      </div>
    </PullToRefresh>

    <!-- 上下文菜单 (长按/右键触发) -->
    <div
      v-if="showContextMenu && contextMenuUser"
      class="fixed z-50 w-32 bg-[#27272a] rounded-lg shadow-xl border border-gray-700 overflow-hidden"
      :style="{ top: contextMenuPos.y + 'px', left: contextMenuPos.x + 'px' }"
      @click.stop
    >
      <button
        @click="handleCheckOnlineStatus(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm text-gray-300 hover:bg-[#3f3f46] hover:text-white flex items-center gap-2 transition border-b border-gray-700"
      >
        <i class="fas fa-signal text-xs text-green-500"></i>
        <span>在线记录</span>
      </button>
      <button
        @click="handleToggleGlobalFavorite(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm hover:bg-[#3f3f46] hover:text-white flex items-center gap-2 transition border-b border-gray-700"
        :class="isGlobalFavorite(contextMenuUser!) ? 'text-yellow-500' : 'text-gray-300'"
      >
        <i class="fas fa-star text-xs"></i>
        <span>{{ isGlobalFavorite(contextMenuUser!) ? '取消全局收藏' : '全局收藏' }}</span>
      </button>
      <button
        @click="confirmDeleteUser(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm text-red-400 hover:bg-[#3f3f46] hover:text-red-300 flex items-center gap-2 transition"
      >
        <i class="fas fa-trash-alt text-xs"></i>
        <span>删除会话</span>
      </button>
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
import { formatTime } from '@/utils/time'
import Toast from '@/components/common/Toast.vue'
import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import Dialog from '@/components/common/Dialog.vue'
import PullToRefresh from '@/components/common/PullToRefresh.vue'
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
const listAreaRef = ref<HTMLElement | null>(null)
const isRefreshing = ref(false)

// 上下文菜单状态
const showContextMenu = ref(false)
const contextMenuPos = ref({ x: 0, y: 0 })
const contextMenuUser = ref<User | null>(null)
let longPressTimer: any = null
let isLongPressHandled = false

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

// 长按/右键菜单逻辑
const startLongPress = (user: User, event: MouseEvent | TouchEvent) => {
  isLongPressHandled = false
  
  // 记录点击位置
  let clientX, clientY
  if (window.TouchEvent && event instanceof TouchEvent) {
    // @ts-ignore
    if (event.touches && event.touches.length > 0) {
      // @ts-ignore
      clientX = event.touches[0].clientX
      // @ts-ignore
      clientY = event.touches[0].clientY
    } else {
      return
    }
  } else if (event instanceof MouseEvent) {
    clientX = event.clientX
    clientY = event.clientY
  } else {
    return
  }

  longPressTimer = setTimeout(() => {
    isLongPressHandled = true
    showContextMenu.value = true
    contextMenuUser.value = user
    // 简单的边界处理，避免溢出屏幕
    const menuWidth = 128
    const menuHeight = 120
    let x = clientX
    let y = clientY
    
    if (x + menuWidth > window.innerWidth) x = window.innerWidth - menuWidth - 10
    if (y + menuHeight > window.innerHeight) y = window.innerHeight - menuHeight - 10
    
    contextMenuPos.value = { x, y }
    // 触发震动反馈 (如果支持)
    if (navigator.vibrate) navigator.vibrate(50)
  }, 500)
}

const endLongPress = () => {
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }
}

const cancelLongPress = () => {
  if (longPressTimer) {
    clearTimeout(longPressTimer)
    longPressTimer = null
  }
}

const handleContextMenu = (user: User, event: MouseEvent) => {
  // PC端右键直接触发
  cancelLongPress() // 清除长按定时器，避免冲突
  isLongPressHandled = true // 标记为已处理，阻止点击进入
  
  showContextMenu.value = true
  contextMenuUser.value = user
  
  const menuWidth = 128
  const menuHeight = 120
  let x = event.clientX
  let y = event.clientY
  
  if (x + menuWidth > window.innerWidth) x = window.innerWidth - menuWidth - 10
  if (y + menuHeight > window.innerHeight) y = window.innerHeight - menuHeight - 10
  
  contextMenuPos.value = { x, y }
}

const closeContextMenu = () => {
  showContextMenu.value = false
  contextMenuUser.value = null
}

const handleCheckOnlineStatus = (user: User) => {
  closeContextMenu()
  checkUserOnlineStatus(user.id)
  show('正在查询...')
}

const isGlobalFavorite = (user: User) => {
  if (!userStore.currentUser) return false
  return favoriteStore.isFavorite(userStore.currentUser.id, user.id)
}

const handleToggleGlobalFavorite = async (user: User) => {
  closeContextMenu()
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
  closeContextMenu()
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

const handleClick = (user: User) => {
  if (isLongPressHandled) {
    isLongPressHandled = false
    return
  }
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
  document.addEventListener('click', closeContextMenu)
})

onUnmounted(() => {
  window.removeEventListener('match-success', handleMatchSuccess)
  window.removeEventListener('check-online-result', onCheckOnlineResult)
  document.removeEventListener('click', closeContextMenu)
})
</script>
