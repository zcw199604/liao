import { ref, watch, onUnmounted } from 'vue'

export const DEFAULT_MODAL_FULLSCREEN_STORAGE_KEY = 'media_modal_fullscreen'

const readFullscreenFromStorage = (key: string): boolean => {
  try {
    return localStorage.getItem(key) === '1'
  } catch {
    return false
  }
}

const writeFullscreenToStorage = (key: string, value: boolean) => {
  try {
    localStorage.setItem(key, value ? '1' : '0')
  } catch {
    // ignore
  }
}

const isEditableTarget = (target: EventTarget | null): boolean => {
  const el = target as HTMLElement | null
  if (!el) return false
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return el.isContentEditable
}

export const useModalFullscreen = (options: {
  isModalOpen: () => boolean
  isBlocked?: () => boolean
  onRequestClose: () => void
  storageKey?: string
}) => {
  const storageKey = options.storageKey || DEFAULT_MODAL_FULLSCREEN_STORAGE_KEY
  const isFullscreen = ref<boolean>(readFullscreenFromStorage(storageKey))

  const syncFromStorage = () => {
    isFullscreen.value = readFullscreenFromStorage(storageKey)
  }

  const setFullscreen = (value: boolean) => {
    isFullscreen.value = value
    writeFullscreenToStorage(storageKey, value)
  }

  const toggleFullscreen = () => {
    setFullscreen(!isFullscreen.value)
  }

  const exitFullscreen = () => {
    if (!isFullscreen.value) return
    setFullscreen(false)
  }

  const handleKeydown = (e: KeyboardEvent) => {
    if (!options.isModalOpen()) return
    if (options.isBlocked?.() === true) return
    if (e.metaKey || e.ctrlKey || e.altKey) return
    if (isEditableTarget(e.target)) return
    if (e.isComposing) return

    if (e.key === 'Escape') {
      if (isFullscreen.value) {
        e.preventDefault()
        exitFullscreen()
        return
      }
      options.onRequestClose()
      return
    }

    if (e.repeat) return
    if (e.code === 'KeyF' || e.key === 'f' || e.key === 'F') {
      e.preventDefault()
      toggleFullscreen()
    }
  }

  watch(
    options.isModalOpen,
    (open) => {
      if (open) {
        syncFromStorage()
        window.addEventListener('keydown', handleKeydown)
      } else {
        window.removeEventListener('keydown', handleKeydown)
      }
    },
    { immediate: true }
  )

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
  })

  return {
    isFullscreen,
    syncFromStorage,
    setFullscreen,
    toggleFullscreen,
    exitFullscreen
  }
}
