import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { reactive, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

import GlobalFavorites from '@/components/settings/GlobalFavorites.vue'

let favoriteStore: any
let identityStore: any

const selectMock = vi.fn()
const enterChatMock = vi.fn()
const disconnectMock = vi.fn()
const toastShow = vi.fn()

vi.mock('@/stores/favorite', () => ({
  useFavoriteStore: () => favoriteStore
}))

vi.mock('@/stores/identity', () => ({
  useIdentityStore: () => identityStore
}))

vi.mock('@/composables/useIdentity', () => ({
  useIdentity: () => ({ select: selectMock })
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => ({ enterChat: enterChatMock })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({ disconnect: disconnectMock })
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

const createTestRouter = async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/chat/:id', component: { template: '<div />' } }
    ]
  })
  await router.push('/')
  await router.isReady()
  return router
}

describe('components/settings/GlobalFavorites.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    favoriteStore = reactive({
      loading: false,
      allFavorites: [] as any[],
      groupedFavorites: {} as Record<string, any[]>,
      loadAllFavorites: vi.fn().mockResolvedValue(undefined),
      removeFavoriteById: vi.fn().mockResolvedValue(true)
    })
    identityStore = reactive({
      identityList: [] as any[],
      loadList: vi.fn().mockResolvedValue(undefined)
    })
  })

  it('onMounted loads favorites and loads identity list when empty', async () => {
    const router = await createTestRouter()

    mount(GlobalFavorites, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          ChatHistoryPreview: true
        }
      }
    })
    await flushAsync()

    expect(favoriteStore.loadAllFavorites).toHaveBeenCalled()
    expect(identityStore.loadList).toHaveBeenCalled()
  })

  it('getIdentityName returns name when found, otherwise fallback', async () => {
    const router = await createTestRouter()
    identityStore.identityList = [{ id: 'me', name: 'MeName' }]

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [router],
        stubs: { PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true, ChatHistoryPreview: true }
      }
    })
    await flushAsync()

    const vm = wrapper.vm as any
    expect(vm.getIdentityName('me')).toBe('MeName')
    expect(vm.getIdentityName('nope')).toBe('未知身份')
  })

  it('openPreview + handlePreviewSwitch switches identity and navigates to chat, otherwise shows toast', async () => {
    vi.useFakeTimers()
    try {
      const router = await createTestRouter()
      const pushSpy = vi.spyOn(router, 'push')

      identityStore.identityList = [{ id: 'id1', name: 'I1' }]
      selectMock.mockResolvedValueOnce(undefined)

      const wrapper = mount(GlobalFavorites, {
        global: {
          plugins: [router],
          stubs: { PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true, ChatHistoryPreview: true }
        }
      })
      await flushAsync()

      const vm = wrapper.vm as any
      vm.openPreview({ identityId: 'id1', targetUserId: 'u1', targetUserName: 'U1' })
      await flushAsync()
      expect(vm.showPreview).toBe(true)

      await vm.handlePreviewSwitch()
      expect(disconnectMock).toHaveBeenCalledWith(true)
      expect(selectMock).toHaveBeenCalled()

      vi.advanceTimersByTime(500)
      await flushAsync()
      expect(enterChatMock).toHaveBeenCalled()
      expect(pushSpy).toHaveBeenCalledWith('/chat/u1')
      expect(vm.showPreview).toBe(false)

      // identity missing branch
      vm.previewIdentityId = 'missing'
      vm.previewTargetId = 'u2'
      vm.previewTargetName = 'U2'
      await vm.handlePreviewSwitch()
      expect(toastShow).toHaveBeenCalledWith('身份不存在，无法切换')
    } finally {
      vi.useRealTimers()
    }
  })

  it('directSwitch handles identity present/missing branches', async () => {
    vi.useFakeTimers()
    try {
      const router = await createTestRouter()
      const pushSpy = vi.spyOn(router, 'push')

      identityStore.identityList = [{ id: 'id1', name: 'I1' }]
      selectMock.mockResolvedValueOnce(undefined)

      const wrapper = mount(GlobalFavorites, {
        global: {
          plugins: [router],
          stubs: { PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true, ChatHistoryPreview: true }
        }
      })
      await flushAsync()

      const vm = wrapper.vm as any
      await vm.directSwitch({ identityId: 'id1', targetUserId: 'u1', targetUserName: '' })
      vi.advanceTimersByTime(500)
      await flushAsync()
      expect(pushSpy).toHaveBeenCalledWith('/chat/u1')

      await vm.directSwitch({ identityId: 'missing', targetUserId: 'u2', targetUserName: 'U2' })
      expect(toastShow).toHaveBeenCalledWith('身份不存在，无法切换')
    } finally {
      vi.useRealTimers()
    }
  })

  it('confirmDelete/executeDelete covers early return and success/failure branches', async () => {
    const router = await createTestRouter()
    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [router],
        stubs: { PullToRefresh: { template: '<div><slot /></div>' }, Skeleton: true, ChatHistoryPreview: true }
      }
    })
    await flushAsync()

    const vm = wrapper.vm as any

    // no target -> early return
    await vm.executeDelete()
    expect(toastShow).not.toHaveBeenCalledWith('已取消收藏')

    vm.confirmDelete({ id: 1, identityId: 'id1', targetUserId: 'u1', targetUserName: 'U1' })
    expect(vm.showDeleteDialog).toBe(true)

    favoriteStore.removeFavoriteById.mockResolvedValueOnce(true)
    await vm.executeDelete()
    expect(toastShow).toHaveBeenCalledWith('已取消收藏')
    expect(vm.deleteTarget).toBe(null)

    vm.confirmDelete({ id: 2, identityId: 'id1', targetUserId: 'u2', targetUserName: 'U2' })
    favoriteStore.removeFavoriteById.mockResolvedValueOnce(false)
    await vm.executeDelete()
    expect(toastShow).toHaveBeenCalledWith('操作失败')
  })

  it('renders skeleton list when loading and no favorites', async () => {
    const router = await createTestRouter()
    favoriteStore.loading = true
    favoriteStore.allFavorites = []
    favoriteStore.groupedFavorites = {}

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          ChatHistoryPreview: true
        }
      }
    })
    await flushAsync()

    // The skeleton list items use a unique background class.
    expect(wrapper.findAll('.bg-surface-active')).toHaveLength(6)
    expect(wrapper.text()).not.toContain('暂无收藏')
  })

  it('renders grouped favorites and covers template fallback branches', async () => {
    const router = await createTestRouter()
    identityStore.identityList = [{ id: 'id1', name: 'I1' }]

    favoriteStore.loading = false
    favoriteStore.allFavorites = [{ id: 1 }] as any
    favoriteStore.groupedFavorites = {
      id1: [
        { id: 1, identityId: 'id1', targetUserId: 'u1', targetUserName: 'Alice' },
        { id: 2, identityId: 'id1', targetUserId: 'u2', targetUserName: '' },
        { id: 3, identityId: 'id1', targetUserId: '', targetUserName: '' }
      ]
    }

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [router],
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          Skeleton: true,
          ChatHistoryPreview: true
        }
      }
    })
    await flushAsync()

    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('未知用户')

    const avatars = wrapper.findAll('div.bg-gradient-to-br')
    const letters = avatars.map(a => a.text().trim())
    expect(letters).toContain('A')
    expect(letters).toContain('U')
    expect(letters).toContain('?')

    // Covers (fav.targetUserName || '') fallback branch.
    const vm = wrapper.vm as any
    vm.openPreview({ identityId: 'id1', targetUserId: 'u2', targetUserName: '' })
    await flushAsync()
    expect(vm.previewTargetName).toBe('')
  })
})
