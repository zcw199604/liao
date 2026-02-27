<template>
  <teleport to="body">
    <transition name="slide-up">
      <div v-if="visible" class="fixed inset-0 z-[111] flex items-end justify-center" @click.self="close">
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>

        <div class="relative w-full max-w-2xl bg-surface rounded-t-3xl shadow-2xl p-6 max-h-[78vh] overflow-y-auto border-t border-line" @click.stop>
          <button @click="close" class="absolute top-4 right-4 text-fg-muted hover:text-fg transition-colors rounded-lg hover:bg-surface-3">
            <i class="fas fa-times text-lg"></i>
          </button>

          <h3 class="text-xl font-bold text-fg mb-2">mtPhoto 相同图片</h3>
          <p v-if="md5" class="text-xs text-fg-subtle font-mono break-all mb-4">MD5: {{ md5 }}</p>

          <div v-if="loading" class="py-10 text-center text-fg-subtle text-sm">
            <i class="fas fa-spinner fa-spin mr-2"></i>
            正在查询相同图片...
          </div>

          <div v-else-if="error" class="py-10 text-center text-red-400 text-sm">
            <p class="mb-3">{{ error }}</p>
            <button class="px-3 py-1.5 rounded-lg bg-surface-3 hover:bg-surface-hover text-fg text-xs" @click="emit('retry')">重试</button>
          </div>

          <div v-else-if="groupedItems.length === 0" class="py-10 text-center text-fg-subtle text-sm">
            未找到 mtPhoto 相同图片
          </div>

          <div v-else class="space-y-5">
            <section v-for="dayGroup in groupedItems" :key="dayGroup.dayKey" class="space-y-3">
              <div class="text-sm font-semibold text-fg">{{ dayGroup.dayLabel }}</div>

              <div class="space-y-3">
                <article
                  v-for="folderGroup in dayGroup.folders"
                  :key="folderGroup.key"
                  class="rounded-xl border border-line bg-surface-2 px-3 py-3"
                >
                  <div class="flex items-center justify-between gap-3">
                    <div class="min-w-0">
                      <p class="text-sm text-fg truncate">{{ folderGroup.name }}</p>
                      <p class="text-xs text-fg-subtle truncate">{{ folderGroup.path }}</p>
                    </div>
                  </div>

                  <div class="mt-2 space-y-2">
                    <div
                      v-for="item in folderGroup.items"
                      :key="itemKey(item)"
                      class="flex items-start justify-between gap-3 rounded-lg border border-line/70 px-3 py-2"
                    >
                      <div class="min-w-0">
                        <p class="text-sm text-fg truncate">{{ item.fileName || fallbackFileName(item.filePath) }}</p>
                        <p class="text-xs text-fg-subtle">{{ formatTime(item.tokenAt, item.day) }}</p>
                      </div>
                      <button
                        class="flex-shrink-0 px-2.5 py-1.5 rounded-md text-xs transition"
                        :class="
                          item.canOpenFolder
                            ? 'bg-emerald-500/15 text-emerald-300 hover:bg-emerald-500/25'
                            : 'bg-surface-3 text-fg-subtle cursor-not-allowed'
                        "
                        :disabled="!item.canOpenFolder"
                        @click="emit('open-folder', item)"
                      >
                        打开目录
                      </button>
                    </div>
                  </div>
                </article>
              </div>
            </section>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatFullTime } from '@/utils/time'
import type { MtPhotoSameMediaItem } from '@/api/mtphoto'

interface Props {
  visible: boolean
  loading?: boolean
  error?: string
  md5?: string
  items?: MtPhotoSameMediaItem[]
}

type FolderGroup = {
  key: string
  name: string
  path: string
  items: MtPhotoSameMediaItem[]
}

type DayGroup = {
  dayKey: string
  dayLabel: string
  folders: FolderGroup[]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  error: '',
  md5: '',
  items: () => []
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'open-folder': [item: MtPhotoSameMediaItem]
  retry: []
}>()

const close = () => emit('update:visible', false)

const fallbackFileName = (pathValue?: string) => {
  const value = String(pathValue || '').trim()
  if (!value) return '未知文件'
  const normalized = value.replace(/\\/g, '/')
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || '未知文件'
}

const normalizeDay = (item: MtPhotoSameMediaItem) => {
  const dayValue = String(item.day || '').trim()
  if (dayValue) return dayValue
  const tokenAt = String(item.tokenAt || '').trim()
  if (tokenAt.length >= 10) return tokenAt.slice(0, 10)
  return '未知日期'
}

const itemSortTs = (item: MtPhotoSameMediaItem) => {
  const tokenAt = String(item.tokenAt || '').trim()
  if (!tokenAt) return 0
  const ts = Date.parse(tokenAt)
  return Number.isFinite(ts) ? ts : 0
}

const itemFolderPath = (item: MtPhotoSameMediaItem) => {
  const folderPath = String(item.folderPath || '').trim()
  if (folderPath) return folderPath
  const directory = String(item.directory || '').trim()
  return directory || '未知目录'
}

const itemFolderName = (item: MtPhotoSameMediaItem) => {
  const folderName = String(item.folderName || '').trim()
  if (folderName) return folderName
  const pathValue = itemFolderPath(item)
  const normalized = pathValue.replace(/\\/g, '/')
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || pathValue
}

const itemKey = (item: MtPhotoSameMediaItem) => {
  return [item.id, item.filePath, item.tokenAt, item.folderId].join('|')
}

const formatTime = (tokenAt?: string, day?: string) => {
  const full = String(tokenAt || '').trim()
  if (full) return formatFullTime(full)
  const short = String(day || '').trim()
  return short || '未知时间'
}

const groupedItems = computed<DayGroup[]>(() => {
  const source = Array.isArray(props.items) ? [...props.items] : []
  source.sort((a, b) => {
    const ta = itemSortTs(a)
    const tb = itemSortTs(b)
    if (ta !== tb) return tb - ta
    return itemFolderPath(a).localeCompare(itemFolderPath(b), 'zh-CN')
  })

  const byDay = new Map<string, MtPhotoSameMediaItem[]>()
  for (const item of source) {
    const dayKey = normalizeDay(item)
    if (!byDay.has(dayKey)) byDay.set(dayKey, [])
    byDay.get(dayKey)!.push(item)
  }

  const dayGroups: DayGroup[] = Array.from(byDay.entries()).map(([dayKey, items]) => {
    const byFolder = new Map<string, MtPhotoSameMediaItem[]>()
    for (const item of items) {
      const folderPath = itemFolderPath(item)
      if (!byFolder.has(folderPath)) byFolder.set(folderPath, [])
      byFolder.get(folderPath)!.push(item)
    }

    const folders: FolderGroup[] = Array.from(byFolder.entries())
      .map(([folderPath, folderItems]) => {
        const firstItem = folderItems[0]
        return {
          key: folderPath,
          name: firstItem ? itemFolderName(firstItem) : folderPath,
          path: folderPath,
          items: [...folderItems].sort((a, b) => itemSortTs(b) - itemSortTs(a))
        }
      })
      .sort((a, b) => a.path.localeCompare(b.path, 'zh-CN'))

    return {
      dayKey,
      dayLabel: dayKey,
      folders
    }
  })

  dayGroups.sort((a, b) => {
    if (a.dayKey === '未知日期') return 1
    if (b.dayKey === '未知日期') return -1
    return b.dayKey.localeCompare(a.dayKey, 'zh-CN')
  })
  return dayGroups
})
</script>

<style scoped>
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
