import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'

// 记录每个弹窗模块被加载（import）的次数，用于验证按需加载与预取
const loadCounts = vi.hoisted(() => ({} as Record<string, number>))

const makeModalMock = (name: string) => {
  loadCounts[name] = (loadCounts[name] || 0) + 1
  return { __esModule: true, default: defineComponent({ name, render: () => h('div', { class: `modal-${name}` }) }) }
}

vi.mock('@/components/media/AllUploadImageModal.vue', () => makeModalMock('allUpload'))
vi.mock('@/components/media/MtPhotoAlbumModal.vue', () => makeModalMock('mtPhoto'))
vi.mock('@/components/media/DouyinDownloadModal.vue', () => makeModalMock('douyin'))
vi.mock('@/components/media/VideoExtractCreateModal.vue', () => makeModalMock('videoExtractCreate'))
vi.mock('@/components/media/VideoExtractTaskModal.vue', () => makeModalMock('videoExtractTask'))

import App from '@/App.vue'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useDouyinStore } from '@/stores/douyin'
import { useVideoExtractStore } from '@/stores/videoExtract'

const Page = defineComponent({ render: () => h('div') })

const createTestRouter = () =>
  createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', component: Page, meta: { requiresAuth: false } },
      { path: '/identity', component: Page, meta: { requiresAuth: true } }
    ]
  })

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('App.vue lazy global modals', () => {
  it('prefetches modal chunks when idle after entering an authenticated route', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/login')
    await router.isReady()

    const idleCallbacks: Array<() => void> = []
    vi.stubGlobal('requestIdleCallback', (cb: () => void) => {
      idleCallbacks.push(cb)
      return 1
    })

    const wrapper = mount(App, { global: { plugins: [pinia, router] } })
    await flushPromises()
    // 登录页不预取
    expect(idleCallbacks).toHaveLength(0)

    await router.push('/identity')
    await flushPromises()
    expect(idleCallbacks).toHaveLength(1)
    // 空闲回调执行前尚未加载任何弹窗代码
    expect(Object.keys(loadCounts)).toHaveLength(0)

    idleCallbacks[0]!()
    await vi.waitFor(() => {
      for (const name of ['allUpload', 'mtPhoto', 'douyin', 'videoExtractCreate', 'videoExtractTask']) {
        expect(loadCounts[name]).toBe(1)
      }
    })

    // 只预取一次
    await router.push('/login')
    await router.push('/identity')
    await flushPromises()
    expect(idleCallbacks).toHaveLength(1)
    wrapper.unmount()
  })

  it('mounts each modal only after it is first opened, and keeps it mounted after closing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/login')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [pinia, router] } })
    await flushPromises()
    for (const name of ['allUpload', 'mtPhoto', 'douyin', 'videoExtractCreate', 'videoExtractTask']) {
      expect(wrapper.find(`.modal-${name}`).exists()).toBe(false)
    }

    const mediaStore = useMediaStore()
    mediaStore.showAllUploadImageModal = true
    useMtPhotoStore().showModal = true
    useDouyinStore().showModal = true
    useVideoExtractStore().showCreateModal = true
    useVideoExtractStore().showTaskModal = true

    // 动态 import 需要真实的异步加载时间
    await vi.waitFor(async () => {
      await flushPromises()
      for (const name of ['allUpload', 'mtPhoto', 'douyin', 'videoExtractCreate', 'videoExtractTask']) {
        expect(wrapper.find(`.modal-${name}`).exists()).toBe(true)
      }
    })

    // 关闭后保持挂载，保留弹窗内部状态
    mediaStore.showAllUploadImageModal = false
    await flushPromises()
    expect(wrapper.find('.modal-allUpload').exists()).toBe(true)
    wrapper.unmount()
  })

  it('falls back to setTimeout when requestIdleCallback is unavailable', async () => {
    vi.useFakeTimers()
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/identity')
    await router.isReady()

    vi.stubGlobal('requestIdleCallback', undefined)
    const timeoutSpy = vi.spyOn(globalThis, 'setTimeout')
    const wrapper = mount(App, { global: { plugins: [pinia, router] } })
    expect(timeoutSpy).toHaveBeenCalledWith(expect.any(Function), 1500)
    vi.runAllTimers()
    wrapper.unmount()
    vi.useRealTimers()
  })
})
