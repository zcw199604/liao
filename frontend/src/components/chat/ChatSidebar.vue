<template>
  <div class="flex flex-col h-full bg-canvas relative">
    <!-- 顶部切换栏 -->
    <div class="ui-glass-topbar flex items-center justify-between pt-4 pb-2 px-4 z-10 shrink-0">
      <!-- 左侧：菜单按钮（下拉） -->
      <div class="relative">
        <button
          @click.stop="handleToggleTopMenu"
          class="ui-icon-btn ui-icon-btn-ghost text-fg-muted"
        >
          <i class="fas fa-bars text-xl"></i>
        </button>

        <!-- 下拉菜单 -->
        <div
          v-if="showTopMenu"
          @click.stop
          class="absolute left-0 top-12 w-48 ui-card-sm z-50"
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
	        class="h-full overflow-y-auto no-scrollbar px-4 pt-2 pb-24" 
	        ref="listAreaRef" 
	        @click="closeContextMenu"
	        :style="{ 
	          transform: `translateX(${listTranslateX}px)`, 
	          transition: isAnimating ? listSwipeTransition : 'none' 
	        }"
	      >
	        <!-- 列表搜索：位于滚动区域顶部，向下滚动时不固定在顶栏 -->
	        <div class="pb-2">
	          <div class="relative">
	            <i class="fas fa-search absolute left-3 top-1/2 -translate-y-1/2 text-xs text-fg-muted"></i>
	            <input
	              v-model="searchKeyword"
	              data-testid="chat-sidebar-search-input"
	              type="text"
	              placeholder="搜索用户（昵称/名称/ID/地址）"
	              class="w-full rounded-xl border border-line bg-surface-2 py-2 pl-9 pr-9 text-sm text-fg placeholder:text-fg-muted outline-none focus:ring-2 focus:ring-blue-500/30"
	            />
	            <button
	              v-if="searchKeyword"
	              @click="searchKeyword = ''"
	              class="absolute right-2 top-1/2 -translate-y-1/2 w-5 h-5 rounded-full bg-surface-3 text-fg-muted hover:text-fg transition"
	              aria-label="清空搜索"
	            >
	              <i class="fas fa-times text-[10px]"></i>
	            </button>
	          </div>
	        </div>
	
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
	            v-for="user in filteredDisplayList"
	            :key="user.id"
	            @click="handleClick(user, $event)"
	            @touchstart="startLongPress(user, $event)"
	            @touchend="endLongPress"
	            @touchmove="cancelLongPress"
	            @mousedown="startLongPress(user, $event)"
	            @mouseup="endLongPress"
	            @mouseleave="cancelLongPress"
	            @contextmenu.prevent="handleContextMenu(user, $event)"
	            class="ui-list-item flex items-center p-4 mb-3 cursor-pointer"
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
	                  <span class="font-bold text-base text-fg truncate">
	                    <template v-for="(part, index) in getHighlightParts(user.nickname)" :key="`nickname-${user.id}-${index}`">
	                      <span :class="part.match ? 'search-highlight rounded px-0.5' : ''">{{ part.text }}</span>
	                    </template>
	                  </span>
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
	                     <template v-for="(part, index) in getHighlightParts(user.address)" :key="`address-${user.id}-${index}`">
	                       <span :class="part.match ? 'search-highlight rounded px-0.5' : ''">{{ part.text }}</span>
	                     </template>
	                   </span>
	                   <p class="text-sm text-fg-muted truncate flex-1">
	                     <template v-for="(part, index) in getHighlightParts(user.lastMsg)" :key="`lastmsg-${user.id}-${index}`">
	                       <span :class="part.match ? 'search-highlight rounded px-0.5' : ''">{{ part.text }}</span>
	                     </template>
	                   </p>
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
	            v-if="showNoSearchMatch"
	            class="ui-empty-state mt-20"
	          >
	            <i class="fas fa-search text-5xl mb-4 opacity-50"></i>
	            <p class="text-sm">未找到匹配用户</p>
	          </div>
	          <div
	            v-else-if="(!chatStore.displayList || chatStore.displayList.length === 0) && !isInitialLoadingUsers"
	            class="ui-empty-state mt-20"
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
        <div class="flex items-center gap-2">
          <button
            @click="toggleSelectAll"
            class="ui-btn-secondary px-4 py-2 text-sm"
          >
            {{ isAllSelected ? '取消全选' : '全选当前列表' }}
          </button>

          <button
            @click="openDaySelector"
            class="ui-btn-secondary px-4 py-2 text-sm flex items-center gap-2"
          >
            <i class="fas fa-calendar-alt text-xs text-purple-400"></i>
            <span>按天...</span>
          </button>
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="exitSelectionMode"
            class="ui-btn-secondary px-3 py-2 text-sm"
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
      class="fixed z-50 w-32 ui-card-sm bg-surface-3/95 overflow-hidden p-0"
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

    <!-- 匹配按钮：选择模式下隐藏，避免遮挡底部批量操作栏 -->
    <MatchButton v-if="!selectionMode" />

    <!-- 匹配蒙层：选择模式下隐藏 -->
    <MatchOverlay v-if="!selectionMode" />

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

    <ChatDayBulkDeleteModal
      v-model:visible="showDayBulkDeleteModal"
      :items="dayDeleteItems"
      :preselect-key="dayDeletePreselectKey"
      @confirm="applyDaySelection"
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
import ChatDayBulkDeleteModal from '@/components/chat/ChatDayBulkDeleteModal.vue'
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

