<template>
  <teleport to="body">
    <transition name="fade">
      <div
        v-if="visible"
        class="fixed inset-0 bg-black/95 flex items-center justify-center z-[100] overflow-hidden select-none"
        @click.self="handleClose"
      >
        <!-- 顶部工具栏 -->
        <div class="absolute top-0 left-0 right-0 p-4 flex justify-between items-center z-20 bg-gradient-to-b from-black/50 to-transparent pointer-events-none">
           <!-- 缩放提示/状态/计数 -->
           <div class="flex items-center gap-3 px-2 pointer-events-auto">
              <span v-if="realMediaList.length > 1" class="text-white/90 font-medium text-sm drop-shadow-md">
                {{ currentIndex + 1 }} / {{ realMediaList.length }}
              </span>
              <span v-if="currentMedia.type === 'image'" class="text-white/50 text-xs shadow-black/50 drop-shadow-md">
                {{ scale > 1 ? '拖动查看 · 点击还原' : '点击放大' }}
              </span>
           </div>

           <div class="flex items-center gap-4 pointer-events-auto">
              <!-- 下载按钮 -->
              <a 
                :href="currentMedia.url" 
                download 
                target="_blank"
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
                title="下载"
                @click.stop
              >
                <i class="fas fa-download text-sm"></i>
              </a>

              <!-- 关闭按钮 -->
              <button
                @click="handleClose"
                class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition backdrop-blur-sm"
              >
                <i class="fas fa-times text-lg"></i>
              </button>
           </div>
        </div>

        <!-- 左右切换按钮 -->
        <template v-if="realMediaList.length > 1">
          <button 
            class="absolute left-2 sm:left-6 top-1/2 -translate-y-1/2 w-10 h-10 sm:w-12 sm:h-12 rounded-full bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center backdrop-blur-md transition z-30 focus:outline-none"
            @click.stop="prev"
            title="上一张 (←)"
          >
            <i class="fas fa-chevron-left text-lg sm:text-xl"></i>
          </button>
          
          <button 
            class="absolute right-2 sm:right-6 top-1/2 -translate-y-1/2 w-10 h-10 sm:w-12 sm:h-12 rounded-full bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center backdrop-blur-md transition z-30 focus:outline-none"
            @click.stop="next"
            title="下一张 (→)"
          >
            <i class="fas fa-chevron-right text-lg sm:text-xl"></i>
          </button>
        </template>

        <!-- 图片预览 (支持点击放大和拖动) -->
        <div 
          v-if="currentMedia.type === 'image'" 
          class="relative w-full h-full flex items-center justify-center p-0 transition-opacity duration-200 pb-20"
          @click.self="handleClose"
        >
          <img
            :key="currentMedia.url"
            :src="currentMedia.url"
            class="max-w-full max-h-full object-contain cursor-grab active:cursor-grabbing select-none"
            :class="{ 'transition-transform duration-300 ease-out': !isDragging }"
            :style="imageStyle"
            alt="预览"
            draggable="false"
            @mousedown="startDrag"
            @touchstart="startDrag"
            @click.stop="handleClick"
          />
        </div>

        <!-- 视频预览 -->
        <div v-else-if="currentMedia.type === 'video'" class="relative w-full h-full flex items-center justify-center pb-20">
             <video
              :key="currentMedia.url + '-video'"
              :src="currentMedia.url"
              controls
              autoplay
              class="max-w-[95%] max-h-[95%] shadow-2xl rounded-lg bg-black"
            ></video>
        </div>

        <!-- 底部缩略图栏 -->
        <div 
          v-if="realMediaList.length > 1"
          class="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-black/90 via-black/50 to-transparent flex items-end justify-center z-40 pb-6 pointer-events-auto"
          @click.stop
        >
           <div 
             ref="thumbnailContainer"
             class="flex gap-3 px-4 overflow-x-auto no-scrollbar max-w-full items-center h-16 w-full sm:w-auto sm:max-w-[80vw]"
           >
             <div 
               v-for="(item, idx) in realMediaList" 
               :key="'thumb-' + idx"
               class="relative w-12 h-12 flex-shrink-0 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-200 shadow-lg"
               :class="idx === currentIndex ? 'border-indigo-500 scale-110 opacity-100 ring-2 ring-indigo-500/30' : 'border-transparent opacity-40 hover:opacity-80 hover:scale-105'"
               @click="jumpTo(idx)"
             >
                <img v-if="item.type === 'image'" :src="item.url" class="w-full h-full object-cover" loading="lazy" />
                <video v-else :src="item.url" class="w-full h-full object-cover"></video>
                <!-- Video indicator -->
                <div v-if="item.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/40">
                  <i class="fas fa-play text-[8px] text-white/90"></i>
                </div>
             </div>
           </div>
        </div>

        <!-- 上传按钮（如果允许上传） -->
        <button
          v-if="canUpload"
          @click="$emit('upload')"
          class="absolute bottom-28 left-1/2 transform -translate-x-1/2 px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-full font-medium transition shadow-lg shadow-indigo-600/30 flex items-center gap-2 z-50"
        >
          <i class="fas fa-cloud-upload-alt"></i>
          <span>上传此{{ currentMedia.type === 'image' ? '图片' : '视频' }}</span>
        </button>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import type { UploadedMedia } from '@/types'

