<template>
  <!-- 输入区域（对齐旧版布局：左侧图片/表情图标 + 输入框 + 圆形发送按钮） -->
  <div class="bg-[#18181b] px-4 py-3 pb-safe flex items-end border-t border-gray-800">
    <div class="flex items-center gap-2 mr-3 mb-1">
      <button
        type="button"
        @click.stop="$emit('showUpload')"
        :disabled="disabled"
        class="text-gray-400 active:text-white transition disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="图片"
      >
        <i class="fas fa-plus-circle text-2xl"></i>
      </button>
      <button
        type="button"
        @click.stop="$emit('showEmoji')"
        :disabled="disabled"
        class="text-gray-400 active:text-yellow-400 transition disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="表情"
      >
        <i class="fas fa-smile text-2xl"></i>
      </button>
    </div>

    <div class="flex-1 bg-[#27272a] rounded-2xl min-h-[40px] flex items-center px-4 py-2 mr-3">
      <textarea
        :value="modelValue"
        @input="handleInput"
        @keydown="handleKeydown"
        @focus="handleFocus"
        @blur="handleBlur"
        placeholder="发消息..."
        class="w-full bg-transparent text-white outline-none text-base resize-none overflow-hidden h-6 leading-6 disabled:text-gray-500 disabled:cursor-not-allowed"
        :disabled="disabled"
        rows="1"
        ref="textareaRef"
      ></textarea>
    </div>

    <!-- 右侧按钮组 -->
    <div class="flex items-center gap-2">
      <!-- 发送按钮 -->
      <button
        type="button"
        @click="$emit('send')"
        :disabled="disabled || !modelValue.trim()"
        class="mb-1 w-9 h-9 rounded-full bg-indigo-600 flex items-center justify-center text-white disabled:opacity-50 disabled:bg-gray-700 disabled:cursor-not-allowed transition shrink-0"
        aria-label="发送"
      >
        <i class="fas fa-paper-plane text-xs"></i>
      </button>

      <!-- 随机匹配按钮 -->
      <button
        type="button"
        @click="$emit('startMatch')"
        :disabled="!wsConnected"
        class="mb-1 w-9 h-9 rounded-full bg-purple-600 flex items-center justify-center text-white transition hover:bg-purple-700 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shrink-0"
        title="匹配新用户"
        aria-label="匹配"
      >
        <i class="fas fa-random text-sm"></i>
      </button>
    </div>
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
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    emit('send')
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
