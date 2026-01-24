import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

import DouyinDownloadModal from '@/components/media/DouyinDownloadModal.vue'
import { useDouyinStore } from '@/stores/douyin'

const flushAsync = async () => {
  await new Promise<void>((resolve) => setImmediate(resolve))
  await nextTick()
}

const defineClipboard = (readText: any) => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      readText
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/media/DouyinDownloadModal.vue', () => {
  it('pastes any non-empty clipboard text (no douyin filter)', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const readText = vi.fn().mockResolvedValue('not douyin')
    defineClipboard(readText)

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await nextTick()

    const pasteBtn = wrapper.findAll('button').find((btn) => btn.text().includes('粘贴'))
    expect(pasteBtn).toBeTruthy()

    await pasteBtn!.trigger('click')
    await flushAsync()

    expect(readText).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('not douyin')
    expect(toastShow).toHaveBeenCalledWith('已从剪贴板填充')
  })

  it('auto-reads clipboard on open and fills any non-empty text', async () => {
    localStorage.setItem('douyin_auto_clipboard', '1')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const readText = vi.fn().mockResolvedValue('not douyin')
    defineClipboard(readText)

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    expect(readText).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('not douyin')
    expect(toastShow).toHaveBeenCalledWith('已从剪贴板读取')
  })
})

