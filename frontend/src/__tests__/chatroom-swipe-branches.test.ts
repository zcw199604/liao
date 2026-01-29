import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const toastShow = vi.fn()

const wsMocks = {
  connect: vi.fn(),
  setScrollToBottom: vi.fn()
}

const chatMocks = {
  toggleFavorite: vi.fn(),
  enterChat: vi.fn(),
  startMatch: vi.fn().mockReturnValue(true)
}

const uploadMocks = {
  uploadFile: vi.fn(),
  getMediaUrl: (input: string) => input
}

const messageMocks = {
  sendText: vi.fn(),
  sendImage: vi.fn(),
  sendVideo: vi.fn(),
  retryMessage: vi.fn(),
  sendTypingStatus: vi.fn()
}

const swipeCalls = vi.hoisted(() => [] as Array<{ opts: any; coordsStart: { x: number; y: number } }>)

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => wsMocks
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => chatMocks
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

vi.mock('@/composables/useMessage', () => ({
  useMessage: () => messageMocks
}))

vi.mock('@/api/media', () => ({
  getImgServerAddress: vi.fn().mockResolvedValue({ state: 'OK', msg: { server: '' } }),
  updateImgServerAddress: vi.fn().mockResolvedValue({ state: 'OK' }),
  getCachedImages: vi.fn().mockResolvedValue([]),
  getAllUploadImages: vi.fn().mockResolvedValue({ data: [], total: 0, page: 1, pageSize: 20, totalPages: 0 }),
  getChatImages: vi.fn(),
  reuploadHistoryImage: vi.fn()
}))

vi.mock('@/composables/useInteraction', async () => {
  const { reactive, ref } = await import('vue')
  return {
    useSwipeAction: (_target: any, opts: any) => {
      const coordsStart = reactive({ x: 0, y: 0 })
      swipeCalls.push({ opts, coordsStart })
      return { coordsStart, isSwiping: ref(false) }
    }
  }
})

import ChatRoomView from '@/views/ChatRoomView.vue'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

const createStubs = () => {
  const scrollToBottom = vi.fn()
  const scrollToTop = vi.fn()
  const getIsAtBottom = vi.fn().mockReturnValue(true)

  const MessageList = {
    name: 'MessageList',
    inheritAttrs: false,
    props: ['messages', 'isTyping', 'loadingMore', 'canLoadMore', 'floatingBottomOffsetPx'],
    emits: ['load-more', 'close-all-panels', 'retry'],
    methods: { scrollToBottom, scrollToTop, getIsAtBottom },
    template: '<div class="message-list-stub" v-bind=\"$attrs\"></div>'
  }

  return {
    MessageList,
    scrollToBottom,
    scrollToTop,
    getIsAtBottom
  }
}

const createTestRouter = () => {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/list', component: { template: '<div />' } },
      { path: '/identity', component: { template: '<div />' } },
      { path: '/chat/:userId?', component: ChatRoomView }
    ]
  })
}

