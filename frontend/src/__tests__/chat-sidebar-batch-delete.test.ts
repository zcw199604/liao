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
import * as chatApi from '@/api/chat'

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

const seedUsers = () => {
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
    area: 'CN',
    address: 'A',
    ip: '',
    isFavorite: false,
    lastMsg: 'hi',
    lastTime: '2026-01-02 09:05:00.000',
    unreadCount: 1
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
    isFavorite: true,
    lastMsg: 'yo',
    lastTime: '2026-01-01 10:00:00.000',
    unreadCount: 0
  } as any)

  return chatStore
}

describe('components/chat/ChatSidebar.vue (batch delete + day selector)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('onMounted returns early when currentUser is missing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: { Toast: true, SettingsDrawer: true, Dialog: true, PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true }
      }
    })

    await flushAsync()

    expect(wsMocks.connect).not.toHaveBeenCalled()
    expect(chatComposableMocks.loadUsers).not.toHaveBeenCalled()
  })

  it('selection mode, day selector apply, and batch delete success/partial-fail branches', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = seedUsers()
    const removeSpy = vi.spyOn(chatStore, 'removeUser')

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

    // Enter selection mode with preselect -> adds id and sets preselect day key.
    vm.handleEnterSelectionMode(chatStore.getUser('u1'))
    await flushAsync()
    expect(vm.selectionMode).toBe(true)
    expect(vm.selectedUserIds).toContain('u1')

    // Apply day selection: non-matching key -> toast "没有可选会话"
    vm.applyDaySelection(['2099-01-01'])
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('没有可选会话')

    // Apply day selection: select both day keys -> toast count
    vm.applyDaySelection(['2026-01-02', '2026-01-01'])
    await flushAsync()
    expect(vm.selectedUserIds.sort()).toEqual(['u1', 'u2'])
    expect(toastShow).toHaveBeenCalledWith('已选中 2 个会话')

    // confirmBatchDelete opens dialog only when selected not empty.
    vm.selectedUserIds = []
    vm.confirmBatchDelete()
    expect(vm.showBatchDeleteDialog).toBe(false)
    vm.selectedUserIds = ['u1']
    vm.confirmBatchDelete()
    expect(vm.showBatchDeleteDialog).toBe(true)

    // Batch delete: success (no failed) -> exitSelectionMode
    vi.mocked(chatApi.batchDeleteUsers).mockResolvedValueOnce({ code: 0, data: { failedItems: [] } } as any)
    vm.selectedUserIds = ['u1', 'u2']
    await vm.executeBatchDelete()
    expect(removeSpy).toHaveBeenCalledWith('u1')
    expect(removeSpy).toHaveBeenCalledWith('u2')
    expect(toastShow).toHaveBeenCalledWith('已删除 2 个会话')
    expect(vm.selectionMode).toBe(false)

    // Re-seed and re-enter selection mode.
    seedUsers()
    vm.handleEnterSelectionMode(chatStore.getUser('u1'))
    vm.selectedUserIds = ['u1', 'u2']

    // Batch delete: partial failures via res.code!=0 keeps failed selected.
    vi.mocked(chatApi.batchDeleteUsers).mockResolvedValueOnce({ code: 1 } as any)
    await vm.executeBatchDelete()
    expect(vm.selectionMode).toBe(true)
    expect(vm.selectedUserIds.sort()).toEqual(['u1', 'u2'])
    expect(toastShow).toHaveBeenCalledWith('已删除 0 个，失败 2 个')

    // Batch delete: failedItems list removes success and keeps failed.
    vi.mocked(chatApi.batchDeleteUsers).mockResolvedValueOnce({
      code: 0,
      data: { failedItems: [{ userToId: 'u2', reason: 'bad' }] }
    } as any)
    await vm.executeBatchDelete()
    expect(vm.selectedUserIds).toEqual(['u2'])
    expect(toastShow).toHaveBeenCalledWith('已删除 1 个，失败 1 个')
  })

  it('executeBatchDelete early returns and parses failedItems edge cases (missing userToId/reason)', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = seedUsers()
    const removeSpy = vi.spyOn(chatStore, 'removeUser')

    const wrapper = mount(ChatSidebar, {
      global: {
        plugins: [pinia, router],
        stubs: { Toast: true, SettingsDrawer: true, Dialog: true, PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true }
      }
    })
    await flushAsync()

    const vm = wrapper.vm as any

    // Early return: no currentUser
    userStore.currentUser = null as any
    vm.selectedUserIds = ['u1']
    await vm.executeBatchDelete()
    expect(vi.mocked(chatApi.batchDeleteUsers)).not.toHaveBeenCalled()

    // Early return: empty selection
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any
    vm.selectedUserIds = []
    await vm.executeBatchDelete()
    expect(vi.mocked(chatApi.batchDeleteUsers)).not.toHaveBeenCalled()

    // Missing failedItems -> fallback to [] (treat as all-success)
    seedUsers()
    vi.mocked(chatApi.batchDeleteUsers).mockResolvedValueOnce({ code: 0, data: {} } as any)
    vm.selectedUserIds = ['u1']
    await vm.executeBatchDelete()
    expect(removeSpy).toHaveBeenCalledWith('u1')

    // Edge failedItems: missing userToId -> skipped; missing reason -> does not populate reason map
    seedUsers()
    vi.mocked(chatApi.batchDeleteUsers).mockResolvedValueOnce({
      code: 0,
      data: { failedItems: [{}, { userToId: 'u2' }] }
    } as any)
    vm.selectedUserIds = ['u1', 'u2']
    await vm.executeBatchDelete()
    expect(removeSpy).toHaveBeenCalledWith('u1')
    expect(vm.selectedUserIds).toEqual(['u2'])
  })
})
