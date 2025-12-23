import { ref } from 'vue'

const toastMessage = ref('')
const toastVisible = ref(false)
let toastTimer: ReturnType<typeof setTimeout> | null = null

export const useToast = () => {
  const show = (message: string, duration: number = 2000) => {
    toastMessage.value = message
    toastVisible.value = true

    if (toastTimer) {
      clearTimeout(toastTimer)
    }

    toastTimer = setTimeout(() => {
      toastVisible.value = false
    }, duration)
  }

  const hide = () => {
    toastVisible.value = false
    if (toastTimer) {
      clearTimeout(toastTimer)
      toastTimer = null
    }
  }

  const error = (message: string) => {
    show(message, 3000)
  }

  const success = (message: string) => {
    show(message, 2000)
  }

  return {
    message: toastMessage,
    visible: toastVisible,
    show,
    hide,
    error,
    success
  }
}
