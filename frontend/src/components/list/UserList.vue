<template>
  <div class="space-y-2 p-4">
    <div
      v-for="user in users"
      :key="user.id"
      @click="$emit('select', user)"
      class="flex items-center gap-3 p-4 bg-[#1a1a1f] hover:bg-[#27272a] rounded-xl cursor-pointer transition-colors border border-white/5"
    >
      <!-- 头像 -->
      <div :class="`w-12 h-12 rounded-full flex items-center justify-center text-white font-bold text-lg ${getColorClass(user.id)}`">
        {{ user.nickname.charAt(0).toUpperCase() }}
      </div>

      <!-- 用户信息 -->
      <div class="flex-1">
        <div class="flex items-center gap-2">
          <span class="text-white font-medium">{{ user.nickname }}</span>
          <i v-if="user.isFavorite" class="fas fa-star text-yellow-400 text-xs"></i>
        </div>
        <div class="text-sm text-gray-400">{{ user.area || '未知地区' }}</div>
      </div>

      <!-- 时间 -->
      <div v-if="user.lastMessageTime" class="text-xs text-gray-500">
        {{ formatTime(user.lastMessageTime) }}
      </div>
    </div>

    <div v-if="users.length === 0" class="text-center py-12 text-gray-500">
      <i class="fas fa-inbox text-4xl mb-3 opacity-50"></i>
      <div>暂无{{ type === 'history' ? '历史' : '收藏' }}用户</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { User } from '@/types'
import { formatTime } from '@/utils/time'
import { getColorClass } from '@/constants/colors'

interface Props {
  users: User[]
  type: 'history' | 'favorite'
}

defineProps<Props>()
defineEmits<{
  'select': [user: User]
}>()
</script>
