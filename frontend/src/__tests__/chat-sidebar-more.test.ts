import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const toastShow = vi.fn()

const wsMocks = {
  connect: vi.fn(),
  disconnect: vi.fn(),
  checkUserOnlineStatus: vi.fn()
}

const chatComposableMocks = {
  loadUsers: vi.fn()
}

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => wsMocks
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => chatComposableMocks
}))

vi.mock('@/composables/useInteraction', async () => {
  const { ref } = await import('vue')
  return {
    useSwipeAction: () => ({ isSwiping: ref(false) })
  }
})

vi.mock('@/api/chat', () => ({
  deleteUser: vi.fn(),
  batchDeleteUsers: vi.fn()
}))

vi.mock('@/api/favorite', () => ({
  listAllFavorites: vi.fn().mockResolvedValue({ code: 0, data: [] }),
  addFavorite: vi.fn(),
  removeFavorite: vi.fn(),
  removeFavoriteById: vi.fn()
}))

import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useFavoriteStore } from '@/stores/favorite'
import { useMessageStore } from '@/stores/message'
import * as chatApi from '@/api/chat'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/chat/ChatSidebar.vue (more coverage)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('refreshCurrentTab switches between history/favorite and shows toast on error', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    const loadHistorySpy = vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined)
    const loadFavSpy = vi.spyOn(chatStore, 'loadFavoriteUsers').mockResolvedValue(undefined)

    const wrapper = mount(ChatSidebar, {
      props: { currentUserId: 'u1' },
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: true,
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })

    await flushAsync()

    chatStore.activeTab = 'history'
    await (wrapper.vm as any).refreshCurrentTab()
    expect(loadHistorySpy).toHaveBeenCalledWith('me', 'me')

    chatStore.activeTab = 'favorite'
    await (wrapper.vm as any).refreshCurrentTab()
    expect(loadFavSpy).toHaveBeenCalledWith('me', 'me')

    loadHistorySpy.mockRejectedValueOnce(new Error('boom'))
    chatStore.activeTab = 'history'
    await (wrapper.vm as any).refreshCurrentTab()
    expect(toastShow).toHaveBeenCalledWith('刷新失败，请稍后重试')
  })

  it('tab switch refreshes when clicking same tab; otherwise updates activeTab', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: true,
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })
    await flushAsync()

    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined)

    await (wrapper.vm as any).handleTabSwitch('history')
    expect(chatStore.loadHistoryUsers).toHaveBeenCalledWith('me', 'me')

    await (wrapper.vm as any).handleTabSwitch('favorite')
    expect(chatStore.activeTab).toBe('favorite')
  })

  it('long press opens context menu and blocks click-to-select on that interaction', async () => {
    vi.useFakeTimers()

    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1', 'u2']
    chatStore.favoriteUserIds = []
    chatStore.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '男',
      age: '18',
      area: 'X',
      address: 'A',
      ip: '',
      isFavorite: true,
      lastMsg: 'hi',
      lastTime: new Date(2026, 0, 4, 9, 5, 0).toISOString(),
      unreadCount: 3
    } as any)
    chatStore.upsertUser({
      id: 'u2',
      name: 'Bob',
      nickname: 'Bob',
      sex: '女',
      age: '0',
      area: '未知',
      address: '未知',
      ip: '',
      isFavorite: false,
      lastMsg: 'yo',
      lastTime: new Date(2026, 0, 4, 9, 6, 0).toISOString(),
      unreadCount: 0
    } as any)

    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: vi.fn()
    })

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: true,
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })

    await flushAsync()

    const vm = wrapper.vm as any
    const user = chatStore.getUser('u1') as any

    vm.startLongPress(user, new MouseEvent('mousedown', { clientX: window.innerWidth - 2, clientY: 5 }))
    vi.advanceTimersByTime(500)
    await flushAsync()

    expect(vm.showContextMenu).toBe(true)
    expect(vm.contextMenuUser?.id).toBe('u1')

    const stopPropagation = vi.fn()
    const preventDefault = vi.fn()
    vm.handleClick(user, { stopPropagation, preventDefault } as any)
    expect(stopPropagation).toHaveBeenCalled()
    expect(preventDefault).toHaveBeenCalled()
    expect(wrapper.emitted('select')).toBeUndefined()

    vi.useRealTimers()
  })

  it('context menu actions: global favorite toggle, check online, and delete user success/fail', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/identity', component: { template: '<div />' } }
      ]
    })
    await router.push('/')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1']
    chatStore.favoriteUserIds = []
    chatStore.upsertUser({ id: 'u1', name: 'Alice', nickname: 'Alice', sex: '未知', ip: '', lastMsg: 'x', lastTime: '', unreadCount: 0 } as any)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = []
    const addFavSpy = vi.spyOn(favoriteStore, 'addFavorite').mockResolvedValue(true)
    const removeFavSpy = vi.spyOn(favoriteStore, 'removeFavorite').mockResolvedValue(true)

    const messageStore = useMessageStore()
    const resetSpy = vi.spyOn(messageStore, 'resetAll').mockImplementation(() => {})

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: true,
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })

    await flushAsync()

    const vm = wrapper.vm as any
    const user = chatStore.getUser('u1') as any

    await vm.handleToggleGlobalFavorite(user)
    expect(addFavSpy).toHaveBeenCalledWith('me', 'u1', 'Alice')
    expect(toastShow).toHaveBeenCalledWith('已加入全局收藏')

    favoriteStore.allFavorites = [{ id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'Alice' } as any]
    await vm.handleToggleGlobalFavorite(user)
    expect(removeFavSpy).toHaveBeenCalledWith('me', 'u1')
    expect(toastShow).toHaveBeenCalledWith('已取消全局收藏')

    vm.handleCheckOnlineStatus(user)
    expect(wsMocks.checkUserOnlineStatus).toHaveBeenCalledWith('u1')
    expect(toastShow).toHaveBeenCalledWith('正在查询...')

    window.dispatchEvent(new CustomEvent('check-online-result', { detail: { data: { IF_Online: '1', TimeAll: 't' } } }))
    await flushAsync()
    expect(vm.showOnlineStatusDialog).toBe(true)

    vm.confirmDeleteUser(user)
    expect(vm.showDeleteUserDialog).toBe(true)

    vi.mocked(chatApi.deleteUser).mockResolvedValueOnce({} as any)
    const removeUserSpy = vi.spyOn(chatStore, 'removeUser')
    await vm.executeDeleteUser()
    expect(removeUserSpy).toHaveBeenCalledWith('u1')
    expect(toastShow).toHaveBeenCalledWith('删除成功')

    vm.confirmDeleteUser(user)
    vi.mocked(chatApi.deleteUser).mockRejectedValueOnce(new Error('nope'))
    await vm.executeDeleteUser()
    expect(toastShow).toHaveBeenCalledWith('删除失败')

    // cover confirmSwitchIdentity path
    await vm.handleSwitchIdentity()
    vm.confirmSwitchIdentity()
    expect(wsMocks.disconnect).toHaveBeenCalledWith(true)
    expect(resetSpy).toHaveBeenCalled()
  })
})
