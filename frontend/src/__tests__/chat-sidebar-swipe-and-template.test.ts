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

const swipeCalls = vi.hoisted(() => [] as any[])

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
    useSwipeAction: (_target: any, opts: any) => {
      swipeCalls.push(opts)
      return { isSwiping: ref(false) }
    }
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
import { useFavoriteStore } from '@/stores/favorite'
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

const DialogStub = {
  name: 'Dialog',
  props: ['visible'],
  emits: ['update:visible', 'confirm'],
  template: `<div v-if="visible" class="dialog-stub"><slot /></div>`
}

describe('components/chat/ChatSidebar.vue (swipe + template branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    swipeCalls.splice(0, swipeCalls.length)
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('covers selection-mode helpers, highlight/sex template branches, and scrollTop restore', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = await createTestRouter()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.listScrollTop = 123
    chatStore.activeTab = 'history'
    chatStore.historyUserIds = ['u1', 'u2', 'u3']
    chatStore.favoriteUserIds = []

    // Seed three users to cover sex label ternary (男/女/other) + highlight class.
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
      lastMsg: 'm1',
      lastTime: '2026-01-02 09:05:00.000',
      unreadCount: 0
    } as any)
    chatStore.upsertUser({
      id: 'u2',
      name: 'Betty',
      nickname: 'Betty',
      sex: '女',
      age: '18',
      area: 'CN',
      address: '保密',
      ip: '',
      isFavorite: false,
      lastMsg: 'm2',
      lastTime: '2026-01-02 09:06:00.000',
      unreadCount: 1
    } as any)
    chatStore.upsertUser({
      id: 'u3',
      name: 'X',
      nickname: 'X',
      sex: '未知',
      age: '18', // show the label even when sex is 未知
      area: 'CN',
      address: '未知',
      ip: '',
      isFavorite: false,
      lastMsg: 'm3',
      lastTime: '',
      unreadCount: 0
    } as any)

    const favoriteStore = useFavoriteStore()
    favoriteStore.allFavorites = []

    const wrapper = mount(ChatSidebar, {
      props: { currentUserId: 'u1' },
      global: {
        plugins: [pinia, router],
        stubs: {
          Toast: true,
          SettingsDrawer: true,
          Dialog: DialogStub,
          PullToRefresh: { template: '<div><slot /></div>' },
          ChatDayBulkDeleteModal: true,
          Skeleton: true,
          MatchButton: true,
          MatchOverlay: true,
          DraggableBadge: true
        }
      }
    })
    await flushAsync()
    await flushAsync()

    // scrollTop restore on mount.
    const listEl = (wrapper.vm as any).listAreaRef as HTMLElement | null
    expect(listEl).toBeTruthy()
    expect(listEl!.scrollTop).toBe(123)

    // currentUser highlight class (only when selectionMode=false).
    const aliceSpan = wrapper.findAll('span').find(s => s.text() === 'Alice')
    expect(aliceSpan).toBeTruthy()
    const aliceRow = aliceSpan!.element.closest('div.flex.items-center.p-4') as HTMLElement
    expect(aliceRow.className).toContain('border-blue-500/30')

    const vm = wrapper.vm as any

    // handleEnterSelectionMode without preselect hits the ternary false branch.
    vm.handleEnterSelectionMode()
    await flushAsync()
    expect(vm.selectionMode).toBe(true)

    // toggleSelection add/remove branches.
    vm.toggleSelection('u1')
    expect(vm.selectedUserIds).toContain('u1')
    vm.toggleSelection('u1')
    expect(vm.selectedUserIds).not.toContain('u1')

    // toggleSelectAll select and clear branches + isAllSelected computed.
    vm.toggleSelectAll()
    expect(vm.selectedUserIds.sort()).toEqual(['u1', 'u2', 'u3'])
    expect(vm.isAllSelected).toBe(true)
    vm.toggleSelectAll()
    expect(vm.selectedUserIds).toEqual([])
    expect(vm.isAllSelected).toBe(false)

    // In selection mode, selected row should use ring class (template ternary branch).
    vm.toggleSelection('u1')
    await flushAsync()
    expect(aliceRow.className).toContain('ring-2')

    // refreshCurrentTab early return when isRefreshing=true
    vm.isRefreshing = true
    const loadHistorySpy = vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined)
    await vm.refreshCurrentTab()
    expect(loadHistorySpy).not.toHaveBeenCalled()
  })

  it('covers dayDeleteItems labels/sorting (today/yesterday/unknown) and grouping merge branch', async () => {
    vi.useFakeTimers()
    try {
      vi.setSystemTime(new Date(2026, 0, 2, 12, 0, 0)) // 2026-01-02

      const pinia = createPinia()
      setActivePinia(pinia)
      const router = await createTestRouter()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const chatStore = useChatStore()
      chatStore.activeTab = 'history'
      chatStore.historyUserIds = ['a1', 'a2', 'y1', 'u']
      chatStore.favoriteUserIds = []

      // Two users on the same day -> hits map.get(key) true branch.
      chatStore.upsertUser({ id: 'a1', nickname: 'A1', name: 'A1', sex: '男', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '2026-01-02 00:00:00.000', unreadCount: 0 } as any)
      chatStore.upsertUser({ id: 'a2', nickname: 'A2', name: 'A2', sex: '男', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '2026-01-02 01:00:00.000', unreadCount: 0 } as any)
      chatStore.upsertUser({ id: 'y1', nickname: 'Y1', name: 'Y1', sex: '女', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '2026-01-01 00:00:00.000', unreadCount: 0 } as any)
      chatStore.upsertUser({ id: 'u', nickname: 'U', name: 'U', sex: '未知', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: 'bad-time', unreadCount: 0 } as any)

      const wrapper = mount(ChatSidebar, {
        global: {
          plugins: [pinia, router],
          stubs: {
            Toast: true,
            SettingsDrawer: true,
            Dialog: DialogStub,
            PullToRefresh: { template: '<div><slot /></div>' },
            ChatDayBulkDeleteModal: true,
            Skeleton: true,
            MatchButton: true,
            MatchOverlay: true,
            DraggableBadge: true
          }
        }
      })
      await flushAsync()
      await flushAsync()

      const items = (wrapper.vm as any).dayDeleteItems as Array<{ key: string; label: string; count: number }>
      const labels = items.map(i => i.label).join('|')
      expect(labels).toContain('（今天）')
      expect(labels).toContain('（昨天）')
      expect(labels).toContain('未知时间')

      // unknown should be sorted last
      expect(items[items.length - 1]?.key).toBe('unknown')

      const todayItem = items.find(i => i.key === '2026-01-02')
      expect(todayItem?.count).toBe(2)
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers swipe callbacks + resetListTranslateX timer branches + unmount cleanup', async () => {
    vi.useFakeTimers()
    try {
      const pinia = createPinia()
      setActivePinia(pinia)
      const router = await createTestRouter()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const chatStore = useChatStore()
      chatStore.activeTab = 'history'
      chatStore.historyUserIds = ['u1']
      chatStore.upsertUser({ id: 'u1', name: 'U1', nickname: 'U1', sex: '男', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '2026-01-02 00:00:00.000', unreadCount: 0 } as any)

      const wrapper = mount(ChatSidebar, {
        global: {
          plugins: [pinia, router],
          stubs: {
            Toast: true,
            SettingsDrawer: true,
            Dialog: DialogStub,
            PullToRefresh: { template: '<div><slot /></div>' },
            ChatDayBulkDeleteModal: true,
            Skeleton: true,
            MatchButton: true,
            MatchOverlay: true,
            DraggableBadge: true
          }
        }
      })
      await flushAsync()

      expect(swipeCalls).toHaveLength(1)
      const opts = swipeCalls[0]!
      const vm = wrapper.vm as any

      // selectionMode early return
      vm.selectionMode = true
      opts.onSwipeProgress?.(999, 0)
      expect(vm.listTranslateX).toBe(0)

      // showContextMenu early return
      vm.selectionMode = false
      vm.showContextMenu = true
      opts.onSwipeProgress?.(999, 0)
      expect(vm.listTranslateX).toBe(0)

      // vertical-dominant -> resets translate
      vm.showContextMenu = false
      vm.listTranslateX = 10
      opts.onSwipeProgress?.(10, 100)
      expect(vm.listTranslateX).toBe(0)

      // clamp/dampening both directions
      opts.onSwipeProgress?.(999, 0)
      expect(vm.listTranslateX).toBeGreaterThan(120)
      opts.onSwipeProgress?.(-999, 0)
      expect(vm.listTranslateX).toBeLessThan(-120)

      // onSwipeEnd switches tab by direction
      chatStore.activeTab = 'history'
      opts.onSwipeEnd?.('left')
      expect(chatStore.activeTab).toBe('favorite')
      chatStore.activeTab = 'favorite'
      opts.onSwipeEnd?.('right')
      expect(chatStore.activeTab).toBe('history')

      // onSwipeEnd no-op branches (direction/tab mismatch)
      chatStore.activeTab = 'history'
      opts.onSwipeEnd?.('right')
      expect(chatStore.activeTab).toBe('history')
      chatStore.activeTab = 'favorite'
      opts.onSwipeEnd?.('left')
      expect(chatStore.activeTab).toBe('favorite')

      // onSwipeEnd closes context menu
      vm.showContextMenu = true
      vm.contextMenuUser = chatStore.getUser('u1')
      opts.onSwipeEnd?.('left')
      expect(vm.showContextMenu).toBe(false)

      // onSwipeFinish selectionMode early return
      vm.selectionMode = true
      vm.showContextMenu = false
      vm.listTranslateX = 55
      opts.onSwipeFinish?.(0, 0, false)
      expect(vm.listTranslateX).toBe(55)
      vm.selectionMode = false

      // onSwipeFinish closes menu and resets translate immediately
      vm.showContextMenu = true
      vm.listTranslateX = 55
      opts.onSwipeFinish?.(0, 0, false)
      expect(vm.listTranslateX).toBe(0)
      expect(vm.isAnimating).toBe(false)

      // resetListTranslateX: already at 0 -> sets isAnimating=false and returns
      vm.showContextMenu = false
      vm.isAnimating = true
      vm.listTranslateX = 0
      vm.resetListTranslateX()
      expect(vm.isAnimating).toBe(false)

      // resetListTranslateX: non-zero -> creates timer and later clears animation
      const clearSpy = vi.spyOn(window, 'clearTimeout')
      vm.listTranslateX = 66
      vm.resetListTranslateX()
      expect(vm.isAnimating).toBe(true)
      expect(vm.listTranslateX).toBe(0)
      // call again while timer exists -> clears previous timer (branch)
      vm.listTranslateX = 66
      vm.resetListTranslateX()
      expect(clearSpy).toHaveBeenCalled()

      // unmount cleanup also clears timer if present
      wrapper.unmount()
      expect(clearSpy).toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers TouchEvent branches and context-menu favorite/online dialog template branches', async () => {
    vi.useFakeTimers()
    try {
      const pinia = createPinia()
      setActivePinia(pinia)
      const router = await createTestRouter()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const chatStore = useChatStore()
      chatStore.activeTab = 'history'
      chatStore.historyUserIds = ['u1']
      chatStore.upsertUser({ id: 'u1', name: 'U1', nickname: 'U1', sex: '女', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '2026-01-02 00:00:00.000', unreadCount: 0 } as any)

      const favoriteStore = useFavoriteStore()
      favoriteStore.allFavorites = []

      // Provide a fake TouchEvent so startLongPress hits the touch path.
      class FakeTouchEvent extends Event {
        touches: Array<{ clientX: number; clientY: number }>
        constructor(type: string, touches: Array<{ clientX: number; clientY: number }>) {
          super(type)
          this.touches = touches
        }
      }
      ;(globalThis as any).TouchEvent = FakeTouchEvent
      ;(window as any).TouchEvent = FakeTouchEvent

      const wrapper = mount(ChatSidebar, {
        global: {
          plugins: [pinia, router],
          stubs: {
            Toast: true,
            SettingsDrawer: true,
            Dialog: DialogStub,
            PullToRefresh: { template: '<div><slot /></div>' },
            ChatDayBulkDeleteModal: true,
            Skeleton: true,
            MatchButton: true,
            MatchOverlay: true,
            DraggableBadge: true
          }
        }
      })
      await flushAsync()

      const vm = wrapper.vm as any
      const user = chatStore.getUser('u1')

      // TouchEvent with empty touches -> returns early (no menu).
      vm.startLongPress(user, new FakeTouchEvent('touchstart', []) as any)
      vi.advanceTimersByTime(500)
      await flushAsync()
      expect(vm.showContextMenu).toBe(false)

      // TouchEvent with touches -> opens menu
      vm.startLongPress(user, new FakeTouchEvent('touchstart', [{ clientX: 5, clientY: 5 }]) as any)
      vi.advanceTimersByTime(500)
      await flushAsync()
      expect(vm.showContextMenu).toBe(true)

      // Template branches: favorite label + class in context menu.
      const favBtn = wrapper.findAll('button').find(b => b.text().includes('全局收藏') || b.text().includes('取消全局收藏'))
      expect(favBtn).toBeTruthy()
      expect(favBtn!.text()).toContain('全局收藏')
      expect(favBtn!.classes().join(' ')).toContain('text-fg-muted')

      favoriteStore.allFavorites = [{ id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'U1' } as any]
      await flushAsync()
      expect(favBtn!.text()).toContain('取消全局收藏')
      expect(favBtn!.classes().join(' ')).toContain('text-yellow-500')

      // Online status dialog branches (在线/离线)
      vm.showOnlineStatusDialog = true
      vm.onlineStatusData = { isOnline: true, lastTime: 't' }
      await flushAsync()
      const onlineDialogText = wrapper.find('.dialog-stub').text()
      expect(onlineDialogText).toContain('当前状态')
      expect(onlineDialogText).toContain('在线')

      vm.onlineStatusData = { isOnline: false, lastTime: '' }
      await flushAsync()
      expect(wrapper.find('.dialog-stub').text()).toContain('离线')
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers remaining edge branches: isAllSelected empty, preselect lastTime fallback, day selection unknown, menu bounds, and global favorite toggle', async () => {
    vi.useFakeTimers()
    try {
      const pinia = createPinia()
      setActivePinia(pinia)
      const router = await createTestRouter()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const chatStore = useChatStore()
      chatStore.activeTab = 'history'
      chatStore.historyUserIds = ['u1']
      chatStore.upsertUser({ id: 'u1', name: 'U1', nickname: '', sex: '男', age: '18', area: '', address: '', ip: '', lastMsg: 'x', lastTime: '', unreadCount: 0 } as any)

      const favoriteStore = useFavoriteStore()
      favoriteStore.allFavorites = []
      const addSpy = vi.spyOn(favoriteStore, 'addFavorite').mockResolvedValue(true as any)
      const removeSpy = vi.spyOn(favoriteStore, 'removeFavorite').mockResolvedValue(true as any)

      const wrapper = mount(ChatSidebar, {
        global: {
          plugins: [pinia, router],
          stubs: {
            Toast: true,
            SettingsDrawer: true,
            Dialog: DialogStub,
            PullToRefresh: { template: '<div><slot /></div>' },
            ChatDayBulkDeleteModal: true,
            Skeleton: true,
            MatchButton: true,
            MatchOverlay: true,
            DraggableBadge: true
          }
        }
      })
      await flushAsync()

      const vm = wrapper.vm as any

      // isAllSelected: empty displayList -> total=0 branch
      chatStore.historyUserIds = []
      await flushAsync()
      expect(vm.isAllSelected).toBe(false)

      // handleEnterSelectionMode: preselect exists but lastTime empty -> getDayKeyFromLastTime('') => 'unknown'
      chatStore.historyUserIds = ['u1']
      await flushAsync()
      vm.handleEnterSelectionMode(chatStore.getUser('u1'))
      expect(vm.dayDeletePreselectKey).toBe('unknown')
      vm.exitSelectionMode()

      // applyDaySelection: unknown day key should select users with empty/invalid lastTime
      vm.openDaySelector()
      vm.applyDaySelection(['unknown'])
      expect(vm.selectedUserIds).toEqual(['u1'])

      // selectionMode blocks long-press.
      vm.selectionMode = true
      vm.startLongPress(chatStore.getUser('u1'), new MouseEvent('mousedown', { clientX: 1, clientY: 1 }) as any)
      vi.advanceTimersByTime(500)
      expect(vm.showContextMenu).toBe(false)
      vm.selectionMode = false

      // Context menu bounds adjustment (mouse path + x/y clamp).
      vm.startLongPress(chatStore.getUser('u1'), new MouseEvent('mousedown', { clientX: window.innerWidth, clientY: window.innerHeight }) as any)
      vi.advanceTimersByTime(500)
      await flushAsync()
      expect(vm.showContextMenu).toBe(true)
      expect(vm.contextMenuPos.x).toBeLessThan(window.innerWidth)
      expect(vm.contextMenuPos.y).toBeLessThan(window.innerHeight)

      // Right-click menu bounds adjustment too.
      vm.handleContextMenu(chatStore.getUser('u1'), new MouseEvent('contextmenu', { clientX: window.innerWidth, clientY: window.innerHeight }) as any)
      expect(vm.contextMenuPos.x).toBeLessThan(window.innerWidth)
      expect(vm.contextMenuPos.y).toBeLessThan(window.innerHeight)

      // Swipe progress: clear reset timer and stop animation.
      // Ensure menu is closed; swipe handlers early-return when context menu is open.
      vm.showContextMenu = false
      expect(swipeCalls).toHaveLength(1)
      const opts = swipeCalls[0]!
      const clearSpy = vi.spyOn(window, 'clearTimeout')
      vm.listTranslateX = 66
      vm.resetListTranslateX() // creates resetTranslateTimer and sets isAnimating=true
      vm.isAnimating = true
      opts.onSwipeProgress?.(20, 0)
      expect(clearSpy).toHaveBeenCalled()
      expect(vm.isAnimating).toBe(false)

      // Swipe finish with menu open clears timer branch
      vm.showContextMenu = true
      vm.listTranslateX = 66
      vm.resetListTranslateX()
      opts.onSwipeFinish?.(0, 0, false)
      expect(vm.listTranslateX).toBe(0)

      // onSwipeEnd selectionMode early return
      vm.selectionMode = true
      chatStore.activeTab = 'history'
      opts.onSwipeEnd?.('left')
      expect(chatStore.activeTab).toBe('history')
      vm.selectionMode = false

      // Global favorite add/remove branches + nickname fallback (nickname is empty -> uses name).
      vm.showContextMenu = true
      await vm.handleToggleGlobalFavorite(chatStore.getUser('u1'))
      expect(addSpy).toHaveBeenCalledWith('me', 'u1', 'U1')

      favoriteStore.allFavorites = [{ id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'U1' } as any]
      await vm.handleToggleGlobalFavorite(chatStore.getUser('u1'))
      expect(removeSpy).toHaveBeenCalledWith('me', 'u1')

      // currentUser missing -> early return
      userStore.currentUser = null as any
      await vm.handleToggleGlobalFavorite(chatStore.getUser('u1'))

      // onCheckOnlineResult only updates when data.data exists
      vm.showOnlineStatusDialog = false
      window.dispatchEvent(new CustomEvent('check-online-result', { detail: { data: { IF_Online: '1', TimeAll: 't' } } }))
      await flushAsync()
      expect(vm.showOnlineStatusDialog).toBe(true)

      vm.showOnlineStatusDialog = false
      window.dispatchEvent(new CustomEvent('check-online-result', { detail: { nope: true } }))
      await flushAsync()
      expect(vm.showOnlineStatusDialog).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })
})
