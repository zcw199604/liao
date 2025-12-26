<template>
  <teleport to="body">
    <transition name="fade">
      <div
        v-if="visible"
        class="fixed inset-0 bg-black/95 flex items-center justify-center z-[100]"
        @click.self="handleClose"
      >
        <!-- 顶部工具栏 -->
        <div class="absolute top-0 left-0 right-0 p-4 flex justify-between items-center z-20 bg-gradient-to-b from-black/50 to-transparent">
           <!-- 缩放提示/状态 -->
           <div v-if="type === 'image'" class="text-white/50 text-xs px-2">
              {{ isZoomed ? '点击缩小' : '点击放大' }}
           </div>
           <div v-else></div>

           <div class="flex items-center gap-4">
              <!-- 下载按钮 -->
              <a 
                :href="url" 
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

        <!-- 图片预览 (支持点击放大) -->
        <div 
          v-if="type === 'image'" 
          class="relative w-full h-full flex items-center justify-center overflow-auto p-4"
          @click.self="handleClose"
        >
          <img
            :src="url"
            class="transition-all duration-300 ease-out cursor-zoom-in shadow-2xl"
            :class="isZoomed ? 'max-w-none scale-150 cursor-zoom-out' : 'max-w-[95%] max-h-[95%] object-contain'"
            alt="预览"
            @click.stop="toggleZoom"
          />
        </div>

        <!-- 视频预览 -->
        <video
          v-else-if="type === 'video'"
          :src="url"
          controls
          autoplay
          class="max-w-[95%] max-h-[95%] shadow-2xl rounded-lg"
        ></video>

        <!-- 上传按钮（如果允许上传） -->
        <button
          v-if="canUpload"
          @click="$emit('upload')"
          class="absolute bottom-8 left-1/2 transform -translate-x-1/2 px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-full font-medium transition shadow-lg shadow-indigo-600/30 flex items-center gap-2"
        >
          <i class="fas fa-cloud-upload-alt"></i>
          <span>上传此{{ type === 'image' ? '图片' : '视频' }}</span>
        </button>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  visible: boolean
  url: string
  type: 'image' | 'video'
  canUpload?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  canUpload: false
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'upload': []
}>()

const isZoomed = ref(false)

const handleClose = () => {
  isZoomed.value = false
  emit('update:visible', false)
}

const toggleZoom = () => {
  isZoomed.value = !isZoomed.value
}

// 每次打开重置缩放状态
watch(() => props.visible, (val) => {
  if (val) isZoomed.value = false
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
