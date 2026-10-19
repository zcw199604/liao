<template>
  <router-view v-slot="{ Component }">
    <Transition name="fade" mode="out-in">
      <component :is="Component" />
    </Transition>
  </router-view>
  <!-- 全局弹窗按需加载：首次打开时才下载并挂载，之后保持挂载以保留内部状态 -->
  <template v-for="modal in modals" :key="modal.key">
    <component :is="modalComponents[modal.key]" v-if="modalMounted[modal.key] && modalComponents[modal.key]" />
  </template>
  <!-- 弹窗代码尚未下载完成时的加载占位，点击遮罩可取消打开 -->
  <div
    v-if="modalLoading"
    class="fixed inset-0 z-[95] bg-black/50 flex items-center justify-center"
    data-testid="modal-loading"
    @click="cancelPendingModals"
  >
    <div class="flex items-center gap-3 px-5 py-3 rounded-2xl bg-surface border border-line text-fg text-sm shadow-lg">
      <span class="w-4 h-4 border-2 border-purple-500 border-t-transparent rounded-full animate-spin"></span>
      <span>加载中…</span>
    </div>
  </div>
</template>

<script setup lang="ts">
// 根组件：所有路由页面的容器
import { computed, markRaw, reactive, shallowReactive, watch, type Component } from 'vue'
import { useRoute } from 'vue-router'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useDouyinStore } from '@/stores/douyin'
import { useVideoExtractStore } from '@/stores/videoExtract'
import { useToast } from '@/composables/useToast'

const mediaStore = useMediaStore()
const mtPhotoStore = useMtPhotoStore()
const douyinStore = useDouyinStore()
const videoExtractStore = useVideoExtractStore()
const { show: showToast } = useToast()

type ModalKey = 'allUpload' | 'mtPhoto' | 'douyin' | 'videoExtractCreate' | 'videoExtractTask'

interface ModalConfig {
  key: ModalKey
  load: () => Promise<{ default: Component }>
  isOpen: () => boolean
  close: () => void
}

// 全局弹窗配置：代码加载方式、打开状态与关闭方式
const modals: ModalConfig[] = [
  {
    key: 'allUpload',
    load: () => import('@/components/media/AllUploadImageModal.vue'),
    isOpen: () => mediaStore.showAllUploadImageModal,
    close: () => { mediaStore.showAllUploadImageModal = false }
  },
  {
    key: 'mtPhoto',
    load: () => import('@/components/media/MtPhotoAlbumModal.vue'),
    isOpen: () => mtPhotoStore.showModal,
    close: () => { mtPhotoStore.showModal = false }
  },
  {
    key: 'douyin',
    load: () => import('@/components/media/DouyinDownloadModal.vue'),
    isOpen: () => douyinStore.showModal,
    close: () => { douyinStore.showModal = false }
  },
  {
    key: 'videoExtractCreate',
    load: () => import('@/components/media/VideoExtractCreateModal.vue'),
    isOpen: () => videoExtractStore.showCreateModal,
    close: () => { videoExtractStore.showCreateModal = false }
  },
  {
    key: 'videoExtractTask',
    load: () => import('@/components/media/VideoExtractTaskModal.vue'),
    isOpen: () => videoExtractStore.showTaskModal,
    close: () => { videoExtractStore.showTaskModal = false }
  }
]

// 已下载完成的弹窗组件
const modalComponents = shallowReactive<Partial<Record<ModalKey, Component>>>({})
// 是否已挂载（首次打开后置为 true，之后保持挂载）
const modalMounted = reactive<Partial<Record<ModalKey, boolean>>>({})
// 进行中的下载，避免重复请求；失败后清除以便下次重试
const pendingLoads: Partial<Record<ModalKey, Promise<void>>> = {}

const ensureLoaded = (modal: ModalConfig): Promise<void> => {
  if (modalComponents[modal.key]) return Promise.resolve()
  pendingLoads[modal.key] ??= modal.load().then(
    (mod) => {
      modalComponents[modal.key] = markRaw(mod.default)
    },
    (err) => {
      delete pendingLoads[modal.key]
      throw err
    }
  )
  return pendingLoads[modal.key]!
}

for (const modal of modals) {
  watch(modal.isOpen, (open) => {
    if (!open) return
    modalMounted[modal.key] = true
    ensureLoaded(modal).catch(() => {
      // 下载失败：关闭弹窗并提示，下次打开时重新加载
      if (!modalComponents[modal.key]) modal.close()
      showToast('加载失败，请检查网络后重试')
    })
  }, { immediate: true })
}

// 有弹窗已打开但代码尚未就绪时显示加载占位
const modalLoading = computed(() => modals.some((modal) => modal.isOpen() && !modalComponents[modal.key]))

const cancelPendingModals = () => {
  for (const modal of modals) {
    if (modal.isOpen() && !modalComponents[modal.key]) modal.close()
  }
}

// 进入登录后的页面时，在浏览器空闲时预取弹窗代码，首次打开无需等待下载
let prefetched = false
const prefetchModals = () => {
  if (prefetched) return
  prefetched = true
  const run = () => {
    for (const modal of modals) {
      ensureLoaded(modal).catch(() => {
        // 预取失败不影响使用，打开时会再次加载
      })
    }
  }
  if (typeof window.requestIdleCallback === 'function') {
    window.requestIdleCallback(run, { timeout: 3000 })
  } else {
    setTimeout(run, 1500)
  }
}

const route = useRoute()
watch(() => route.meta.requiresAuth, (requiresAuth) => {
  if (requiresAuth) prefetchModals()
}, { immediate: true })
</script>

<style>
/* 路由过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
