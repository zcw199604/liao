import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import ChatDayBulkDeleteModal from '@/components/chat/ChatDayBulkDeleteModal.vue'

describe('components/chat/ChatDayBulkDeleteModal.vue', () => {
  it('does not render when visible=false', () => {
    const wrapper = mount(ChatDayBulkDeleteModal, {
      props: {
        visible: false,
        items: []
      }
    })

    expect(wrapper.find('.fixed.inset-0').exists()).toBe(false)
  })

  it('shows empty state and emits close from multiple buttons', async () => {
    const wrapper = mount(ChatDayBulkDeleteModal, {
      props: {
        visible: true,
        items: []
      }
    })

    expect(wrapper.text()).toContain('暂无可选择的日期')
    expect(wrapper.get('button[aria-label=\"close\"]').exists()).toBe(true)

    await wrapper.get('button[aria-label=\"close\"]').trigger('click')
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])

    // Click backdrop also closes (first emitted already, so check last)
    await wrapper.setProps({ visible: true })
    await nextTick()
    await wrapper.get('.fixed.inset-0').trigger('click')
    const updates = wrapper.emitted('update:visible') || []
    expect(updates[updates.length - 1]).toEqual([false])
  })

  it('toggles selection, toggleAll, and confirm emits keys', async () => {
    const wrapper = mount(ChatDayBulkDeleteModal, {
      props: {
        visible: true,
        items: [
          { key: '2026-01-01', label: '2026-01-01', count: 2 },
          { key: '2026-01-02', label: '2026-01-02', count: 0 }
        ]
      }
    })

    // Confirm is disabled when nothing selected.
    expect(wrapper.get('button:disabled').text()).toContain('选中会话')

    // Select one item, then toggle again to remove selection branch.
    const itemButtons = wrapper.findAll('button').filter(b => b.text().includes('个会话'))
    await itemButtons[0]!.trigger('click')
    await nextTick()
    await itemButtons[0]!.trigger('click')
    await nextTick()
    expect(wrapper.findAll('button').find(b => b.text().includes('选中会话'))?.attributes('disabled')).toBeDefined()

    await itemButtons[0]!.trigger('click')
    await nextTick()

    // Toggle all -> selects all.
    const toggleAllBtn = wrapper.findAll('button').find(b => b.text().includes('全选') || b.text().includes('取消全选'))
    expect(toggleAllBtn).toBeTruthy()
    await toggleAllBtn!.trigger('click')
    await nextTick()
    expect(toggleAllBtn!.text()).toContain('取消全选')

    // Confirm emits confirm + closes.
    const confirmBtn = wrapper.findAll('button').find(b => b.text().includes('选中会话'))
    expect(confirmBtn).toBeTruthy()
    await confirmBtn!.trigger('click')

    expect(wrapper.emitted('confirm')?.[0]).toEqual([['2026-01-01', '2026-01-02']])
    expect(wrapper.emitted('update:visible')?.at(-1)).toEqual([false])
  })

  it('applies preselectKey when reopening', async () => {
    const wrapper = mount(ChatDayBulkDeleteModal, {
      props: {
        visible: false,
        items: [
          { key: 'unknown', label: '未知时间', count: 1 },
          { key: '2026-01-01', label: '2026-01-01', count: 2 }
        ],
        preselectKey: '2026-01-01'
      }
    })

    // Re-open triggers watch(visible) and applies preselectKey.
    await wrapper.setProps({ visible: true })
    await nextTick()

    const confirmBtn = wrapper.findAll('button').find(b => b.text().includes('选中会话'))
    expect(confirmBtn).toBeTruthy()
    expect(confirmBtn!.attributes('disabled')).toBeUndefined()
    expect(confirmBtn!.text()).toContain('(2)')

    // Hide clears; next open without preselectKey resets selection.
    await wrapper.setProps({ visible: false })
    await nextTick()
    await wrapper.setProps({ visible: true, preselectKey: undefined })
    await nextTick()

    expect(wrapper.findAll('button').find(b => b.text().includes('选中会话'))?.attributes('disabled')).toBeDefined()
  })

  it('toggleAll clears selection when already fully selected and confirm returns early when empty', async () => {
    const wrapper = mount(ChatDayBulkDeleteModal, {
      props: {
        visible: true,
        items: [
          { key: 'a', label: 'a', count: 1 },
          { key: 'b', label: 'b', count: 1 }
        ]
      }
    })

    const vm = wrapper.vm as any
    await vm.handleConfirm()
    expect(wrapper.emitted('confirm')).toBeUndefined()

    const toggleAllBtn = wrapper.findAll('button').find(b => b.text().includes('全选') || b.text().includes('取消全选'))
    await toggleAllBtn!.trigger('click')
    await nextTick()
    await toggleAllBtn!.trigger('click')
    await nextTick()
    expect(wrapper.findAll('button').find(b => b.text().includes('选中会话'))?.attributes('disabled')).toBeDefined()
  })
})
