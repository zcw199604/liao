<template>
  <!-- 输入区域 -->
  <div class="bg-[#18181b] px-4 py-3 pb-safe flex items-end border-t border-gray-800">
    <!-- 左侧工具栏 -->
    <div class="flex items-center gap-3 mr-3 mb-1.5">
      <!-- 随机匹配按钮 (移至左侧以防误触) -->
      <button
        type="button"
        @click="$emit('startMatch')"
        :disabled="!wsConnected"
        class="w-9 h-9 rounded-full bg-[#27272a] hover:bg-[#3f3f46] text-purple-500 hover:text-purple-400 flex items-center justify-center transition active:scale-95 disabled:opacity-30 disabled:cursor-not-allowed"
        title="匹配新用户"
        aria-label="匹配"
      >
        <i class="fas fa-random text-sm"></i>
      </button>

      <div class="w-[1px] h-6 bg-gray-700 mx-1"></div>

      <button
        type="button"
        @click.stop="$emit('showUpload')"
        :disabled="disabled"
        class="text-gray-400 hover:text-white transition disabled:opacity-50 disabled:cursor-not-allowed p-1"
        aria-label="图片"
      >
        <i class="fas fa-plus-circle text-2xl"></i>
      </button>
      <button
        type="button"
        @click.stop="$emit('showEmoji')"
        :disabled="disabled"
        class="text-gray-400 hover:text-yellow-400 transition disabled:opacity-50 disabled:cursor-not-allowed p-1"
        aria-label="表情"
      >
        <i class="fas fa-smile text-2xl"></i>
      </button>
    </div>

    <!-- 输入框容器 -->
    <div 
      class="flex-1 bg-[#27272a] rounded-2xl min-h-[40px] flex items-center px-4 py-2 mr-3 transition-colors border border-transparent focus-within:border-indigo-500/50 focus-within:bg-[#2f2f32]"
    >
      <textarea
        :value="modelValue"
        @input="handleInput"
        @keydown="handleKeydown"
        @focus="handleFocus"
        @blur="handleBlur"
        placeholder="发消息..."
        class="w-full bg-transparent text-white outline-none text-base resize-none overflow-hidden h-6 leading-6 disabled:text-gray-500 disabled:cursor-not-allowed placeholder-gray-500"
        :disabled="disabled"
        rows="1"
        ref="textareaRef"
      ></textarea>
    </div>

    <!-- 发送按钮 -->
    <button
      type="button"
      @click="$emit('send')"
      :disabled="disabled || !modelValue.trim()"
      class="mb-1 w-10 h-10 rounded-full bg-indigo-600 hover:bg-indigo-500 flex items-center justify-center text-white disabled:opacity-50 disabled:bg-gray-700 disabled:cursor-not-allowed transition shrink-0 shadow-lg shadow-indigo-500/20 active:scale-95"
      aria-label="发送"
    >
      <i class="fas fa-paper-plane text-sm translate-x-[-1px] translate-y-[1px]"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

interface Props {
  modelValue: string
  disabled: boolean
  wsConnected?: boolean  // 新增：WebSocket连接状态
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'send': []
  'showUpload': []
  'showEmoji': []
  'typingStart': []
  'typingEnd': []
  'startMatch': []  // 新增：匹配事件
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
let typingTimer: ReturnType<typeof setTimeout> | null = null
let isTypingStatus = false

const handleInput = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)

  // 自动调整高度
  autoResize()

  // 正在输入状态
  if (!isTypingStatus) {
    emit('typingStart')
    isTypingStatus = true
  }

  if (typingTimer) {
    clearTimeout(typingTimer)
  }

  typingTimer = setTimeout(() => {
    emit('typingEnd')
    isTypingStatus = false
  }, 3000)
}

const autoResize = () => {
  nextTick(() => {
    if (textareaRef.value) {
      textareaRef.value.style.height = 'auto'
      textareaRef.value.style.height = textareaRef.value.scrollHeight + 'px'
    }
  })
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter') {
    if (e.ctrlKey || e.metaKey) {
      // Ctrl+Enter or Cmd+Enter to send
      e.preventDefault()
      emit('send')
    } else if (!e.shiftKey) {
       // Regular Enter sends too (current behavior), shift+enter inserts newline (default)
       // If you want ONLY Ctrl+Enter to send, remove this block. 
       // But usually, standard Enter sends, Shift+Enter new line.
       // Let's keep standard Enter behavior but ensure Ctrl+Enter also works explicitly if needed
       // (though standard Enter usually covers it unless we want to change standard behavior)
       
       // Current implementation: Enter sends, Shift+Enter new line
       e.preventDefault()
       emit('send')
    }
  }
}

const handleFocus = () => {
  if (!isTypingStatus) {
    emit('typingStart')
    isTypingStatus = true
  }
}

const handleBlur = () => {
  if (typingTimer) {
    clearTimeout(typingTimer)
  }
  if (isTypingStatus) {
    emit('typingEnd')
    isTypingStatus = false
  }
}

// 监听modelValue变化，重置高度
watch(() => props.modelValue, () => {
  autoResize()
})
</script>
