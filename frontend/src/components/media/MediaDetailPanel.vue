<template>
  <teleport to="body">
    <transition name="slide-up">
      <div v-if="visible" class="fixed inset-0 z-[110] flex items-end justify-center" @click.self="close">
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>

        <div class="relative w-full max-w-lg bg-[#18181b] rounded-t-3xl shadow-2xl p-6 max-h-[70vh] overflow-y-auto" @click.stop>
          <button @click="close" class="absolute top-4 right-4 text-gray-400 hover:text-white transition-colors">
            <i class="fas fa-times text-lg"></i>
          </button>

          <h3 class="text-xl font-bold text-white mb-6">文件详细信息</h3>

          <div class="space-y-4">
            <div v-if="media.originalFilename" class="detail-item">
              <label>原始文件名</label>
              <div class="value">{{ media.originalFilename || '未知' }}</div>
            </div>

            <div v-if="media.localFilename" class="detail-item">
              <label>本地存储名</label>
              <div class="value text-gray-500 text-sm">{{ media.localFilename || '未知' }}</div>
            </div>

            <div v-if="media.fileSize !== undefined" class="detail-item">
              <label>文件大小</label>
              <div class="value">{{ formatFileSize(media.fileSize || 0) }}</div>
            </div>

            <div v-if="media.fileExtension || media.fileType" class="detail-item">
              <label>文件格式</label>
              <div class="value">
                <span class="text-blue-400">{{ media.fileExtension?.toUpperCase() || 'N/A' }}</span>
                <span v-if="media.fileType" class="text-gray-500 text-sm ml-2">({{ media.fileType }})</span>
              </div>
            </div>

            <div v-if="media.width !== undefined && media.height !== undefined" class="detail-item">
              <label>分辨率</label>
              <div class="value">{{ media.width }} × {{ media.height }}</div>
            </div>

            <div v-if="media.duration !== undefined" class="detail-item">
              <label>时长</label>
              <div class="value">{{ media.duration }}s</div>
            </div>

            <div v-if="media.day" class="detail-item">
              <label>日期</label>
              <div class="value">{{ media.day }}</div>
            </div>

            <div v-if="media.uploadTime" class="detail-item">
              <label>首次上传</label>
              <div class="value">{{ formatFullTime(media.uploadTime || '') }}</div>
            </div>

            <div v-if="media.updateTime" class="detail-item">
              <label>最后更新</label>
              <div class="value">{{ formatFullTime(media.updateTime || '') }}</div>
            </div>

            <div v-if="media.md5" class="detail-item">
              <label>MD5</label>
              <div class="value font-mono text-xs">{{ media.md5 }}</div>
            </div>

            <div v-if="media.pHash" class="detail-item">
              <label>pHash</label>
              <div class="value font-mono text-xs">{{ media.pHash }}</div>
            </div>

            <div v-if="media.similarity !== undefined" class="detail-item">
              <label>相似度</label>
              <div class="value font-bold text-blue-400">{{ (media.similarity * 100).toFixed(2) }}%</div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { formatFileSize } from '@/utils/file'
import { formatFullTime } from '@/utils/time'
import type { UploadedMedia } from '@/types'

interface Props {
  visible: boolean
  media: UploadedMedia
}

defineProps<Props>()
const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const close = () => emit('update:visible', false)
</script>

<style scoped>
.detail-item {
  @apply border-b border-white/5 pb-3;
}
.detail-item:last-child {
  @apply border-b-0;
}
.detail-item label {
  @apply text-gray-500 text-sm mb-1 block font-medium;
}
.detail-item .value {
  @apply text-white break-all;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
