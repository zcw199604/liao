import { defineStore } from 'pinia'
import { ref } from 'vue'

export type DouyinEntryMode = 'default' | 'favorites'
export type DouyinFavoritesTab = 'users' | 'awemes'
export type DouyinTargetMode = 'detail' | 'account' | 'favorites'

export interface DouyinOpenOptions {
  prefill?: string
  entryMode?: DouyinEntryMode
  favoritesTab?: DouyinFavoritesTab
  targetMode?: DouyinTargetMode
  accountSecUserId?: string
  autoFetchAccount?: boolean
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

  // 弹窗打开行为扩展：支持直达“用户作品”并自动拉取
  const targetMode = ref<DouyinTargetMode>('detail')
  const accountSecUserId = ref('')
  const autoFetchAccount = ref(false)

  const open = (input?: string | DouyinOpenOptions) => {
    const opts = normalizeOpenOptions(input)

    if (opts.prefill && !draftInput.value) {
      draftInput.value = String(opts.prefill)
    }

    entryMode.value = opts.entryMode === 'favorites' ? 'favorites' : 'default'
    favoritesTab.value = opts.favoritesTab === 'awemes' ? 'awemes' : 'users'

    // 若显式要求直达收藏模式，优先落到 favorites。
    if (opts.targetMode === 'favorites') {
      targetMode.value = 'favorites'
    } else if (opts.targetMode === 'account') {
      targetMode.value = 'account'
    } else {
      targetMode.value = 'detail'
    }

    accountSecUserId.value = String(opts.accountSecUserId || '').trim()
    autoFetchAccount.value = !!opts.autoFetchAccount

    showModal.value = true
  }

  const close = () => {
    showModal.value = false
    draftInput.value = ''
    entryMode.value = 'default'
    favoritesTab.value = 'users'
    targetMode.value = 'detail'
    accountSecUserId.value = ''
    autoFetchAccount.value = false
  }

  return {
    showModal,
    draftInput,
    entryMode,
    favoritesTab,
    targetMode,
    accountSecUserId,
    autoFetchAccount,
    open,
    close
  }
})
