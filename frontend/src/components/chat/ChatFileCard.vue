<template>
  <!-- 聊天消息中的文件卡片：点击下载 -->
  <div
    class="p-3 rounded-lg flex items-center gap-3 min-w-[200px] max-w-sm cursor-pointer transition border group"
    :class="isSelf ? 'bg-white/10 hover:bg-white/20 border-white/10 text-white' : 'bg-surface-3/70 hover:bg-surface-hover border-line text-fg'"
    @click="downloadFile"
  >
    <div
      class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
      :class="isSelf ? 'bg-gray-800 text-indigo-400' : 'bg-surface text-indigo-600 dark:text-indigo-400 border border-line'"
    >
      <i class="fas fa-file text-2xl"></i>
    </div>
    <div class="flex-1 overflow-hidden min-w-0">
      <div
        class="text-sm truncate font-medium"
        :class="isSelf ? 'text-white/90' : 'text-fg'"
        :title="fileName"
      >
        {{ fileName }}
      </div>
      <div class="text-xs mt-0.5" :class="isSelf ? 'text-white/50' : 'text-fg-muted'">点击下载</div>
    </div>
    <div
      class="w-8 h-8 rounded-full flex items-center justify-center transition"
      :class="isSelf ? 'bg-white/5 text-gray-300 group-hover:bg-white/10 group-hover:text-white' : 'bg-surface/60 text-fg-muted group-hover:bg-surface-hover group-hover:text-fg border border-line'"
    >
      <i class="fas fa-download text-sm"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  url: string
  isSelf: boolean
}

const props = defineProps<Props>()

// 从 URL 中提取文件名
const getDownloadFileName = (url: string): string => {
  const raw = String(url || '')
  if (!raw) return '未知文件'
  try {
    const u = new URL(raw)
    return decodeURIComponent(u.pathname.split('/').pop() || '未知文件')
  } catch {
    return raw.split('/').pop() || '未知文件'
  }
}

const fileName = computed(() => getDownloadFileName(props.url))

const downloadFile = () => {
  const href = String(props.url || '')
  if (!href) return

  const link = document.createElement('a')
  link.href = href
  link.download = fileName.value
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
</script>