// 列表搜索：仅在当前 tab 的本地用户列表中过滤
const searchKeyword = ref('')
const normalizedSearchKeyword = computed(() => searchKeyword.value.trim().toLowerCase())
const filteredDisplayList = computed(() => {
  const keyword = normalizedSearchKeyword.value
  if (!keyword) return chatStore.displayList

  return chatStore.displayList.filter((user) => {
    const fields = [user.nickname, user.name, user.id, user.address]
    return fields.some((field) => String(field || '').toLowerCase().includes(keyword))
  })
})

const showNoSearchMatch = computed(() => {
  return normalizedSearchKeyword.value.length > 0 &&
    chatStore.displayList.length > 0 &&
    filteredDisplayList.value.length === 0 &&
    !isInitialLoadingUsers.value
})

interface HighlightPart {
  text: string
  match: boolean
}

/**
 * 将文本拆分为“普通片段 + 高亮片段”，用于模板安全渲染。
 * Args:
 *   raw: 原始文本（支持空值）
 * Returns:
 *   HighlightPart[]: 可直接 v-for 渲染的文本片段数组
 */
const getHighlightParts = (raw: string | null | undefined): HighlightPart[] => {
  const text = String(raw || '')
  if (!text) {
    return [{ text: '', match: false }]
  }

  const keyword = normalizedSearchKeyword.value
  if (!keyword) {
    return [{ text, match: false }]
  }

  const lowerText = text.toLowerCase()
  if (!lowerText.includes(keyword)) {
    return [{ text, match: false }]
  }

  const parts: HighlightPart[] = []
  let lastIndex = 0
  let index = lowerText.indexOf(keyword)

  while (index !== -1) {
    if (index > lastIndex) {
      parts.push({
        text: text.slice(lastIndex, index),
        match: false
      })
    }

    parts.push({
      text: text.slice(index, index + keyword.length),
      match: true
    })

    lastIndex = index + keyword.length
    index = lowerText.indexOf(keyword, lastIndex)
  }

  if (lastIndex < text.length) {
    parts.push({
      text: text.slice(lastIndex),
      match: false
    })
  }

  return parts
}

// 批量删除（多选）状态
const selectionMode = ref(false)
const selectedUserIds = ref<string[]>([])
const showBatchDeleteDialog = ref(false)

const isSelected = (userId: string) => selectedUserIds.value.includes(userId)
const isAllSelected = computed(() => {
  const visibleIds = filteredDisplayList.value.map(user => user.id)
  if (visibleIds.length === 0) return false
  return visibleIds.every(id => selectedUserIds.value.includes(id))
})

const handleEnterSelectionMode = (preselect?: User) => {
  closeContextMenu()
  showTopMenu.value = false
  selectionMode.value = true
  dayDeletePreselectKey.value = preselect ? getDayKeyFromLastTime(preselect.lastTime || '') : undefined
  if (preselect && !isSelected(preselect.id)) {
    selectedUserIds.value = [preselect.id, ...selectedUserIds.value]
  }
}

