import { defineStore } from 'pinia'
import { ref } from 'vue'

export type DouyinEntryMode = 'default' | 'favorites'
export type DouyinFavoritesTab = 'users' | 'awemes'

export interface DouyinOpenOptions {
  prefill?: string
  entryMode?: DouyinEntryMode
  favoritesTab?: DouyinFavoritesTab
}

const normalizeOpenOptions = (input?: string | DouyinOpenOptions): DouyinOpenOptions => {
  // 兼容旧签名：open('xxx')
  if (typeof input === 'string') {
    return { prefill: input }
  }
  if (!input) {
    return {}
  }
  return input
}

export const useDouyinStore = defineStore('douyin', () => {
  const showModal = ref(false)
  const draftInput = ref('')

  // 入口上下文：用于从聊天上传菜单直达“收藏/用户收藏”
  const entryMode = ref<DouyinEntryMode>('default')
  const favoritesTab = ref<DouyinFavoritesTab>('users')

  const open = (input?: string | DouyinOpenOptions) => {
    const opts = normalizeOpenOptions(input)

    if (opts.prefill && !draftInput.value) {
      draftInput.value = String(opts.prefill)
    }

    entryMode.value = opts.entryMode === 'favorites' ? 'favorites' : 'default'
    favoritesTab.value = opts.favoritesTab === 'awemes' ? 'awemes' : 'users'
    showModal.value = true
  }

  const close = () => {
    showModal.value = false
    draftInput.value = ''
    entryMode.value = 'default'
    favoritesTab.value = 'users'
  }

  return {
    showModal,
    draftInput,
    entryMode,
    favoritesTab,
    open,
    close
  }
})
