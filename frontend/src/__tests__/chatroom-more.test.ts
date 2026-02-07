import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const toastShow = vi.fn()

const messageMocks = {
  sendText: vi.fn(),
  sendImage: vi.fn(),
  sendVideo: vi.fn(),
  retryMessage: vi.fn(),
  sendTypingStatus: vi.fn()
}

const uploadMocks = {
  uploadFile: vi.fn(),
  getMediaUrl: (input: string) => input
}

const wsMocks = {
  connect: vi.fn(),
  setScrollToBottom: vi.fn()
}

const chatMocks = {
  toggleFavorite: vi.fn(),
  enterChat: vi.fn(),
  startMatch: vi.fn()
}

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useMessage', () => ({
  useMessage: () => messageMocks
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => wsMocks
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => chatMocks
}))

vi.mock('@/api/media', () => ({
  getChatImages: vi.fn(),
  reuploadHistoryImage: vi.fn()
}))

import ChatRoomView from '@/views/ChatRoomView.vue'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useDouyinStore } from '@/stores/douyin'
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
      { path: '/identity', component: { template: '<div />' } },
      { path: '/list', component: { template: '<div />' } },
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

  const ChatInput = {
    name: 'ChatInput',
    props: ['modelValue', 'disabled', 'wsConnected'],
    emits: ['update:modelValue', 'send', 'show-upload', 'show-emoji', 'typing-start', 'typing-end', 'start-match'],
    template: '<div data-testid=\"chat-input\"></div>'
  }

  const UploadMenu = {
    name: 'UploadMenu',
    props: ['visible', 'uploadedMedia', 'canOpenChatHistory'],
    emits: ['update:visible', 'send', 'upload-file', 'open-chat-history', 'open-all-uploads', 'open-mt-photo', 'open-douyin-favorite-authors'],
    template: '<div data-testid=\"upload-menu\"></div>'
  }

  const EmojiPanel = {
    name: 'EmojiPanel',
    props: ['visible'],
    emits: ['update:visible', 'select'],
    template: '<div data-testid=\"emoji-panel\"></div>'
  }

  const ChatHeader = {
    name: 'ChatHeader',
    props: ['user', 'connected'],
    emits: ['back', 'toggle-favorite', 'clear-and-reload', 'toggle-sidebar'],
    template: '<div data-testid=\"chat-header\"></div>'
  }

  return {
    MessageList,
    ChatInput,
    UploadMenu,
    EmojiPanel,
    ChatHeader,
    scrollToBottom,
    scrollToTop,
    getIsAtBottom
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('views/ChatRoomView.vue (more coverage)', () => {
  it('send text: ignores empty, blocks on disconnected, and scrolls when at bottom', async () => {
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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    ;(wrapper.vm as any).inputText = '   '
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('send')
    await flushAsync()
    expect(messageMocks.sendText).not.toHaveBeenCalled()

    ;(wrapper.vm as any).inputText = 'hi'
    chatStore.wsConnected = false
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('send')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('连接已断开，请刷新页面重试')

    chatStore.wsConnected = true
    ;(wrapper.vm as any).inputText = 'hello'
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('send')
    await flushAsync()
    await flushAsync()

    expect(messageMocks.sendText).toHaveBeenCalledWith('hello', chatStore.currentChatUser)
    expect((wrapper.vm as any).inputText).toBe('')
    expect(stubs.scrollToBottom).toHaveBeenCalled()
  })

  it('retry: blocks on disconnected and retries when connected', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false
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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    const msg = { tid: '1', content: 'x' } as any
    wrapper.findComponent({ name: 'MessageList' }).vm.$emit('retry', msg)
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('连接已断开，请刷新页面重试')
    expect(messageMocks.retryMessage).not.toHaveBeenCalled()

    chatStore.wsConnected = true
    wrapper.findComponent({ name: 'MessageList' }).vm.$emit('retry', msg)
    await flushAsync()
    await flushAsync()
    expect(messageMocks.retryMessage).toHaveBeenCalledWith(msg)
    expect(stubs.scrollToBottom).toHaveBeenCalled()
  })

  it('renders sidebar + history media modal conditional branches', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    const vm = wrapper.vm as any

    // sidebar branches (v-if + style ternary)
    expect(wrapper.find('chat-sidebar-stub').exists()).toBe(false)
    vm.pageTranslateX = 20
    vm.isPageAnimating = true
    vm.sidebarTranslateX = -10
    vm.isSidebarAnimating = true
    vm.showSidebar = true
    await flushAsync()
    expect(wrapper.find('chat-sidebar-stub').exists()).toBe(true)
    expect(wrapper.find('.backdrop-blur-sm').exists()).toBe(true)

    vm.isPageAnimating = false
    vm.isSidebarAnimating = false
    vm.showSidebar = false
    await flushAsync()
    expect(wrapper.find('chat-sidebar-stub').exists()).toBe(false)

    // history modal branches: loading -> empty -> list
    vm.showHistoryMediaModal = true
    vm.historyMediaLoading = true
    await flushAsync()
    expect(wrapper.text()).toContain('加载中...')

    vm.historyMediaLoading = false
    vm.historyMedia = []
    await flushAsync()
    expect(wrapper.text()).toContain('暂无聊天历史图片')

    vm.historyMedia = [
      { url: 'http://x/1.png', type: 'image' },
      { url: 'http://x/2.mp4', type: 'video' },
      { url: 'http://x/3.bin', type: 'file' }
    ]
    await flushAsync()
    expect(wrapper.findAll('media-tile-stub')).toHaveLength(3)

    vm.showHistoryMediaModal = false
    await flushAsync()
    expect(wrapper.findAll('media-tile-stub')).toHaveLength(0)
  })

  it('toggles panels, emoji insert, typing status and open upload menu side-effects', async () => {
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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    expect((wrapper.vm as any).showUploadMenu).toBe(false)
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('show-upload')
    await flushAsync()
    await flushAsync()
    expect((wrapper.vm as any).showUploadMenu).toBe(true)
    expect(mediaStore.loadImgServer).toHaveBeenCalled()
    expect(mediaStore.loadCachedImages).toHaveBeenCalled()

    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('show-emoji')
    await flushAsync()
    expect((wrapper.vm as any).showEmojiPanel).toBe(true)
    expect((wrapper.vm as any).showUploadMenu).toBe(false)

    ;(wrapper.vm as any).inputText = 'a'
    wrapper.findComponent({ name: 'EmojiPanel' }).vm.$emit('select', '😀')
    await flushAsync()
    expect((wrapper.vm as any).inputText).toBe('a😀')
    expect((wrapper.vm as any).showEmojiPanel).toBe(false)

    chatStore.wsConnected = false
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('typing-start')
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('typing-end')
    expect(messageMocks.sendTypingStatus).not.toHaveBeenCalled()

    chatStore.wsConnected = true
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('typing-start')
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('typing-end')
    expect(messageMocks.sendTypingStatus).toHaveBeenCalledWith(true, chatStore.currentChatUser)
    expect(messageMocks.sendTypingStatus).toHaveBeenCalledWith(false, chatStore.currentChatUser)

    wrapper.findComponent({ name: 'MessageList' }).vm.$emit('close-all-panels')
    await flushAsync()
    expect((wrapper.vm as any).showUploadMenu).toBe(false)
    expect((wrapper.vm as any).showEmojiPanel).toBe(false)

    const douyinStore = useDouyinStore()
    ;(wrapper.vm as any).showUploadMenu = true
    wrapper.findComponent({ name: 'UploadMenu' }).vm.$emit('open-douyin-favorite-authors')
    await flushAsync()
    expect((wrapper.vm as any).showUploadMenu).toBe(false)
    expect(douyinStore.showModal).toBe(true)
    expect(douyinStore.entryMode).toBe('favorites')
    expect(douyinStore.favoritesTab).toBe('users')
  })

  it('uploads file and handles send media branch', async () => {
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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    const inputEl = wrapper.get('input[type=\"file\"]').element as HTMLInputElement
    const clickSpy = vi.spyOn(inputEl, 'click').mockImplementation(() => {})
    wrapper.findComponent({ name: 'UploadMenu' }).vm.$emit('upload-file')
    expect(clickSpy).toHaveBeenCalled()

    const img = new File(['x'], 'a.png', { type: 'image/png' })
    uploadMocks.uploadFile.mockResolvedValueOnce({ url: 'u', type: 'image' })
    await (wrapper.vm as any).handleFileChange({ target: { files: [img], value: 'x' } } as any)
    expect(toastShow).toHaveBeenCalledWith('图片上传成功')

    uploadMocks.uploadFile.mockResolvedValueOnce({ url: 'u', type: 'video' })
    await (wrapper.vm as any).handleFileChange({ target: { files: [img], value: 'x' } } as any)
    expect(toastShow).toHaveBeenCalledWith('视频上传成功')

    uploadMocks.uploadFile.mockResolvedValueOnce(null)
    await (wrapper.vm as any).handleFileChange({ target: { files: [img], value: 'x' } } as any)
    expect(toastShow).toHaveBeenCalledWith('文件上传失败')

    ;(wrapper.vm as any).showUploadMenu = true
    ;(wrapper.vm as any).handleSendMedia({ url: 'v', type: 'video', localFilename: 'v.mp4' })
    expect(messageMocks.sendVideo).toHaveBeenCalled()
    expect((wrapper.vm as any).showUploadMenu).toBe(false)

    ;(wrapper.vm as any).handleSendMedia({ url: 'i', type: 'image', localFilename: 'i.png' })
    expect(messageMocks.sendImage).toHaveBeenCalled()
  })

  it('loads chat history media and re-uploads preview into uploaded list', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1', unreadCount: 2 } as any)

    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(2 as any)
    vi.spyOn(messageStore, 'clearHistory').mockImplementation(() => {})

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)
    const addUploadedSpy = vi.spyOn(mediaStore, 'addUploadedMedia')
    mediaStore.imgServer = ''

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9007' as any)

    vi.mocked(mediaApi.getChatImages).mockResolvedValue(['/x/a.png', '/x/b.mp4'] as any)
    vi.mocked(mediaApi.reuploadHistoryImage).mockResolvedValue({ state: 'OK', msg: '2026/01/a.png' } as any)

    const mtPhotoStore = useMtPhotoStore()
    vi.spyOn(mtPhotoStore, 'open').mockResolvedValue(undefined)

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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()
    await flushAsync()

    await (wrapper.vm as any).handleOpenChatHistory()
    await flushAsync()
    await flushAsync()

    expect((wrapper.vm as any).showHistoryMediaModal).toBe(true)
    expect((wrapper.vm as any).historyMediaLoading).toBe(false)
    expect((wrapper.vm as any).historyMedia.length).toBe(2)

    vi.spyOn(mediaStore, 'loadImgServer').mockImplementation(async () => {
      mediaStore.imgServer = 'img.local'
    })

    await (wrapper.vm as any).openPreviewUpload({ url: 'http://localhost:8080/upload/images/2026/01/a.png', type: 'image' })
    await (wrapper.vm as any).confirmPreviewUpload()
    await flushAsync()

    expect(addUploadedSpy).toHaveBeenCalled()
    expect(addUploadedSpy.mock.calls[0]?.[0]?.url).toContain('http://img.local:9007/img/Upload/2026/01/a.png')
    expect((wrapper.vm as any).showMediaPreview).toBe(false)
    expect((wrapper.vm as any).showUploadMenu).toBe(true)
  })

  it('preview-media event toggles gallery list only when clicked url exists in message media list', async () => {
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

    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'getMessages').mockReturnValue([
      { isImage: true, isVideo: false, imageUrl: 'http://img/1.png', content: '', time: '', fromuser: {} } as any
    ])

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
          UploadMenu: stubs.UploadMenu,
          EmojiPanel: stubs.EmojiPanel,
          ChatInput: stubs.ChatInput,
          ChatHeader: stubs.ChatHeader,
          MessageList: stubs.MessageList
        }
      }
    })

    await flushAsync()

    window.dispatchEvent(new CustomEvent('preview-media', { detail: { url: 'http://img/1.png', type: 'image' } }))
    await flushAsync()

    expect((wrapper.vm as any).showMediaPreview).toBe(true)
    expect((wrapper.vm as any).previewMediaList.length).toBe(1)

    window.dispatchEvent(new CustomEvent('preview-media', { detail: { url: 'http://img/404.png', type: 'image' } }))
    await flushAsync()
    expect((wrapper.vm as any).previewMediaList.length).toBe(0)
  })
})
