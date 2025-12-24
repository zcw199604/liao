<template>
  <div class="bg-[#18181b]/90 backdrop-blur-sm border-b border-gray-800 sticky top-0 z-20 px-4 py-3">
    <div class="flex items-center justify-between mb-2">
      <button
        @click="$emit('back')"
        class="w-10 h-10 flex items-center justify-start text-gray-300"
      >
        <i class="fas fa-chevron-left text-xl"></i>
      </button>
      <div class="font-bold text-base text-white">{{ user?.nickname || '未知用户' }}</div>

      <!-- 右侧按钮组 -->
      <div class="flex items-center gap-2">
        <!-- 清空记录按钮 -->
        <button
          @click="$emit('clearAndReload')"
          class="w-10 h-10 flex items-center justify-center text-gray-400 hover:text-white transition"
          title="清空并重新加载聊天记录"
        >
          <i class="fas fa-sync-alt"></i>
        </button>

        <!-- 收藏按钮 -->
        <button
          @click="$emit('toggleFavorite')"
          class="w-10 h-10 flex items-center justify-center"
        >
          <i
            :class="user?.isFavorite ? 'fas fa-star text-yellow-500' : 'far fa-star text-gray-400'"
          ></i>
        </button>
      </div>
    </div>
    <div class="flex items-center justify-center gap-3 text-xs text-gray-400">
      <div v-if="user?.sex" class="flex items-center gap-1">
        <i class="fas fa-venus-mars"></i>
        <span>{{ user.sex }}</span>
      </div>
      <div v-if="user?.age && user.age !== '0'" class="flex items-center gap-1">
        <i class="fas fa-birthday-cake"></i>
        <span>{{ user.age }}岁</span>
      </div>
      <div v-if="user?.address || user?.area" class="flex items-center gap-1">
        <i class="fas fa-map-marker-alt"></i>
        <span>{{ user.address || user.area }}</span>
      </div>
      <div class="flex items-center gap-1" :class="connected ? 'text-green-500' : 'text-red-500'">
        <span class="w-1.5 h-1.5 rounded-full" :class="connected ? 'bg-green-500' : 'bg-red-500'"></span>
        <span>{{ connected ? '已连接' : '未连接' }}</span>
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
}>()
</script>
