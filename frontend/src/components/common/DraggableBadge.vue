<template>
  <div
    class="select-none touch-none cursor-move relative z-20 w-fit -m-2 p-2"
    :style="{ ...wrapperStyle, touchAction: 'none' }"
    @mousedown.stop="startDrag"
    @touchstart.stop.prevent="startDrag"
    @click.stop
  >
    <div
      class="px-2 py-0.5 bg-red-500 text-white text-xs rounded-full min-w-[20px] text-center font-medium shadow-sm transition-colors duration-200 flex items-center justify-center"
      :class="{ 'bg-red-600 opacity-60 scale-90': isThresholdReached }"
    >
      {{ count }}
    </div>
    
    <!-- 拖拽时的原点提示 -->
    <div 
        v-if="isDragging" 
        class="absolute top-1/2 left-1/2 w-1.5 h-1.5 bg-red-500/30 rounded-full -translate-x-1/2 -translate-y-1/2 pointer-events-none"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'

const props = defineProps<{
  count: number
}>()

const emit = defineEmits<{
  (e: 'clear'): void
}>()

const isDragging = ref(false)
const position = ref({ x: 0, y: 0 })
const startPos = ref({ x: 0, y: 0 })
const isAnimating = ref(false)

const THRESHOLD = 60

const distance = computed(() => {
  return Math.sqrt(position.value.x * position.value.x + position.value.y * position.value.y)
})

const isThresholdReached = computed(() => distance.value > THRESHOLD)

const wrapperStyle = computed(() => {
  const style: Record<string, string> = {
    transform: `translate(${position.value.x}px, ${position.value.y}px)`
  }
  
  if (isAnimating.value) {
    style.transition = 'transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275)'
  } else if (!isDragging.value) {
    style.transition = 'transform 0.3s'
  } else {
    style.transition = 'none'
  }
  
  return style
})

const startDrag = (event: MouseEvent | TouchEvent) => {
  if (isAnimating.value) return
  
  isDragging.value = true
  let clientX: number, clientY: number
  
  if (window.MouseEvent && event instanceof MouseEvent) {
    clientX = event.clientX
    clientY = event.clientY
  } else {
    const touch = (event as TouchEvent).touches[0]
    if (!touch) return
    clientX = touch.clientX
    clientY = touch.clientY
  }
  
  startPos.value = { x: clientX, y: clientY }
  
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('touchmove', onDrag, { passive: false })
  window.addEventListener('mouseup', endDrag)
  window.addEventListener('touchend', endDrag)
}

const onDrag = (event: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return
  
  // 阻止页面滚动
  if (event.cancelable) {
     event.preventDefault();
  }
  
  let clientX: number, clientY: number
  
  if (window.MouseEvent && event instanceof MouseEvent) {
    clientX = event.clientX
    clientY = event.clientY
  } else {
    const touch = (event as TouchEvent).touches[0]
    if (!touch) return
    clientX = touch.clientX
    clientY = touch.clientY
  }
  
  position.value = {
    x: clientX - startPos.value.x,
    y: clientY - startPos.value.y
  }
}

const endDrag = () => {
  if (!isDragging.value) return
  isDragging.value = false
  
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('touchmove', onDrag)
  window.removeEventListener('mouseup', endDrag)
  window.removeEventListener('touchend', endDrag)
  
  if (isThresholdReached.value) {
    emit('clear')
  } else {
    // 回弹
    isAnimating.value = true
    position.value = { x: 0, y: 0 }
    setTimeout(() => {
      isAnimating.value = false
    }, 300)
  }
}

onUnmounted(() => {
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('touchmove', onDrag)
  window.removeEventListener('mouseup', endDrag)
  window.removeEventListener('touchend', endDrag)
})
</script>