interface Props {
  visible: boolean
  url: string
  type: 'image' | 'video'
  canUpload?: boolean
  mediaList?: UploadedMedia[]
}

const props = withDefaults(defineProps<Props>(), {
  canUpload: false,
  mediaList: () => []
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'upload': []
}>()

// 状态管理
const scale = ref(1)
const translateX = ref(0)
const translateY = ref(0)
const isDragging = ref(false)
const currentIndex = ref(0)
const thumbnailContainer = ref<HTMLElement | null>(null)

// 整合后的媒体列表
const realMediaList = computed<UploadedMedia[]>(() => {
  if (props.mediaList && props.mediaList.length > 0) {
    return props.mediaList
  }
  // 兼容旧模式：单张图片构造成列表
  return [{ url: props.url, type: props.type }]
})

const currentMedia = computed<UploadedMedia>(() => {
  if (realMediaList.value.length === 0) {
    return { url: '', type: 'image' }
  }
  const item = realMediaList.value[currentIndex.value]
  if (item) return item
  return realMediaList.value[0] || { url: '', type: 'image' }
})

// 导航逻辑
const next = () => {
  resetZoom()
  if (currentIndex.value < realMediaList.value.length - 1) {
    currentIndex.value++
  } else {
    currentIndex.value = 0 // 循环
  }
}

const prev = () => {
  resetZoom()
  if (currentIndex.value > 0) {
    currentIndex.value--
  } else {
    currentIndex.value = realMediaList.value.length - 1 // 循环
  }
}

const jumpTo = (index: number) => {
  if (index === currentIndex.value) return
  resetZoom()
  currentIndex.value = index
}

const handleKeydown = (e: KeyboardEvent) => {
  if (!props.visible) return
  
  if (e.key === 'ArrowRight') next()
  if (e.key === 'ArrowLeft') prev()
  if (e.key === 'Escape') handleClose()
}

// 自动滚动缩略图
watch(currentIndex, (newIndex) => {
  if (!props.visible) return
  nextTick(() => {
    if (thumbnailContainer.value && realMediaList.value.length > 1) {
      const container = thumbnailContainer.value
      const children = container.children
      if (children[newIndex]) {
        const target = children[newIndex] as HTMLElement
        // Scroll to center
        const containerWidth = container.clientWidth
        const targetLeft = target.offsetLeft
        const targetWidth = target.clientWidth
        
        container.scrollTo({
          left: targetLeft - containerWidth / 2 + targetWidth / 2,
          behavior: 'smooth'
        })
      }
    }
  })
})

// 拖动辅助变量
let startX = 0
let startY = 0
let initialTranslateX = 0
let initialTranslateY = 0
let hasMoved = false

const imageStyle = computed(() => {
  return {
    transform: `translate3d(${translateX.value}px, ${translateY.value}px, 0) scale(${scale.value})`
  }
})

const handleClose = () => {
  resetZoom()
  emit('update:visible', false)
}

const resetZoom = () => {
  scale.value = 1
  translateX.value = 0
  translateY.value = 0
  isDragging.value = false
}

const handleClick = () => {
  if (scale.value === 1) {
    scale.value = 3 // 放大倍数
  } else {
    // 再次点击还原
    resetZoom()
  }
}

const startDrag = (e: MouseEvent | TouchEvent) => {
  // 允许 scale=1 时进行拖动以支持滑动切换
  // if (scale.value <= 1) return 
  
  // 对于触摸事件，不立即阻止默认行为，以便允许点击
  // 但在移动时会阻止默认行为
  
  isDragging.value = true
  hasMoved = false
  
  const clientX = e instanceof MouseEvent ? e.clientX : (e.touches?.[0]?.clientX || 0)
  const clientY = e instanceof MouseEvent ? e.clientY : (e.touches?.[0]?.clientY || 0)
  
  startX = clientX
  startY = clientY
  initialTranslateX = translateX.value
  initialTranslateY = translateY.value
  
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', stopDrag)
  window.addEventListener('touchmove', onDrag, { passive: false })
  window.addEventListener('touchend', stopDrag)
}

const onDrag = (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return
  
  const clientX = e instanceof MouseEvent ? e.clientX : (e.touches?.[0]?.clientX || 0)
  const clientY = e instanceof MouseEvent ? e.clientY : (e.touches?.[0]?.clientY || 0)
  
  const deltaX = clientX - startX
  const deltaY = clientY - startY
  
  // 防抖阈值
  if (Math.abs(deltaX) > 5 || Math.abs(deltaY) > 5) {
      hasMoved = true
      // 移动时阻止默认行为（如滚动）
      if (e.cancelable) e.preventDefault()
  }

  if (scale.value > 1) {
    // 放大模式：自由拖拽
    translateX.value = initialTranslateX + deltaX
    translateY.value = initialTranslateY + deltaY
  } else {
    // 未放大模式：仅水平滑动（Swipe）
    // 增加阻尼感，除以 1.5 还是 1.0 看手感，这里用 1:1 跟随更自然
    translateX.value = deltaX 
    // Y轴保持不动
  }
}

const stopDrag = () => {
  isDragging.value = false
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', stopDrag)
  window.removeEventListener('touchmove', onDrag)
  window.removeEventListener('touchend', stopDrag)
  
  if (!hasMoved) {
      // 如果没有移动，视为点击
      // 如果是放大状态下的点击，应该不需要在这里处理，handleClick 会处理
      // 但如果是在 scale=1 下的点击，handleClick 也会处理
      return
  }
  
  if (scale.value === 1) {
    // 滑动切换判定
    const threshold = 80 // 滑动阈值
    if (translateX.value > threshold) {
      // 向右滑 -> 上一张
      prev()
    } else if (translateX.value < -threshold) {
      // 向左滑 -> 下一张
      next()
    } else {
      // 未达到阈值，回弹
      translateX.value = 0
    }
  } else {
    // 放大状态下的松手，不需要额外逻辑，保持当前位置
    // (后续可以加边缘回弹逻辑，这里暂不处理)
  }
}

// 监听visible变化
watch(() => props.visible, (val) => {
  if (val) {
    resetZoom()
    window.addEventListener('keydown', handleKeydown)
    
    // 初始化 currentIndex
    // 如果有传入 mediaList，尝试找到 url 对应的 index
    if (props.mediaList && props.mediaList.length > 0 && props.url) {
      const idx = props.mediaList.findIndex(m => m.url === props.url)
      currentIndex.value = idx >= 0 ? idx : 0
      
      // Initial scroll to active thumbnail
      nextTick(() => {
        if (thumbnailContainer.value && realMediaList.value.length > 1) {
             const idx = currentIndex.value
             const container = thumbnailContainer.value
             const children = container.children
             if (children[idx]) {
                const target = children[idx] as HTMLElement
                const containerWidth = container.clientWidth
                const targetLeft = target.offsetLeft
                const targetWidth = target.clientWidth
                container.scrollTo({
                    left: targetLeft - containerWidth / 2 + targetWidth / 2,
                    behavior: 'instant' as ScrollBehavior // Instant for initial load
                })
             }
        }
      })
    } else {
      currentIndex.value = 0
    }
  } else {
    window.removeEventListener('keydown', handleKeydown)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
