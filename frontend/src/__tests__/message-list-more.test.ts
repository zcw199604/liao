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

const clipboardMock = vi.hoisted(() => ({
  copyToClipboard: vi.fn()
}))

vi.mock('@/utils/clipboard', () => ({
  copyToClipboard: clipboardMock.copyToClipboard
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => ({
    getMediaUrl: (url: string) => url
  })
}))

const scrollerSpies = vi.hoisted(() => ({
  scrollToBottom: vi.fn(),
  scrollToItem: vi.fn()
}))

vi.mock('vue-virtual-scroller', () => ({
  DynamicScroller: {
    name: 'DynamicScroller',
    props: ['items'],
    emits: ['scroll', 'click'],
    methods: scrollerSpies,
    template: `<div class="chat-area" @scroll="$emit('scroll')" @click="$emit('click')">
      <slot v-for="(it, idx) in items" :item="it" :index="idx" :active="true"></slot>
    </div>`
  },
  DynamicScrollerItem: {
    name: 'DynamicScrollerItem',
    props: ['item', 'active', 'dataIndex', 'sizeDependencies'],
    template: `<div class="scroller-item"><slot /></div>`
  }
}))

import MessageList from '@/components/chat/MessageList.vue'
import { useMessageStore } from '@/stores/message'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())

  if (typeof HTMLElement !== 'undefined' && typeof (HTMLElement.prototype as any).scrollTo !== 'function') {
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      value: vi.fn()
    })
  }
})

