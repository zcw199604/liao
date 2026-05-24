import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/chat', () => ({
  getContactCandidates: vi.fn()
}))

vi.mock('@/utils/cookie', () => ({
  generateCookie: vi.fn(() => 'generated-cookie')
}))

import CrossIdentityContactPicker from '@/components/chat/CrossIdentityContactPicker.vue'
import * as chatApi from '@/api/chat'
import { useIdentityStore } from '@/stores/identity'
import { useUserStore } from '@/stores/user'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/chat/CrossIdentityContactPicker.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('loads candidates for another identity and emits selected candidate', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'B', name: 'Bee', nickname: 'Bee' } as any

    const identityStore = useIdentityStore()
    identityStore.identityList = [
      { id: 'B', name: 'Bee', sex: '女' },
      { id: 'A', name: 'Alice', sex: '男' }
    ]
    identityStore.saveIdentityCookie('A', 'cookie-a')

    vi.mocked(chatApi.getContactCandidates).mockResolvedValue({
      code: 0,
      msg: 'success',
      data: {
        sourceIdentityId: 'A',
        items: [
          {
            targetUserId: 'test',
            nickname: 'Target',
            sources: ['archive', 'history'],
            localArchived: true,
            lastMsg: 'matched'
          }
        ]
      },
      warnings: ['favorite: upstream status 500']
    } as any)

    const wrapper = mount(CrossIdentityContactPicker, {
      props: { visible: false },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()
    await flushAsync()

    expect(chatApi.getContactCandidates).toHaveBeenCalledWith(expect.objectContaining({
      sourceIdentityId: 'A',
      includeUpstream: true,
      limit: 300,
      cookieData: 'cookie-a'
    }))
    expect(wrapper.text()).toContain('Target')
    expect(wrapper.text()).toContain('归档')
    expect(wrapper.text()).toContain('历史')
    expect(wrapper.text()).toContain('部分上游数据不可用')

    await wrapper.findAll('button').find(button => button.text().includes('Target'))?.trigger('click')

    const selected = wrapper.emitted('select')?.[0]
    expect(selected?.[0]).toMatchObject({ targetUserId: 'test', nickname: 'Target' })
    expect(selected?.[1]).toBe('A')
    expect(wrapper.emitted('update:visible')?.at(-1)).toEqual([false])
  })

  it('filters candidates locally after loading', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'B', name: 'Bee', nickname: 'Bee' } as any

    const identityStore = useIdentityStore()
    identityStore.identityList = [
      { id: 'A', name: 'Alice', sex: '男' },
      { id: 'B', name: 'Bee', sex: '女' }
    ]

    vi.mocked(chatApi.getContactCandidates).mockResolvedValue({
      code: 0,
      msg: 'success',
      data: {
        sourceIdentityId: 'A',
        items: [
          { targetUserId: 'u1', nickname: 'Alpha', sources: ['history'] },
          { targetUserId: 'u2', nickname: 'Beta', sources: ['favorite'] }
        ]
      }
    } as any)

    const wrapper = mount(CrossIdentityContactPicker, {
      props: { visible: false },
      global: {
        stubs: { teleport: true }
      }
    })

    await wrapper.setProps({ visible: true })
    await flushAsync()
    await flushAsync()

    await wrapper.find('input[placeholder="搜索用户"]').setValue('beta')

    expect(wrapper.text()).not.toContain('Alpha')
    expect(wrapper.text()).toContain('Beta')
  })
})
