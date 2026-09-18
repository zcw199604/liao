<template>
  <div class="fixed inset-0 z-[85] bg-black/70 flex items-center justify-center p-3" @click.stop>
    <section role="dialog" aria-modal="true" aria-label="移动文件夹内全部文件" class="bg-surface rounded-xl border border-line w-full max-w-xl max-h-[90dvh] flex flex-col p-4 gap-3 text-fg">
      <h3 class="font-semibold">移动文件夹内全部文件</h3>
      <p class="text-xs break-all">源目录：{{ sourcePath }}</p>
      <label class="flex gap-2 items-center text-sm">
        <input v-model="includeSubfolders" type="checkbox" :disabled="saving" class="accent-pink-500" />包含子文件夹中的文件
      </label>
      <p class="text-xs text-fg-subtle">{{ includeSubfolders ? '子目录中的文件也会集中移入目标目录，保留原文件夹结构（可能变为空目录）。' : '只移动当前目录直属文件，子文件夹保持原样。' }}重名自动重命名，不覆盖目标文件。</p>
      <div class="flex flex-wrap gap-2 text-xs">
        <button v-for="(item, index) in history" :key="item.id" class="border border-line rounded px-2 py-1" :disabled="loading || saving" @click="navigate(item.id, index)">{{ item.name }}</button>
      </div>
      <p class="text-xs break-all">目标目录：{{ targetId ? targetPath : '请选择下方文件夹' }}</p>
      <div class="overflow-y-auto min-h-20 max-h-[35vh] border border-line rounded p-2 space-y-1">
        <p v-if="loading" class="text-sm text-fg-subtle">加载中...</p>
        <button v-for="folder in folders" v-else :key="folder.id" :data-testid="`target-${folder.id}`"
          class="block w-full text-left rounded p-2 bg-surface-3 hover:bg-surface-2 text-sm disabled:opacity-50" :disabled="saving" @click="navigate(folder.id)">
          {{ folder.name }} →
        </button>
        <p v-if="!loading && !folders.length" class="text-xs text-fg-subtle">无子文件夹，可将当前目录作为目标</p>
      </div>
      <p v-if="invalidTarget" class="text-xs text-amber-500">目标不能是源目录；包含子目录时，目标也不能位于源目录内。</p>
      <p v-if="error" role="alert" class="text-xs text-red-400 break-all">{{ error }}</p>
      <div class="flex justify-end gap-2">
        <button class="px-3 py-2 rounded border border-line text-sm disabled:opacity-50" :disabled="saving" @click="$emit('close')">取消</button>
        <button data-testid="confirm-move" class="px-3 py-2 rounded bg-pink-500 text-white text-sm disabled:opacity-50" :disabled="!targetId || loading || saving || invalidTarget" @click="move">
          {{ saving ? '移动中，请稍候...' : '确认移动全部文件到此目录' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import * as api from '@/api/mtphoto'
const props = defineProps<{ sourceId: number; sourcePath: string }>()
const emit = defineEmits<{ close: []; moved: [count: number]; refresh: [] }>()
const includeSubfolders = ref(false)
const targetId = ref(0)
const targetPath = ref('')
const folders = ref<api.MtPhotoFolderNode[]>([])
const history = ref([{ id: 0, name: '根目录' }])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const invalidTarget = computed(() => {
  const source = props.sourcePath.replace(/\/$/, '')
  const target = targetPath.value.replace(/\/$/, '')
  return targetId.value === props.sourceId || (!!targetId.value && (target === source || (includeSubfolders.value && target.startsWith(source + '/'))))
})
const navigate = async (id: number, historyIndex?: number) => {
  if (loading.value || saving.value) return
  loading.value = true
  error.value = ''
  const name = folders.value.find(folder => folder.id === id)?.name || `目录 ${id}`
  try {
    const result = id ? await api.getMtPhotoFolderContent(id, 1, 1, false) : await api.getMtPhotoFolderRoot()
    targetId.value = id
    targetPath.value = result.path || ''
    folders.value = result.folderList || []
    if (historyIndex !== undefined) history.value = history.value.slice(0, historyIndex + 1)
    else if (id) history.value.push({ id, name })
  } catch (e: any) {
    error.value = e?.response?.data?.error || e?.message || '读取目录失败'
  } finally { loading.value = false }
}
const move = async () => {
  if (!targetId.value || invalidTarget.value || saving.value || loading.value) return
  saving.value = true
  error.value = ''
  try {
    const result = await api.moveMtPhotoFolderFiles({ sourceId: props.sourceId, targetId: targetId.value, includeSubfolders: includeSubfolders.value })
    if (!result.success) throw new Error('移动结果未确认，请刷新目录检查后再操作')
    emit('moved', result.count)
  } catch (e: any) {
    error.value = e?.response?.data?.error || '移动结果未确认，请刷新源目录和目标目录检查后再操作'
    emit('refresh')
  } finally { saving.value = false }
}
onMounted(() => navigate(0))
</script>
