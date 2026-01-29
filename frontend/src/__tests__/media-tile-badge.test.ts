import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import MediaTileBadge from '@/components/common/MediaTileBadge.vue'

describe('components/common/MediaTileBadge.vue', () => {
  it('renders slot content, forwards attrs, and toggles pointer events by interactive', () => {
    const wrapper = mount(MediaTileBadge, {
      props: { interactive: false },
      attrs: { 'data-x': '1' },
      slots: { default: 'Hello' }
    })

    expect(wrapper.text()).toBe('Hello')
    expect(wrapper.attributes('data-x')).toBe('1')
    expect(wrapper.classes()).toContain('pointer-events-none')

    const wrapper2 = mount(MediaTileBadge, {
      props: { interactive: true },
      slots: { default: 'Hi' }
    })
    expect(wrapper2.classes()).toContain('pointer-events-auto')
  })

  it('applies variant classes for all variants', () => {
    const variants = [
      { variant: 'neutral' as const, className: 'bg-black/50' },
      { variant: 'success' as const, className: 'bg-emerald-600/80' },
      { variant: 'info' as const, className: 'bg-indigo-600/80' },
      { variant: 'warn' as const, className: 'bg-amber-600/80' },
      { variant: 'danger' as const, className: 'bg-red-600/80' }
    ]

    for (const v of variants) {
      const wrapper = mount(MediaTileBadge, {
        props: { variant: v.variant },
        slots: { default: 'x' }
      })
      expect(wrapper.classes()).toContain(v.className)
    }
  })
})

