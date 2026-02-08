<template>
  <teleport to="body">
    <transition name="slide-up">
      <div v-if="visible" class="fixed inset-0 z-[110] flex items-end justify-center" @click.self="close">
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>

        <div class="relative w-full max-w-lg bg-surface rounded-t-3xl shadow-2xl p-6 max-h-[70vh] overflow-y-auto border-t border-line" @click.stop>
          <button @click="close" class="absolute top-4 right-4 text-fg-muted hover:text-fg transition-colors rounded-lg hover:bg-surface-3">
            <i class="fas fa-times text-lg"></i>
          </button>

          <h3 class="text-xl font-bold text-fg mb-6">文件详细信息</h3>

          <div v-if="douyinWork" class="mb-6">
            <h4 class="text-base font-semibold text-fg mb-3">作品信息</h4>
            <div class="space-y-4">
              <div v-if="douyinWork.detailId" class="detail-item">
                <label>作品ID</label>
                <div class="value font-mono text-xs">{{ douyinWork.detailId }}</div>
              </div>

              <div v-if="douyinWork.desc" class="detail-item">
                <label>文案/标题</label>
                <div class="value text-sm">{{ douyinWork.desc }}</div>
              </div>

              <div v-if="douyinWork.authorName" class="detail-item">
                <label>作者名称</label>
                <div class="value">{{ douyinWork.authorName }}</div>
              </div>

              <div v-if="douyinWork.authorUniqueId" class="detail-item">
                <label>抖音号</label>
                <div class="value font-mono text-xs">{{ douyinWork.authorUniqueId }}</div>
              </div>

              <div v-if="douyinWork.authorSecUserId" class="detail-item">
                <label>作者ID（sec_user_id）</label>
                <div class="value font-mono text-xs">{{ douyinWork.authorSecUserId }}</div>
              </div>

              <div v-if="douyinWork.status" class="detail-item">
                <label>状态</label>
                <div class="value">{{ douyinWork.status }}</div>
              </div>

              <div v-if="douyinWork.publishAt" class="detail-item">
                <label>发布时间</label>
                <div class="value">{{ formatFullTime(douyinWork.publishAt) }}</div>
              </div>

              <div v-if="douyinWork.isPinned !== undefined" class="detail-item">
                <label>是否置顶</label>
                <div class="value">
                  <span :class="douyinWork.isPinned ? 'text-emerald-600 dark:text-emerald-300' : 'text-fg-muted'">
                    {{ douyinWork.isPinned ? '是' : '否' }}
                  </span>
                </div>
              </div>

              <div v-if="douyinWork.pinnedRank !== undefined && douyinWork.pinnedRank !== null" class="detail-item">
                <label>置顶顺序</label>
                <div class="value">{{ douyinWork.pinnedRank }}</div>
              </div>

              <div v-if="douyinWork.pinnedAt" class="detail-item">
                <label>置顶时间</label>
                <div class="value">{{ formatFullTime(douyinWork.pinnedAt) }}</div>
              </div>

              <div v-if="douyinWork.crawledAt" class="detail-item">
                <label>采集时间</label>
                <div class="value">{{ formatFullTime(douyinWork.crawledAt) }}</div>
              </div>

              <div v-if="douyinWork.lastSeenAt" class="detail-item">
                <label>最近可见</label>
                <div class="value">{{ formatFullTime(douyinWork.lastSeenAt) }}</div>
              </div>
            </div>
          </div>

          <h4 class="text-base font-semibold text-fg mb-3">文件信息</h4>
          <div class="space-y-4">
            <div v-if="media.originalFilename" class="detail-item">
              <label>原始文件名</label>
              <div class="value">{{ media.originalFilename || '未知' }}</div>
            </div>

            <div v-if="media.localFilename" class="detail-item">
              <label>本地存储名</label>
              <div class="value text-fg-subtle text-sm">{{ media.localFilename || '未知' }}</div>
            </div>

            <div v-if="media.fileSize !== undefined" class="detail-item">
              <label>文件大小</label>
              <div class="value">{{ formatFileSize(media.fileSize || 0) }}</div>
            </div>

            <div v-if="media.fileExtension || media.fileType" class="detail-item">
              <label>文件格式</label>
              <div class="value">
                <span class="text-blue-400">{{ media.fileExtension?.toUpperCase() || 'N/A' }}</span>
                <span v-if="media.fileType" class="text-fg-subtle text-sm ml-2">({{ media.fileType }})</span>
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
import { computed } from 'vue'
import { formatFileSize } from '@/utils/file'
import { formatFullTime } from '@/utils/time'
import type { UploadedMedia } from '@/types'

interface Props {
  visible: boolean
  media: UploadedMedia
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const close = () => emit('update:visible', false)

const douyinWork = computed(() => {
  const media = props.media
  const ctx = media?.context
  if (!ctx || ctx.provider !== 'douyin') return null
  const w = ctx.work
  if (!w) return null
  // Only show when there is at least one useful field.
  const hasAny =
    !!w.detailId ||
    !!w.desc ||
    !!w.authorName ||
    !!w.authorUniqueId ||
    !!w.authorSecUserId ||
    !!w.publishAt ||
    !!w.pinnedAt ||
    !!w.crawledAt ||
    !!w.lastSeenAt ||
    !!w.status ||
    w.isPinned !== undefined ||
    w.pinnedRank !== undefined
  return hasAny ? w : null
})
</script>

<style scoped>
.detail-item {
  @apply border-b border-line pb-3;
}
.detail-item:last-child {
  @apply border-b-0;
}
.detail-item label {
  @apply text-fg-subtle text-sm mb-1 block font-medium;
}
.detail-item .value {
  @apply text-fg break-all;
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
