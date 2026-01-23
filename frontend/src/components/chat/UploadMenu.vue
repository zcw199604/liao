<template>
  <div v-if="visible" class="bg-[#1a1a1d] border-t border-gray-800 p-4 max-h-96 overflow-y-auto" @click.stop>
    <!-- 区域1：已上传的文件（点击发送） -->
    <div v-if="uploadedMedia && uploadedMedia.length > 0" class="mb-4">
      <div class="text-xs text-gray-500 mb-2">
        <i class="fas fa-check-circle text-indigo-500 mr-1"></i>已上传的文件（点击发送）
      </div>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="(media, idx) in uploadedMedia"
          :key="idx"
          @click="$emit('send', media)"
          class="w-16 h-16 rounded-lg overflow-hidden cursor-pointer border border-gray-700 hover:border-indigo-500 transition relative"
        >
          <MediaTile
            :src="media.url"
            :type="media.type"
            class="w-full h-full"
            :show-skeleton="false"
            :indicator-size="'sm'"
            :muted="true"
          >
            <template #file>
              <i class="fas fa-file text-2xl"></i>
            </template>
          </MediaTile>
        </div>
      </div>
    </div>

    <!-- 空状态提示 -->
    <div v-else class="text-xs text-gray-500 mb-3">
      暂无已上传的文件
    </div>

    <!-- 区域2：操作按钮 -->
    <div class="space-y-2">
      <button
        @click="$emit('uploadFile')"
        class="w-full py-3 bg-[#27272a] text-white rounded-xl hover:bg-[#333] transition flex items-center justify-center gap-2"
      >
        <i class="fas fa-folder-open"></i>
        <span>选择文件上传</span>
      </button>

      <button
        v-if="canOpenChatHistory"
        @click="$emit('openChatHistory')"
        class="w-full py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 flex items-center justify-center gap-2 transition"
      >
        <i class="fas fa-history"></i>
        <span>历史聊天图片</span>
      </button>

      <button
        @click="$emit('openAllUploads')"
        class="w-full py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 flex items-center justify-center gap-2 transition"
      >
        <i class="fas fa-images"></i>
        <span>所有上传图片</span>
      </button>

      <button
        @click="$emit('openMtPhoto')"
        class="w-full py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 flex items-center justify-center gap-2 transition"
      >
        <i class="fas fa-photo-video"></i>
        <span>mtPhoto 相册</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { UploadedMedia } from '@/types'
import MediaTile from '@/components/common/MediaTile.vue'

interface Props {
  visible: boolean
  uploadedMedia: UploadedMedia[]
  canOpenChatHistory?: boolean
}

withDefaults(defineProps<Props>(), {
  canOpenChatHistory: false
})
defineEmits<{
  'update:visible': [value: boolean]
  'send': [media: UploadedMedia]
  'uploadFile': []
  'openChatHistory': []
  'openAllUploads': []
  'openMtPhoto': []
}>()
</script>
