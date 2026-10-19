import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const toastShow = vi.fn()

const wsMocks = {
  connect: vi.fn(),
  setScrollToBottom: vi.fn(),
  reconnectNow: vi.fn()
}

const chatMocks = {
  loadUsers: vi.fn().mockResolvedValue(undefined),
  toggleFavorite: vi.fn(),
  enterChat: vi.fn(),
  startMatch: vi.fn()
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
import { useMediaStore } from '@/stores/media'
import { useMessageStore } from '@/stores/message'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useUserStore } from '@/stores/user'
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

const createMessageListStub = (opts?: { isAtBottom?: boolean }) => {
  const scrollToBottom = vi.fn()
  const scrollToTop = vi.fn()
  const captureScrollAnchor = vi.fn().mockReturnValue({ scrollHeight: 100, scrollTop: 0 })
  const restoreScrollAnchor = vi.fn()
  const getIsAtBottom = vi.fn().mockReturnValue(opts?.isAtBottom ?? true)

  const MessageList = {
    name: 'MessageList',
    inheritAttrs: false,
    props: ['messages', 'isTyping', 'loadingMore', 'canLoadMore', 'floatingBottomOffsetPx'],
    emits: ['load-more', 'close-all-panels', 'retry', 'preview'],
    methods: { scrollToBottom, scrollToTop, getIsAtBottom, captureScrollAnchor, restoreScrollAnchor },
    template: '<div class=\"message-list-stub\" v-bind=\"$attrs\"></div>'
  }

  return { MessageList, scrollToBottom, scrollToTop, getIsAtBottom }
}

const DialogStub = {
  name: 'Dialog',
  props: ['visible'],
  emits: ['update:visible', 'confirm'],
  template: '<div v-if=\"visible\" class=\"dialog-stub\"><slot /></div>'
}

const TeleportStub = {
  name: 'teleport',
  template: '<div class=\"teleport-stub\"><slot /></div>'
}

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type'],
  emits: ['click'],
  template: `<div class="media-tile-stub" @click="$emit('click')"><slot name="file" /></div>`
}

beforeEach(() => {
  vi.clearAllMocks()
  swipeCalls.splice(0, swipeCalls.length)
  localStorage.clear()
  setActivePinia(createPinia())
  chatMocks.startMatch.mockReturnValue(true)
})