const exitSelectionMode = () => {
  selectionMode.value = false
  selectedUserIds.value = []
  showBatchDeleteDialog.value = false
  showDayBulkDeleteModal.value = false
  dayDeletePreselectKey.value = undefined
}

const toggleSelection = (userId: string) => {
  if (isSelected(userId)) {
    selectedUserIds.value = selectedUserIds.value.filter(id => id !== userId)
    return
  }
  selectedUserIds.value = [...selectedUserIds.value, userId]
}

const toggleSelectAll = () => {
  const visibleIds = filteredDisplayList.value.map(user => user.id)
  if (visibleIds.length === 0) return

  const isVisibleAllSelected = visibleIds.every(id => selectedUserIds.value.includes(id))
  if (isVisibleAllSelected) {
    selectedUserIds.value = selectedUserIds.value.filter(id => !visibleIds.includes(id))
    return
  }

  selectedUserIds.value = Array.from(new Set([...selectedUserIds.value, ...visibleIds]))
}

const confirmBatchDelete = () => {
  if (selectedUserIds.value.length === 0) return
  showBatchDeleteDialog.value = true
}

// 按天批量删除（从当前列表按 lastTime 分组）
const showDayBulkDeleteModal = ref(false)
const dayDeletePreselectKey = ref<string | undefined>(undefined)

const parseLastTimeToDate = (timeStr: string): Date | null => {
  const raw = (timeStr || '').trim()
  if (!raw) return null

  let date = new Date(raw)
  // 兼容 iOS/Safari：yyyy-MM-dd
  if (Number.isNaN(date.getTime())) {
    date = new Date(raw.replace(/-/g, '/'))
  }
  if (Number.isNaN(date.getTime())) return null
  return date
}

const toDayKey = (d: Date) => {
  const y = d.getFullYear().toString()
  const m = (d.getMonth() + 1).toString().padStart(2, '0')
  const day = d.getDate().toString().padStart(2, '0')
  return `${y}-${m}-${day}`
}

const getDayKeyFromLastTime = (timeStr: string) => {
  const d = parseLastTimeToDate(timeStr)
  if (!d) return 'unknown'
  return toDayKey(d)
}

const getTodayKey = () => toDayKey(new Date())

const getYesterdayKey = () => {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return toDayKey(d)
}

const dayDeleteGroupMap = computed(() => {
  const map = new Map<string, string[]>()
  for (const user of filteredDisplayList.value) {
    const key = getDayKeyFromLastTime(user.lastTime || '')
    const arr = map.get(key)
    if (arr) {
      arr.push(user.id)
    } else {
      map.set(key, [user.id])
    }
  }
  return map
})

const dayDeleteItems = computed(() => {
  const todayKey = getTodayKey()
  const yesterdayKey = getYesterdayKey()

  const items = Array.from(dayDeleteGroupMap.value.entries()).map(([key, ids]) => {
    let label = key === 'unknown' ? '未知时间' : key
    if (key !== 'unknown') {
      if (key === todayKey) label = `${key}（今天）`
      else if (key === yesterdayKey) label = `${key}（昨天）`
    }
    return { key, label, count: ids.length }
  })

  items.sort((a, b) => {
    if (a.key === 'unknown') return 1
    if (b.key === 'unknown') return -1
    return a.key < b.key ? 1 : -1 // 新日期在前
  })
  return items
})

const openDaySelector = () => {
  showDayBulkDeleteModal.value = true
}

const applyDaySelection = (dayKeys: string[]) => {
  const keySet = new Set(dayKeys)
  const ids: string[] = []
  for (const user of filteredDisplayList.value) {
    const key = getDayKeyFromLastTime(user.lastTime || '')
    if (keySet.has(key)) ids.push(user.id)
  }

  selectedUserIds.value = ids
  showDayBulkDeleteModal.value = false
  if (ids.length === 0) {
    show('没有可选会话')
  } else {
    show(`已选中 ${ids.length} 个会话`)
  }
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


// 切换 tab 后清空搜索，避免把上一个 tab 的筛选条件带到新列表
watch(() => chatStore.activeTab, () => {
  searchKeyword.value = ''
})

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
