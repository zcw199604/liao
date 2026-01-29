import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DraggableBadge from '@/components/common/DraggableBadge.vue'

describe('components/common/DraggableBadge.vue', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('bounces back when drag distance is below threshold, and ignores startDrag while animating', async () => {
    vi.useFakeTimers()

    const wrapper = mount(DraggableBadge, { props: { count: 3 } })
    const root = wrapper.element as HTMLElement

    // idle transition (not dragging / not animating)
    expect(root.style.transition).toContain('transform 0.3s')

    const addSpy = vi.spyOn(window, 'addEventListener')

    await wrapper.trigger('mousedown', { clientX: 0, clientY: 0 })
    expect(root.style.transition).toBe('none')

    const onDragHandler = addSpy.mock.calls.find(([type]) => type === 'mousemove')?.[1] as any

    // Cover onDrag early return branch: call the registered handler while isDragging=false.
    ;(wrapper.vm as any).isDragging = false
    onDragHandler?.(new MouseEvent('mousemove', { clientX: 1, clientY: 1 }))
    ;(wrapper.vm as any).isDragging = true

    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 10, clientY: 0, cancelable: true }))
    await wrapper.vm.$nextTick()

    window.dispatchEvent(new MouseEvent('mouseup'))
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('clear')).toBeUndefined()
    expect(root.style.transform).toContain('translate(0px, 0px)')
    expect(root.style.transition).toContain('cubic-bezier')

    // While animating, startDrag is a no-op.
    await wrapper.trigger('mousedown', { clientX: 0, clientY: 0 })
    expect(root.style.transition).toContain('cubic-bezier')

    vi.advanceTimersByTime(300)
    await wrapper.vm.$nextTick()
    expect(root.style.transition).toContain('transform 0.3s')

    wrapper.unmount()
  })

  it('emits clear when dragged beyond threshold', async () => {
    const wrapper = mount(DraggableBadge, { props: { count: 1 } })

    await wrapper.trigger('mousedown', { clientX: 0, clientY: 0 })
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 120, clientY: 0 }))
    window.dispatchEvent(new MouseEvent('mouseup'))

    expect(wrapper.emitted('clear')?.length).toBe(1)
    wrapper.unmount()
  })

  it('supports touch events and respects cancelable/non-cancelable move events', async () => {
    const wrapper1 = mount(DraggableBadge, { props: { count: 9 } })
    const addSpy = vi.spyOn(window, 'addEventListener')

    // touchstart without touches is ignored
    await wrapper1.trigger('touchstart', { touches: [] })
    // early-return happens before registering window listeners
    expect(addSpy.mock.calls.some(([type]) => type === 'touchmove')).toBe(false)
    wrapper1.unmount()

    const wrapper = mount(DraggableBadge, { props: { count: 9 } })
    // touchstart with touches begins dragging
    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    expect(wrapper.find('div.absolute').exists()).toBe(true)

    const moveCancelable: any = new Event('touchmove', { cancelable: true })
    moveCancelable.touches = [{ clientX: 100, clientY: 0 }]
    moveCancelable.preventDefault = vi.fn()
    window.dispatchEvent(moveCancelable)
    expect(moveCancelable.preventDefault).toHaveBeenCalled()

    const moveNotCancelable: any = new Event('touchmove', { cancelable: false })
    moveNotCancelable.touches = [{ clientX: 101, clientY: 0 }]
    moveNotCancelable.preventDefault = vi.fn()
    window.dispatchEvent(moveNotCancelable)
    expect(moveNotCancelable.preventDefault).not.toHaveBeenCalled()

    window.dispatchEvent(new Event('touchend'))
    expect(wrapper.emitted('clear')?.length).toBe(1)
    wrapper.unmount()
  })
})
