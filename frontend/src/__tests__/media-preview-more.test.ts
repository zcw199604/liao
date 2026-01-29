import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()

const uploadMocks = {
  uploadFile: vi.fn()
}

const openCreateFromMediaMock = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

vi.mock('@/stores/videoExtract', () => ({
  useVideoExtractStore: () => ({
    openCreateFromMedia: openCreateFromMediaMock
  })
}))

vi.mock('plyr', () => {
  class MockPlyr {
    media: any
    speed = 1
    volume = 1
    currentTime = 0
    fullscreen = {
      active: false,
      toggle: () => {
        this.fullscreen.active = !this.fullscreen.active
      },
      exit: () => {
        this.fullscreen.active = false
      }
    }

    constructor(media: any) {
      this.media = media
    }

    play() {
      return Promise.resolve()
    }

    pause() {}

    destroy() {}
  }

  return { default: MockPlyr }
})

import MediaPreview from '@/components/media/MediaPreview.vue'
import { useUserStore } from '@/stores/user'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())

  if (typeof (HTMLElement.prototype as any).scrollTo !== 'function') {
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      value: () => {}
    })
  }
})

describe('components/media/MediaPreview.vue (more coverage)', () => {
  it('upload button text covers default per type and custom uploadText', async () => {
    const mountWith = (type: 'image' | 'video' | 'file', uploadText = '') =>
      mount(MediaPreview, {
        props: {
          visible: true,
          url: type === 'image' ? 'http://x/1.png' : type === 'video' ? '/upload/videos/2026/01/a.mp4' : '/upload/files/a.pdf',
          type,
          canUpload: true,
          uploadText,
          mediaList: [
            {
              url: type === 'image' ? 'http://x/1.png' : type === 'video' ? '/upload/videos/2026/01/a.mp4' : '/upload/files/a.pdf',
              type
            }
          ]
        },
        global: {
          stubs: {
            teleport: true,
            MediaDetailPanel: { template: '<div />' }
          }
        }
      })

    const wrapperImage = mountWith('image')
    await flushAsync()
    expect(wrapperImage.findAll('button').some((b) => b.text().includes('上传此图片'))).toBe(true)

    const wrapperVideo = mountWith('video')
    await flushAsync()
    expect(wrapperVideo.findAll('button').some((b) => b.text().includes('上传此视频'))).toBe(true)

    const wrapperFile = mountWith('file')
    await flushAsync()
    expect(wrapperFile.findAll('button').some((b) => b.text().includes('上传此文件'))).toBe(true)

    const wrapperCustom = mountWith('image', '自定义上传')
    await flushAsync()
    expect(wrapperCustom.findAll('button').some((b) => b.text().includes('自定义上传'))).toBe(true)
  })

  it('image click toggles zoom and swipe-drag navigates next/prev', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [
          { url: 'http://x/1.png', type: 'image' },
          { url: 'http://x/2.png', type: 'image' }
        ]
      },
      global: { stubs: { teleport: true } }
    })

    await flushAsync()

    const img = wrapper.get('img[alt="预览"]')
    await img.trigger('click')
    expect((wrapper.vm as any).scale).toBe(3)

    await img.trigger('click')
    expect((wrapper.vm as any).scale).toBe(1)

    // swipe left -> next
    await img.trigger('mousedown', { clientX: 200, clientY: 10 })
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 0, clientY: 10, cancelable: true }))
    window.dispatchEvent(new MouseEvent('mouseup', { clientX: 0, clientY: 10 }))
    await flushAsync()
    expect(wrapper.get('img[alt="预览"]').attributes('src')).toBe('http://x/2.png')
    expect(wrapper.emitted('media-change')?.length).toBeGreaterThan(0)

    // swipe right -> prev (back to first)
    await img.trigger('mousedown', { clientX: 0, clientY: 10 })
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 200, clientY: 10, cancelable: true }))
    window.dispatchEvent(new MouseEvent('mouseup', { clientX: 200, clientY: 10 }))
    await flushAsync()
    expect(wrapper.get('img[alt="预览"]').attributes('src')).toBe('http://x/1.png')
  })

  it('touch swipe navigates next for image mode (touch branches)', async () => {
    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: 'http://x/1.png',
        type: 'image',
        mediaList: [
          { url: 'http://x/1.png', type: 'image' },
          { url: 'http://x/2.png', type: 'image' }
        ]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    const img = wrapper.get('img[alt="预览"]')
    await img.trigger('touchstart', { touches: [{ clientX: 200, clientY: 10 }] })

    const move = new Event('touchmove', { bubbles: true, cancelable: true })
    Object.defineProperty(move, 'touches', { configurable: true, value: [{ clientX: 0, clientY: 10 }] })
    window.dispatchEvent(move)

    window.dispatchEvent(new Event('touchend', { bubbles: true }))
    await flushAsync()

    expect(wrapper.get('img[alt="预览"]').attributes('src')).toBe('http://x/2.png')
  })

  it('speed menu stays open on inside click and closes on outside pointerdown', async () => {
    if (!(globalThis as any).PointerEvent) {
      ;(globalThis as any).PointerEvent = MouseEvent as any
    }

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: '/upload/videos/2026/01/a.mp4',
        type: 'video',
        mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
      },
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()
    await flushAsync()

    const speedBtn = wrapper.find('button[title*="倍速"]')
    expect(speedBtn.exists()).toBe(true)

    await speedBtn.trigger('click')
    await flushAsync()

    const option025 = wrapper.findAll('button').find(b => b.text().trim() === 'x0.25')
    expect(option025).toBeTruthy()

    option025!.element.dispatchEvent(new (globalThis as any).PointerEvent('pointerdown', { bubbles: true, button: 0 }))
    await flushAsync()
    expect(wrapper.findAll('button').some(b => b.text().trim() === 'x0.25')).toBe(true)

    window.dispatchEvent(new (globalThis as any).PointerEvent('pointerdown', { bubbles: true, button: 0 }))
    await flushAsync()
    expect(wrapper.findAll('button').some(b => b.text().trim() === 'x0.25')).toBe(false)
  })

  it('download triggers direct anchor for non-api urls, blocks api without token, and shows backend msg on non-ok', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
    ;(globalThis as any).fetch = fetchMock

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: 'https://example.com/a.jpg',
        type: 'image',
        mediaList: [{ url: 'https://example.com/a.jpg', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    await wrapper.get('button[title="下载"]').trigger('click')
    await flushAsync()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()

    // api without token
    const wrapper2 = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/api/x',
        type: 'image',
        mediaList: [{ url: '/api/x', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()
    await wrapper2.get('button[title="下载"]').trigger('click')
    expect(toastShow).toHaveBeenCalledWith('未登录或Token缺失')

    // api non-ok with json error
    localStorage.setItem('authToken', 't')
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 403,
      json: async () => ({ error: 'forbidden' })
    } as any)
    await wrapper2.get('button[title="下载"]').trigger('click')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('forbidden')

    localStorage.removeItem('authToken')
    clickSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
  })

  it('download proxies invalid img/Upload url via catch branch and rejects dangerous path', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn().mockReturnValue('blob:mock') })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })

    localStorage.setItem('authToken', 't')

    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: { get: () => '' },
      blob: async () => new Blob(['x'], { type: 'image/png' })
    } as any)

    const badHref = 'http://[bad]/img/Upload/2026/01/a.png?x=1'
    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: badHref,
        type: 'image',
        mediaList: [{ url: badHref, type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    await wrapper.get('button[title="下载"]').trigger('click')
    await flushAsync()

    expect(fetchMock).toHaveBeenCalled()
    expect(String(fetchMock.mock.calls[0]?.[0] || '')).toContain('/api/downloadImgUpload?path=2026%2F01%2Fa.png')

    // path contains ".." -> do not proxy; keep direct download path
    fetchMock.mockClear()
    clickSpy.mockClear()

    const badPathHref = 'http://[bad]/img/Upload/../secret.png'
    const wrapper2 = mount(MediaPreview, {
      props: {
        visible: true,
        url: badPathHref,
        type: 'image',
        mediaList: [{ url: badPathHref, type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()
    await wrapper2.get('button[title="下载"]').trigger('click')
    await flushAsync()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()

    localStorage.removeItem('authToken')
    clickSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: originalCreateObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: originalRevokeObjectURL })
  })

  it('download fallback name uses extension from mime for multiple types', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const originalCreateElement = document.createElement
    const anchors: HTMLAnchorElement[] = []
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: any) => {
      const el = originalCreateElement.call(document, tag) as any
      if (tag === 'a') anchors.push(el as HTMLAnchorElement)
      return el
    })

    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn().mockReturnValue('blob:mock') })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })

    localStorage.setItem('authToken', 't')

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/api/x',
        type: 'image',
        mediaList: [{ url: '/api/x', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    const cases: Array<[string, string]> = [
      ['image/png', 'download.png'],
      ['image/gif', 'download.gif'],
      ['image/webp', 'download.webp'],
      ['video/mp4', 'download.mp4'],
      ['application/octet-stream', 'download']
    ]

    for (const [mime] of cases) {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => '' },
        blob: async () => new Blob(['x'], { type: mime })
      } as any)
      await (wrapper.vm as any).handleDownload()
      await flushAsync()
    }

    for (const [, expected] of cases) {
      expect(anchors.some((a) => a.download === expected)).toBe(true)
    }

    localStorage.removeItem('authToken')
    clickSpy.mockRestore()
    createElementSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: originalCreateObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: originalRevokeObjectURL })
  })

  it('video pointer gestures cover tap, seek, volume, and cancel branches', async () => {
    const originalRaf = (globalThis as any).requestAnimationFrame
    const originalCaf = (globalThis as any).cancelAnimationFrame
    ;(globalThis as any).requestAnimationFrame = (cb: any) => {
      cb(0)
      return 1
    }
    ;(globalThis as any).cancelAnimationFrame = () => {}

    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))

    try {
      const wrapper = mount(MediaPreview, {
        props: {
          visible: true,
          url: '/upload/videos/2026/01/a.mp4',
          type: 'video',
          mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
        },
        global: { stubs: { teleport: true } }
      })
      await flushAsync()

      const videoWrapperEl = wrapper.get('.media-preview-video-wrapper').element as HTMLElement
      const video = wrapper.get('video').element as HTMLVideoElement
      const vm = wrapper.vm as any

      let paused = true
      Object.defineProperty(video, 'paused', { configurable: true, get: () => paused })
      Object.defineProperty(video, 'ended', { configurable: true, get: () => false })
      Object.defineProperty(video, 'duration', { configurable: true, value: 100 })
      Object.defineProperty(video, 'currentTime', { configurable: true, writable: true, value: 5 })
      Object.defineProperty(video, 'volume', { configurable: true, writable: true, value: 0.5 })

      video.play = vi.fn().mockImplementation(async () => {
        paused = false
      })
      video.pause = vi.fn().mockImplementation(() => {
        paused = true
      })

      ;(videoWrapperEl as any).getBoundingClientRect = () => ({ width: 500, height: 300 })

      const makeEvent = (pointerId: number, clientX: number, clientY: number, extra: Record<string, any> = {}) =>
        ({
          pointerId,
          clientX,
          clientY,
          button: 0,
          cancelable: true,
          preventDefault: vi.fn(),
          target: null,
          currentTarget: videoWrapperEl,
          ...extra
        } as any)

      // tap -> schedules toggleVideoPlay after window
      vm.handleVideoPointerDown(makeEvent(1, 10, 10))
      vm.handleVideoPointerUp(makeEvent(1, 10, 10))
      await vi.advanceTimersByTimeAsync(300)
      await flushAsync()
      expect(video.play).toHaveBeenCalled()

      // another tap -> pause branch
      vm.handleVideoPointerDown(makeEvent(2, 10, 10))
      vm.handleVideoPointerUp(makeEvent(2, 10, 10))
      await vi.advanceTimersByTimeAsync(300)
      await flushAsync()
      expect(video.pause).toHaveBeenCalled()

      // gesture: below-threshold move returns early, then horizontal seek applies
      vm.handleVideoPointerDown(makeEvent(3, 0, 0))
      vm.handleVideoPointerMove(makeEvent(999, 30, 0))
      vm.handleVideoPointerMove(makeEvent(3, 10, 0))
      vm.handleVideoPointerMove(makeEvent(3, 120, 0))
      await flushAsync()
      vm.handleVideoPointerUp(makeEvent(3, 120, 0))
      await flushAsync()

      // gesture: vertical volume applies and pointercancel clears state
      vm.handleVideoPointerDown(makeEvent(4, 0, 0))
      vm.handleVideoPointerMove(makeEvent(4, 0, 80))
      await flushAsync()
      vm.handleVideoPointerCancel(makeEvent(4, 0, 80))
      await flushAsync()
    } finally {
      vi.useRealTimers()
      ;(globalThis as any).requestAnimationFrame = originalRaf
      ;(globalThis as any).cancelAnimationFrame = originalCaf
    }
  })

  it('overlay controls cancel pending single-tap toggle (no unexpected play/pause flip)', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))

    try {
      const wrapper = mount(MediaPreview, {
        props: {
          visible: true,
          url: '/upload/videos/2026/01/a.mp4',
          type: 'video',
          mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
        },
        global: { stubs: { teleport: true } }
      })
      await flushAsync()

      const videoWrapperEl = wrapper.get('.media-preview-video-wrapper').element as HTMLElement
      const video = wrapper.get('video').element as HTMLVideoElement
      const vm = wrapper.vm as any

      let paused = true
      Object.defineProperty(video, 'paused', { configurable: true, get: () => paused })
      Object.defineProperty(video, 'ended', { configurable: true, get: () => false })
      Object.defineProperty(video, 'duration', { configurable: true, value: 100 })
      Object.defineProperty(video, 'currentTime', { configurable: true, writable: true, value: 5 })

      video.play = vi.fn().mockImplementation(async () => {
        paused = false
      })
      video.pause = vi.fn().mockImplementation(() => {
        paused = true
      })

      const makeEvent = (pointerId: number, clientX: number, clientY: number) =>
        ({
          pointerId,
          clientX,
          clientY,
          button: 0,
          cancelable: true,
          preventDefault: vi.fn(),
          target: null,
          currentTarget: videoWrapperEl
        } as any)

      // Case 1: Playing -> tap schedules pause, but user hits overlay pause quickly -> should only pause once.
      paused = false
      vm.handleVideoPointerDown(makeEvent(1, 10, 10))
      vm.handleVideoPointerUp(makeEvent(1, 10, 10))
      vm.handleOverlayTogglePlay()
      await flushAsync()
      await vi.advanceTimersByTimeAsync(300)
      await flushAsync()
      expect(video.pause).toHaveBeenCalledTimes(1)
      expect(paused).toBe(true)

      // Case 2: Paused -> tap would schedule play, but user seeks quickly -> should stay paused and just change time.
      paused = true
      ;(video.currentTime as any) = 5
      vm.handleVideoPointerDown(makeEvent(2, 10, 10))
      vm.handleVideoPointerUp(makeEvent(2, 10, 10))
      vm.handleOverlaySeek(1)
      await flushAsync()
      await vi.advanceTimersByTimeAsync(300)
      await flushAsync()
      expect(video.play).toHaveBeenCalledTimes(0)
      expect(paused).toBe(true)
      expect(video.currentTime).toBe(6)
    } finally {
      vi.useRealTimers()
    }
  })

  it('live photo download handles missing token and error response (zip via contextmenu)', async () => {
    if (!(globalThis as any).PointerEvent) {
      ;(globalThis as any).PointerEvent = MouseEvent as any
    }

    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
    ;(globalThis as any).fetch = fetchMock

    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/api/douyin/download?key=k1&index=0',
        type: 'image',
        mediaList: [
          { url: '/api/douyin/download?key=k1&index=0', type: 'image', context: { provider: 'douyin', key: 'k1', index: 0, liveVideoIndex: 1 } },
          { url: '/api/douyin/download?key=k1&index=1', type: 'video', context: { provider: 'douyin', key: 'k1', index: 1 } }
        ] as any
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    const liveBtn = wrapper.find('button[title^="下载实况"]')
    expect(liveBtn.exists()).toBe(true)

    await liveBtn.trigger('contextmenu')
    expect(toastShow).toHaveBeenCalledWith('未登录或Token缺失')

    localStorage.setItem('authToken', 't')
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ msg: 'bad' })
    } as any)
    await liveBtn.trigger('contextmenu')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('bad')

    fetchMock.mockRejectedValueOnce(new Error('boom'))
    await liveBtn.trigger('click')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('下载失败')

    localStorage.removeItem('authToken')
    clickSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
  })

  it('capture frame covers early exits and uploads when user is available', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const wrapper = mount(MediaPreview, {
      props: { visible: true, url: 'http://x/1.png', type: 'image' },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    await (wrapper.vm as any).handleCaptureFrame()
    expect(toastShow).toHaveBeenCalledWith('视频未就绪')

    // video mode: readyState < 2
    const wrapper2 = mount(MediaPreview, {
      props: { visible: false, url: '/upload/videos/2026/01/a.mp4', type: 'video', mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }] },
      global: { stubs: { teleport: true } }
    })
    await wrapper2.setProps({ visible: true })
    await flushAsync()

    const video = wrapper2.get('video').element as HTMLVideoElement
    Object.defineProperty(video, 'paused', { configurable: true, value: false })
    Object.defineProperty(video, 'ended', { configurable: true, value: false })
    Object.defineProperty(video, 'readyState', { configurable: true, value: 1 })
    Object.defineProperty(video, 'videoWidth', { configurable: true, value: 0 })
    Object.defineProperty(video, 'videoHeight', { configurable: true, value: 0 })
    Object.defineProperty(video, 'currentTime', { configurable: true, value: 0, writable: true })
    Object.defineProperty(video, 'duration', { configurable: true, value: 10 })
    video.pause = vi.fn()

    await (wrapper2.vm as any).handleCaptureFrame()
    expect(toastShow).toHaveBeenCalledWith('视频未加载完成，无法抓帧')

    // force a successful flow: mock canvas + toBlob
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    const originalToBlob = HTMLCanvasElement.prototype.toBlob
    const ctx: any = { drawImage: vi.fn() }
    Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', { configurable: true, value: vi.fn().mockReturnValue(ctx) })
    Object.defineProperty(HTMLCanvasElement.prototype, 'toBlob', {
      configurable: true,
      value: (cb: (b: Blob | null) => void) => cb(new Blob(['x'], { type: 'image/png' }))
    })

    Object.defineProperty(video, 'paused', { configurable: true, value: true })
    Object.defineProperty(video, 'readyState', { configurable: true, value: 2 })
    Object.defineProperty(video, 'videoWidth', { configurable: true, value: 100 })
    Object.defineProperty(video, 'videoHeight', { configurable: true, value: 100 })

    uploadMocks.uploadFile.mockResolvedValueOnce({ url: 'u', type: 'image' })
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    await (wrapper2.vm as any).handleCaptureFrame()
    expect(clickSpy).toHaveBeenCalled()
    expect(uploadMocks.uploadFile).toHaveBeenCalled()

    Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', { configurable: true, value: originalGetContext })
    Object.defineProperty(HTMLCanvasElement.prototype, 'toBlob', { configurable: true, value: originalToBlob })

    clickSpy.mockRestore()
  })

  it('extract frames shows unsupported toast when store returns false', async () => {
    openCreateFromMediaMock.mockResolvedValueOnce(false)

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/upload/videos/2026/01/a.mp4',
        type: 'video',
        mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
      },
      global: { stubs: { teleport: true } }
    })

    await flushAsync()
    await (wrapper.vm as any).handleExtractFrames()
    expect(toastShow).toHaveBeenCalledWith('当前媒体不支持抽帧')
  })

  it('keydown navigates media list and Escape closes', async () => {
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
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
    await flushAsync()
    expect((wrapper.vm as any).currentIndex).toBe(1)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }))
    await flushAsync()
    expect((wrapper.vm as any).currentIndex).toBe(0)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushAsync()
    expect(wrapper.emitted('update:visible')?.some((e) => e[0] === false)).toBe(true)
  })

  it('details button resolves original filename and opens detail panel', async () => {
    const resolver = vi.fn().mockResolvedValue('/tmp/path/a.png?token=1')

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: 'http://x/1.png',
        type: 'image',
        resolveOriginalFilename: resolver,
        mediaList: [{ url: 'http://x/1.png', type: 'image', md5: 'm1' } as any]
      },
      global: {
        stubs: {
          teleport: true,
          MediaDetailPanel: { template: '<div data-testid="detail-panel"></div>' }
        }
      }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()

    const infoBtn = wrapper.get('button[title="查看详细信息"]')
    await infoBtn.trigger('click')
    await flushAsync()

    expect(resolver).toHaveBeenCalledTimes(1)
    expect(((wrapper.vm as any).currentMedia || {}).originalFilename).toBe('a.png')
    expect((wrapper.vm as any).showDetails).toBe(true)
    expect(wrapper.find('[data-testid="detail-panel"]').exists()).toBe(true)
  })

  it('download proxies img/Upload url to api, decodes filename, and falls back to md5 name', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const originalCreateElement = document.createElement
    const anchors: HTMLAnchorElement[] = []
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: any) => {
      const el = originalCreateElement.call(document, tag) as any
      if (tag === 'a') anchors.push(el as HTMLAnchorElement)
      return el
    })

    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn().mockReturnValue('blob:mock') })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })

    localStorage.setItem('authToken', 't')

    // img/Upload -> /api/downloadImgUpload + filename from header
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: { get: () => "attachment; filename*=UTF-8''%E4%B8%AD%E6%96%87.png" },
      blob: async () => new Blob(['x'], { type: 'image/png' })
    } as any)

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: 'http://img.local:9006/img/Upload/2026/01/a.png?x=1',
        type: 'image',
        mediaList: [{ url: 'http://img.local:9006/img/Upload/2026/01/a.png?x=1', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    await wrapper.get('button[title="下载"]').trigger('click')
    await flushAsync()

    expect(fetchMock).toHaveBeenCalled()
    expect(anchors.some((a) => a.download === '中文.png')).toBe(true)

    // /api url fallback to mtphoto_${md5}${ext}
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: { get: () => '' },
      blob: async () => new Blob(['x'], { type: 'image/jpeg' })
    } as any)

    const wrapper2 = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/api/x',
        type: 'image',
        mediaList: [{ url: '/api/x', type: 'image', md5: 'm1' } as any]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()
    await wrapper2.get('button[title="下载"]').trigger('click')
    await flushAsync()
    expect(anchors.some((a) => a.download === 'mtphoto_m1.jpg')).toBe(true)

    localStorage.removeItem('authToken')
    clickSpy.mockRestore()
    createElementSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: originalCreateObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: originalRevokeObjectURL })
  })

  it('media error retries at most twice and updates display url with cache buster', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: 'http://x/1.png?x=1',
        type: 'image',
        mediaList: [{ url: 'http://x/1.png?x=1', type: 'image' }]
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    const vm = wrapper.vm as any
    expect(vm.currentMediaDisplayUrl).toBe('http://x/1.png?x=1')

    vm.handleMediaError()
    vm.handleMediaError()
    vm.handleMediaError()

    await vi.runAllTimersAsync()
    await flushAsync()

    expect(vm.mediaRetryCount).toBe(2)
    expect(String(vm.currentMediaDisplayUrl)).toContain('&_=')

    vi.useRealTimers()
  })

  it('video gesture fallbacks cover stepPx/volume fallback and volume gesture unsupported hint', async () => {
    const originalRaf = (globalThis as any).requestAnimationFrame
    const originalCaf = (globalThis as any).cancelAnimationFrame
    ;(globalThis as any).requestAnimationFrame = (cb: any) => {
      cb(0)
      return 1
    }
    ;(globalThis as any).cancelAnimationFrame = () => {}

    try {
      const wrapper = mount(MediaPreview, {
        props: {
          visible: false,
          url: '/upload/videos/2026/01/a.mp4',
          type: 'video',
          mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
        },
        global: { stubs: { teleport: true } }
      })
      await wrapper.setProps({ visible: true })
      await flushAsync()
      await flushAsync()

      const videoWrapperEl = wrapper.get('.media-preview-video-wrapper').element as HTMLElement
      const video = wrapper.get('video').element as HTMLVideoElement
      const vm = wrapper.vm as any

      let paused = true
      Object.defineProperty(video, 'paused', { configurable: true, get: () => paused })
      Object.defineProperty(video, 'ended', { configurable: true, get: () => false })
      Object.defineProperty(video, 'duration', { configurable: true, value: 100 })
      Object.defineProperty(video, 'currentTime', { configurable: true, writable: true, value: 5 })
      Object.defineProperty(video, 'volume', { configurable: true, writable: true, value: 0.5 })

      video.play = vi.fn().mockImplementation(async () => {
        paused = false
      })
      video.pause = vi.fn().mockImplementation(() => {
        paused = true
      })

      // width/height=0 forces stepPx/volume fallbacks inside applyVideoGestureFrame()
      ;(videoWrapperEl as any).getBoundingClientRect = () => ({ width: 0, height: 0 })

      const makeEvent = (pointerId: number, clientX: number, clientY: number, extra: Record<string, any> = {}) =>
        ({
          pointerId,
          clientX,
          clientY,
          button: 0,
          cancelable: true,
          preventDefault: vi.fn(),
          target: null,
          currentTarget: videoWrapperEl,
          ...extra
        } as any)

      // First vertical gesture sets volumeGestureSupported=false (plyr branch does not mutate video.volume)
      vm.handleVideoPointerDown(makeEvent(1, 0, 0))
      vm.handleVideoPointerMove(makeEvent(1, 0, 200))
      vm.handleVideoPointerUp(makeEvent(1, 0, 200))
      await flushAsync()
      expect(toastShow).toHaveBeenCalledWith('当前浏览器限制网页调节音量，请使用实体音量键')

      // Second vertical gesture hits volumeGestureSupported === false early-return path
      vm.handleVideoPointerDown(makeEvent(2, 0, 0))
      vm.handleVideoPointerMove(makeEvent(2, 0, 200))
      vm.handleVideoPointerUp(makeEvent(2, 0, 200))
      await flushAsync()

      // Horizontal gesture uses stepPx fallback (width=0) and should not throw
      vm.handleVideoPointerDown(makeEvent(3, 0, 0))
      vm.handleVideoPointerMove(makeEvent(3, 200, 0))
      vm.handleVideoPointerUp(makeEvent(3, 200, 0))
      await flushAsync()
    } finally {
      ;(globalThis as any).requestAnimationFrame = originalRaf
      ;(globalThis as any).cancelAnimationFrame = originalCaf
    }
  })

  it('live photo long-press shows motion and suppresses click', async () => {
    vi.useFakeTimers()

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/api/douyin/download?key=k1&index=0',
        type: 'image',
        mediaList: [
          { url: '/api/douyin/download?key=k1&index=0', type: 'image', context: { provider: 'douyin', key: 'k1', index: 0, liveVideoIndex: 1 } },
          { url: '/api/douyin/download?key=k1&index=1', type: 'video', context: { provider: 'douyin', key: 'k1', index: 1 } }
        ] as any
      },
      global: { stubs: { teleport: true } }
    })
    await flushAsync()

    const img = wrapper.get('img[alt="预览"]')
    await img.trigger('mousedown', { clientX: 0, clientY: 0 })
    await flushAsync()

    await vi.advanceTimersByTimeAsync(300)
    await flushAsync()
    expect((wrapper.vm as any).livePhotoVisible).toBe(true)

    window.dispatchEvent(new MouseEvent('mouseup'))
    await flushAsync()

    vi.useRealTimers()
  })

  it('file preview + virtual thumbnails branch render (useVirtualThumbnails)', async () => {
    const bigList = Array.from({ length: 201 }, (_, i) => ({
      url: i === 0 ? '/upload/files/a.pdf' : `http://x/${i}.png`,
      type: i === 0 ? 'file' : 'image'
    }))

    const wrapper = mount(MediaPreview, {
      props: {
        visible: true,
        url: '/upload/files/a.pdf',
        type: 'file',
        mediaList: bigList as any
      },
      global: {
        stubs: {
          teleport: true,
          RecycleScroller: {
            props: ['items'],
            template: '<div data-testid="recycle-scroller"></div>'
          },
          MediaTile: { template: '<div />' },
          MediaDetailPanel: { template: '<div />' }
        }
      }
    })
    await flushAsync()

    expect(wrapper.text()).toContain('暂不支持预览此文件类型')
    expect(wrapper.find('[data-testid="recycle-scroller"]').exists()).toBe(true)

    // currentIndex out-of-range fallback branch
    ;(wrapper.vm as any).currentIndex = 999
    await flushAsync()
    expect((wrapper.vm as any).currentIndex).toBe(999)
  })

  it('single tap shows overlay then auto-hides; double tap toggles fullscreen', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))

    if (!(globalThis as any).PointerEvent) {
      ;(globalThis as any).PointerEvent = MouseEvent as any
    }

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: '/upload/videos/2026/01/a.mp4',
        type: 'video',
        mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
      },
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    const videoWrapperEl = wrapper.get('.media-preview-video-wrapper').element as HTMLElement

    const makeEvent = (pointerId: number, clientX: number, clientY: number) =>
      ({
        pointerId,
        clientX,
        clientY,
        button: 0,
        cancelable: true,
        preventDefault: vi.fn(),
        target: null,
        currentTarget: videoWrapperEl
      } as any)

    // single tap -> overlay visible then auto hides
    vm.handleVideoPointerDown(makeEvent(1, 0, 0))
    vm.handleVideoPointerUp(makeEvent(1, 0, 0))
    await flushAsync()
    expect(vm.showVideoOverlayControls).toBe(true)

    vi.advanceTimersByTime(1000)
    await flushAsync()
    expect(vm.showVideoOverlayControls).toBe(false)

    // double tap within the window -> fullscreen toggles
    vm.handleVideoPointerDown(makeEvent(2, 10, 10))
    vm.handleVideoPointerUp(makeEvent(2, 10, 10))
    vi.advanceTimersByTime(100)
    vm.handleVideoPointerDown(makeEvent(3, 12, 12))
    vm.handleVideoPointerUp(makeEvent(3, 12, 12))
    await flushAsync()
    expect(vm.isVideoFullscreen).toBe(true)

    vi.useRealTimers()
  })

  it('speed long-press boosts temp speed and suppresses next speed-menu click', async () => {
    vi.useFakeTimers()

    if (!(globalThis as any).PointerEvent) {
      ;(globalThis as any).PointerEvent = MouseEvent as any
    }

    const wrapper = mount(MediaPreview, {
      props: {
        visible: false,
        url: '/upload/videos/2026/01/a.mp4',
        type: 'video',
        mediaList: [{ url: '/upload/videos/2026/01/a.mp4', type: 'video' }]
      },
      global: { stubs: { teleport: true } }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any

    // non-left click -> ignored
    vm.handleSpeedPressStart({ button: 1 } as any)

    // long-press triggers boost
    vm.handleSpeedPressStart({ button: 0 } as any)
    vi.advanceTimersByTime(320)
    await flushAsync()
    expect(vm.isTempSpeedBoosting).toBe(true)

    vm.handleSpeedPressEnd()
    await flushAsync()
    expect(vm.isTempSpeedBoosting).toBe(false)

    // suppressSpeedClick -> click toggling speed menu should be ignored once
    expect(vm.showSpeedMenu).toBe(false)
    vm.handleToggleSpeedMenu()
    await flushAsync()
    expect(vm.showSpeedMenu).toBe(false)

    // cancel path while boosting
    vm.handleSpeedPressStart({ button: 0 } as any)
    vi.advanceTimersByTime(320)
    await flushAsync()
    expect(vm.isTempSpeedBoosting).toBe(true)

    vm.handleSpeedPressCancel()
    await flushAsync()
    expect(vm.isTempSpeedBoosting).toBe(false)

    vi.useRealTimers()
  })
})
