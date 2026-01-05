<template>
  <div 
    class="relative h-full flex flex-col overflow-hidden select-none"
    @touchstart="handleTouchStart"
    @touchmove="handleTouchMove"
    @touchend="handleTouchEnd"
    @touchcancel="handleTouchEnd"
  >
    <!-- 下拉指示器 (绝对定位在顶部上方) -->
    <div 
      class="absolute top-0 left-0 w-full flex justify-center items-end pointer-events-none z-10"
      :style="{ 
        height: threshold + 'px', 
        transform: `translateY(${currentPull - threshold}px)`,
        opacity: Math.min(currentPull / threshold, 1)
      }"
    >
      <div class="pb-3 flex items-center gap-2 text-gray-400 text-sm font-medium">
        <div v-if="status === 'refreshing'" class="flex items-center gap-2">
          <svg class="animate-spin h-4 w-4 text-indigo-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span>正在刷新...</span>
        </div>
        <div v-else-if="status === 'reaching'" class="flex items-center gap-2 text-indigo-400">
          <i class="fas fa-arrow-up animate-bounce"></i>
          <span>释放立即刷新</span>
        </div>
        <div v-else class="flex items-center gap-2">
          <i class="fas fa-arrow-down" :style="{ transform: `rotate(${currentPull / threshold * 180}deg)` }"></i>
          <span>下拉刷新</span>
        </div>
      </div>
    </div>

    <!-- 内容容器 (应用位移) -->
    <div 
      ref="contentRef"
      class="flex-1 h-full relative"
      :style="{ 
        transform: currentPull > 0 ? `translateY(${currentPull}px)` : 'none',
        transition: isTouching ? 'none' : 'transform 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94)'
      }"
    >
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = defineProps({
  onRefresh: {
    type: Function,
    required: true
  },
  threshold: {
    type: Number,
    default: 60 // 触发刷新的阈值 px
  },
  maxPull: {
    type: Number,
    default: 120 // 最大下拉距离 px
  }
})

const contentRef = ref<HTMLElement | null>(null)
const scrollContainer = ref<HTMLElement | null>(null)

// 状态
const status = ref<'idle' | 'pulling' | 'reaching' | 'refreshing'>('idle')
const currentPull = ref(0)
const isTouching = ref(false)
let startY: number | null = null
let startScrollTop = 0

// 尝试查找内部的滚动容器
const findScrollContainer = () => {
  if (!contentRef.value) return null
  // 1. 检查 slot 根元素是否可滚动
  const child = contentRef.value.firstElementChild as HTMLElement
  if (child && (getComputedStyle(child).overflowY === 'auto' || getComputedStyle(child).overflowY === 'scroll')) {
    return child
  }
  // 2. 查找内部第一个可滚动元素
  return contentRef.value.querySelector('.overflow-y-auto') as HTMLElement
}

onMounted(() => {
  scrollContainer.value = findScrollContainer()
})

const handleTouchStart = (e: TouchEvent) => {
  if (status.value === 'refreshing') return
  
  // 重新获取滚动容器（以防动态渲染）
  if (!scrollContainer.value) {
    scrollContainer.value = findScrollContainer()
  }

  const container = scrollContainer.value
  if (!container) return

  // 只有当滚动条在顶部时才允许下拉
  if (container.scrollTop <= 0 && e.touches.length > 0) {
    startY = e.touches[0]!.clientY
    startScrollTop = container.scrollTop
    // 不立即设置 isTouching，等到 move 确定是下拉意图
  }
}

const handleTouchMove = (e: TouchEvent) => {
  if (status.value === 'refreshing' || startY === null || e.touches.length === 0) return
  
  const container = scrollContainer.value
  if (!container) return
  
  // 如果过程中滚动条不再是0，说明用户在往回滚或者列表自己动了，取消下拉逻辑
  if (container.scrollTop > 0) {
    startY = null // 重置，防止后续误判
    currentPull.value = 0
    status.value = 'idle'
    return
  }

  const currentY = e.touches[0]!.clientY
  const deltaY = currentY - startY

  // 只有向下滑动才处理
  if (deltaY > 0) {
    // 阻止默认滚动行为（关键！防止页面整体橡皮筋效果）
    if (e.cancelable) {
       e.preventDefault() 
    }
    
    isTouching.value = true
    
    // 增加阻尼感：拉得越长，越难拉
    // 简单的非线性公式
    const dampening = 0.5
    const move = Math.min(deltaY * dampening, props.maxPull)
    
    currentPull.value = move
    
    if (move >= props.threshold) {
      status.value = 'reaching'
    } else {
      status.value = 'pulling'
    }
  }
}

const handleTouchEnd = async () => {
  isTouching.value = false
  startY = null
  
  if (status.value === 'reaching') {
    status.value = 'refreshing'
    currentPull.value = props.threshold // 停留在加载位置
    
    // 震动反馈
    if (navigator.vibrate) navigator.vibrate(20)

    try {
      await props.onRefresh()
    } finally {
      // 刷新结束，回弹
      setTimeout(() => {
        status.value = 'idle'
        currentPull.value = 0
      }, 300) // 延迟一点让用户看到完成（可选）
    }
  } else {
    // 未达到阈值或取消
    status.value = 'idle'
    currentPull.value = 0
  }
}
</script>
