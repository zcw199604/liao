<template>
  <div
    class="flex mb-3"
    :class="message.isSelf ? 'justify-end' : 'justify-start'"
  >
    <div
      class="msg-bubble shadow-sm"
      :class="message.isSelf ? 'msg-right' : 'msg-left'"
    >
      <template v-if="message.segments && message.segments.length">
        <div class="flex flex-col gap-2">
          <template v-for="(seg, idx) in message.segments" :key="idx">
            <div v-if="seg.kind === 'text'" v-html="parseEmoji(seg.text, emojiMap)"></div>

            <div
              v-else-if="seg.kind === 'image'"
              class="cursor-pointer"
            >
              <MediaTile
                :src="getMediaUrl(seg.url)"
                type="image"
                :fill="false"
                class="inline-block rounded-lg bg-surface-3/50"
                :media-class="[
                  'max-w-full sm:max-w-sm',
                  'max-h-[40vh] min-h-[100px] min-w-[100px]'
                ].join(' ')"
                :show-skeleton="false"
                @click="previewMedia(getMediaUrl(seg.url), 'image')"
                @error="handleImageError"
              >
                <template #center>
                  <div class="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition flex items-center justify-center opacity-0 group-hover:opacity-100 rounded-lg">
                    <i class="fas fa-search-plus text-white/80 drop-shadow-md"></i>
                  </div>
                </template>
              </MediaTile>
            </div>

            <div
              v-else-if="seg.kind === 'video'"
              class="cursor-pointer"
            >
              <MediaTile
                :src="getMediaUrl(seg.url)"
                type="video"
                :fill="false"
                class="inline-block rounded-lg bg-black"
                media-class="max-w-full sm:max-w-sm max-h-[40vh]"
                :show-skeleton="false"
                :muted="true"
                :show-video-indicator="false"
                @click="previewMedia(getMediaUrl(seg.url), 'video')"
              >
                <template #center>
                  <div class="absolute inset-0 flex items-center justify-center bg-black/20 group-hover:bg-black/30 transition rounded-lg">
                    <div class="w-10 h-10 rounded-full bg-black/50 backdrop-blur-sm flex items-center justify-center text-white border border-white/20 shadow-lg group-hover:scale-110 transition">
                      <i class="fas fa-play text-xs ml-0.5"></i>
                    </div>
                  </div>
                </template>
              </MediaTile>
            </div>

            <div
              v-else-if="seg.kind === 'file'"
              class="p-3 rounded-lg flex items-center gap-3 min-w-[200px] max-w-sm cursor-pointer transition border group"
              :class="message.isSelf ? 'bg-white/10 hover:bg-white/20 border-white/10 text-white' : 'bg-surface-3/70 hover:bg-surface-hover border-line text-fg'"
              @click="downloadUrl(getMediaUrl(seg.url))"
            >
              <div
                class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
                :class="message.isSelf ? 'bg-gray-800 text-indigo-400' : 'bg-surface text-indigo-600 dark:text-indigo-400 border border-line'"
              >
                <i class="fas fa-file text-2xl"></i>
              </div>
              <div class="flex-1 overflow-hidden min-w-0">
                <div
                  class="text-sm truncate font-medium"
                  :class="message.isSelf ? 'text-white/90' : 'text-fg'"
                  :title="getFileNameFromUrl(getMediaUrl(seg.url))"
                >
                  {{ getFileNameFromUrl(getMediaUrl(seg.url)) }}
                </div>
                <div class="text-xs mt-0.5" :class="message.isSelf ? 'text-white/50' : 'text-fg-muted'">点击下载</div>
              </div>
              <div
                class="w-8 h-8 rounded-full flex items-center justify-center transition"
                :class="message.isSelf ? 'bg-white/5 text-gray-300 group-hover:bg-white/10 group-hover:text-white' : 'bg-surface/60 text-fg-muted group-hover:bg-surface-hover group-hover:text-fg border border-line'"
              >
                <i class="fas fa-download text-sm"></i>
              </div>
            </div>
          </template>
        </div>
      </template>

      <template v-else>
        <!-- 文本消息 -->
        <div v-if="!message.isImage && !message.isVideo && !message.isFile" v-html="parsedContent"></div>

        <!-- 图片消息 -->
        <div v-else-if="message.isImage" class="cursor-pointer" @click="previewImage">
          <MediaTile
            :src="imageUrl"
            type="image"
            :fill="false"
            class="inline-block rounded-lg bg-surface-3/50"
            :media-class="[
               'max-w-full sm:max-w-sm',
               'max-h-[40vh] min-h-[100px] min-w-[100px]'
            ].join(' ')"
            :show-skeleton="false"
            @error="handleImageError"
          >
            <template #center>
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition flex items-center justify-center opacity-0 group-hover:opacity-100 rounded-lg">
                <i class="fas fa-search-plus text-white/80 drop-shadow-md"></i>
              </div>
            </template>
          </MediaTile>
        </div>

        <!-- 视频消息 -->
        <div v-else-if="message.isVideo" class="cursor-pointer" @click="previewVideo">
          <MediaTile
            :src="videoUrl"
            type="video"
            :fill="false"
            class="inline-block rounded-lg bg-black"
            media-class="max-w-full sm:max-w-sm max-h-[40vh]"
            :show-skeleton="false"
            :muted="true"
            :show-video-indicator="false"
          >
            <template #center>
              <div class="absolute inset-0 flex items-center justify-center bg-black/20 group-hover:bg-black/30 transition rounded-lg">
                <div class="w-10 h-10 rounded-full bg-black/50 backdrop-blur-sm flex items-center justify-center text-white border border-white/20 shadow-lg group-hover:scale-110 transition">
                  <i class="fas fa-play text-xs ml-0.5"></i>
                </div>
              </div>
            </template>
          </MediaTile>
        </div>

        <!-- 文件消息 -->
        <div
          v-else-if="message.isFile"
          class="p-3 rounded-lg flex items-center gap-3 min-w-[200px] max-w-sm cursor-pointer transition border group"
          :class="message.isSelf ? 'bg-white/10 hover:bg-white/20 border-white/10 text-white' : 'bg-surface-3/70 hover:bg-surface-hover border-line text-fg'"
          @click="downloadFile"
        >
          <div
            class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
            :class="message.isSelf ? 'bg-gray-800 text-indigo-400' : 'bg-surface text-indigo-600 dark:text-indigo-400 border border-line'"
          >
             <i class="fas fa-file text-2xl"></i>
          </div>
          <div class="flex-1 overflow-hidden min-w-0">
             <div class="text-sm truncate font-medium" :class="message.isSelf ? 'text-white/90' : 'text-fg'" :title="fileName">{{ fileName }}</div>
             <div class="text-xs mt-0.5" :class="message.isSelf ? 'text-white/50' : 'text-fg-muted'">点击下载</div>
          </div>
          <div
            class="w-8 h-8 rounded-full flex items-center justify-center transition"
            :class="message.isSelf ? 'bg-white/5 text-gray-300 group-hover:bg-white/10 group-hover:text-white' : 'bg-surface/60 text-fg-muted group-hover:bg-surface-hover group-hover:text-fg border border-line'"
          >
            <i class="fas fa-download text-sm"></i>
          </div>
        </div>
      </template>

      <!-- 时间戳 -->
      <div
        class="text-[10px] mt-1.5 opacity-50 select-none"
        :class="message.isSelf ? 'text-right' : 'text-left'"
      >
        {{ formatTime(message.time) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ChatMessage } from '@/types'
import { formatTime } from '@/utils/time'
import { parseEmoji } from '@/utils/string'
import { emojiMap } from '@/constants/emoji'
import { useUpload } from '@/composables/useUpload'
import MediaTile from '@/components/common/MediaTile.vue'

interface Props {
  message: ChatMessage
}

const props = defineProps<Props>()
const { getMediaUrl } = useUpload()

const parsedContent = computed(() => {
  if (!props.message.content) return ''
  return parseEmoji(props.message.content, emojiMap)
})

const imageUrl = computed(() => {
  return getMediaUrl(props.message.imageUrl || props.message.content || '')
})

const videoUrl = computed(() => {
  return getMediaUrl(props.message.videoUrl || props.message.content || '')
})

const fileUrl = computed(() => {
  return getMediaUrl(props.message.fileUrl || props.message.content || '')
})

const getFileNameFromUrl = (url: string): string => {
  const raw = String(url || '')
  if (!raw) return '未知文件'
  try {
    const u = new URL(raw)
    return decodeURIComponent(u.pathname.split('/').pop() || '未知文件')
  } catch {
    return raw.split('/').pop() || '未知文件'
  }
}

const fileName = computed(() => {
  if (props.message.isFile) {
     return getFileNameFromUrl(fileUrl.value)
  }
  return ''
})

const downloadUrl = (url: string) => {
  const href = String(url || '')
  if (!href) return

  const link = document.createElement('a')
  link.href = href
  link.download = getFileNameFromUrl(href)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const downloadFile = () => {
  downloadUrl(fileUrl.value)
}

const previewMedia = (url: string, type: 'image' | 'video') => {
  window.dispatchEvent(new CustomEvent('preview-media', {
    detail: { url, type }
  }))
}

const previewImage = () => {
  previewMedia(imageUrl.value, 'image')
}

const previewVideo = () => {
  previewMedia(videoUrl.value, 'video')
}

const handleImageError = (e: Event) => {
  const target = e.target as HTMLImageElement | null
  console.error('图片加载失败:', target?.src || imageUrl.value)
}
</script>
