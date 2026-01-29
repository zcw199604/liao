import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

import { SWIPE_RESET_DURATION_MS } from '@/constants/interaction'

let capturedSwipeOptions: any = null

vi.mock('@/composables/useInteraction', async () => {
  const { ref } = await import('vue')
  return {
    useSwipeAction: (_target: unknown, options: unknown) => {
      capturedSwipeOptions = options
      return { isSwiping: ref(false) }
    }
  }
})

import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import { useChatStore } from '@/stores/chat'

describe('components/chat/ChatSidebar.vue - swipe reset', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setActivePinia(createPinia())
    capturedSwipeOptions = null
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('resets translateX on swipe finish and clears transition after duration', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const wrapper = shallowMount(ChatSidebar, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' }
        }
      }
    })

    await nextTick()
    expect(capturedSwipeOptions).toBeTruthy()

    const listEl = wrapper.find('div.overflow-y-auto.no-scrollbar')
    expect(listEl.exists()).toBe(true)
    const listDom = listEl.element as unknown as HTMLElement

    ;(capturedSwipeOptions as any).onSwipeProgress?.(100, 0)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(50')

    ;(capturedSwipeOptions as any).onSwipeFinish?.(100, 0, false)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(0')
    expect(listDom.style.transition).toContain(`${SWIPE_RESET_DURATION_MS}ms`)

    vi.advanceTimersByTime(SWIPE_RESET_DURATION_MS)
    await nextTick()
    expect(listDom.style.transition).toBe('none')
  })

  it('snaps back to 0 without animation when context menu is open', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const wrapper = shallowMount(ChatSidebar, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' }
        }
      }
    })

    await nextTick()
    expect(capturedSwipeOptions).toBeTruthy()

    const listEl = wrapper.find('div.overflow-y-auto.no-scrollbar')
    expect(listEl.exists()).toBe(true)
    const listDom = listEl.element as unknown as HTMLElement

    ;(capturedSwipeOptions as any).onSwipeProgress?.(100, 0)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(50')

    ;(wrapper.vm as any).showContextMenu = true
    await nextTick()

    ;(capturedSwipeOptions as any).onSwipeFinish?.(100, 0, false)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(0')
    expect(listDom.style.transition).toBe('none')
  })

  it('onSwipeEnd switches tabs and closes context menu when open', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const wrapper = shallowMount(ChatSidebar, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' }
        }
      }
    })

    await nextTick()
    expect(capturedSwipeOptions).toBeTruthy()

    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    ;(capturedSwipeOptions as any).onSwipeEnd?.('left')
    await nextTick()
    expect(chatStore.activeTab).toBe('favorite')

    ;(capturedSwipeOptions as any).onSwipeEnd?.('right')
    await nextTick()
    expect(chatStore.activeTab).toBe('history')

    // context menu open -> closes and does not switch tab
    chatStore.activeTab = 'history'
    ;(wrapper.vm as any).showContextMenu = true
    ;(wrapper.vm as any).contextMenuUser = { id: 'u1' }
    await nextTick()
    ;(capturedSwipeOptions as any).onSwipeEnd?.('left')
    await nextTick()
    expect((wrapper.vm as any).showContextMenu).toBe(false)
    expect((wrapper.vm as any).contextMenuUser).toBe(null)
    expect(chatStore.activeTab).toBe('history')
  })

  it('onSwipeProgress ignores vertical swipes and clamps overscroll', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const wrapper = shallowMount(ChatSidebar, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' }
        }
      }
    })

    await nextTick()
    expect(capturedSwipeOptions).toBeTruthy()

    const listEl = wrapper.find('div.overflow-y-auto.no-scrollbar')
    expect(listEl.exists()).toBe(true)
    const listDom = listEl.element as unknown as HTMLElement

    // vertical dominated -> ignore horizontal
    ;(capturedSwipeOptions as any).onSwipeProgress?.(10, 100)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(0')

    // positive overscroll clamps with dampening
    ;(capturedSwipeOptions as any).onSwipeProgress?.(1000, 0)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(196')

    // negative overscroll clamps with dampening
    ;(capturedSwipeOptions as any).onSwipeProgress?.(-1000, 0)
    await nextTick()
    expect(listDom.style.transform).toContain('translateX(-196')

    // context menu open -> ignore progress
    ;(wrapper.vm as any).showContextMenu = true
    ;(wrapper.vm as any).listTranslateX = 12
    await nextTick()
    ;(capturedSwipeOptions as any).onSwipeProgress?.(100, 0)
    await nextTick()
    expect((wrapper.vm as any).listTranslateX).toBe(12)
  })
})
