import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import PullToRefresh from '@/components/common/PullToRefresh.vue'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/common/PullToRefresh.vue', () => {
  it('handles pull gestures, reaching threshold triggers refresh, and resets after completion', async () => {
    vi.useFakeTimers()
    try {
      const onRefresh = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(PullToRefresh, {
        props: { onRefresh },
        slots: {
          default: '<div class="overflow-y-auto" style="overflow-y:auto; height:100px;"></div>'
        }
      })
      await flushAsync()

      const scrollEl = wrapper.get('.overflow-y-auto').element as HTMLElement
      scrollEl.scrollTop = 0

      Object.defineProperty(navigator, 'vibrate', { configurable: true, value: vi.fn() })

      const vm = wrapper.vm as any

      // Horizontal movement cancels pull
      vm.handleTouchStart({ touches: [{ clientX: 0, clientY: 0 }] } as any)
      const preventDefault = vi.fn()
      vm.handleTouchMove({ touches: [{ clientX: 100, clientY: 1 }], cancelable: true, preventDefault } as any)
      expect(vm.currentPull).toBe(0)
      expect(vm.status).toBe('idle')

      // Vertical pull below threshold -> pulling state, preventDefault called
      vm.handleTouchStart({ touches: [{ clientX: 0, clientY: 0 }] } as any)
      vm.handleTouchMove({ touches: [{ clientX: 0, clientY: 50 }], cancelable: true, preventDefault } as any)
      expect(preventDefault).toHaveBeenCalled()
      expect(vm.status).toBe('pulling')
      expect(vm.currentPull).toBeGreaterThan(0)

      // Pull beyond threshold -> reaching state
      vm.handleTouchMove({ touches: [{ clientX: 0, clientY: 200 }], cancelable: true, preventDefault } as any)
      expect(vm.status).toBe('reaching')

      // End gesture triggers refresh flow
      await vm.handleTouchEnd()
      expect(vm.status).toBe('refreshing')
      expect(onRefresh).toHaveBeenCalled()

      // After the internal delay, resets to idle
      vi.advanceTimersByTime(300)
      await flushAsync()
      expect(vm.status).toBe('idle')
      expect(vm.currentPull).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers early return branches: refreshing state, missing scroll container, and scrollTop>0 cancellation', async () => {
    vi.useFakeTimers()
    try {
      const onRefresh = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(PullToRefresh, {
        props: { onRefresh },
        slots: {
          default: '<div class="overflow-y-auto" style="overflow-y:auto; height:100px;"></div>'
        }
      })
      await flushAsync()

      const scrollEl = wrapper.get('.overflow-y-auto').element as HTMLElement
      scrollEl.scrollTop = 0

      const vm = wrapper.vm as any

      // status=refreshing blocks touchstart/move
      vm.status = 'refreshing'
      vm.handleTouchStart({ touches: [{ clientX: 0, clientY: 0 }] } as any)
      vm.handleTouchMove({ touches: [{ clientX: 0, clientY: 20 }], cancelable: true, preventDefault: vi.fn() } as any)
      expect(vm.currentPull).toBe(0)

      // Missing scroll container -> start does nothing (container null branch)
      vm.status = 'idle'
      vm.scrollContainer = null
      vm.handleTouchStart({ touches: [{ clientX: 0, clientY: 0 }] } as any)

      // scrollTop>0 during move cancels pull and resets state
      vm.scrollContainer = scrollEl
      vm.handleTouchStart({ touches: [{ clientX: 0, clientY: 0 }] } as any)
      scrollEl.scrollTop = 10
      vm.currentPull = 20
      vm.status = 'pulling'
      vm.handleTouchMove({ touches: [{ clientX: 0, clientY: 50 }], cancelable: true, preventDefault: vi.fn() } as any)
      expect(vm.currentPull).toBe(0)
      expect(vm.status).toBe('idle')

      // handleTouchEnd on non-reaching state resets without calling refresh
      vm.status = 'pulling'
      vm.currentPull = 10
      await vm.handleTouchEnd()
      expect(vm.status).toBe('idle')
      expect(vm.currentPull).toBe(0)
      expect(onRefresh).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })
})

