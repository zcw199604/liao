<template>
  <div v-if="visible" class="bg-surface/60 backdrop-blur-md border-t border-line p-4 max-h-96 overflow-y-auto" @click.stop>
    <!-- 区域1：已上传的文件（点击发送） -->
    <div v-if="uploadedMedia && uploadedMedia.length > 0" class="mb-4">
      <div class="text-xs text-fg-subtle mb-2">
        <i class="fas fa-check-circle text-indigo-500 mr-1"></i>已上传的文件（点击发送）
      </div>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="(media, idx) in uploadedMedia"
          :key="idx"
          @click="$emit('send', media)"
          class="w-16 h-16 rounded-lg overflow-hidden cursor-pointer border border-line-strong hover:border-indigo-500 transition-colors relative"
        >
          <MediaTile
            :src="media.type === 'video' && media.posterUrl ? media.posterUrl : media.url"
            :type="media.type === 'video' && media.posterUrl ? 'image' : media.type"
            :poster="media.type === 'video' ? media.posterUrl : undefined"
            class="w-full h-full"
            :show-skeleton="false"
            :indicator-size="'sm'"
            :muted="true"
            :fit="media.type === 'video' ? 'contain' : 'cover'"
          >
            <template #center>
              <div
                v-if="media.type === 'video'"
                class="rounded-full bg-black/40 backdrop-blur-sm flex items-center justify-center border border-white/15 shadow-lg w-6 h-6"
              >
                <i class="fas fa-play text-white text-[10px] ml-0.5"></i>
              </div>
            </template>
            <template #file>
              <i class="fas fa-file text-2xl"></i>
            </template>
          </MediaTile>
        </div>
      </div>
    </div>

    <!-- 空状态提示 -->
    <div v-else class="text-xs text-fg-subtle mb-3">
      暂无已上传的文件
    </div>

    <!-- 区域2：操作按钮 -->
    <div class="space-y-2">
      <button
        @click="$emit('uploadFile')"
        class="w-full py-3 bg-surface/70 hover:bg-surface/90 text-fg rounded-xl transition-colors flex items-center justify-center gap-2 border border-line"
      >
        <i class="fas fa-folder-open"></i>
        <span>选择文件上传</span>
      </button>

      <button
        v-if="canOpenChatHistory"
        @click="$emit('openChatHistory')"
        class="w-full py-3 bg-surface/70 hover:bg-surface/90 text-fg rounded-xl border border-line flex items-center justify-center gap-2 transition-colors"
      >
        <i class="fas fa-history"></i>
        <span>历史聊天图片</span>
      </button>

      <button
        @click="$emit('openAllUploads')"
        class="w-full py-3 bg-surface/70 hover:bg-surface/90 text-fg rounded-xl border border-line flex items-center justify-center gap-2 transition-colors"
      >
        <i class="fas fa-images"></i>
        <span>所有上传图片</span>
      </button>

      <button
        @click="$emit('openMtPhoto')"
        class="w-full py-3 bg-surface/70 hover:bg-surface/90 text-fg rounded-xl border border-line flex items-center justify-center gap-2 transition-colors"
      >
        <i class="fas fa-photo-video"></i>
        <span>mtPhoto 相册</span>
      </button>

      <button
        @click="$emit('openDouyinFavoriteAuthors')"
        class="w-full py-3 bg-surface/70 hover:bg-surface/90 text-fg rounded-xl border border-line flex items-center justify-center gap-2 transition-colors"
      >
        <i class="fas fa-star"></i>
        <span>抖音收藏作者</span>
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
  'openDouyinFavoriteAuthors': []
}>()
</script>
