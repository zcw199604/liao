<template>
  <div>
    <div
      v-for="identity in identities"
      :key="identity.id"
      @click="$emit('select', identity)"
      class="identity-card flex items-center p-4 mb-3 bg-[#18181b] rounded-2xl border border-transparent cursor-pointer"
    >
      <!-- 头像 -->
      <div
        :class="getColorClass(identity.id)"
        class="w-14 h-14 rounded-xl flex items-center justify-center text-white font-bold text-xl shadow-lg shrink-0"
      >
        {{ (identity.name || '?').charAt(0).toUpperCase() }}
      </div>

      <!-- 信息 -->
      <div class="ml-4 flex-1 min-w-0">
        <div class="flex items-center gap-2 mb-1">
          <span class="font-bold text-lg text-white">{{ identity.name }}</span>
          <span
            class="px-2 py-0.5 text-xs rounded-full"
            :class="identity.sex === '男' ? 'bg-blue-500/20 text-blue-400' : 'bg-pink-500/20 text-pink-400'"
          >
            {{ identity.sex }}
          </span>
        </div>
        <p class="text-xs text-gray-500">ID: {{ identity.id.substring(0, 8) }}...</p>
        <p v-if="identity.created_at || identity.createdAt" class="text-xs text-gray-600 mt-1">
          最后使用: {{ identity.created_at || identity.createdAt }}
        </p>
      </div>

      <!-- 操作按钮 -->
      <div class="flex items-center gap-2">
        <button
          @click.stop="$emit('delete', identity)"
          class="w-8 h-8 flex items-center justify-center text-red-500 hover:bg-red-500/10 rounded-lg transition"
        >
          <i class="fas fa-trash-alt text-sm"></i>
        </button>
        <i class="fas fa-chevron-right text-gray-600"></i>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="identities.length === 0" class="flex flex-col items-center justify-center mt-20 text-gray-600">
      <i class="fas fa-user-plus text-5xl mb-4 opacity-50"></i>
      <p class="text-sm">还没有身份，点击上方按钮创建</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Identity } from '@/types'
import { getColorClass } from '@/constants/colors'

interface Props {
  identities: Identity[]
}

defineProps<Props>()
defineEmits<{
  'select': [identity: Identity]
  'delete': [identity: Identity]
}>()
</script>