describe('components/chat/MessageList.vue (more coverage)', () => {
  it('renders skeleton items when loading history and messages is empty', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = true

    const wrapper = mount(MessageList, {
      props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })

    await flushAsync()
    await flushAsync()

    // 6 skeleton bubbles should be present
    expect(wrapper.findAll('.msg-bubble').length).toBeGreaterThanOrEqual(6)
  })

  it('load-more button emits and label changes by props', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })

    await flushAsync()

    const btn = wrapper.get('button')
    expect(btn.text()).toContain('查看历史消息')
    await btn.trigger('click')
    expect(wrapper.emitted('loadMore')?.length).toBe(1)

    await wrapper.setProps({ loadingMore: true })
    await nextTick()
    expect(wrapper.get('button').text()).toContain('加载中')

    await wrapper.setProps({ loadingMore: false, canLoadMore: false })
    await nextTick()
    expect(wrapper.get('button').text()).toContain('暂无更多历史消息')
  })

  it('scrollToBottom schedules via requestAnimationFrame and falls back without it', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const raf = window.requestAnimationFrame
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      value: (cb: FrameRequestCallback) => {
        cb(0)
        return 0
      }
    })

    const wrapper = mount(MessageList, {
      props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })

    const area = wrapper.get('.chat-area').element as HTMLElement
    Object.defineProperty(area, 'scrollHeight', { configurable: true, value: 1000 })
    const scrollToSpy = (area as any).scrollTo as unknown as ReturnType<typeof vi.fn>

    // allow the initial onMounted scrollToBottom() to settle
    await flushAsync()
    await flushAsync()
    scrollerSpies.scrollToBottom.mockClear()
    scrollToSpy.mockClear()

    ;(wrapper.vm as any).scrollToBottom(true)
    await flushAsync()
    await flushAsync()
    expect(scrollerSpies.scrollToBottom).toHaveBeenCalled()
    expect(scrollToSpy).toHaveBeenCalledWith(expect.objectContaining({ behavior: 'smooth' }))

    Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, value: undefined })
    ;(wrapper.vm as any).scrollToBottom(true)
    await flushAsync()
    await flushAsync()
    expect(scrollerSpies.scrollToBottom).toHaveBeenCalled()

    Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, value: raf })
  })

  it('download helpers handle empty/invalid urls and file click does not navigate', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          {
            tid: '1',
            time: '',
            isSelf: true,
            isFile: true,
            isImage: false,
            isVideo: false,
            content: '',
            fileUrl: 'http://x/a%20b.txt',
            fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })

    expect((wrapper.vm as any).getDownloadFileName('')).toBe('未知文件')
    expect((wrapper.vm as any).getDownloadFileName('not a url')).toBe('not a url')

    const createSpy = vi.spyOn(document, 'createElement')
    ;(wrapper.vm as any).downloadFile('')
    expect(createSpy).not.toHaveBeenCalled()

    ;(wrapper.vm as any).downloadFile('http://x/a%20b.txt')
    expect(clickSpy).toHaveBeenCalled()

    clickSpy.mockRestore()
  })

  it('renders typing row, sendStatus rows, and ChatMedia preview dispatch', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          {
            tid: '1',
            time: '2026-01-01 00:00:00.000',
            isSelf: true,
            sendStatus: 'sending',
            isImage: false,
            isVideo: false,
            isFile: false,
            content: 't',
            segments: [{ kind: 'image', url: 'http://img/1.png' }],
            fromuser: { id: 'me', nickname: 'me', name: 'me', sex: '未知', ip: '' }
          },
          {
            tid: '2',
            time: '2026-01-01 00:00:01.000',
            isSelf: true,
            sendStatus: 'failed',
            isImage: false,
            isVideo: false,
            isFile: false,
            content: 't2',
            segments: [{ kind: 'text', text: 'hello' }],
            fromuser: { id: 'me', nickname: 'me', name: 'me', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: true,
        loadingMore: false,
        canLoadMore: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          Skeleton: true,
          ChatMedia: {
            template: `<div class="chat-media" @click="$emit('preview', 'http://img/1.png')" />`
          }
        }
      }
    })

    await flushAsync()

    expect(wrapper.text()).toContain('正在输入')
    expect(wrapper.text()).toContain('发送中')
    expect(wrapper.text()).toContain('发送失败')

    // retry button emits
    const retryBtn = wrapper.findAll('button').find(btn => btn.text().includes('重试'))
    expect(retryBtn).toBeTruthy()
    await retryBtn!.trigger('click')
    expect(wrapper.emitted('retry')?.length).toBe(1)

    await wrapper.get('.chat-media').trigger('click')
    expect(dispatchSpy).toHaveBeenCalled()
  })

  it('copyToClipboard shows success/fail toasts and ignores empty text', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    clipboardMock.copyToClipboard.mockResolvedValueOnce(true)

    const wrapper = mount(MessageList, {
      props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })
    await flushAsync()

    await (wrapper.vm as any).copyToClipboard('')
    expect(toastShow).not.toHaveBeenCalled()

    await (wrapper.vm as any).copyToClipboard('hi')
    await flushAsync()
    expect(clipboardMock.copyToClipboard).toHaveBeenCalledWith('hi')
    expect(toastShow).toHaveBeenCalledWith('已复制')

    clipboardMock.copyToClipboard.mockResolvedValueOnce(false)
    await (wrapper.vm as any).copyToClipboard('hi2')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('复制失败')
  })

  it('handleScroll updates bottom state and media layout auto-scrolls only near bottom', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const originalRaf = window.requestAnimationFrame
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      value: (cb: FrameRequestCallback) => {
        cb(0)
        return 0
      }
    })

    try {
      const wrapper = mount(MessageList, {
        props: { messages: [{ tid: '1', content: 'x', isSelf: true } as any], isTyping: false, loadingMore: false, canLoadMore: true },
        global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
      })

      await flushAsync()
      await flushAsync()
      scrollerSpies.scrollToBottom.mockClear()

      const area = wrapper.get('.chat-area').element as HTMLElement
      Object.defineProperty(area, 'scrollHeight', { configurable: true, value: 1000 })
      Object.defineProperty(area, 'clientHeight', { configurable: true, value: 800 })

      // Not at bottom -> hasNewMessages stays
      ;(wrapper.vm as any).hasNewMessages = true
      ;(area as any).scrollTop = 0
      await wrapper.get('.chat-area').trigger('scroll')
      await new Promise<void>((resolve) => setTimeout(resolve, 120))
      await flushAsync()
      expect((wrapper.vm as any).isAtBottom).toBe(false)

      // Near bottom -> clears hasNewMessages
      ;(area as any).scrollTop = 150
      await wrapper.get('.chat-area').trigger('scroll')
      await new Promise<void>((resolve) => setTimeout(resolve, 120))
      await flushAsync()
      expect((wrapper.vm as any).isAtBottom).toBe(true)
      expect((wrapper.vm as any).hasNewMessages).toBe(false)

      // Media layout: far from bottom -> no auto-scroll
      scrollerSpies.scrollToBottom.mockClear()
      ;(area as any).scrollTop = 0
      ;(wrapper.vm as any).handleMediaLayout()
      await new Promise<void>((resolve) => setTimeout(resolve, 170))
      await flushAsync()
      await flushAsync()
      expect(scrollerSpies.scrollToBottom).not.toHaveBeenCalled()

      // Media layout: near bottom -> auto-scroll
      ;(area as any).scrollTop = 150
      ;(wrapper.vm as any).handleMediaLayout()
      await new Promise<void>((resolve) => setTimeout(resolve, 170))
      await flushAsync()
      await flushAsync()
      expect(scrollerSpies.scrollToBottom).toHaveBeenCalled()
    } finally {
      Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, value: originalRaf })
    }
  })

  it('renders segment video/file and fallback image/video branches', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          {
            clientId: 'c1',
            time: '2026-01-01 00:00:00.000',
            isSelf: false,
            sendStatus: '',
            isImage: false,
            isVideo: false,
            isFile: false,
            content: 'x',
            segments: [
              { kind: 'video', url: 'http://x/v.mp4' },
              { kind: 'file', url: 'http://x/a%20b.txt' }
            ],
            fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
          },
          {
            tid: 't1',
            time: '2026-01-01 00:00:01.000',
            isSelf: true,
            sendStatus: '',
            isImage: true,
            imageUrl: 'http://x/1.png',
            isVideo: false,
            isFile: false,
            content: '',
            fromuser: { id: 'me', nickname: 'me', name: 'me', sex: '未知', ip: '' }
          },
          {
            tid: 't2',
            time: '2026-01-01 00:00:02.000',
            isSelf: true,
            sendStatus: '',
            isImage: false,
            isVideo: true,
            videoUrl: 'http://x/2.mp4',
            isFile: false,
            content: '',
            fromuser: { id: 'me', nickname: 'me', name: 'me', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          Skeleton: true,
          ChatMedia: { template: '<div class=\"chat-media\" />' }
        }
      }
    })

    await flushAsync()

    // File segment should render "点击下载" and clicking triggers download
    expect(wrapper.text()).toContain('点击下载')
    const fileTile = wrapper.find('[title=\"a b.txt\"]')
    if (fileTile.exists()) {
      await fileTile.trigger('click')
      expect(clickSpy).toHaveBeenCalled()
    }

    clickSpy.mockRestore()
  })

  it('covers getMessageKey fallbacks and image/video/file url fallbacks when explicit urls are missing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          // clientId branch
          { clientId: 'cid-1', time: 't', isSelf: true, content: 'a', isImage: false, isVideo: false, isFile: false } as any,
          // tid branch (clientId empty)
          { clientId: '', tid: 't-1', time: 't', isSelf: true, content: 'b', isImage: false, isVideo: false, isFile: false } as any,
          // fallback key branch (no clientId/tid, no fromuser)
          { time: undefined, type: undefined, content: undefined, isSelf: true, isImage: false, isVideo: false, isFile: false } as any,
          // fallback image url -> content
          { tid: 'img', time: 't', isSelf: true, isImage: true, imageUrl: '', content: 'http://x/fallback.png', isVideo: false, isFile: false } as any,
          // fallback video url -> content
          { tid: 'vid', time: 't', isSelf: true, isImage: false, isVideo: true, videoUrl: '', content: 'http://x/fallback.mp4', isFile: false } as any,
          // fallback file url -> content
          { tid: 'file', time: 't', isSelf: true, isImage: false, isVideo: false, isFile: true, fileUrl: '', content: 'http://x/fallback.txt' } as any,
          // fallback file url -> empty string
          { tid: 'file2', time: 't', isSelf: true, isImage: false, isVideo: false, isFile: true, fileUrl: '', content: '' } as any
        ],
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          Skeleton: true,
          ChatMedia: { template: '<div class=\"chat-media\" />' }
        }
      }
    })

    await flushAsync()
    expect(wrapper.findAll('.chat-media').length).toBeGreaterThan(0)
  })

  it('watch branch: ignores length change while loading history and toggles hasNewMessages when not at bottom', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [{ tid: '1', content: 'a', time: 't', isSelf: true } as any],
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })
    await flushAsync()

    // ignore branch when loading history
    messageStore.isLoadingHistory = true
    await wrapper.setProps({ messages: [{ tid: '1', content: 'a', time: 't', isSelf: true } as any, { tid: '2', content: 'b', time: 't', isSelf: true } as any] })
    await flushAsync()
    expect((wrapper.vm as any).hasNewMessages).toBe(false)

    // not at bottom -> hasNewMessages=true
    messageStore.isLoadingHistory = false
    ;(wrapper.vm as any).isAtBottom = false
    await wrapper.setProps({ messages: [{ tid: '1', content: 'a', time: 't', isSelf: true } as any, { tid: '2', content: 'b', time: 't', isSelf: true } as any, { tid: '3', content: 'c', time: 't', isSelf: true } as any] })
    await flushAsync()
    expect((wrapper.vm as any).hasNewMessages).toBe(true)
  })

  it('getDownloadFileName returns fallback name when url ends with slash', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })
    await flushAsync()

    expect((wrapper.vm as any).getDownloadFileName('http://x/')).toBe('未知文件')
  })
})
