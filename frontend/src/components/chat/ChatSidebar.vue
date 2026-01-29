<template>
  <div class="flex flex-col h-full bg-canvas relative">
    <!-- 顶部切换栏 -->
    <div class="flex items-center justify-between pt-4 pb-2 px-4 bg-canvas z-10 shrink-0">
      <!-- 左侧：菜单按钮（下拉） -->
      <div class="relative">
        <button
          @click.stop="handleToggleTopMenu"
          class="w-10 h-10 flex items-center justify-center text-fg-muted hover:text-fg transition"
        >
          <i class="fas fa-bars text-xl"></i>
        </button>

        <!-- 下拉菜单 -->
        <div
          v-if="showTopMenu"
          @click.stop
          class="absolute left-0 top-12 w-48 bg-surface rounded-xl shadow-2xl border border-line-strong z-50"
        >
          <button
            @click="handleOpenSettings"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 rounded-t-xl transition"
          >
            <i class="fas fa-user-edit text-blue-400"></i>
            <span>身份信息</span>
          </button>
          <button
            @click="handleOpenSystemSettings"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line-strong transition"
          >
            <i class="fas fa-cog text-fg-muted"></i>
            <span>系统设置</span>
          </button>
          <button
            @click="handleOpenMediaManagement"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line-strong transition"
          >
            <i class="fas fa-images text-purple-400"></i>
            <span>图片管理</span>
          </button>
          <button
            @click="handleOpenFavorites"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line-strong transition"
          >
            <i class="fas fa-star text-yellow-500"></i>
            <span>全局收藏</span>
          </button>
          <button
            @click="handleEnterSelectionMode()"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-red-500 flex items-center gap-3 border-t border-line-strong transition"
          >
            <i class="fas fa-check-square text-red-500"></i>
            <span>批量删除</span>
          </button>
          <button
            @click="handleSwitchIdentity"
            class="w-full px-4 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line-strong rounded-b-xl transition"
          >
            <i class="fas fa-user-circle text-indigo-400"></i>
            <span>切换身份</span>
          </button>
        </div>
      </div>

      <!-- 中间：切换栏 -->
      <div class="flex bg-surface-2 p-1 rounded-full border border-line">
        <button
          @click="handleTabSwitch('history')"
          :class="chatStore.activeTab === 'history' ? 'bg-surface-active text-fg shadow-md' : 'text-fg-subtle'"
          class="px-6 py-1.5 rounded-full text-sm font-medium transition-all duration-300"
        >
          消息
        </button>
        <button
          @click="handleTabSwitch('favorite')"
          :class="chatStore.activeTab === 'favorite' ? 'bg-surface-active text-fg shadow-md' : 'text-fg-subtle'"
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
    <div v-if="isRefreshing" class="h-0.5 w-full bg-surface overflow-hidden shrink-0">
       <div class="h-full bg-blue-500 animate-progress-indeterminate"></div>
    </div>

    <!-- 列表内容 -->
	    <PullToRefresh :on-refresh="refreshCurrentTab" class="flex-1 min-h-0">
	      <div 
	        class="h-full overflow-y-auto no-scrollbar px-4 pt-2" 
	        ref="listAreaRef" 
	        @click="closeContextMenu"
	        :style="{ 
	          transform: `translateX(${listTranslateX}px)`, 
	          transition: isAnimating ? listSwipeTransition : 'none' 
	        }"
	      >
	        <!-- 骨架屏：首次加载用户列表时占位，减少跳动 -->
	        <template v-if="isInitialLoadingUsers && (!chatStore.displayList || chatStore.displayList.length === 0)">
	          <div v-for="i in 8" :key="'sk-user-' + i" class="flex items-center p-4 mb-3 bg-surface rounded-2xl">
	            <Skeleton class="w-12 h-12 rounded-xl" />
	            <div class="ml-4 flex-1 min-w-0">
	              <Skeleton class="h-4 w-28 rounded" />
	              <Skeleton class="h-3 w-44 rounded mt-2" />
	            </div>
	            <div class="ml-2 w-10 flex justify-end">
	              <Skeleton class="h-4 w-6 rounded" />
	            </div>
	          </div>
	        </template>

	        <template v-else>
	          <div
	            v-for="user in chatStore.displayList"
	            :key="user.id"
	            @click="handleClick(user, $event)"
	            @touchstart="startLongPress(user, $event)"
	            @touchend="endLongPress"
	            @touchmove="cancelLongPress"
	            @mousedown="startLongPress(user, $event)"
	            @mouseup="endLongPress"
	            @mouseleave="cancelLongPress"
	            @contextmenu.prevent="handleContextMenu(user, $event)"
	            class="flex items-center p-4 mb-3 bg-surface rounded-2xl active:scale-[0.98] transition-transform duration-100 cursor-pointer select-none"
	            :class="[
	              { 'border border-blue-500/30': currentUserId === user.id && !selectionMode },
	              selectionMode && isSelected(user.id) ? 'ring-2 ring-blue-500/30 border border-blue-500/30' : ''
	            ]"
	          >
	            <!-- 多选标记 -->
	            <div
	              v-if="selectionMode"
	              class="w-6 h-6 rounded-full border border-line-strong flex items-center justify-center shrink-0 mr-3"
	              :class="isSelected(user.id) ? 'bg-blue-600 border-blue-500 text-white' : 'bg-surface-3 text-fg-muted'"
	            >
	              <i v-if="isSelected(user.id)" class="fas fa-check text-xs"></i>
	            </div>

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
	                  <span class="font-bold text-base text-fg truncate">{{ user.nickname }}</span>
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
	                <span class="text-xs text-fg-subtle shrink-0 ml-2">{{ formatTime(user.lastTime || '') }}</span>
	              </div>
	              
	              <div class="flex justify-between items-center">
	                <div class="flex items-center gap-2 min-w-0 flex-1">
	                   <!-- 地址显示 (如果有) -->
	                   <span v-if="user.address && user.address !== '未知' && user.address !== '保密'" class="px-1.5 py-0.5 bg-surface-3/70 rounded text-[10px] text-fg-muted truncate max-w-[80px]">
	                     {{ user.address }}
	                   </span>
	                   <p class="text-sm text-fg-muted truncate flex-1">{{ user.lastMsg }}</p>
	                </div>
	                <!-- 收藏标识 -->
	                <i v-if="user.isFavorite && chatStore.activeTab === 'history'" class="fas fa-star text-xs text-yellow-500 ml-2 shrink-0"></i>
	              </div>
	            </div>
	  
	            <!-- 右侧操作栏 -->
	            <div class="flex flex-col items-end justify-center ml-2 gap-2 relative">
	                <!-- 未读消息数气泡 -->
	                <DraggableBadge
	                  v-if="user.unreadCount && user.unreadCount > 0"
	                  :count="user.unreadCount"
	                  @clear="handleClearUnread(user)"
	                />
	            </div>
	          </div>
	  
	          <!-- 空状态提示 -->
	          <div
	            v-if="(!chatStore.displayList || chatStore.displayList.length === 0) && !isInitialLoadingUsers"
	            class="flex flex-col items-center justify-center mt-20 text-fg-subtle"
	          >
	            <i class="far fa-comments text-5xl mb-4 opacity-50"></i>
	            <p class="text-sm">暂无{{ chatStore.activeTab === 'history' ? '消息' : '收藏' }}</p>
	          </div>
	        </template>
	      </div>
	    </PullToRefresh>

    <!-- 批量删除底栏 -->
    <div v-if="selectionMode" class="shrink-0 px-4 py-3 border-t border-line bg-canvas">
      <div class="flex items-center justify-between gap-3">
        <button
          @click="toggleSelectAll"
          class="px-4 py-2 bg-surface-3 text-fg rounded-lg hover:bg-surface-hover transition text-sm border border-line"
        >
          {{ isAllSelected ? '取消全选' : '全选当前列表' }}
        </button>

        <div class="flex items-center gap-2">
          <button
            @click="exitSelectionMode"
            class="px-3 py-2 bg-surface-3 text-fg rounded-lg hover:bg-surface-hover transition text-sm border border-line"
          >
            取消
          </button>

          <button
            @click="confirmBatchDelete"
            :disabled="selectedUserIds.length === 0"
            class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            <i class="fas fa-trash"></i>
            <span>删除 ({{ selectedUserIds.length }})</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 上下文菜单 (长按/右键触发) -->
    <div
      v-if="showContextMenu && contextMenuUser"
      class="fixed z-50 w-32 bg-surface-3 rounded-lg shadow-xl border border-line-strong overflow-hidden"
      :style="{ top: contextMenuPos.y + 'px', left: contextMenuPos.x + 'px' }"
      @click.stop
    >
      <button
        @click="handleCheckOnlineStatus(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm text-fg-muted hover:bg-surface-hover hover:text-fg flex items-center gap-2 transition border-b border-line-strong"
      >
        <i class="fas fa-signal text-xs text-green-500"></i>
        <span>在线记录</span>
      </button>
      <button
        @click="handleToggleGlobalFavorite(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm hover:bg-surface-hover hover:text-fg flex items-center gap-2 transition border-b border-line-strong"
        :class="isGlobalFavorite(contextMenuUser!) ? 'text-yellow-500' : 'text-fg-muted'"
      >
        <i class="fas fa-star text-xs"></i>
        <span>{{ isGlobalFavorite(contextMenuUser!) ? '取消全局收藏' : '全局收藏' }}</span>
      </button>
      <button
        @click="handleEnterSelectionMode(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm text-fg-muted hover:bg-surface-hover hover:text-fg flex items-center gap-2 transition border-b border-line-strong"
      >
        <i class="fas fa-check-square text-xs text-blue-500"></i>
        <span>多选删除</span>
      </button>
      <button
        @click="confirmDeleteUser(contextMenuUser!)"
        class="w-full px-4 py-2 text-left text-sm text-red-500 hover:bg-surface-hover hover:text-red-400 flex items-center gap-2 transition"
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

    <Dialog
      v-model:visible="showBatchDeleteDialog"
      title="批量删除会话"
      :content="`确定要删除选中的 ${selectedUserIds.length} 个会话吗？`"
      show-warning
      @confirm="executeBatchDelete"
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
           <div class="text-fg-muted">当前状态:</div>
           <div :class="onlineStatusData.isOnline ? 'text-green-500 font-bold' : 'text-fg-subtle font-bold'">
             {{ onlineStatusData.isOnline ? '在线' : '离线' }}
           </div>
        </div>
        <div class="text-sm text-fg-subtle">
          最近登录: {{ onlineStatusData.lastTime || '未知' }}
        </div>
      </div>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useFavoriteStore } from '@/stores/favorite'
