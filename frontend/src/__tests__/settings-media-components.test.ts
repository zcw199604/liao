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

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({
    disconnect: vi.fn(),
    connect: vi.fn(),
    send: vi.fn()
  })
}))

vi.mock('@/composables/useSettings', () => ({
  useSettings: () => ({
    connectionStats: { value: {} },
    forceoutUserCount: { value: 0 },
    disconnectAllLoading: { value: false },
    loadConnectionStats: vi.fn().mockResolvedValue(undefined),
    loadForceoutUserCount: vi.fn().mockResolvedValue(undefined),
    disconnectAll: vi.fn().mockResolvedValue(true),
    clearForceout: vi.fn().mockResolvedValue({ success: true, message: 'ok' })
  })
}))

import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'

import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())

  if (!(HTMLElement.prototype as any).scrollTo) {
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      value: () => {}
    })
  }
})

describe('components/settings/SettingsDrawer.vue', () => {
  it('emits update:visible=false when clicking overlay', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN',
      cookie: 'c'
    } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const wrapper = mount(SettingsDrawer, {
      props: { visible: true, mode: 'identity' },
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          Dialog: true,
          SystemSettings: true,
          GlobalFavorites: true
        }
      }
    })

    await wrapper.get('div.fixed.inset-0').trigger('click')
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('shows toast when saving without changes and exits edit mode', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN',
      cookie: 'c'
    } as any

    const wrapper = mount(SettingsDrawer, {
      props: { visible: true, mode: 'identity' },
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          Dialog: true,
          SystemSettings: true,
          GlobalFavorites: true
        }
      }
    })

    const editBtn = wrapper.findAll('button').find(btn => btn.text().includes('编辑'))
    expect(editBtn).toBeTruthy()
    await editBtn!.trigger('click')
    await nextTick()

    const saveBtn = wrapper.findAll('button').find(btn => btn.text().includes('保存'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    await nextTick()

    expect(toastShow).toHaveBeenCalledWith('没有任何修改')
    expect(wrapper.findAll('button').some(btn => btn.text().includes('编辑'))).toBe(true)
  })
})

