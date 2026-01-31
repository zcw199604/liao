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

vi.mock('@/utils/clipboard', () => ({
  copyToClipboard: vi.fn()
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

describe('components/chat/MessageList.vue (template branch gaps)', () => {
  it('covers segment/file self branches, fallback file url selection, and floating-button variants', async () => {
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
            sendStatus: '',
            isImage: false,
            isVideo: false,
            isFile: false,
            content: 'x',
            segments: [{ kind: 'file', url: 'http://x/self.txt' }],
            fromuser: { id: 'me', nickname: '', name: 'me', sex: '未知', ip: '' }
          },
          {
            tid: '2',
            time: '',
            isSelf: false,
            sendStatus: '',
            isImage: false,
            isVideo: false,
            isFile: true,
            fileUrl: '',
            content: 'http://x/peer.txt',
            fromuser: { id: 'u1', nickname: '', name: 'u1', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: false,
        loadingMore: false,
        canLoadMore: true,
        floatingBottomOffsetPx: 40
      },
      global: {
        plugins: [pinia],
        stubs: {
          Skeleton: true,
          ChatMedia: {
            props: ['type', 'src'],
            template: `<div class="chat-media" :data-type="type" :data-src="src"></div>`
          }
        }
      }
    })

    await flushAsync()

    expect(wrapper.text()).toContain('点击下载')

    // Cover floating button UI: not at bottom -> "回到底部" variant.
    ;(wrapper.vm as any).isAtBottom = false
    ;(wrapper.vm as any).hasNewMessages = false
    await flushAsync()
    const btn1 = wrapper.find('button[title=\"回到底部\"]')
    expect(btn1.exists()).toBe(true)

    // hasNewMessages -> "新消息" variant.
    ;(wrapper.vm as any).hasNewMessages = true
    await flushAsync()
    expect(wrapper.find('button[title=\"有新消息\"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('新消息')
  })

  it('scheduleScrollToBottom guards duplicate scheduling when called twice before raf tick', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const originalRaf = window.requestAnimationFrame
    let rafCb: FrameRequestCallback | null = null
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      value: (cb: FrameRequestCallback) => {
        rafCb = cb
        return 0
      }
    })

    try {
      const wrapper = mount(MessageList, {
        props: { messages: [] as any, isTyping: false, loadingMore: false, canLoadMore: true },
        global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
      })

      await flushAsync()
      scrollerSpies.scrollToBottom.mockClear()

      ;(wrapper.vm as any).scrollToBottom(true)
      ;(wrapper.vm as any).scrollToBottom(true)

      // Only one scheduled run should execute.
      rafCb?.(0)
      await flushAsync()
      await flushAsync()
      expect(scrollerSpies.scrollToBottom).toHaveBeenCalledTimes(1)
    } finally {
      Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, value: originalRaf })
    }
  })
})