import { useChat } from '@/composables/useChat'
import { useWebSocket } from '@/composables/useWebSocket'
import { useToast } from '@/composables/useToast'
import { useSwipeAction } from '@/composables/useInteraction'
import { getColorClass } from '@/constants/colors'
import { TAB_SWIPE_THRESHOLD, SWIPE_RESET_DURATION_MS, CONTEXT_MENU_WIDTH, CONTEXT_MENU_HEIGHT } from '@/constants/interaction'
import { formatTime } from '@/utils/time'
import Toast from '@/components/common/Toast.vue'
import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import Dialog from '@/components/common/Dialog.vue'
	import PullToRefresh from '@/components/common/PullToRefresh.vue'
	import Skeleton from '@/components/common/Skeleton.vue'
	import MatchButton from '@/components/chat/MatchButton.vue'
	import MatchOverlay from '@/components/chat/MatchOverlay.vue'
import DraggableBadge from '@/components/common/DraggableBadge.vue'
import type { User } from '@/types'
import { deleteUser, batchDeleteUsers } from '@/api/chat'

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
const settingsMode = ref<'identity' | 'system' | 'media' | 'favorites'>('identity')
const showSwitchIdentityDialog = ref(false)
const showDeleteUserDialog = ref(false)
	const userToDelete = ref<User | null>(null)
	const listAreaRef = ref<HTMLElement | null>(null)
