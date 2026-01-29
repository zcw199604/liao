import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()
const routerPush = vi.fn()
const disconnect = vi.fn()
const selectIdentity = vi.fn()
const enterChat = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({ disconnect })
}))

vi.mock('@/composables/useIdentity', () => ({
  useIdentity: () => ({ select: selectIdentity })
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => ({ enterChat })
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<any>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: routerPush })
  }
})

import GlobalFavorites from '@/components/settings/GlobalFavorites.vue'
import { useFavoriteStore } from '@/stores/favorite'
import { useIdentityStore } from '@/stores/identity'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/settings/GlobalFavorites.vue', () => {
  it('shows empty state when no favorites', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = []
    favoriteStore.loading = false
    vi.spyOn(favoriteStore, 'loadAllFavorites').mockImplementation(async () => {})

    const identityStore = useIdentityStore()
    identityStore.identityList = []
    vi.spyOn(identityStore, 'loadList').mockImplementation(async () => {})

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [pinia],
        stubs: {
          PullToRefresh: { template: `<div><slot /></div>` },
          ChatHistoryPreview: true,
          Skeleton: true
        }
      }
    })

    await nextTick()
    expect(wrapper.text()).toContain('暂无收藏')
  })

  it('shows loading skeleton when loading and list is empty', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = []
    favoriteStore.loading = true
    vi.spyOn(favoriteStore, 'loadAllFavorites').mockImplementation(async () => {})

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [pinia],
        stubs: {
          PullToRefresh: { template: `<div><slot /></div>` },
          ChatHistoryPreview: true,
          Skeleton: { template: `<div data-testid="sk"></div>` }
        }
      }
    })

    await nextTick()
    expect(wrapper.findAll('[data-testid="sk"]').length).toBeGreaterThan(0)
  })

  it('switches identity and enters chat when identity exists; shows toast when identity missing', async () => {
    vi.useFakeTimers()
    const pinia = createPinia()
    setActivePinia(pinia)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = [
      { id: 1, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' }
    ] as any
    favoriteStore.loading = false
    vi.spyOn(favoriteStore, 'loadAllFavorites').mockImplementation(async () => {})

    const identityStore = useIdentityStore()
    identityStore.identityList = [{ id: 'i1', name: 'Me', sex: '男' } as any]
    vi.spyOn(identityStore, 'loadList').mockImplementation(async () => {})

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [pinia],
        stubs: {
          PullToRefresh: { template: `<div><slot /></div>` },
          ChatHistoryPreview: {
            name: 'ChatHistoryPreview',
            props: ['visible'],
            emits: ['close', 'switch'],
            template: `<div data-testid="preview"></div>`
          },
          Skeleton: true
        }
      }
    })

    await nextTick()

    // direct switch button
    await wrapper.find('button[title="切换并聊天"]').trigger('click')
    expect(disconnect).toHaveBeenCalledWith(true)
    expect(selectIdentity).toHaveBeenCalledWith({ id: 'i1', name: 'Me', sex: '男' })

    vi.advanceTimersByTime(500)
    await nextTick()

    expect(enterChat).toHaveBeenCalled()
    expect(routerPush).toHaveBeenCalledWith('/chat/u1')

    // identity missing branch via preview switch event
    identityStore.identityList = []
    await wrapper.find('button[title="预览聊天"]').trigger('click')
    expect(wrapper.find('[data-testid="preview"]').exists()).toBe(true)

    wrapper.findComponent({ name: 'ChatHistoryPreview' }).vm.$emit('switch')
    await nextTick()
    expect(toastShow).toHaveBeenCalledWith('身份不存在，无法切换')

    vi.useRealTimers()
  })

  it('confirms delete and shows toast based on result', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = [
      { id: 1, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' }
    ] as any
    favoriteStore.loading = false
    vi.spyOn(favoriteStore, 'loadAllFavorites').mockImplementation(async () => {})
    const removeSpy = vi.spyOn(favoriteStore, 'removeFavoriteById').mockResolvedValue(true as any)

    const wrapper = mount(GlobalFavorites, {
      global: {
        plugins: [pinia],
        stubs: {
          PullToRefresh: { template: `<div><slot /></div>` },
          ChatHistoryPreview: true,
          Skeleton: true
        }
      }
    })

    await nextTick()
    await wrapper.find('button[title="取消收藏"]').trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('确认删除')

    await wrapper.findAll('button').find(b => b.text().trim() === '删除')!.trigger('click')
    await nextTick()

    expect(removeSpy).toHaveBeenCalledWith(1)
    expect(toastShow).toHaveBeenCalledWith('已取消收藏')
  })
})
