<template>
  <div class="bg-[#18181b]/90 backdrop-blur-md border-b border-gray-800 sticky top-0 z-20 px-4 py-2 shadow-sm">
    <div class="flex items-center justify-between mb-1">
      <div class="flex items-center">
        <!-- 侧边栏开关 -->
        <button
          @click="$emit('toggleSidebar')"
          class="w-10 h-10 -ml-2 flex items-center justify-center text-gray-300 hover:text-white hover:bg-white/5 rounded-full transition mr-1"
          aria-label="显示列表"
        >
          <i class="fas fa-bars text-lg"></i>
        </button>

        <!-- 返回按钮 -->
        <button
          @click="$emit('back')"
          class="w-10 h-10 flex items-center justify-center text-gray-300 hover:text-white hover:bg-white/5 rounded-full transition"
          aria-label="返回"
        >
          <i class="fas fa-chevron-left text-lg"></i>
        </button>
      </div>
      
      <div class="flex flex-col items-center">
        <div class="font-bold text-base text-white leading-tight mb-0.5">{{ user?.nickname || '未知用户' }}</div>
        
        <!-- 状态信息行 -->
        <div class="flex items-center justify-center gap-2 text-[11px] text-gray-400">
          <div class="flex items-center gap-1.5" :class="connected ? 'text-emerald-500' : 'text-rose-500'">
            <span class="relative flex h-2 w-2">
              <span v-if="connected" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2" :class="connected ? 'bg-emerald-500' : 'bg-rose-500'"></span>
            </span>
            <span>{{ connected ? '已连接' : '未连接' }}</span>
          </div>

          <span v-if="user?.sex || (user?.age && user.age !== '0')" class="text-gray-600">|</span>

          <div v-if="user?.sex" class="flex items-center">
            <i class="fas fa-venus-mars mr-1 opacity-70"></i>{{ user.sex }}
          </div>
          <div v-if="user?.age && user.age !== '0'" class="flex items-center">
            {{ user.age }}岁
          </div>
        </div>
      </div>

      <!-- 右侧按钮组 -->
      <div class="flex items-center gap-1 -mr-2">
        <!-- 清空记录按钮 -->
        <button
          @click="$emit('clearAndReload')"
          class="w-10 h-10 flex items-center justify-center text-gray-400 hover:text-white hover:bg-white/5 rounded-full transition"
          title="清空并重新加载聊天记录"
        >
          <i class="fas fa-sync-alt text-sm"></i>
        </button>

        <!-- 收藏按钮 -->
        <button
          @click="$emit('toggleFavorite')"
          class="w-10 h-10 flex items-center justify-center hover:bg-white/5 rounded-full transition"
        >
          <i
            class="text-lg transition-transform active:scale-125"
            :class="user?.isFavorite ? 'fas fa-star text-yellow-500' : 'far fa-star text-gray-400'"
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
