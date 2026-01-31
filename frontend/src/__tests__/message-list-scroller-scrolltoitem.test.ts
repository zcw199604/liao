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
  scrollToItem: vi.fn()
}))

// This mock intentionally omits scrollToBottom so MessageList falls back to scrollToItem.
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

describe('components/chat/MessageList.vue (scrollToItem fallback)', () => {
  it('uses DynamicScroller.scrollToItem when scrollToBottom is unavailable', async () => {
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

    try {
      const wrapper = mount(MessageList, {
        props: {
          messages: [{ tid: 't1', content: 'x', time: '', isSelf: true } as any],
          isTyping: false,
          loadingMore: false,
          canLoadMore: true
        },
        global: { plugins: [pinia], stubs: { Skeleton: true, ChatMedia: true } }
      })

      await flushAsync()
      await flushAsync()
      scrollerSpies.scrollToItem.mockClear()

      ;(wrapper.vm as any).scrollToBottom(true)
      await flushAsync()
      await flushAsync()

      // renderItems = [loadMore, message]
      expect(scrollerSpies.scrollToItem).toHaveBeenCalledWith(1)
    } finally {
      Object.defineProperty(window, 'requestAnimationFrame', { configurable: true, value: raf })
    }
  })
})

