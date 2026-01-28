<template>
  <div v-if="visible" class="bg-surface/60 backdrop-blur-md border-t border-line p-4" @click.stop>
    <div class="text-xs text-fg-subtle mb-3 flex items-center justify-between">
      <span><i class="fas fa-smile mr-1"></i>选择表情</span>
      <button @click="$emit('update:visible', false)" class="text-fg/40 hover:text-fg transition-colors">
        <i class="fas fa-times"></i>
      </button>
    </div>

    <!-- 表情网格 -->
    <div class="grid grid-cols-6 gap-2 max-h-64 overflow-y-auto p-1">
      <div
        v-for="(emojiUrl, text) in emojiMap"
        :key="text"
        @click="handleSelect(text)"
        class="flex flex-col items-center gap-1 p-3 hover:bg-surface-3 rounded-xl cursor-pointer transition active:scale-95"
      >
        <img :src="emojiUrl" :alt="text" class="w-8 h-8" />
        <span class="text-[10px] text-fg-subtle text-center leading-tight">{{ text }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { emojiMap } from '@/constants/emoji'

interface Props {
  visible: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'select': [text: string]
}>()

const handleSelect = (text: string) => {
  emit('select', text)
  // 不自动关闭面板，让用户可以连续选择
}
</script>
