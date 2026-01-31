import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import ChatInput from '@/components/chat/ChatInput.vue'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/chat/ChatInput.vue', () => {
  it('emits typingStart/typingEnd with debounce on input and clears timer on blur', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(ChatInput, {
        props: { modelValue: '', disabled: false, wsConnected: true }
      })

      const textarea = wrapper.get('textarea')
      Object.defineProperty(textarea.element, 'scrollHeight', { configurable: true, get: () => 42 })

      ;(textarea.element as HTMLTextAreaElement).value = 'hi'
      await textarea.trigger('input')
      await flushAsync()

      expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe('hi')
      expect(wrapper.emitted('typingStart')).toHaveLength(1)
      expect(wrapper.emitted('typingEnd')).toBeUndefined()

      // Second input resets the timer and does not re-emit typingStart.
      vi.advanceTimersByTime(1000)
      ;(textarea.element as HTMLTextAreaElement).value = 'hi2'
      await textarea.trigger('input')
      await flushAsync()
      expect(wrapper.emitted('typingStart')).toHaveLength(1)

      // Not yet 3s since last input -> no typingEnd
      vi.advanceTimersByTime(2999)
      expect(wrapper.emitted('typingEnd')).toBeUndefined()

      vi.advanceTimersByTime(1)
      expect(wrapper.emitted('typingEnd')).toHaveLength(1)

      // Blur clears pending timer and emits typingEnd if needed.
      ;(textarea.element as HTMLTextAreaElement).value = 'x'
      await textarea.trigger('input')
      await flushAsync()
      expect(wrapper.emitted('typingStart')).toHaveLength(2)

      await textarea.trigger('blur')
      await flushAsync()
      expect(wrapper.emitted('typingEnd')!.length).toBeGreaterThanOrEqual(2)

      vi.advanceTimersByTime(4000)
      // no extra typingEnd from the cleared timer
      expect(wrapper.emitted('typingEnd')!.length).toBeGreaterThanOrEqual(2)
    } finally {
      vi.useRealTimers()
    }
  })

  it('handles focus typingStart, keydown send shortcuts, and button disabled states', async () => {
    const wrapper = mount(ChatInput, {
      props: { modelValue: '  ', disabled: false, wsConnected: false }
    })

    const textarea = wrapper.get('textarea')

    // focus triggers typingStart when not already typing
    await textarea.trigger('focus')
    await flushAsync()
    expect(wrapper.emitted('typingStart')).toHaveLength(1)

    // startMatch disabled when wsConnected is false
    const buttons = wrapper.findAll('button')
    const matchBtn = buttons.find(b => b.attributes('aria-label') === '匹配')
    expect(matchBtn?.attributes('disabled')).toBeDefined()

    // send disabled when modelValue is whitespace only
    const sendBtn = buttons.find(b => b.attributes('aria-label') === '发送')
    expect(sendBtn?.attributes('disabled')).toBeDefined()

    await wrapper.setProps({ modelValue: 'hello', wsConnected: true })
    await flushAsync()
    expect(matchBtn?.attributes('disabled')).toBeUndefined()
    expect(sendBtn?.attributes('disabled')).toBeUndefined()

    // Ctrl/Cmd+Enter sends when not disabled
    await textarea.trigger('keydown', { key: 'Enter', ctrlKey: true })
    await textarea.trigger('keydown', { key: 'Enter', metaKey: true })
    expect(wrapper.emitted('send')?.length).toBe(2)

    // Regular Enter sends (Shift+Enter should not)
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(wrapper.emitted('send')?.length).toBe(3)

    // disabled=true blocks send emissions
    await wrapper.setProps({ disabled: true })
    await flushAsync()
    await textarea.trigger('keydown', { key: 'Enter', ctrlKey: true })
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })
    expect(wrapper.emitted('send')?.length).toBe(3)
  })

  it('autoResize runs on modelValue change (watch)', async () => {
    const wrapper = mount(ChatInput, {
      props: { modelValue: '', disabled: false, wsConnected: true }
    })

    const textarea = wrapper.get('textarea')
    Object.defineProperty(textarea.element, 'scrollHeight', { configurable: true, get: () => 60 })

    await wrapper.setProps({ modelValue: 'hello' })
    await flushAsync()
    expect((textarea.element as HTMLTextAreaElement).style.height).toContain('60px')
  })
})

