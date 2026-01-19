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
})
