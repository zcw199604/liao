<template>
  <div class="bg-glass border-b border-line sticky top-0 z-20 px-4 py-2 backdrop-blur-xl shadow-sm">
    <div class="flex items-center justify-between mb-1">
      <div class="flex items-center shrink-0">
        <!-- 返回按钮 -->
        <button
          @click="$emit('back')"
          class="ui-icon-btn ui-icon-btn-ghost -ml-2"
          aria-label="返回"
          title="返回"
        >
          <i class="fas fa-chevron-left text-lg"></i>
        </button>

        <!-- 会话列表抽屉开关 -->
        <button
          @click="$emit('toggleSidebar')"
          class="ui-icon-btn ui-icon-btn-ghost"
          aria-label="会话列表"
          title="会话列表"
        >
          <i class="far fa-comments text-lg"></i>
        </button>
      </div>

      <div class="flex flex-col items-center min-w-0 flex-1 px-2">
        <div class="font-bold text-base text-fg leading-tight mb-0.5 truncate max-w-full">{{ user?.nickname || '未知用户' }}</div>

        <!-- 状态信息行 -->
        <div class="flex items-center justify-center gap-2 text-[11px] text-fg/60 min-w-0">
          <button
            type="button"
            data-testid="chat-header-status"
            class="flex items-center gap-1.5 shrink-0"
            :class="statusClass.text"
            :disabled="!canReconnect"
            :title="canReconnect ? '点击立即重连' : undefined"
            @click="$emit('reconnect')"
          >
            <span class="relative inline-flex rounded-full h-2 w-2" :class="statusClass.dot"></span>
            <span>{{ resolvedStatusText }}</span>
          </button>

          <!-- 性别年龄 -->
          <div v-if="user?.sex && user.sex !== '未知'" class="flex items-center gap-1 px-1.5 py-0.5 rounded"
             :class="user.sex === '男' ? 'bg-blue-500/10 text-blue-400' : (user.sex === '女' ? 'bg-pink-500/10 text-pink-400' : 'bg-gray-700/50')">
            <i class="opacity-60" :class="user.sex === '男' ? 'fas fa-mars' : (user.sex === '女' ? 'fas fa-venus' : 'fas fa-genderless')"></i>
            <span v-if="user.age && user.age !== '0'">{{ user.age }}</span>
          </div>

          <!-- 地址 -->
          <div v-if="user?.address && user.address !== '未知' && user.address !== '保密'" class="flex items-center gap-1 min-w-0">
             <span class="text-fg/25">|</span>
             <i class="fas fa-map-marker-alt text-[10px] opacity-60"></i>
             <span class="truncate max-w-[120px] sm:max-w-[240px]">{{ user.address }}</span>
          </div>
        </div>
      </div>

      <!-- 右侧按钮组 -->
      <div class="flex items-center gap-1 -mr-2 shrink-0">
        <!-- 收藏按钮 -->
        <button
          @click="$emit('toggleFavorite')"
          class="ui-icon-btn ui-icon-btn-ghost"
          :aria-label="user?.isFavorite ? '取消收藏' : '收藏'"
          :title="user?.isFavorite ? '取消收藏' : '收藏'"
        >
          <i
            class="text-lg transition-transform active:scale-125"
            :class="user?.isFavorite ? 'fas fa-star text-yellow-500' : 'far fa-star text-fg/40'"
          ></i>
        </button>

        <!-- 更多操作：低频且有破坏性的操作收进菜单，避免误触 -->
        <div ref="moreMenuRef" class="relative">
          <button
            data-testid="chat-header-more"
            @click="showMoreMenu = !showMoreMenu"
            class="ui-icon-btn ui-icon-btn-ghost"
            aria-label="更多操作"
            title="更多操作"
          >
            <i class="fas fa-ellipsis-v text-base"></i>
          </button>

          <div
            v-if="showMoreMenu"
            class="absolute right-0 top-11 w-44 ui-card-sm z-50 py-1"
          >
            <button
              data-testid="chat-header-clear-reload"
              @click="handleMenuAction('clearAndReload')"
              class="w-full px-4 py-2.5 text-left text-sm hover:bg-surface-3 text-fg flex items-center gap-3 transition"
            >
              <i class="fas fa-redo-alt text-fg-muted w-4"></i>
              <span>清空并重新加载</span>
            </button>
            <button
              data-testid="chat-header-blacklist"
              @click="handleMenuAction('blacklist')"
              class="w-full px-4 py-2.5 text-left text-sm hover:bg-surface-3 text-rose-500 flex items-center gap-3 border-t border-line transition"
            >
              <i class="fas fa-user-slash w-4"></i>
              <span>拉黑该用户</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { User } from '@/types'

// 连接状态：online 在线 / reconnecting 重连中 / forbidden 暂时禁止 / offline 离线
export type ConnectionState = 'online' | 'reconnecting' | 'forbidden' | 'offline'

interface Props {
  user: User | null
  connected: boolean
  connectionState?: ConnectionState
  statusText?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'back': []
  'toggleFavorite': []
  'blacklist': []
  'clearAndReload': []
  'toggleSidebar': []
  'reconnect': []
}>()

const showMoreMenu = ref(false)
const moreMenuRef = ref<HTMLElement | null>(null)

const resolvedState = computed<ConnectionState>(() => props.connectionState || (props.connected ? 'online' : 'offline'))

const resolvedStatusText = computed(() => {
  if (props.statusText) return props.statusText
  return resolvedState.value === 'online' ? '在线' : '离线'
})

// 只有普通离线/重连中可以手动触发立即重连
const canReconnect = computed(() => resolvedState.value === 'offline' || resolvedState.value === 'reconnecting')

const statusClass = computed(() => {
  switch (resolvedState.value) {
    case 'online':
      return { text: 'text-emerald-500', dot: 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.55)]' }
    case 'reconnecting':
      return { text: 'text-amber-500', dot: 'bg-amber-500 animate-pulse' }
    default:
      return { text: 'text-rose-500', dot: 'bg-rose-500 shadow-[0_0_6px_rgba(244,63,94,0.35)]' }
  }
})

const handleMenuAction = (action: 'clearAndReload' | 'blacklist') => {
  showMoreMenu.value = false
  if (action === 'clearAndReload') emit('clearAndReload')
  else emit('blacklist')
}

// 点击菜单外部时关闭
const handleDocumentClick = (e: MouseEvent) => {
  if (!showMoreMenu.value) return
  if (moreMenuRef.value && !moreMenuRef.value.contains(e.target as Node)) {
    showMoreMenu.value = false
  }
}

onMounted(() => document.addEventListener('click', handleDocumentClick, true))
onUnmounted(() => document.removeEventListener('click', handleDocumentClick, true))
</script>
