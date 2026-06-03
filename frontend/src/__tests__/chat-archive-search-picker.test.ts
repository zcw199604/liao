import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('@/api/chat', () => ({
  searchChatArchive: vi.fn()
}))

import ChatArchiveSearchPicker from '@/components/chat/ChatArchiveSearchPicker.vue'
import * as chatApi from '@/api/chat'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/chat/ChatArchiveSearchPicker.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('searches archive by keyword and emits selected result with owner identity', async () => {
    vi.mocked(chatApi.searchChatArchive).mockResolvedValue({
      code: 0,
      msg: 'success',
      data: {
        items: [
          {
            ownerUserId: 'owner-a',
            targetUserId: 'target-1',
            nickname: 'Target One',
            sources: ['archive', 'history'],
            localArchived: true,
            lastMsg: 'hello'
          }
        ]
      }
    } as any)

    const wrapper = mount(ChatArchiveSearchPicker, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.find('input[placeholder="搜索归档用户 ID 或名称"]').setValue('target')
    await wrapper.find('input[placeholder="搜索归档用户 ID 或名称"]').trigger('keyup.enter')
    await flushAsync()

    expect(chatApi.searchChatArchive).toHaveBeenCalledWith({ q: 'target', limit: 100 })
    expect(wrapper.text()).toContain('Target One')
    expect(wrapper.text()).toContain('owner-a')
    expect(wrapper.text()).toContain('归档')
    expect(wrapper.text()).toContain('历史')

    await wrapper.findAll('button').find(button => button.text().includes('Target One'))?.trigger('click')
    const selected = wrapper.emitted('select')?.[0]
    expect(selected?.[0]).toMatchObject({ ownerUserId: 'owner-a', targetUserId: 'target-1' })
    expect(wrapper.emitted('update:visible')?.at(-1)).toEqual([false])
  })

  it('clears search results and renders empty state', async () => {
    vi.mocked(chatApi.searchChatArchive).mockResolvedValue({
      code: 0,
      msg: 'success',
      data: { items: [] }
    } as any)

    const wrapper = mount(ChatArchiveSearchPicker, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    const input = wrapper.find('input[placeholder="搜索归档用户 ID 或名称"]')
    await input.setValue('missing')
    await input.trigger('keyup.enter')
    await flushAsync()

    expect(wrapper.text()).toContain('未找到归档用户')

    await wrapper.find('button[aria-label="清空搜索"]').trigger('click')
    expect(wrapper.text()).toContain('输入用户 ID 或名称开始搜索')
  })
})
