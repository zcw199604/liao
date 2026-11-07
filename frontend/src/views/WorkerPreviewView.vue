<template>
  <main class="page-container bg-canvas text-fg">
    <header class="flex shrink-0 items-center gap-3 border-b border-line bg-surface px-4 py-3">
      <button class="ui-icon-btn ui-icon-btn-ghost" aria-label="返回" @click="goBack">
        <i class="fas fa-arrow-left"></i>
      </button>
      <h1 class="min-w-0 flex-1 truncate text-base font-bold">X 媒体预览</h1>
      <button
        class="ui-icon-btn ui-icon-btn-ghost disabled:opacity-50"
        aria-label="刷新预览"
        title="刷新预览"
        data-testid="worker-preview-refresh"
        :disabled="connecting"
        @click="openPreview"
      >
        <i class="fas fa-sync-alt" :class="{ 'animate-spin': connecting }"></i>
      </button>
    </header>

    <div class="relative min-h-0 flex-1">
      <iframe
        v-if="frameUrl"
        :key="frameKey"
        :src="frameUrl"
        title="X 媒体预览"
        class="h-full w-full border-0 bg-black"
        allow="fullscreen; autoplay; picture-in-picture"
        allowfullscreen
        referrerpolicy="same-origin"
        @load="frameLoading = false"
        @error="onFrameError"
      ></iframe>
      <div v-if="connecting || frameLoading" class="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-canvas text-fg-muted" role="status">
        <span class="h-7 w-7 animate-spin rounded-full border-2 border-line border-t-blue-500"></span>
        <span class="text-sm">正在打开媒体预览…</span>
      </div>
      <div v-else-if="error" class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center" role="alert">
        <i class="fas fa-exclamation-circle text-2xl text-amber-400"></i>
        <p class="max-w-md text-sm text-fg-muted">{{ error }}</p>
        <button
          class="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-blue-500"
          data-testid="worker-preview-retry"
          @click="openPreview"
        >重试</button>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createWorkerPreviewSession } from '@/api/workerPreview'

const router = useRouter()
const connecting = ref(false)
const frameLoading = ref(false)
const frameUrl = ref('')
const frameKey = ref(0)
const error = ref('')
let active = true

const openPreview = async () => {
  if (connecting.value) return
  connecting.value = true
  frameLoading.value = false
  frameUrl.value = ''
  error.value = ''
  try {
    const response = await createWorkerPreviewSession()
    if (!active) return
    if (response.code !== 0) throw new Error(response.msg || '暂时无法打开媒体预览')
    if (response.data?.url !== '/worker-preview/preview') throw new Error('预览地址无效，请联系管理员')
    frameKey.value += 1
    frameLoading.value = true
    frameUrl.value = response.data.url
  } catch (cause: unknown) {
    if (!active) return
    const failure = cause as { response?: { data?: { msg?: string } }; message?: string }
    error.value = failure?.response?.data?.msg || failure?.message || '暂时无法打开媒体预览，请稍后重试'
  } finally {
    if (active) connecting.value = false
  }
}

const onFrameError = () => {
  frameUrl.value = ''
  frameLoading.value = false
  error.value = '页面加载失败，请检查网络后重试'
}

const goBack = () => {
  // 返回应用页面，避免 iframe 内部的浏览记录影响外层返回行为。
  const previous = window.history.state?.back
  if (typeof previous === 'string' && previous.startsWith('/') && !previous.startsWith('//') && !previous.startsWith('/worker-preview')) {
    void router.replace(previous)
  } else {
    void router.replace('/identity')
  }
}

onMounted(openPreview)
onBeforeUnmount(() => {
  active = false
  frameUrl.value = ''
})
</script>
