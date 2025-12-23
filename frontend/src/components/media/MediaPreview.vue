<template>
  <teleport to="body">
    <transition name="fade">
      <div
        v-if="visible"
        class="fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-50"
        @click.self="$emit('update:visible', false)"
      >
        <button
          @click="$emit('update:visible', false)"
          class="absolute top-4 right-4 w-12 h-12 bg-gray-800 hover:bg-gray-700 rounded-full flex items-center justify-center text-white transition"
        >
          <i class="fas fa-times text-xl"></i>
        </button>

        <!-- 图片预览 -->
        <img
          v-if="type === 'image'"
          :src="url"
          class="max-w-[90%] max-h-[90%] object-contain"
          alt="预览"
        />

        <!-- 视频预览 -->
        <video
          v-else-if="type === 'video'"
          :src="url"
          controls
          class="max-w-[90%] max-h-[90%]"
        ></video>

        <!-- 上传按钮（如果允许上传） -->
        <button
          v-if="canUpload"
          @click="$emit('upload')"
          class="absolute bottom-8 left-1/2 transform -translate-x-1/2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition"
        >
          <i class="fas fa-upload mr-2"></i>
          上传此{{ type === 'image' ? '图片' : '视频' }}
        </button>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  url: string
  type: 'image' | 'video'
  canUpload?: boolean
}

withDefaults(defineProps<Props>(), {
  canUpload: false
})

defineEmits<{
  'update:visible': [value: boolean]
  'upload': []
}>()
</script>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
