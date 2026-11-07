<template>
  <div v-if="visible" class="bg-surface/60 backdrop-blur-md border-t border-line p-4" @click.stop>
    <div class="text-xs text-fg-subtle mb-3 flex items-center justify-between">
      <span><i class="fas fa-smile mr-1"></i>选择表情{{ isWanfeng ? '（晚风）' : '' }}</span>
      <button @click="$emit('update:visible', false)" class="text-fg/40 hover:text-fg transition-colors">
        <i class="fas fa-times"></i>
      </button>
    </div>

    <!-- 晚风线路：用对方站点同一套 Unicode 表情（点击直接插入文本） -->
    <div v-if="isWanfeng" class="grid grid-cols-8 gap-1 max-h-64 overflow-y-auto p-1">
      <button
        v-for="emoji in wanfengEmojiList"
        :key="emoji"
        type="button"
        data-testid="wanfeng-emoji-item"
        @click="handleSelect(emoji)"
        class="flex items-center justify-center h-10 rounded-xl text-2xl leading-none hover:bg-surface-3 active:scale-95 transition"
      >
        {{ emoji }}
      </button>
    </div>

    <!-- chat2019 线路：保持原有图片表情（emojiMap） -->
    <div v-else class="grid grid-cols-6 gap-2 max-h-64 overflow-y-auto p-1">
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
import { computed } from 'vue'
import { emojiMap, wanfengEmojiList } from '@/constants/emoji'
import { useProviderStore } from '@/stores/provider'

interface Props {
  visible: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'select': [text: string]
}>()

// 表情按线路分开一套：晚风用对方站点的 Unicode 表情，chat2019 保持图片表情
const providerStore = useProviderStore()
const isWanfeng = computed(() => String(providerStore.currentId || '') === 'wanfeng')

const handleSelect = (text: string) => {
  emit('select', text)
  // 不自动关闭面板，让用户可以连续选择
}
</script>
