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
  getChatImages: vi.fn().mockResolvedValue([]),
  reuploadHistoryImage: vi.fn()
}))

vi.mock('@/composables/useInteraction', async () => {
  const { reactive, ref } = await import('vue')
  return {
    useSwipeAction: () => ({ coordsStart: reactive({ x: 0, y: 0 }), isSwiping: ref(false) })
  }
})

import ChatRoomView from '@/views/ChatRoomView.vue'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'
import { useSystemConfigStore } from '@/stores/systemConfig'
import * as mediaApi from '@/api/media'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
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

const createStubs = (opts?: { isAtBottom?: boolean }) => {
  const scrollToBottom = vi.fn()
  const scrollToTop = vi.fn()
  const getIsAtBottom = vi.fn().mockReturnValue(opts?.isAtBottom ?? true)

  const MessageList = {
    name: 'MessageList',
    inheritAttrs: false,
    props: ['messages', 'isTyping', 'loadingMore', 'canLoadMore', 'floatingBottomOffsetPx'],
    emits: ['load-more', 'close-all-panels', 'retry'],
    methods: { scrollToBottom, scrollToTop, getIsAtBottom },
    template: '<div class="message-list-stub" v-bind=\"$attrs\"></div>'
  }

  return { MessageList, scrollToBottom, scrollToTop, getIsAtBottom }
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('views/ChatRoomView.vue (branch coverage gaps)', () => {
  it('redirects to /identity when no currentUser (onMounted early return)', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    const pushSpy = vi.spyOn(router, 'push')
    await router.push('/chat')
    await router.isReady()

    const stubs = createStubs()
    mount(ChatRoomView, {
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
    expect(pushSpy).toHaveBeenCalledWith('/identity')
    expect(wsMocks.connect).not.toHaveBeenCalled()
  })

  it('covers initLayoutResizeObserver branches (undefined vs callback triggers scrollToBottom)', async () => {
    const originalRO = (globalThis as any).ResizeObserver

    try {
      // Branch: ResizeObserver is undefined -> returns early.
      ;(globalThis as any).ResizeObserver = undefined

      const pinia = createPinia()
      setActivePinia(pinia)
      const router = createTestRouter()
      await router.push('/chat')
      await router.isReady()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const mediaStore = useMediaStore()
      vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

      const stubs = createStubs({ isAtBottom: true })
      mount(ChatRoomView, {
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

      // Branch: ResizeObserver exists and callback invokes scrollToBottom only when at bottom.
      let roCallback: (() => void) | null = null
      const disconnectSpy = vi.fn()
      const observeSpy = vi.fn()
      class FakeResizeObserver {
        constructor(cb: () => void) {
          roCallback = cb
        }
        observe = observeSpy
        disconnect = disconnectSpy
      }

      ;(globalThis as any).ResizeObserver = FakeResizeObserver

      const pinia2 = createPinia()
      setActivePinia(pinia2)
      const router2 = createTestRouter()
      await router2.push('/chat')
      await router2.isReady()

      const userStore2 = useUserStore()
      userStore2.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const chatStore2 = useChatStore()
      chatStore2.wsConnected = true
      chatStore2.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

      const mediaStore2 = useMediaStore()
      vi.spyOn(mediaStore2, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore2, 'loadCachedImages').mockResolvedValue(undefined)

      const stubs2 = createStubs({ isAtBottom: true })
      const wrapper2 = mount(ChatRoomView, {
        global: {
          plugins: [pinia2, router2],
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
            MessageList: stubs2.MessageList
          }
        }
      })
      await flushAsync()

      expect(observeSpy).toHaveBeenCalled()
      roCallback?.()
      expect(stubs2.scrollToBottom).toHaveBeenCalled()

      // When refs are missing, observe() should be skipped.
      const prevObserveCalls = observeSpy.mock.calls.length
      ;(wrapper2.vm as any).chatHeaderWrapRef = null
      ;(wrapper2.vm as any).bottomDockRef = null
      ;(wrapper2.vm as any).initLayoutResizeObserver()
      expect(observeSpy.mock.calls.length).toBe(prevObserveCalls)

      wrapper2.unmount()
      expect(disconnectSpy).toHaveBeenCalled()

      // When user is not at bottom, callback should NOT force scroll.
      const pinia3 = createPinia()
      setActivePinia(pinia3)
      const router3 = createTestRouter()
      await router3.push('/chat')
      await router3.isReady()

      const userStore3 = useUserStore()
      userStore3.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

      const mediaStore3 = useMediaStore()
      vi.spyOn(mediaStore3, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore3, 'loadCachedImages').mockResolvedValue(undefined)

      const stubs3 = createStubs({ isAtBottom: false })
      const wrapper3 = mount(ChatRoomView, {
        global: {
          plugins: [pinia3, router3],
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
            MessageList: stubs3.MessageList
          }
        }
      })
      await flushAsync()
      stubs3.scrollToBottom.mockClear()
      roCallback?.()
      expect(stubs3.scrollToBottom).not.toHaveBeenCalled()

      wrapper3.unmount()
    } finally {
      ;(globalThis as any).ResizeObserver = originalRO
    }
  })

  it('covers sidebar select same-user branch and some early-return handlers', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    const replaceSpy = vi.spyOn(router, 'replace')
    await router.push('/chat/u1')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createStubs({ isAtBottom: true })
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

    // same user -> close sidebar only (no replace/enterChat call)
    ;(wrapper.vm as any).showSidebar = true
    ;(wrapper.vm as any).handleSidebarSelect({ id: 'u1' })
    expect((wrapper.vm as any).showSidebar).toBe(false)
    expect(chatMocks.enterChat).not.toHaveBeenCalled()
    expect(replaceSpy).not.toHaveBeenCalled()

    // different user -> switch chat + update route param
    ;(wrapper.vm as any).showSidebar = true
    ;(wrapper.vm as any).handleSidebarSelect({ id: 'u2', name: 'U2', nickname: 'U2' })
    expect((wrapper.vm as any).showSidebar).toBe(false)
    expect(chatMocks.enterChat).toHaveBeenCalled()
    expect(replaceSpy).toHaveBeenCalledWith('/chat/u2')

    // handleRetry early-return when no currentChatUser
    chatStore.exitChat()
    await (wrapper.vm as any).handleRetry({ tid: '1', content: 'x' })
    expect(messageMocks.retryMessage).not.toHaveBeenCalled()

    // handleSendMedia early-return when no currentChatUser
    ;(wrapper.vm as any).handleSendMedia({ url: 'x', type: 'image' })
    expect(messageMocks.sendImage).not.toHaveBeenCalled()

    // ws disconnected branch
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    chatStore.wsConnected = false
    ;(wrapper.vm as any).handleSendMedia({ url: 'x', type: 'image' })
    expect(toastShow).toHaveBeenCalledWith('连接已断开，请刷新页面重试')
  })

  it('covers route-param enterChat vs redirect-to-list branches on mount', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    const pushSpy = vi.spyOn(router, 'push')

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    // user exists -> enterChat(user,true)
    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.upsertUser({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    chatStore.historyUserIds = ['u1']

    await router.push('/chat/u1')
    await router.isReady()

    const stubs = createStubs({ isAtBottom: true })
    mount(ChatRoomView, {
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
    expect(chatMocks.enterChat).toHaveBeenCalled()

    // user missing -> redirect to /list
    const pinia2 = createPinia()
    setActivePinia(pinia2)
    const router2 = createTestRouter()
    const pushSpy2 = vi.spyOn(router2, 'push')
    await router2.push('/chat/missing')
    await router2.isReady()

    const userStore2 = useUserStore()
    userStore2.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const mediaStore2 = useMediaStore()
    vi.spyOn(mediaStore2, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore2, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs2 = createStubs({ isAtBottom: true })
    mount(ChatRoomView, {
      global: {
        plugins: [pinia2, router2],
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
          MessageList: stubs2.MessageList
        }
      }
    })
    await flushAsync()
    expect(pushSpy2).toHaveBeenCalledWith('/list')
    expect(pushSpy).not.toHaveBeenCalledWith('/list')
  })

  it('covers confirmPreviewUpload failure message branches and handlePreview rawUrl branches', async () => {
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
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue(9006 as any)

    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'getMessages').mockReturnValue([
      { tid: '1', isImage: true, isVideo: false, imageUrl: 'http://x/img.png', content: '' } as any,
      { tid: '2', isImage: false, isVideo: true, videoUrl: 'http://x/v.mp4', content: '' } as any,
      { tid: '3', isImage: true, isVideo: false, content: 'http://x/from-content.png' } as any
    ])

    const stubs = createStubs({ isAtBottom: true })
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

    // confirmPreviewUpload: failure message branches (msg, error, default)
    ;(wrapper.vm as any).previewTarget = { url: '/upload/images/a.png', type: 'image' }
    vi.mocked(mediaApi.reuploadHistoryImage).mockResolvedValueOnce({ state: 'NO', msg: 'm' } as any)
    await (wrapper.vm as any).confirmPreviewUpload()
    expect(toastShow).toHaveBeenCalledWith('重新上传失败: m')

    vi.mocked(mediaApi.reuploadHistoryImage).mockResolvedValueOnce({ state: 'NO', error: 'e' } as any)
    await (wrapper.vm as any).confirmPreviewUpload()
    expect(toastShow).toHaveBeenCalledWith('重新上传失败: e')

    vi.mocked(mediaApi.reuploadHistoryImage).mockResolvedValueOnce({ state: 'NO' } as any)
    await (wrapper.vm as any).confirmPreviewUpload()
    expect(toastShow).toHaveBeenCalledWith('重新上传失败: 未知错误')

    // handlePreview: rawUrl chooses imageUrl/videoUrl/content, and found toggles list mode.
    ;(wrapper.vm as any).handlePreview({ detail: { url: 'http://x/img.png', type: 'image' } })
    await flushAsync()
    expect((wrapper.vm as any).previewMediaList.length).toBeGreaterThan(0)

    ;(wrapper.vm as any).handlePreview({ detail: { url: 'http://x/not-found.png', type: 'image' } })
    await flushAsync()
    expect((wrapper.vm as any).previewMediaList).toEqual([])
  })

  it('handleFileChange early returns when no file or no currentUser', async () => {
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

    await (wrapper.vm as any).handleFileChange({ target: { files: [] } } as any)
    expect(uploadMocks.uploadFile).not.toHaveBeenCalled()

    userStore.currentUser = null as any
    const file = new File(['x'], 'a.png', { type: 'image/png' })
    await (wrapper.vm as any).handleFileChange({ target: { files: [file] } } as any)
    expect(uploadMocks.uploadFile).not.toHaveBeenCalled()
  })

  it('handleRetry does not force scroll when message list is not at bottom', async () => {
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

    const stubs = createStubs({ isAtBottom: false })
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
    // Immediate watch on currentChatUser triggers a scroll once during mount.
    stubs.scrollToBottom.mockClear()

    await (wrapper.vm as any).handleRetry({ tid: '1', content: 'x' })
    expect(messageMocks.retryMessage).toHaveBeenCalled()
    expect(stubs.scrollToBottom).not.toHaveBeenCalled()
  })
})
