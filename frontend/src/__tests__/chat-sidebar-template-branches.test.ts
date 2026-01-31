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

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

const createTestRouter = async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/identity', component: { template: '<div />' } }
    ]
  })
  await router.push('/')
  await router.isReady()
  return router
}

describe('components/chat/ChatSidebar.vue (template branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('top menu toggles and opens each settings mode', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false

    const DialogStub = { name: 'Dialog', template: '<div class=\"dialog-stub\"><slot /></div>' }

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: DialogStub,
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })
    await flushAsync()

    // wsConnected false -> red dot; toggle to true -> green.
    expect(wrapper.find('.bg-red-500').exists()).toBe(true)
    chatStore.wsConnected = true
    await flushAsync()
    expect(wrapper.find('.bg-green-500').exists()).toBe(true)

    // Open dropdown menu.
    await wrapper.get('button').trigger('click')
    await flushAsync()
    expect(wrapper.text()).toContain('身份信息')
    expect(wrapper.text()).toContain('系统设置')
    expect(wrapper.text()).toContain('图片管理')
    expect(wrapper.text()).toContain('全局收藏')
    expect(wrapper.text()).toContain('切换身份')

    const vm = wrapper.vm as any

    // Identity mode.
    await wrapper.findAll('button').find(b => b.text().includes('身份信息'))!.trigger('click')
    expect(vm.showSettings).toBe(true)
    expect(vm.settingsMode).toBe('identity')

    // System mode.
    vm.showTopMenu = true
    await flushAsync()
    await wrapper.findAll('button').find(b => b.text().includes('系统设置'))!.trigger('click')
    expect(vm.settingsMode).toBe('system')

    // Media mode.
    vm.showTopMenu = true
    await flushAsync()
    await wrapper.findAll('button').find(b => b.text().includes('图片管理'))!.trigger('click')
    expect(vm.settingsMode).toBe('media')

    // Favorites mode.
    vm.showTopMenu = true
    await flushAsync()
    await wrapper.findAll('button').find(b => b.text().includes('全局收藏'))!.trigger('click')
    expect(vm.settingsMode).toBe('favorites')

    // Switch identity opens dialog.
    vm.showTopMenu = true
    await flushAsync()
    await wrapper.findAll('button').find(b => b.text().includes('切换身份'))!.trigger('click')
    expect(vm.showSwitchIdentityDialog).toBe(true)
  })

  it('renders skeleton during initial load, then empty states for history/favorite tabs', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    // Keep loadUsers pending so isInitialLoadingUsers stays true.
    let resolveLoad: (() => void) | null = null
    chatComposableMocks.loadUsers.mockImplementationOnce(
      () => new Promise<void>(resolve => (resolveLoad = resolve))
    )

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: { template: '<div><slot /></div>' },
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })

    await flushAsync()
    expect(wsMocks.connect).toHaveBeenCalled()
    // Skeletons should render while loadUsers unresolved.
    expect(wrapper.findAll('skeleton-stub').length).toBeGreaterThan(0)

    resolveLoad?.()
    await flushAsync()
    await flushAsync()

    // After load, show empty state for history tab.
    expect(wrapper.text()).toContain('暂无消息')

    const chatStore = useChatStore()
    chatStore.activeTab = 'favorite'
    await flushAsync()
    expect(wrapper.text()).toContain('暂无收藏')
  })

  it('covers list item branches for sex/age/address/star/unread and selection UI', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1', 'u2', 'u3']

    chatStore.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '男',
      age: '18',
      area: 'CN',
      address: 'A',
      ip: '',
      isFavorite: true,
      lastMsg: 'hi',
      lastTime: '2026-01-02 09:05:00.000',
      unreadCount: 2
    } as any)
    chatStore.upsertUser({
      id: 'u2',
      name: 'Bob',
      nickname: 'Bob',
      sex: '女',
      age: '0',
      area: 'CN',
      address: '未知',
      ip: '',
      isFavorite: false,
      lastMsg: 'yo',
      lastTime: '2026-01-01 10:00:00.000',
      unreadCount: 0
    } as any)
    chatStore.upsertUser({
      id: 'u3',
      name: 'Mystery',
      nickname: 'Mystery',
      sex: '未知',
      age: '0',
      area: 'CN',
      address: '保密',
      ip: '',
      isFavorite: false,
      lastMsg: '...',
      lastTime: 'bad-date',
      unreadCount: 1
    } as any)

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: { template: '<div class=\"dialog-stub\"><slot /></div>' },
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })
    await flushAsync()

    // Male + female icons rendered, unknown has neither mars/venus icon.
    expect(wrapper.findAll('i.fas.fa-mars').length).toBeGreaterThanOrEqual(1)
    expect(wrapper.findAll('i.fas.fa-venus').length).toBeGreaterThanOrEqual(1)
    // Address badge: user1 has "A" so present; user2/3 should not show address chip.
    expect(wrapper.text()).toContain('A')
    // Favorite star (history tab only)
    expect(wrapper.findAll('i.fas.fa-star').length).toBeGreaterThan(0)

    // Unread badges show for users with unreadCount>0.
    expect(wrapper.findAll('draggable-badge-stub').length).toBeGreaterThanOrEqual(2)

    // Enter selection mode: checkboxes appear and match button/overlay hidden.
    const vm = wrapper.vm as any
    vm.handleEnterSelectionMode(chatStore.getUser('u1'))
    await flushAsync()
    expect(wrapper.findAll('i.fas.fa-check').length).toBeGreaterThanOrEqual(1)
    expect(wrapper.find('match-button-stub').exists()).toBe(false)
    expect(wrapper.find('match-overlay-stub').exists()).toBe(false)

    // Exit selection mode: match button/overlay visible again.
    vm.exitSelectionMode()
    await flushAsync()
    expect(wrapper.find('match-button-stub').exists()).toBe(true)
    expect(wrapper.find('match-overlay-stub').exists()).toBe(true)

    // Online status dialog slot branches (offline + lastTime fallback).
    vm.showOnlineStatusDialog = true
    vm.onlineStatusData = { isOnline: false, lastTime: '' }
    await flushAsync()
    expect(wrapper.text()).toContain('离线')
    expect(wrapper.text()).toContain('未知')
  })
})

