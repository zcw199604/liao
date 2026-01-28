<template>
  <div class="absolute bottom-6 left-0 right-0 flex justify-center z-20 pointer-events-none">
    <div class="pointer-events-auto">
    <!-- 匹配按钮 -->
    <button
      v-if="!chatStore.isMatching"
      @mousedown="handleMouseDown"
      @mouseup="handleMouseUp"
      @mouseleave="handleMouseLeave"
      @touchstart.prevent="handleTouchStart"
      @touchend.prevent="handleTouchEnd"
      @touchcancel="handleTouchCancel"
      class="flex items-center px-8 py-4 bg-blue-600 hover:bg-blue-500 rounded-full text-white font-bold text-lg shadow-xl transition active:scale-95"
    >
      <i class="fas fa-random mr-2"></i> 匹配新用户
    </button>

    <!-- 取消按钮（带进度显示） -->
    <button
      v-else
      @click="handleCancelMatch"
      class="flex flex-col items-center px-8 py-4 bg-red-600 rounded-full text-white font-bold text-lg shadow-2xl animate-pulse"
    >
      <div class="flex items-center">
        <i class="fas fa-stop mr-2"></i>
        <span v-if="!chatStore.continuousMatchConfig.enabled || chatStore.continuousMatchConfig.total === 1">取消匹配</span>
        <span v-else>取消连续匹配</span>
      </div>
      <span v-if="chatStore.continuousMatchConfig.enabled && chatStore.continuousMatchConfig.total > 1" class="text-xs mt-1 opacity-90">
        第 {{ chatStore.continuousMatchConfig.current }}/{{ chatStore.continuousMatchConfig.total }} 次
      </span>
    </button>

    <!-- 长按菜单 -->
    <div
      v-if="showMenu"
      class="absolute bottom-full left-1/2 -translate-x-1/2 mb-3 bg-[#18181b] rounded-xl shadow-2xl border border-white/10 overflow-hidden"
    >
      <button
        v-for="option in menuOptions"
        :key="option.count"
        @click="handleSelectCount(option.count)"
        class="w-full px-6 py-3 text-left hover:bg-[#27272a] text-white flex items-center gap-3 transition"
      >
        <i :class="option.icon" class="text-blue-400"></i>
        <span>连续匹配 {{ option.count }} 次</span>
      </button>
    </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useChat } from '@/composables/useChat'
import { useToast } from '@/composables/useToast'

const chatStore = useChatStore()
const { startContinuousMatch, cancelMatch, handleAutoMatch } = useChat()
const { show } = useToast()

const showMenu = ref(false)
const longPressTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const isLongPress = ref(false)

const menuOptions = [
  { count: 3, icon: 'fas fa-dice-three' },
  { count: 5, icon: 'fas fa-hand-spock' },
  { count: 10, icon: 'fas fa-fire' }
]

// 长按检测 - 鼠标事件
const handleMouseDown = () => {
  isLongPress.value = false
  longPressTimer.value = setTimeout(() => {
    isLongPress.value = true
    showMenu.value = true
  }, 300)
}

const handleMouseUp = () => {
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
    longPressTimer.value = null
  }

  if (!isLongPress.value) {
    // 短按 - 单次匹配
    handleStartMatch()
  }
}

const handleMouseLeave = () => {
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
    longPressTimer.value = null
  }
}

// 长按检测 - 触摸事件
const handleTouchStart = (e: TouchEvent) => {
  e.preventDefault()
  handleMouseDown()
}

const handleTouchEnd = (e: TouchEvent) => {
  e.preventDefault()
  handleMouseUp()
}

const handleTouchCancel = () => {
  handleMouseLeave()
}

// 单次匹配
const handleStartMatch = () => {
  const ok = startContinuousMatch(1)
  if (ok) {
    show('正在匹配...')
  }
}

// 选择连续匹配次数
const handleSelectCount = (count: number) => {
  showMenu.value = false
  const ok = startContinuousMatch(count)
  if (ok) {
    show(`开始连续匹配 ${count} 次...`)
  }
}

// 取消匹配
const handleCancelMatch = () => {
  cancelMatch()
  show('已取消匹配')
}

// 监听匹配成功事件，触发自动匹配
const handleMatchAutoCheck = () => {
  handleAutoMatch()
}

// 点击外部关闭菜单
const handleClickOutside = (e: MouseEvent) => {
  if (showMenu.value) {
    showMenu.value = false
  }
}

onMounted(() => {
  window.addEventListener('match-auto-check', handleMatchAutoCheck)
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  window.removeEventListener('match-auto-check', handleMatchAutoCheck)
  document.removeEventListener('click', handleClickOutside)
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
  }
})
</script>