const isRefreshing = ref(false)
	const isInitialLoadingUsers = ref(false)

// 批量删除（多选）状态
const selectionMode = ref(false)
const selectedUserIds = ref<string[]>([])
const showBatchDeleteDialog = ref(false)

const isSelected = (userId: string) => selectedUserIds.value.includes(userId)
const isAllSelected = computed(() => {
  const total = chatStore.displayList?.length || 0
  return total > 0 && selectedUserIds.value.length === total
})

const handleEnterSelectionMode = (preselect?: User) => {
  closeContextMenu()
  showTopMenu.value = false
  selectionMode.value = true
  if (preselect && !isSelected(preselect.id)) {
    selectedUserIds.value = [preselect.id, ...selectedUserIds.value]
  }
}

const exitSelectionMode = () => {
  selectionMode.value = false
  selectedUserIds.value = []
  showBatchDeleteDialog.value = false
}

const toggleSelection = (userId: string) => {
  if (isSelected(userId)) {
    selectedUserIds.value = selectedUserIds.value.filter(id => id !== userId)
    return
  }
  selectedUserIds.value = [...selectedUserIds.value, userId]
}

const toggleSelectAll = () => {
  const ids = (chatStore.displayList || []).map(u => u.id)
  if (selectedUserIds.value.length === ids.length) {
    selectedUserIds.value = []
    return
  }
  selectedUserIds.value = ids
}