describe('components/media/MediaPreview.vue', () => {
  it('navigates between media items and closes on Escape', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [
          { url: 'http://x/1.png', type: 'image' },
          { url: 'http://x/2.png', type: 'image' }
        ]
      },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    const img = wrapper.get('img[alt=\"预览\"]')
    expect(img.attributes('src')).toBe('http://x/1.png')

    await wrapper.get('button[title=\"下一张 (→)\"]').trigger('click')
    await nextTick()
    expect(wrapper.get('img[alt=\"预览\"]').attributes('src')).toBe('http://x/2.png')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('retries media load by appending cache buster on error', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 5, 12, 0, 0))

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image'
      },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    const img = wrapper.get('img[alt=\"预览\"]')
    expect(img.attributes('src')).toBe('http://x/1.png')

    await img.trigger('error')
    vi.advanceTimersByTime(600)
    await nextTick()

    expect(wrapper.get('img[alt=\"预览\"]').attributes('src')).toMatch(/^http:\/\/x\/1\.png\?_=\d+$/)

    vi.useRealTimers()
  })

  it('shows detail button when media has md5 without fileSize', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [{ url: 'http://x/1.png', type: 'image', md5: 'abc' }]
      },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    const detailBtn = wrapper.find('button[title=\"查看详细信息\"]')
    expect(detailBtn.exists()).toBe(true)

    await detailBtn.trigger('click')
    await nextTick()

    expect(wrapper.find('h3').text()).toContain('文件详细信息')
  })

  it('does not show detail button when media has no metadata fields', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image'
      },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    expect(wrapper.find('button[title=\"查看详细信息\"]').exists()).toBe(false)
  })

  it('resets detail panel visibility when closing preview', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [{ url: 'http://x/1.png', type: 'image', md5: 'abc' }]
      },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    await wrapper.get('button[title=\"查看详细信息\"]').trigger('click')
    await nextTick()
    expect(wrapper.find('h3').text()).toContain('文件详细信息')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])

    await wrapper.setProps({ visible: false })
    await nextTick()

    await wrapper.setProps({ visible: true })
    await nextTick()

    expect(wrapper.find('h3').exists()).toBe(false)
  })

  it('downloads /api resource with Authorization when downloadUrl is provided', async () => {
    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => 'attachment; filename=%E4%B8%AD%E6%96%87.jpg' },
      blob: async () => new Blob(['x'], { type: 'image/jpeg' })
    } as any)
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    const needRestoreCreateObjectURL = originalCreateObjectURL === undefined
    const needRestoreRevokeObjectURL = originalRevokeObjectURL === undefined

    if (!(URL as any).createObjectURL) {
      Object.defineProperty(URL, 'createObjectURL', {
        configurable: true,
        value: vi.fn().mockReturnValue('blob:mock')
      })
    }
    if (!(URL as any).revokeObjectURL) {
      Object.defineProperty(URL, 'revokeObjectURL', {
        configurable: true,
        value: vi.fn()
      })
    }

    const createObjectURLSpy = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock' as any)
    const revokeObjectURLSpy = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const originalCreateElement = document.createElement.bind(document)
    let createdAnchor: HTMLAnchorElement | null = null
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tagName: any) => {
      const el = originalCreateElement(tagName)
      if (String(tagName).toLowerCase() === 'a') {
        createdAnchor = el as HTMLAnchorElement
      }
      return el
    })

    localStorage.setItem('authToken', 't')

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [
          {
            url: 'http://x/1.png',
            type: 'image',
            md5: 'abc',
            downloadUrl: '/api/downloadMtPhotoOriginal?id=1&md5=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
          }
        ]
      },
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    await wrapper.get('button[title=\"下载\"]').trigger('click')
    await nextTick()
    await Promise.resolve()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [href, options] = fetchMock.mock.calls[0]!
    expect(String(href)).toContain('/api/downloadMtPhotoOriginal')
    expect(options?.headers?.Authorization).toBe('Bearer t')
    expect((createdAnchor as any)?.download).toBe('中文.jpg')

    localStorage.removeItem('authToken')
    createElementSpy.mockRestore()
    clickSpy.mockRestore()
    createObjectURLSpy.mockRestore()
    revokeObjectURLSpy.mockRestore()
    if (needRestoreCreateObjectURL) {
      ;(URL as any).createObjectURL = originalCreateObjectURL
    }
    if (needRestoreRevokeObjectURL) {
      ;(URL as any).revokeObjectURL = originalRevokeObjectURL
    }
    ;(globalThis as any).fetch = originalFetch
  })

  it('downloads upstream /img/Upload resource via /api/downloadImgUpload proxy', async () => {
    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => "attachment; filename*=UTF-8''a.jpg" },
      blob: async () => new Blob(['x'], { type: 'image/jpeg' })
    } as any)
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    const needRestoreCreateObjectURL = originalCreateObjectURL === undefined
    const needRestoreRevokeObjectURL = originalRevokeObjectURL === undefined

    if (!(URL as any).createObjectURL) {
      Object.defineProperty(URL, 'createObjectURL', {
        configurable: true,
        value: vi.fn().mockReturnValue('blob:mock')
      })
    }
    if (!(URL as any).revokeObjectURL) {
      Object.defineProperty(URL, 'revokeObjectURL', {
        configurable: true,
        value: vi.fn()
      })
    }

    const createObjectURLSpy = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock' as any)
    const revokeObjectURLSpy = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const originalCreateElement = document.createElement.bind(document)
    let createdAnchor: HTMLAnchorElement | null = null
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tagName: any) => {
      const el = originalCreateElement(tagName)
      if (String(tagName).toLowerCase() === 'a') {
        createdAnchor = el as HTMLAnchorElement
      }
      return el
    })

    localStorage.setItem('authToken', 't')

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://img:9006/img/Upload/2026/01/a.jpg',
        type: 'image',
        mediaList: [{ url: 'http://img:9006/img/Upload/2026/01/a.jpg', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()

    await wrapper.get('button[title=\"下载\"]').trigger('click')
    await nextTick()
    await Promise.resolve()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [href, options] = fetchMock.mock.calls[0]!
    expect(String(href)).toContain('/api/downloadImgUpload?path=2026%2F01%2Fa.jpg')
    expect(options?.headers?.Authorization).toBe('Bearer t')
    expect((createdAnchor as any)?.download).toBe('a.jpg')

    localStorage.removeItem('authToken')
    createElementSpy.mockRestore()
    clickSpy.mockRestore()
    createObjectURLSpy.mockRestore()
    revokeObjectURLSpy.mockRestore()
    if (needRestoreCreateObjectURL) {
      ;(URL as any).createObjectURL = originalCreateObjectURL
    }
    if (needRestoreRevokeObjectURL) {
      ;(URL as any).revokeObjectURL = originalRevokeObjectURL
    }
    ;(globalThis as any).fetch = originalFetch
  })
})
