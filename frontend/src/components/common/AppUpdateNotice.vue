<template>
  <div
    v-if="updateAvailable"
    role="status"
    class="fixed top-4 left-1/2 -translate-x-1/2 z-[350] flex items-center gap-3 w-max max-w-[calc(100vw_-_2rem)] px-4 py-3 rounded-2xl border border-line bg-surface text-fg text-sm shadow-xl"
  >
    <span>{{ message }}</span>
    <button class="shrink-0 rounded-lg bg-blue-600 px-3 py-2 text-white" @click="refresh">刷新</button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { watchAppUpdates } from '@/utils/appUpdate'

const updateAvailable = ref(false)
const message = ref('页面已更新，刷新即可使用最新版本')
let stop: (() => void) | undefined
onMounted(() => {
  stop = watchAppUpdates((reason) => {
    message.value = reason === 'load-error' ? '页面资源加载失败，请刷新重试' : '页面已更新，刷新即可使用最新版本'
    updateAvailable.value = true
  })
})
onUnmounted(() => { stop?.() })
const refresh = () => { window.location.reload() }
</script>