const confirmBatchDelete = () => {
  if (selectedUserIds.value.length === 0) return
  showBatchDeleteDialog.value = true
}

// 列表偏移量 (用于跟手滑动)
const listTranslateX = ref(0)
const isAnimating = ref(false)
let resetTranslateTimer: number | null = null
const listSwipeTransition = `transform ${SWIPE_RESET_DURATION_MS}ms cubic-bezier(0.25, 0.46, 0.45, 0.94)`

const resetListTranslateX = () => {
  if (resetTranslateTimer !== null) {
    clearTimeout(resetTranslateTimer)
    resetTranslateTimer = null
  }

  // 已经处于 0 偏移时无需回弹，避免误触发动画状态
  if (listTranslateX.value === 0) {
    isAnimating.value = false
    return
  }

  isAnimating.value = true
  listTranslateX.value = 0
  resetTranslateTimer = window.setTimeout(() => {
    isAnimating.value = false
    resetTranslateTimer = null
  }, SWIPE_RESET_DURATION_MS)
}

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
  if (selectionMode.value) return
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
    let x = clientX
    let y = clientY
    
    if (x + CONTEXT_MENU_WIDTH > window.innerWidth) x = window.innerWidth - CONTEXT_MENU_WIDTH - 10
    if (y + CONTEXT_MENU_HEIGHT > window.innerHeight) y = window.innerHeight - CONTEXT_MENU_HEIGHT - 10
    
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
  if (selectionMode.value) return
  // PC端右键直接触发
  cancelLongPress() // 清除长按定时器，避免冲突
  isLongPressHandled = true // 标记为已处理，阻止点击进入
  
  showContextMenu.value = true
  contextMenuUser.value = user
  
  let x = event.clientX
  let y = event.clientY
  
  if (x + CONTEXT_MENU_WIDTH > window.innerWidth) x = window.innerWidth - CONTEXT_MENU_WIDTH - 10
  if (y + CONTEXT_MENU_HEIGHT > window.innerHeight) y = window.innerHeight - CONTEXT_MENU_HEIGHT - 10
  
  contextMenuPos.value = { x, y }
}

const closeContextMenu = () => {
  showContextMenu.value = false
  contextMenuUser.value = null
}

// 列表滑动交互
const { isSwiping } = useSwipeAction(listAreaRef, {
  threshold: TAB_SWIPE_THRESHOLD,
  passive: true, // 保持滚动流畅，但在横滑判定后需要注意
  onSwipeProgress: (deltaX, deltaY) => {
    if (selectionMode.value) return
    if (showContextMenu.value) return // 菜单打开时不滑动

    // 如用户在回弹过程中再次滑动，立即取消回弹动画，确保跟手
    if (resetTranslateTimer !== null) {
      clearTimeout(resetTranslateTimer)
      resetTranslateTimer = null
    }
    if (isAnimating.value) isAnimating.value = false
    
    // 简单的方向锁定：如果垂直移动明显，则忽略水平滑动
    if (Math.abs(deltaY) > Math.abs(deltaX)) {
      listTranslateX.value = 0
      return
    }
    
    // 限制最大滑动距离，增加阻尼感
    const limit = 120
    const dampening = 0.5
    let move = deltaX * dampening
    if (move > limit) move = limit + (move - limit) * 0.2
    if (move < -limit) move = -limit + (move + limit) * 0.2
    
    listTranslateX.value = move
  },
  onSwipeEnd: (direction) => {
    if (selectionMode.value) return
    if (showContextMenu.value) {
      closeContextMenu()
      return
    }

    showTopMenu.value = false
    
    // 触发切换
    if (direction === 'left' && chatStore.activeTab === 'history') {
      chatStore.activeTab = 'favorite'
    } else if (direction === 'right' && chatStore.activeTab === 'favorite') {
      chatStore.activeTab = 'history'
    }
  },
  onSwipeFinish: (_deltaX, _deltaY, _isTriggered) => {
    if (selectionMode.value) return
    if (showContextMenu.value) {
      // 菜单打开时不允许横滑，避免依赖其它路径来复位位移：这里直接收敛到 0，且不做动画
      if (resetTranslateTimer !== null) {
        clearTimeout(resetTranslateTimer)
        resetTranslateTimer = null
      }
      isAnimating.value = false
      listTranslateX.value = 0
      return
    }

    // 无论是否触发 Tab 切换，手势结束都需要回弹复位，避免残留偏移卡住
    resetListTranslateX()
  }
})

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

