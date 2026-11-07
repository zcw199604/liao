<template>
  <!-- 输入区域 -->
  <div class="bg-glass backdrop-blur-xl px-4 py-3 pb-safe flex items-end border-t border-line transition-[padding] duration-200">
    <!-- 左侧工具栏 -->
    <div class="flex items-center gap-2 mr-3 mb-1.5">
      <button
        type="button"
        @click.stop="$emit('showUpload')"
        class="ui-icon-btn ui-icon-btn-ghost"
        aria-label="图片"
      >
        <i class="fas fa-plus-circle text-2xl scale-90"></i>
      </button>
      <button
        type="button"
        @click.stop="$emit('showEmoji')"
        class="ui-icon-btn ui-icon-btn-ghost hover:text-yellow-500"
        aria-label="表情"
      >
        <i class="fas fa-smile text-2xl scale-90"></i>
      </button>
      <!-- 掷骰子：晚风线路才有（对方站点在输入栏有同样按钮，发一条「🎲 掷出了 N」） -->
      <button
        v-if="showDice"
        type="button"
        data-testid="dice-btn"
        :disabled="diceCooldown > 0"
        @click.stop="rollDice"
        class="ui-icon-btn ui-icon-btn-ghost hover:text-sky-400 relative"
        :title="diceCooldown > 0 ? `${diceCooldown} 秒后可再掷` : '掷骰子'"
        aria-label="掷骰子"
      >
        <span v-if="diceCooldown > 0" class="text-xs font-semibold tabular-nums">{{ diceCooldown }}</span>
        <i v-else class="fas fa-dice text-2xl scale-90"></i>
      </button>
    </div>

    <!-- 输入框容器 -->
    <div 
      class="ui-input-shell mr-3"
    >
      <textarea
        :value="modelValue"
        @input="handleInput"
        @keydown="handleKeydown"
        @focus="handleFocus"
        @blur="handleBlur"
        @paste="handlePaste"
        placeholder="发消息..."
        class="w-full bg-transparent text-fg outline-none text-base resize-none overflow-hidden h-6 leading-6 placeholder-fg-subtle"
        rows="1"
        ref="textareaRef"
      ></textarea>
    </div>

    <!-- 随机匹配按钮 -->
    <button
      type="button"
      @click="$emit('startMatch')"
      :disabled="!wsConnected"
      class="ui-icon-btn mb-1 mr-2 shrink-0 disabled:opacity-30"
      title="匹配新用户"
      aria-label="匹配"
    >
      <i class="fas fa-random text-sm"></i>
    </button>

    <!-- 发送按钮 -->
    <button
      type="button"
      @click="$emit('send')"
      :disabled="disabled || !modelValue.trim()"
      class="ui-fab-primary mb-1 shrink-0"
      aria-label="发送"
    >
      <i class="fas fa-paper-plane text-sm translate-x-[-1px] translate-y-[1px]"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'

interface Props {
  modelValue: string
  disabled: boolean
  wsConnected?: boolean  // 新增：WebSocket连接状态
  showDice?: boolean     // 是否显示「掷骰子」按钮（晚风线路才有该功能）
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
  'pasteFiles': [files: File[]]  // 粘贴图片/文件
  'dice': [value: number]  // 掷骰子（参数是 1~100 的点数，由上层拼成文本发出）
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
let typingTimer: ReturnType<typeof setTimeout> | null = null
let isTypingStatus = false

// 掷骰子：与对方站点一致——随机 1~100 点，冷却 5 秒（对方是 5000ms）
const DICE_COOLDOWN_SECONDS = 5
const diceCooldown = ref(0)
let diceTimer: ReturnType<typeof setInterval> | null = null

const rollDice = () => {
  if (props.disabled || diceCooldown.value > 0) return
  const value = Math.floor(Math.random() * 100) + 1
  emit('dice', value)

  diceCooldown.value = DICE_COOLDOWN_SECONDS
  if (diceTimer) clearInterval(diceTimer)
  diceTimer = setInterval(() => {
    diceCooldown.value -= 1
    if (diceCooldown.value <= 0) {
      diceCooldown.value = 0
      if (diceTimer) {
        clearInterval(diceTimer)
        diceTimer = null
      }
    }
  }, 1000)
}

onBeforeUnmount(() => {
  if (diceTimer) clearInterval(diceTimer)
  if (typingTimer) clearTimeout(typingTimer)
})

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

// Enter 发送，Shift+Enter 换行；输入法组字过程中的 Enter 不触发发送
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key !== 'Enter' || e.shiftKey || e.isComposing) return
  e.preventDefault()
  if (!props.disabled) {
    emit('send')
  }
}

// 粘贴板中带文件（截图/复制的图片）时直接交给上层上传，纯文本粘贴保持默认行为
const handlePaste = (e: ClipboardEvent) => {
  const files = Array.from(e.clipboardData?.files || [])
  if (files.length === 0) return
  e.preventDefault()
  emit('pasteFiles', files)
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
