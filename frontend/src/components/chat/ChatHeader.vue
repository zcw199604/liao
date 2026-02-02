<template>
  <div class="bg-surface/60 backdrop-blur-md border-b border-line sticky top-0 z-20 px-4 py-2 shadow-sm">
    <div class="flex items-center justify-between mb-1">
      <div class="flex items-center shrink-0">
        <!-- 侧边栏开关 -->
        <button
          @click="$emit('toggleSidebar')"
          class="w-10 h-10 -ml-2 flex items-center justify-center text-fg/70 hover:text-fg hover:bg-surface/60 rounded-full transition mr-1"
          aria-label="显示列表"
        >
          <i class="fas fa-bars text-lg"></i>
        </button>

        <!-- 返回按钮 -->
        <button
          @click="$emit('back')"
          class="w-10 h-10 flex items-center justify-center text-fg/70 hover:text-fg hover:bg-surface/60 rounded-full transition"
          aria-label="返回"
        >
          <i class="fas fa-chevron-left text-lg"></i>
        </button>
      </div>
      
      <div class="flex flex-col items-center min-w-0 flex-1 px-2">
        <div class="font-bold text-base text-fg leading-tight mb-0.5 truncate max-w-full">{{ user?.nickname || '未知用户' }}</div>
        
        <!-- 状态信息行 -->
        <div class="flex items-center justify-center gap-2 text-[11px] text-fg/60 min-w-0">
          <div class="flex items-center gap-1.5" :class="connected ? 'text-emerald-500' : 'text-rose-500'">
            <span
              class="relative inline-flex rounded-full h-2 w-2"
              :class="connected
                ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.55)]'
                : 'bg-rose-500 shadow-[0_0_6px_rgba(244,63,94,0.35)]'"
            ></span>
            <span>{{ connected ? '在线' : '离线' }}</span>
          </div>

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
        <!-- 清空记录按钮 -->
        <button
          @click="$emit('clearAndReload')"
          class="w-10 h-10 flex items-center justify-center text-fg/40 hover:text-fg hover:bg-surface/60 rounded-full transition"
          title="清空并重新加载聊天记录"
        >
          <i class="fas fa-sync-alt text-sm"></i>
        </button>

        <!-- 收藏按钮 -->
        <button
          @click="$emit('toggleFavorite')"
          class="w-10 h-10 flex items-center justify-center hover:bg-surface/60 rounded-full transition"
        >
          <i
            class="text-lg transition-transform active:scale-125"
            :class="user?.isFavorite ? 'fas fa-star text-yellow-500' : 'far fa-star text-fg/40'"
          ></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { User } from '@/types'

interface Props {
  user: User | null
  connected: boolean
}

defineProps<Props>()
defineEmits<{
  'back': []
  'toggleFavorite': []
  'clearAndReload': []
  'toggleSidebar': []
}>()
</script>