const handleToggleTopMenu = () => {
  closeContextMenu()
  showTopMenu.value = !showTopMenu.value
}

// 处理tab切换
const handleTabSwitch = async (tab: 'history' | 'favorite') => {
  if (selectionMode.value) {
    show('请先退出选择模式')
    return
  }
  closeContextMenu()
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

const handleClearUnread = (user: User) => {
  if (user.unreadCount && user.unreadCount > 0) {
    chatStore.updateUser(user.id, { unreadCount: 0 })
    if (navigator.vibrate) navigator.vibrate(50)
  }
}

const handleClick = (user: User, event?: MouseEvent) => {
  if (selectionMode.value) {
    toggleSelection(user.id)
    event?.stopPropagation()
    return
  }
  if (isLongPressHandled) {
    isLongPressHandled = false
    event?.stopPropagation()
    event?.preventDefault()
    return
  }
  chatStore.listScrollTop = listAreaRef.value?.scrollTop || 0
  emit('select', user)
}

const executeBatchDelete = async () => {
  if (!userStore.currentUser) return

  const myUserId = userStore.currentUser.id
  const ids = [...selectedUserIds.value]
  if (ids.length === 0) return

  const batchSize = 200
  const failed = new Set<string>()
  const failedReasons = new Map<string, string>()

  try {
    // 前端分批，避免 payload 过大/单次请求过慢。
    for (let i = 0; i < ids.length; i += batchSize) {
      const chunk = ids.slice(i, i + batchSize)
      try {
        const res: any = await batchDeleteUsers(myUserId, chunk)
        if (!res || res.code !== 0) {
          chunk.forEach(id => failed.add(id))
          continue
        }

        const items: any[] = res.data?.failedItems || []
        items.forEach(item => {
          const id = String(item.userToId || '').trim()
          if (!id) return
          failed.add(id)
          const reason = item.reason ? String(item.reason) : ''
          if (reason) failedReasons.set(id, reason)
        })
      } catch (e) {
        chunk.forEach(id => failed.add(id))
      }
    }

    const successIds = ids.filter(id => !failed.has(id))
    successIds.forEach(id => chatStore.removeUser(id))

    if (failed.size === 0) {
      show(`已删除 ${successIds.length} 个会话`)
      exitSelectionMode()
    } else {
      selectedUserIds.value = ids.filter(id => failed.has(id))
      show(`已删除 ${successIds.length} 个，失败 ${failed.size} 个`)
    }
  } finally {
    showBatchDeleteDialog.value = false
  }
}

const handleOpenSettings = () => {
  closeContextMenu()
  showTopMenu.value = false
  settingsMode.value = 'identity'
  showSettings.value = true
}

const handleOpenSystemSettings = () => {
  closeContextMenu()
  showTopMenu.value = false
  settingsMode.value = 'system'
  showSettings.value = true
}

const handleOpenMediaManagement = () => {
  closeContextMenu()
  showTopMenu.value = false
  settingsMode.value = 'media'
  showSettings.value = true
}

const handleOpenFavorites = () => {
  closeContextMenu()
  showTopMenu.value = false
  settingsMode.value = 'favorites'
  showSettings.value = true
}

const handleSwitchIdentity = () => {
  closeContextMenu()
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
	  isInitialLoadingUsers.value = true
	  try {
	    await loadUsers()
	  } finally {
	    isInitialLoadingUsers.value = false
	  }
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
  if (resetTranslateTimer !== null) {
    clearTimeout(resetTranslateTimer)
    resetTranslateTimer = null
  }
  window.removeEventListener('match-success', handleMatchSuccess)
  window.removeEventListener('check-online-result', onCheckOnlineResult)
  document.removeEventListener('click', closeContextMenu)
})
</script>