describe('views/ChatRoomView.vue (panels/upload/remaining branches)', () => {
  it('toggle upload/emoji panels and only scrolls when MessageList is at bottom', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    const loadImgSpy = vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    const loadCachedSpy = vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    // Toggle upload -> loads server + cached images (currentUser exists) and scrolls to bottom.
    await (wrapper.vm as any).handleToggleUpload()
    expect(loadImgSpy).toHaveBeenCalled()
    expect(loadCachedSpy).toHaveBeenCalledWith('me')
    expect(stubs.scrollToBottom).toHaveBeenCalled()

    // Toggle emoji -> hides upload and scrolls to bottom.
    await (wrapper.vm as any).handleToggleEmoji()
    expect(stubs.scrollToBottom).toHaveBeenCalled()

    // Close panels -> scrolls.
    await (wrapper.vm as any).handleCloseAllPanels()
    expect(stubs.scrollToBottom).toHaveBeenCalled()

    // Emoji select appends and closes panel.
    ;(wrapper.vm as any).showEmojiPanel = true
    await (wrapper.vm as any).handleEmojiSelect('🙂')
    expect((wrapper.vm as any).inputText).toContain('🙂')

    // When user is missing, toggle upload does not call loadImgServer/loadCachedImages.
    userStore.currentUser = null as any
    loadImgSpy.mockClear()
    await (wrapper.vm as any).handleToggleUpload()
    expect(loadImgSpy).not.toHaveBeenCalled()
  })

  it('covers handleSend/handleRetry/typing branches for wsConnected true/false', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    chatStore.wsConnected = false

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: false })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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
    // currentChatUser watch (immediate) may scroll on mount; clear before assertions for handler branches.
    stubs.scrollToBottom.mockClear()

    ;(wrapper.vm as any).inputText = 'hi'
    await (wrapper.vm as any).handleSend()
    expect(toastShow).toHaveBeenCalledWith('连接已断开，正在重连…')
    expect(messageMocks.sendText).not.toHaveBeenCalled()

    chatStore.wsConnected = true
    await (wrapper.vm as any).handleSend()
    expect(messageMocks.sendText).toHaveBeenCalled()

    // 自己发送消息后总是回到底部（即便当前不在底部）
    expect(stubs.scrollToBottom).toHaveBeenCalledWith(true)
    stubs.scrollToBottom.mockClear()

    // typing status only sent when connected and has currentChatUser
    ;(wrapper.vm as any).handleTypingStart()
    ;(wrapper.vm as any).handleTypingEnd()
    expect(messageMocks.sendTypingStatus).toHaveBeenCalledTimes(2)

    chatStore.wsConnected = false
    messageMocks.sendTypingStatus.mockClear()
    ;(wrapper.vm as any).handleTypingStart()
    expect(messageMocks.sendTypingStatus).not.toHaveBeenCalled()

    // retry connected/disconnected
    await (wrapper.vm as any).handleRetry({ tid: 't1', content: 'x' })
    expect(toastShow).toHaveBeenCalledWith('连接已断开，正在重连…')
  })

  it('handleFileChange covers upload success (image/video/file) and failure branches', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    const file = new File(['x'], 'a.bin', { type: 'application/octet-stream' })

    // 单个文件：上传成功后直接发送给当前会话，并清空 input
    uploadMocks.uploadFile.mockResolvedValueOnce({ url: 'v', type: 'video' } as any)
    const target1: any = { files: [file], value: 'x' }
    await (wrapper.vm as any).handleFileChange({ target: target1 } as any)
    expect(messageMocks.sendVideo).toHaveBeenCalledTimes(1)
    expect(target1.value).toBe('')
    expect((wrapper.vm as any).uploadProgress).toBeNull()

    // 多个文件：逐个上传，进度回调写入 uploadProgress；部分失败时提示失败个数
    uploadMocks.uploadFile.mockImplementationOnce(async (_f: File, _id: string, _name: string, opts: any) => {
      opts.onProgress(50)
      expect((wrapper.vm as any).uploadProgress).toEqual({ current: 1, total: 2, percent: 50 })
      return { url: 'i', type: 'image' }
    })
    uploadMocks.uploadFile.mockResolvedValueOnce(null as any)
    await (wrapper.vm as any).uploadAndSendFiles([file, file])
    expect(messageMocks.sendImage).toHaveBeenCalledTimes(1)
    expect(toastShow).toHaveBeenCalledWith('1 个文件上传失败')

    // 单文件失败
    uploadMocks.uploadFile.mockResolvedValueOnce(null as any)
    const target4: any = { files: [file], value: 'x' }
    await (wrapper.vm as any).handleFileChange({ target: target4 } as any)
    expect(toastShow).toHaveBeenCalledWith('文件上传失败')
    expect(target4.value).toBe('')

    // 断线时上传成功：不自动发送，保留在已上传列表
    chatStore.wsConnected = false
    messageMocks.sendImage.mockClear()
    uploadMocks.uploadFile.mockResolvedValueOnce({ url: 'i2', type: 'file' } as any)
    await (wrapper.vm as any).uploadAndSendFiles([file])
    expect(messageMocks.sendImage).not.toHaveBeenCalled()
    expect(toastShow).toHaveBeenCalledWith('已上传，可在“已上传的文件”中点击发送')
    chatStore.wsConnected = true

    // 上传期间切换会话：不发给新会话
    let resolveUpload: (v: any) => void = () => {}
    uploadMocks.uploadFile.mockImplementationOnce(() => new Promise(r => { resolveUpload = r }))
    const pending = (wrapper.vm as any).uploadAndSendFiles([file])
    await flushAsync()
    // 上传进行中再次触发：提示稍候
    await (wrapper.vm as any).uploadAndSendFiles([file])
    expect(toastShow).toHaveBeenCalledWith('正在上传，请稍候')
    chatStore.enterChat({ id: 'u2', name: 'U2', nickname: 'U2' } as any)
    resolveUpload({ url: 'i3', type: 'image' })
    await pending
    expect(messageMocks.sendImage).not.toHaveBeenCalled()

    // 空列表/无当前用户：直接返回
    uploadMocks.uploadFile.mockClear()
    await (wrapper.vm as any).uploadAndSendFiles([])
    expect(uploadMocks.uploadFile).not.toHaveBeenCalled()
  })

  it('keeps per-conversation drafts when switching chats and on unmount', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    const messageStore = useMessageStore()

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    ;(wrapper.vm as any).inputText = '给 u1 的半截话'
    chatStore.enterChat({ id: 'u2', name: 'U2', nickname: 'U2' } as any)
    await flushAsync()
    expect((wrapper.vm as any).inputText).toBe('')

    ;(wrapper.vm as any).inputText = '给 u2'
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)
    await flushAsync()
    expect((wrapper.vm as any).inputText).toBe('给 u1 的半截话')

    // 离开页面时保存当前草稿
    ;(wrapper.vm as any).inputText = '改过的草稿'
    wrapper.unmount()
    expect(messageStore.getDraft('me:u1')).toBe('改过的草稿')
    expect(messageStore.getDraft('me:u2')).toBe('给 u2')

    // 清空草稿 / resetAll
    messageStore.setDraft('me:u2', '')
    expect(messageStore.getDraft('me:u2')).toBe('')
    messageStore.setDraft('', 'x')
    expect(messageStore.getDraft('')).toBe('')
    messageStore.resetAll()
    expect(messageStore.getDraft('me:u1')).toBe('')
  })

  it('history media modal closes on Escape and header reconnect triggers reconnectNow', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
          ChatSidebar: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          ChatHeader: { name: 'ChatHeader', emits: ['reconnect'], template: '<div />' },
          MessageList: stubs.MessageList
        }
      }
    })
    await flushAsync()

    ;(wrapper.vm as any).showHistoryMediaModal = true
    await flushAsync()
    // 预览打开时 Esc 交给预览处理
    ;(wrapper.vm as any).showMediaPreview = true
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect((wrapper.vm as any).showHistoryMediaModal).toBe(true)
    ;(wrapper.vm as any).showMediaPreview = false
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect((wrapper.vm as any).showHistoryMediaModal).toBe(true)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect((wrapper.vm as any).showHistoryMediaModal).toBe(false)

    wsMocks.reconnectNow.mockReturnValueOnce(true)
    wrapper.findComponent({ name: 'ChatHeader' }).vm.$emit('reconnect')
    expect(toastShow).toHaveBeenCalledWith('正在重新连接…')
    toastShow.mockClear()
    wsMocks.reconnectNow.mockReturnValueOnce(false)
    wrapper.findComponent({ name: 'ChatHeader' }).vm.$emit('reconnect')
    expect(toastShow).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('supports paste files from ChatInput and drag-and-drop upload', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const ChatInputStub = { name: 'ChatInput', emits: ['paste-files'], template: '<div />' }
    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
          ChatSidebar: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: ChatInputStub,
          ChatHeader: true,
          MessageList: stubs.MessageList
        }
      }
    })
    await flushAsync()

    const file = new File(['x'], 'p.png', { type: 'image/png' })
    uploadMocks.uploadFile.mockResolvedValue({ url: 'p', type: 'image' } as any)
    wrapper.findComponent({ name: 'ChatInput' }).vm.$emit('paste-files', [file])
    await flushAsync()
    expect(messageMocks.sendImage).toHaveBeenCalledTimes(1)

    const makeDragEvent = (type: string, types: string[], files: File[] = []) => {
      const e = new Event(type, { bubbles: true, cancelable: true }) as any
      e.dataTransfer = { types, files }
      return e
    }
    const root = wrapper.element as HTMLElement

    // 非文件拖拽忽略
    root.dispatchEvent(makeDragEvent('dragenter', ['text/plain']))
    await flushAsync()
    expect(wrapper.find('[data-testid="drop-overlay"]').exists()).toBe(false)

    root.dispatchEvent(makeDragEvent('dragenter', ['Files']))
    const over = makeDragEvent('dragover', ['Files'])
    root.dispatchEvent(over)
    await flushAsync()
    expect(over.defaultPrevented).toBe(true)
    expect(wrapper.find('[data-testid="drop-overlay"]').exists()).toBe(true)

    root.dispatchEvent(makeDragEvent('dragleave', ['Files']))
    await flushAsync()
    expect(wrapper.find('[data-testid="drop-overlay"]').exists()).toBe(false)

    root.dispatchEvent(makeDragEvent('dragenter', ['Files']))
    root.dispatchEvent(makeDragEvent('drop', ['Files'], [file]))
    await flushAsync()
    expect(wrapper.find('[data-testid="drop-overlay"]').exists()).toBe(false)
    expect(messageMocks.sendImage).toHaveBeenCalledTimes(2)
    uploadMocks.uploadFile.mockReset()
  })

  it('executeClearAndReload covers count==0 and catch branches; Dialog/MediaTile template lines render', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'clearHistory').mockImplementation(() => {})

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    // Render the clear dialog body (template line with <br/>).
    ;(wrapper.vm as any).showClearDialog = true
    await flushAsync()
    expect(wrapper.find('.dialog-stub').text()).toContain('本地缓存的消息将被清除')

    // executeClearAndReload count == 0
    vi.spyOn(messageStore, 'loadHistory').mockResolvedValueOnce(0 as any)
    await (wrapper.vm as any).executeClearAndReload()
    expect(toastShow).toHaveBeenCalledWith('暂无聊天记录')

    // executeClearAndReload catch branch
    vi.spyOn(messageStore, 'loadHistory').mockRejectedValueOnce(new Error('boom'))
    await (wrapper.vm as any).executeClearAndReload()
    expect(toastShow).toHaveBeenCalledWith('重新加载失败，请稍后重试')

    // Render the history modal and MediaTile file slot (template line in #file slot).
    ;(wrapper.vm as any).showHistoryMediaModal = true
    ;(wrapper.vm as any).historyMediaLoading = false
    ;(wrapper.vm as any).historyMedia = [{ url: '/upload/files/a.bin', type: 'file' }]
    await flushAsync()
    expect(wrapper.find('.media-tile-stub').exists()).toBe(true)
  })

  it('handleOpenChatHistory covers non-array response and catch; confirmPreviewUpload covers missing imgServer + success', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.enterChat({ id: 'u1', name: 'U1', nickname: 'U1' } as any)

    const mediaStore = useMediaStore()
    mediaStore.imgServer = ''
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)
    const addUploadedSpy = vi.spyOn(mediaStore, 'addUploadedMedia').mockImplementation(() => {})

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue(9006 as any)

    const stubs = createMessageListStub({ isAtBottom: true })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    // open chat history: non-array response falls back to []
    vi.mocked(mediaApi.getChatImages).mockResolvedValueOnce({ hello: 'x' } as any)
    await (wrapper.vm as any).handleOpenChatHistory()
    expect((wrapper.vm as any).historyMedia).toEqual([])

    // catch branch
    vi.mocked(mediaApi.getChatImages).mockRejectedValueOnce(new Error('boom'))
    await (wrapper.vm as any).handleOpenChatHistory()
    expect((wrapper.vm as any).historyMedia).toEqual([])

    // confirmPreviewUpload: missing imgServer after load -> toast and return
    ;(wrapper.vm as any).previewTarget = { url: '/upload/images/2026/01/a.png', type: 'image' }
    mediaStore.imgServer = ''
    await (wrapper.vm as any).confirmPreviewUpload()
    expect(toastShow).toHaveBeenCalledWith('图片服务器地址未获取')

    // success path
    mediaStore.imgServer = 'img.local'
    vi.mocked(mediaApi.reuploadHistoryImage).mockResolvedValueOnce({ state: 'OK', msg: 'remote.png' } as any)
    await (wrapper.vm as any).confirmPreviewUpload()
    expect(addUploadedSpy).toHaveBeenCalled()
    expect(toastShow).toHaveBeenCalledWith('图片已加载，点击可发送')
  })

  it('covers startMatch branches, openAllUploads/openMtPhoto, and openUploadMenuSeq watcher early return', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)
    const loadAllSpy = vi.spyOn(mediaStore, 'loadAllUploadImages').mockResolvedValue(undefined as any)

    const mtPhotoStore = useMtPhotoStore()
    const openMtSpy = vi.spyOn(mtPhotoStore, 'open').mockResolvedValue(undefined as any)

    const stubs = createMessageListStub({ isAtBottom: false })
    const wrapper = mount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          teleport: TeleportStub,
          Dialog: DialogStub,
          Toast: true,
          MediaPreview: true,
          MediaTile: MediaTileStub,
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

    // ws not connected -> toast
    ;(wrapper.vm as any).handleStartMatch()
    expect(toastShow).toHaveBeenCalledWith('WebSocket 未连接，无法匹配')

    // connected, startMatch=false -> does not throw
    chatStore.wsConnected = true
    chatMocks.startMatch.mockReturnValueOnce(false)
    ;(wrapper.vm as any).handleStartMatch()

    // open all uploads / mtPhoto
    await (wrapper.vm as any).handleOpenAllUploads()
    expect(loadAllSpy).toHaveBeenCalledWith(1)
    await (wrapper.vm as any).handleOpenMtPhoto()
    expect(openMtSpy).toHaveBeenCalled()

    // openUploadMenuSeq watcher early return when no currentChatUser
    mediaStore.openUploadMenuSeq += 1
    await flushAsync()
    expect((wrapper.vm as any).showUploadMenu).toBe(false)
  })

  it('covers remaining swipe branches for edge-back and drawer-close callbacks', async () => {
    vi.useFakeTimers()
    try {
      const pinia = createPinia()
      setActivePinia(pinia)
      const router = createTestRouter()
      const pushSpy = vi.spyOn(router, 'replace')
      await router.push('/chat')
      await router.isReady()

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const chatStore = useChatStore()
      chatStore.wsConnected = true

      const mediaStore = useMediaStore()
      vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

      const stubs = createMessageListStub({ isAtBottom: true })
      const wrapper = mount(ChatRoomView, {
        global: {
          plugins: [pinia, router],
          stubs: {
            teleport: TeleportStub,
            Dialog: DialogStub,
            Toast: true,
            MediaPreview: true,
            MediaTile: MediaTileStub,
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

      expect(swipeCalls.length).toBeGreaterThanOrEqual(2)
      const pageSwipe = swipeCalls[0]!
      const drawerSwipe = swipeCalls[1]!

      // edge-back: deltaX <= 0 ignored
      pageSwipe.coordsStart.x = 0
      pageSwipe.opts.onSwipeProgress?.(-10, 0)
      expect((wrapper.vm as any).pageTranslateX).toBe(0)

      // edge-back: onSwipeEnd start outside hot zone resets translate and returns
      pageSwipe.coordsStart.x = 999
      ;(wrapper.vm as any).pageTranslateX = 120
      pageSwipe.opts.onSwipeEnd?.('right')
      expect((wrapper.vm as any).pageTranslateX).toBe(0)

      // edge-back: bounce path when below threshold
      pageSwipe.coordsStart.x = 0
      ;(wrapper.vm as any).pageTranslateX = 10
      pageSwipe.opts.onSwipeEnd?.('right')
      expect((wrapper.vm as any).isPageAnimating).toBe(true)
      vi.advanceTimersByTime(310)
      expect((wrapper.vm as any).isPageAnimating).toBe(false)

      // edge-back: swipe finish when not triggered and translate != 0
      ;(wrapper.vm as any).pageTranslateX = 10
      pageSwipe.opts.onSwipeFinish?.(0, 0, false)
      expect((wrapper.vm as any).isPageAnimating).toBe(true)

      // drawer: deltaX >= 0 ignored; vertical-dominant ignored
      ;(wrapper.vm as any).showSidebar = true
      await flushAsync()
      const sidebarWrap = wrapper.find('.absolute.inset-y-0.left-0')
      expect(sidebarWrap.exists()).toBe(true)
      const sidebarEl = sidebarWrap.element as HTMLElement
      ;(sidebarEl as any).getBoundingClientRect = () => ({ right: 300, width: 300, height: 500, left: 0, top: 0, bottom: 500 })
      drawerSwipe.coordsStart.x = 300
      drawerSwipe.opts.onSwipeProgress?.(10, 0)
      drawerSwipe.opts.onSwipeProgress?.(-10, 100)

      // drawer: onSwipeEnd when translate not enough -> bounce path
      ;(wrapper.vm as any).sidebarTranslateX = -10
      drawerSwipe.opts.onSwipeEnd?.('left')
      expect((wrapper.vm as any).isSidebarAnimating).toBe(true)
      vi.advanceTimersByTime(310)
      expect((wrapper.vm as any).isSidebarAnimating).toBe(false)

      // drawer: onSwipeFinish when not triggered and showSidebar=false resets state
      ;(wrapper.vm as any).showSidebar = false
      ;(wrapper.vm as any).sidebarTranslateX = -10
      drawerSwipe.opts.onSwipeFinish?.(0, 0, false)
      expect((wrapper.vm as any).sidebarTranslateX).toBe(0)

      // ensure handleBack still works on threshold met
      ;(wrapper.vm as any).pageTranslateX = 120
      pageSwipe.coordsStart.x = 0
      pageSwipe.opts.onSwipeEnd?.('right')
      await flushAsync()
      expect(pushSpy).toHaveBeenCalledWith('/list')
    } finally {
      vi.useRealTimers()
    }
  })
})
