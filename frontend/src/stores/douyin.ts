import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useDouyinStore = defineStore('douyin', () => {
  const showModal = ref(false)
  const draftInput = ref('')

  const open = (prefill?: string) => {
    if (prefill && !draftInput.value) {
      draftInput.value = String(prefill)
    }
    showModal.value = true
  }

  const close = () => {
    showModal.value = false
    draftInput.value = ''
  }

  return {
    showModal,
    draftInput,
    open,
    close
  }
})
