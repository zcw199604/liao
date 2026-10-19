import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'

// 控制弹窗模块的加载：抖音弹窗等待 gate 放行，任务弹窗首次加载失败
const control = vi.hoisted(() => {
  let release: () => void = () => {}
  const gate = new Promise<void>((resolve) => { release = resolve })
  return { gate, release: () => release(), taskFailNext: true, taskLoads: 0 }
})

const toastShow = vi.hoisted(() => vi.fn())
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ show: toastShow }) }))

const modalMock = (name: string) => ({
  __esModule: true,
  default: defineComponent({ name, render: () => h('div', { class: `modal-${name}` }) })
})

vi.mock('@/components/media/AllUploadImageModal.vue', () => modalMock('allUpload'))
vi.mock('@/components/media/MtPhotoAlbumModal.vue', () => modalMock('mtPhoto'))
vi.mock('@/components/media/VideoExtractCreateModal.vue', () => modalMock('videoExtractCreate'))
vi.mock('@/components/media/DouyinDownloadModal.vue', async () => {
  await control.gate
  return modalMock('douyin')
})
vi.mock('@/components/media/VideoExtractTaskModal.vue', () => {
  control.taskLoads += 1
  if (control.taskFailNext) {
    control.taskFailNext = false
    throw new Error('network')
  }
  return modalMock('videoExtractTask')
})

import App from '@/App.vue'
import { useDouyinStore } from '@/stores/douyin'
import { useVideoExtractStore } from '@/stores/videoExtract'

const mountApp = async () => {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/login', component: defineComponent({ render: () => h('div') }), meta: { requiresAuth: false } }]
  })
  await router.push('/login')
  await router.isReady()
  return mount(App, { global: { plugins: [pinia, router] } })
}

describe('App.vue modal loading placeholder', () => {
  it('shows a loading placeholder until the modal chunk is ready; clicking it cancels', async () => {
    const wrapper = await mountApp()
    const douyinStore = useDouyinStore()
    expect(wrapper.find('[data-testid="modal-loading"]').exists()).toBe(false)

    douyinStore.showModal = true
    await flushPromises()
    expect(wrapper.find('[data-testid="modal-loading"]').text()).toContain('加载中')

    // 点击遮罩取消打开
    await wrapper.find('[data-testid="modal-loading"]').trigger('click')
    expect(douyinStore.showModal).toBe(false)
    expect(wrapper.find('[data-testid="modal-loading"]').exists()).toBe(false)

    // 再次打开，下载完成后占位消失、弹窗出现
    douyinStore.showModal = true
    await flushPromises()
    expect(wrapper.find('[data-testid="modal-loading"]').exists()).toBe(true)
    control.release()
    await vi.waitFor(async () => {
      await flushPromises()
      expect(wrapper.find('.modal-douyin').exists()).toBe(true)
    })
    expect(wrapper.find('[data-testid="modal-loading"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('closes the modal with a toast when loading fails, and retries on next open', async () => {
    const wrapper = await mountApp()
    const store = useVideoExtractStore()

    store.showTaskModal = true
    await vi.waitFor(async () => {
      await flushPromises()
      expect(store.showTaskModal).toBe(false)
    })
    expect(toastShow).toHaveBeenCalledWith('加载失败，请检查网络后重试')
    expect(wrapper.find('[data-testid="modal-loading"]').exists()).toBe(false)

    store.showTaskModal = true
    await vi.waitFor(async () => {
      await flushPromises()
      expect(wrapper.find('.modal-videoExtractTask').exists()).toBe(true)
    })
    expect(control.taskLoads).toBe(2)
    wrapper.unmount()
  })
})
