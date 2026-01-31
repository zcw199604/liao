import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: vi.fn()
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

describe('components/chat/MessageList.vue (branch gaps)', () => {
  it('covers scrollToBottom fallback when scroller has no scrollToBottom/scrollToItem and covers getScrollerEl null early returns', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const raf = window.requestAnimationFrame
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      writable: true,
      value: (cb: FrameRequestCallback) => {
        cb(0)
        return 0
      }
    })

    try {
      const wrapper = mount(MessageList, {
        props: { messages: [{ tid: 't1', content: 'x', time: '', isSelf: true } as any], isTyping: false, loadingMore: false, canLoadMore: true },
        global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
      })

      await flushAsync()
      await flushAsync()
      scrollerSpies.scrollToBottom.mockClear()
      scrollerSpies.scrollToItem.mockClear()

      const area = wrapper.get('.chat-area').element as HTMLElement
      Object.defineProperty(area, 'scrollHeight', { configurable: true, value: 1000 })
      const scrollToSpy = (area as any).scrollTo as unknown as ReturnType<typeof vi.fn>
      scrollToSpy.mockClear()

      // Replace scrollerRef with a minimal object that has $el but no scrollToBottom/scrollToItem.
      ;(wrapper.vm as any).scrollerRef = { $el: area }
      ;(wrapper.vm as any).scrollToBottom(true)
      await flushAsync()
      await flushAsync()
      expect(scrollerSpies.scrollToBottom).not.toHaveBeenCalled()
      expect(scrollerSpies.scrollToItem).not.toHaveBeenCalled()
      expect(scrollToSpy).toHaveBeenCalled()

      // Force getScrollerEl() to return null, then both handlers should early-return safely.
      vi.useFakeTimers()
      ;(wrapper.vm as any).scrollerRef = { $el: undefined }
      ;(wrapper.vm as any).handleScroll()
      ;(wrapper.vm as any).handleMediaLayout()
      vi.runAllTimers()
      vi.useRealTimers()
      wrapper.unmount()
    } finally {
      Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, writable: true, value: raf })
    }
  })

  it('covers watch branches (append at bottom triggers scroll; shrink does nothing) and scrollToTop both paths', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const raf = window.requestAnimationFrame
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      writable: true,
      value: (cb: FrameRequestCallback) => {
        cb(0)
        return 0
      }
    })

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          { tid: '1', content: 'a', time: 't', isSelf: true } as any,
          { tid: '2', content: 'b', time: 't', isSelf: true } as any
        ],
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
    })

    await flushAsync()
    await flushAsync()
    scrollerSpies.scrollToBottom.mockClear()
    scrollerSpies.scrollToItem.mockClear()

    // scrollToTop uses scrollToItem when available.
    ;(wrapper.vm as any).scrollToTop()
    await flushAsync()
    expect(scrollerSpies.scrollToItem).toHaveBeenCalledWith(0)

    // Watch: append while at bottom -> scrollToBottom.
    ;(wrapper.vm as any).isAtBottom = true
    await wrapper.setProps({
      messages: [
        { tid: '1', content: 'a', time: 't', isSelf: true } as any,
        { tid: '2', content: 'b', time: 't', isSelf: true } as any,
        { tid: '3', content: 'c', time: 't', isSelf: true } as any
      ]
    })
    await flushAsync()
    await flushAsync()
    expect(scrollerSpies.scrollToBottom).toHaveBeenCalled()

    // Watch: shrink -> no scroll/no new badge changes.
    scrollerSpies.scrollToBottom.mockClear()
    await wrapper.setProps({ messages: [{ tid: '1', content: 'a', time: 't', isSelf: true } as any] })
    await flushAsync()
    await flushAsync()
    expect(scrollerSpies.scrollToBottom).not.toHaveBeenCalled()

    // scrollToTop falls back to element scrollTop when scrollToItem is unavailable.
    const area = wrapper.get('.chat-area').element as HTMLElement
    ;(area as any).scrollTop = 123
    ;(wrapper.vm as any).scrollerRef = { $el: area } // no scrollToItem
    ;(wrapper.vm as any).scrollToTop()
    await flushAsync()
    expect((area as any).scrollTop).toBe(0)

    Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, writable: true, value: raf })
  })

  it('covers template fallbacks for image/video src empty and file click branches (fileUrl/content/empty)', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          // image src fallback -> ''
          { tid: 'img-empty', time: 't', isSelf: true, isImage: true, isVideo: false, isFile: false, imageUrl: '', content: '' } as any,
          // video src fallback -> ''
          { tid: 'vid-empty', time: 't', isSelf: true, isImage: false, isVideo: true, isFile: false, videoUrl: '', content: '' } as any,
          // file click: fileUrl branch
          { tid: 'f1', time: 't', isSelf: true, isImage: false, isVideo: false, isFile: true, fileUrl: 'http://x/a.txt', content: '' } as any,
          // file click: content branch
          { tid: 'f2', time: 't', isSelf: true, isImage: false, isVideo: false, isFile: true, fileUrl: '', content: 'http://x/b.txt' } as any,
          // file click: empty branch (downloadFile early return)
          { tid: 'f3', time: 't', isSelf: true, isImage: false, isVideo: false, isFile: true, fileUrl: '', content: '' } as any
        ],
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: { template: '<div class="chat-media" />' } } }
    })

    await flushAsync()

    // Click file titles to trigger the click handler (downloadFile(getMediaUrl(fileUrl||content||''))).
    await wrapper.get('[title="a.txt"]').trigger('click')
    await wrapper.get('[title="b.txt"]').trigger('click')
    await wrapper.get('[title="未知文件"]').trigger('click')

    // Only the non-empty URLs should attempt a download.
    expect(clickSpy).toHaveBeenCalledTimes(2)

    // getDownloadFileName catch fallback (raw.split('/').pop() || '未知文件')
    expect((wrapper.vm as any).getDownloadFileName('not-a-url/')).toBe('未知文件')

    clickSpy.mockRestore()
  })
})
