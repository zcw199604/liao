import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

import Loading from '@/components/common/Loading.vue'
import Dialog from '@/components/common/Dialog.vue'
import Toast from '@/components/common/Toast.vue'
import UserList from '@/components/list/UserList.vue'
import ChatSidebar from '@/components/chat/ChatSidebar.vue'

import { useToast } from '@/composables/useToast'
import { useChatStore } from '@/stores/chat'

describe('components/common/Loading.vue', () => {
  it('renders text when provided', () => {
    const wrapper = mount(Loading, { props: { text: '加载中' } })
    expect(wrapper.text()).toContain('加载中')
  })

  it('does not render text when omitted', () => {
    const wrapper = mount(Loading)
    expect(wrapper.text()).toBe('')
  })
})

describe('components/common/Dialog.vue', () => {
  it('emits update:visible=false when clicking overlay', async () => {
    const wrapper = mount(Dialog, { props: { visible: true, title: 't', content: 'c' } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('emits confirm and closes when clicking confirm', async () => {
    const wrapper = mount(Dialog, { props: { visible: true, title: 't', content: 'c' } })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(2)

    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('confirm')).toBeTruthy()
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('emits cancel and closes when clicking cancel', async () => {
    const wrapper = mount(Dialog, { props: { visible: true, title: 't', content: 'c' } })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(2)

    await buttons[0]?.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('renders warning when showWarning=true', () => {
    const wrapper = mount(Dialog, { props: { visible: true, title: 't', content: 'c', showWarning: true } })
    expect(wrapper.text()).toContain('此操作无法撤销')
  })
})

describe('components/common/Toast.vue', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useToast().hide()
  })

  afterEach(() => {
    useToast().hide()
    vi.useRealTimers()
  })

  it('renders message when visible and hides after duration', async () => {
    const wrapper = mount(Toast, {
      global: {
        stubs: { teleport: true }
      }
    })

    const toast = useToast()
    toast.show('你好', 2000)
    await nextTick()
    expect(wrapper.text()).toContain('你好')

    vi.advanceTimersByTime(2000)
    await nextTick()
    expect(wrapper.text()).not.toContain('你好')
  })
})

describe('components/list/UserList.vue', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders user and emits select on click', async () => {
    const user = {
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      ip: '',
      lastMessageTime: new Date(2026, 0, 4, 9, 5, 0).toISOString()
    }

    const wrapper = mount(UserList, { props: { users: [user], type: 'history' } })
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('09:05')

    await wrapper.get('div.cursor-pointer').trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([user])
  })

  it('renders empty state for history and favorite', () => {
    const historyWrapper = mount(UserList, { props: { users: [], type: 'history' } })
    expect(historyWrapper.text()).toContain('暂无历史用户')

    const favoriteWrapper = mount(UserList, { props: { users: [], type: 'favorite' } })
    expect(favoriteWrapper.text()).toContain('暂无收藏用户')
  })
})

describe('components/chat/ChatSidebar.vue', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
    pinia = createPinia()
    setActivePinia(pinia)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const mountChatSidebar = async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    return shallowMount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: true,
          MatchButton: true,
          MatchOverlay: true,
          PullToRefresh: {
            template: '<div><slot /></div>'
          }
        }
      }
    })
  }

  it('formats lastTime and emits select on click', async () => {
    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1']
    chatStore.favoriteUserIds = []
    chatStore.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      ip: '',
      isFavorite: false,
      lastMsg: 'hi',
      lastTime: new Date(2026, 0, 4, 9, 5, 0).toISOString(),
      unreadCount: 0
    })

    const wrapper = await mountChatSidebar()

    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('09:05')

    await wrapper.get('div.cursor-pointer').trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toEqual(expect.objectContaining({ id: 'u1' }))
  })

  it('filters users by keyword and supports nickname/name/id/address matching', async () => {
    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1', 'USER-BETA-02']
    chatStore.favoriteUserIds = []
    chatStore.upsertUser({
      id: 'u1',
      name: 'Alice Name',
      nickname: 'Ali',
      sex: '未知',
      ip: '',
      address: 'Hangzhou',
      isFavorite: false,
      lastMsg: 'hello',
      lastTime: new Date(2026, 0, 4, 9, 5, 0).toISOString(),
      unreadCount: 0
    })
    chatStore.upsertUser({
      id: 'USER-BETA-02',
      name: 'Beta Name',
      nickname: 'Bobby',
      sex: '未知',
      ip: '',
      address: 'Shenzhen',
      isFavorite: false,
      lastMsg: 'hi',
      lastTime: new Date(2026, 0, 4, 9, 6, 0).toISOString(),
      unreadCount: 0
    })

    const wrapper = await mountChatSidebar()
    const searchInput = wrapper.get('[data-testid="chat-sidebar-search-input"]')

    expect(wrapper.text()).toContain('Ali')
    expect(wrapper.text()).toContain('Bobby')

    await searchInput.setValue('boB')
    expect(wrapper.text()).toContain('Bobby')
    expect(wrapper.text()).not.toContain('Ali')

    await searchInput.setValue('alice name')
    expect(wrapper.text()).toContain('Ali')
    expect(wrapper.text()).not.toContain('Bobby')

    await searchInput.setValue('user-beta-02')
    expect(wrapper.text()).toContain('Bobby')
    expect(wrapper.text()).not.toContain('Ali')

    await searchInput.setValue('SHENZHEN')
    expect(wrapper.text()).toContain('Bobby')
    expect(wrapper.text()).not.toContain('Ali')

    await wrapper.get('div.cursor-pointer').trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toEqual(expect.objectContaining({ id: 'USER-BETA-02' }))

    await searchInput.setValue('')
    expect(wrapper.text()).toContain('Ali')
    expect(wrapper.text()).toContain('Bobby')
  })

  it('shows no-match message when keyword has no results', async () => {
    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1']
    chatStore.favoriteUserIds = []
    chatStore.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      ip: '',
      address: 'Hangzhou',
      isFavorite: false,
      lastMsg: 'hello',
      lastTime: new Date(2026, 0, 4, 9, 5, 0).toISOString(),
      unreadCount: 0
    })

    const wrapper = await mountChatSidebar()
    const searchInput = wrapper.get('[data-testid="chat-sidebar-search-input"]')

    await searchInput.setValue('not-exist-user')
    expect(wrapper.text()).toContain('未找到匹配用户')
    expect(wrapper.text()).not.toContain('暂无消息')

    await searchInput.setValue('')
    expect(wrapper.text()).toContain('Alice')
  })
})

