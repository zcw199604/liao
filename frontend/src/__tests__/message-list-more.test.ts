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
})