describe('views/ChatRoomView.vue (swipe branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    swipeCalls.splice(0, swipeCalls.length)
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('covers edge-back swipe and drawer-close swipe callbacks', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    const pushSpy = vi.spyOn(router, 'push')
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createStubs()
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: true,
          Dialog: true,
          Toast: true,
          MediaPreview: true,
          MediaTile: true,
          ChatSidebar: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          ChatHeader: true,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()
    await flushAsync()

    expect(swipeCalls.length).toBeGreaterThanOrEqual(2)
    const pageSwipe = swipeCalls[0]!
    const drawerSwipe = swipeCalls[1]!

    // Edge-back swipe progress: start outside hot zone -> no move
    pageSwipe.coordsStart.x = 999
    pageSwipe.opts.onSwipeProgress?.(120, 0)
    expect((wrapper.vm as any).pageTranslateX).toBe(0)

    // Edge-back swipe progress: vertical dominated -> ignore
    pageSwipe.coordsStart.x = 0
    pageSwipe.opts.onSwipeProgress?.(10, 100)
    expect((wrapper.vm as any).pageTranslateX).toBe(0)

    // Edge-back swipe progress: right swipe within hot zone -> follow (clamped)
    pageSwipe.opts.onSwipeProgress?.(999, 0)
    expect((wrapper.vm as any).pageTranslateX).toBe(150)

    // Edge-back swipe end: triggers handleBack() when threshold met
    ;(wrapper.vm as any).pageTranslateX = 120
    pageSwipe.opts.onSwipeEnd?.('right')
    await flushAsync()
    expect(pushSpy).toHaveBeenCalledWith('/list')

    // Swipe finish: triggered -> early return
    pageSwipe.opts.onSwipeFinish?.(0, 0, true)

    // Drawer swipe progress: showSidebar false -> ignore
    ;(wrapper.vm as any).showSidebar = false
    drawerSwipe.opts.onSwipeProgress?.(-100, 0)
    expect((wrapper.vm as any).sidebarTranslateX).toBe(0)

    // Drawer swipe progress: showSidebar true but starts away from right edge -> ignore
    ;(wrapper.vm as any).showSidebar = true
    await flushAsync()
    const sidebarEl = wrapper.find('.absolute.inset-y-0.left-0')?.element as HTMLElement
    if (sidebarEl) {
      ;(sidebarEl as any).getBoundingClientRect = () => ({ right: 300, width: 300, height: 500, left: 0, top: 0, bottom: 500 })
    }
    drawerSwipe.coordsStart.x = 0
    drawerSwipe.opts.onSwipeProgress?.(-100, 0)
    expect((wrapper.vm as any).sidebarTranslateX).toBe(0)

    // Drawer swipe progress: valid close gesture updates translate
    drawerSwipe.coordsStart.x = 300
    drawerSwipe.opts.onSwipeProgress?.(-60, 0)
    expect((wrapper.vm as any).sidebarTranslateX).toBe(-60)

    // Drawer swipe end: close when translate beyond threshold
    ;(wrapper.vm as any).sidebarTranslateX = -120
    drawerSwipe.opts.onSwipeEnd?.('left')
    await flushAsync()
    expect((wrapper.vm as any).showSidebar).toBe(false)
  })

  it('covers clear/reload and loadMore count branches + early returns', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'clearHistory').mockImplementation(() => {})

    const stubs = createStubs()
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: true,
          Dialog: true,
          Toast: true,
          MediaPreview: true,
          MediaTile: true,
          ChatSidebar: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          ChatHeader: true,
          MessageList: stubs.MessageList
        }
      }
    })
    await flushAsync()

    // handleClearAndReload early-return when missing currentChatUser
    chatStore.exitChat()
    ;(wrapper.vm as any).handleClearAndReload()
    expect((wrapper.vm as any).showClearDialog).toBe(false)

    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    ;(wrapper.vm as any).handleClearAndReload()
    expect((wrapper.vm as any).showClearDialog).toBe(true)

    // executeClearAndReload: count > 0 branch
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValueOnce(2 as any)
    await (wrapper.vm as any).executeClearAndReload()
    await flushAsync()
    expect(stubs.scrollToBottom).toHaveBeenCalled()

    // handleLoadMore: count > 0 branch triggers scrollToTop
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValueOnce(1 as any)
    await (wrapper.vm as any).handleLoadMore()
    await flushAsync()
    expect(stubs.scrollToTop).toHaveBeenCalled()

    // handleLoadMore: count == 0 branch
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValueOnce(0 as any)
    await (wrapper.vm as any).handleLoadMore()
    expect(toastShow).toHaveBeenCalledWith('没有更多历史消息了')

    // handleLoadMore: count < 0 branch
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValueOnce(-1 as any)
    await (wrapper.vm as any).handleLoadMore()
    expect(toastShow).toHaveBeenCalledWith('加载失败')
  })
})
