import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const toastShow = vi.fn()

const identityMocks = {
  loadList: vi.fn(),
  select: vi.fn(),
  create: vi.fn(),
  quickCreate: vi.fn(),
  deleteIdentity: vi.fn()
}

const chatMocks = {
  toggleFavorite: vi.fn(),
  enterChat: vi.fn(),
  startMatch: vi.fn(),
  loadUsers: vi.fn().mockResolvedValue(undefined)
}

const wsMocks = {
  connect: vi.fn(),
  setScrollToBottom: vi.fn(),
  disconnect: vi.fn()
}

const messageMocks = {
  sendText: vi.fn(),
  sendImage: vi.fn(),
  sendVideo: vi.fn(),
  sendTypingStatus: vi.fn()
}

const uploadMocks = {
  uploadFile: vi.fn(),
  getMediaUrl: (input: string) => input
}

const authStoreMocks = {
  login: vi.fn(),
  checkToken: vi.fn(),
  isAuthenticated: false,
  loginLoading: false,
  logout: vi.fn()
}

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow,
    hide: vi.fn(),
    error: toastShow,
    success: toastShow,
    message: { value: '' },
    visible: { value: false }
  })
}))

vi.mock('@/composables/useIdentity', () => ({
  useIdentity: () => identityMocks
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => chatMocks
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => wsMocks
}))

vi.mock('@/composables/useMessage', () => ({
  useMessage: () => messageMocks
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreMocks
}))

import LoginPage from '@/views/LoginPage.vue'
import ChatListView from '@/views/ChatListView.vue'
import IdentityPicker from '@/views/IdentityPicker.vue'
import ChatRoomView from '@/views/ChatRoomView.vue'

import { useChatStore } from '@/stores/chat'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'

const createTestRouter = () => {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/login', component: LoginPage },
      { path: '/identity', component: { template: '<div />' } },
      { path: '/list', component: { template: '<div />' } },
      { path: '/chat/:userId?', component: ChatRoomView }
    ]
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

describe('views/LoginPage.vue', () => {
  it('shows error query and replaces url, without checking token', async () => {
    const router = createTestRouter()
    await router.push(`/login?error=${encodeURIComponent('被踢下线')}`)
    await router.isReady()

    const replaceSpy = vi.spyOn(router, 'replace')

    mount(LoginPage, {
      global: {
        plugins: [router],
        stubs: { Toast: true }
      }
    })

    await nextTick()

    expect(toastShow).toHaveBeenCalledWith('被踢下线')
    expect(replaceSpy).toHaveBeenCalledWith('/')
    expect(authStoreMocks.checkToken).not.toHaveBeenCalled()
  })

  it('redirects to /identity when token is valid', async () => {
    authStoreMocks.checkToken.mockResolvedValue(true)

    const router = createTestRouter()
    await router.push('/login')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    mount(LoginPage, {
      global: {
        plugins: [router],
        stubs: { Toast: true }
      }
    })

    await Promise.resolve()
    await nextTick()

    expect(authStoreMocks.checkToken).toHaveBeenCalledOnce()
    expect(pushSpy).toHaveBeenCalledWith('/identity')
  })

  it('calls login and navigates on success', async () => {
    authStoreMocks.checkToken.mockResolvedValue(false)
    authStoreMocks.login.mockResolvedValue(true)

    const router = createTestRouter()
    await router.push('/login')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    const wrapper = mount(LoginPage, {
      global: {
        plugins: [router],
        stubs: { Toast: true }
      }
    })

    await wrapper.get('input').setValue('code')
    await wrapper.get('button').trigger('click')

    await Promise.resolve()
    await nextTick()

    expect(authStoreMocks.login).toHaveBeenCalledWith('code')
    expect(pushSpy).toHaveBeenCalledWith('/identity')
  })
})

describe('views/IdentityPicker.vue', () => {
  it('loads identity list on mount', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    identityMocks.loadList.mockResolvedValue(undefined)

    mount(IdentityPicker, {
      global: {
        plugins: [pinia],
        stubs: {
          IdentityList: true,
          CreateDialog: true,
          Dialog: true,
          SettingsDrawer: true,
          Toast: true
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    expect(identityMocks.loadList).toHaveBeenCalledOnce()
  })

  it('quick create shows success and reloads list', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    identityMocks.loadList.mockResolvedValue(undefined)
    identityMocks.quickCreate.mockResolvedValue(true)

    const wrapper = mount(IdentityPicker, {
      global: {
        plugins: [pinia],
        stubs: {
          IdentityList: true,
          CreateDialog: true,
          Dialog: true,
          SettingsDrawer: true,
          Toast: true
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    identityMocks.loadList.mockClear()

    const quickCreateBtn = wrapper.findAll('button').find(btn => btn.text().includes('快速进入'))
    expect(quickCreateBtn).toBeTruthy()
    await quickCreateBtn!.trigger('click')

    await Promise.resolve()
    await nextTick()

    expect(identityMocks.quickCreate).toHaveBeenCalledOnce()
    expect(toastShow).toHaveBeenCalledWith('创建成功')
    // store 内部创建成功后已刷新列表，页面不再重复加载
    expect(identityMocks.loadList).not.toHaveBeenCalled()
  })
})

describe('views/IdentityPicker.vue (actions)', () => {
  const mountPicker = async (pinia: any, router: any) => {
    const wrapper = mount(IdentityPicker, {
      global: {
        plugins: [pinia, router],
        stubs: { CreateDialog: true, Dialog: true, SettingsDrawer: true, Toast: true }
      }
    })
    await Promise.resolve()
    await nextTick()
    return wrapper
  }

  it('keeps list visible during actions, blocks repeated clicks, and reports failures', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    identityMocks.loadList.mockResolvedValue(undefined)

    const { useIdentityStore } = await import('@/stores/identity')
    useIdentityStore().identityList = [{ id: 'id-1', name: 'Alice', sex: '女' } as any]

    const wrapper = await mountPicker(pinia, router)
    expect(wrapper.text()).toContain('Alice')

    let resolveSelect: () => void = () => {}
    identityMocks.select.mockImplementationOnce(() => new Promise<void>(r => { resolveSelect = r }))
    await wrapper.get('.identity-card').trigger('click')
    await nextTick()
    // 操作中列表仍可见，并且再次点击不会重复触发
    expect(wrapper.text()).toContain('Alice')
    expect((wrapper.vm as any).busyIdentityId).toBe('id-1')
    await wrapper.get('.identity-card').trigger('click')
    expect(identityMocks.select).toHaveBeenCalledTimes(1)
    resolveSelect()
    await Promise.resolve()
    await nextTick()
    expect((wrapper.vm as any).actionLoading).toBe(false)

    // 选择失败
    identityMocks.select.mockRejectedValueOnce(new Error('x'))
    await (wrapper.vm as any).handleSelect({ id: 'id-1' })
    expect(toastShow).toHaveBeenCalledWith('选择身份失败')

    // 快速创建失败 / 异常
    identityMocks.quickCreate.mockResolvedValueOnce(false)
    await (wrapper.vm as any).handleQuickCreate()
    expect(toastShow).toHaveBeenLastCalledWith('创建失败')
    identityMocks.quickCreate.mockRejectedValueOnce(new Error('x'))
    await (wrapper.vm as any).handleQuickCreate()
    expect(toastShow).toHaveBeenLastCalledWith('创建失败')

    // 自定义创建：成功 / 失败 / 异常
    identityMocks.create.mockResolvedValueOnce(true)
    await (wrapper.vm as any).handleCreate({ name: 'B', sex: '男' })
    expect(toastShow).toHaveBeenLastCalledWith('创建成功')
    identityMocks.create.mockRejectedValueOnce(new Error('x'))
    await (wrapper.vm as any).handleCreate({ name: 'B', sex: '男' })
    expect(toastShow).toHaveBeenLastCalledWith('创建失败')

    // 删除：成功 / 失败 / 异常 / 无目标
    await (wrapper.vm as any).handleDelete()
    expect(identityMocks.deleteIdentity).not.toHaveBeenCalled()
    for (const [impl, msg] of [
      [() => Promise.resolve(true), '删除成功'],
      [() => Promise.resolve(false), '删除失败'],
      [() => Promise.reject(new Error('x')), '删除失败']
    ] as const) {
      identityMocks.deleteIdentity.mockImplementationOnce(impl as any)
      ;(wrapper.vm as any).handleDeleteConfirm({ id: 'id-1', name: 'Alice' })
      await (wrapper.vm as any).handleDelete()
      expect(toastShow).toHaveBeenLastCalledWith(msg)
      expect((wrapper.vm as any).deleteTarget).toBeNull()
    }
  })

  it('logout disconnects ws, clears identity and token, and returns to login', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    identityMocks.loadList.mockResolvedValue(undefined)
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'me' } as any

    const replaceSpy = vi.spyOn(router, 'replace')
    const wrapper = await mountPicker(pinia, router)
    await wrapper.get('[data-testid="identity-logout"]').trigger('click')
    expect((wrapper.vm as any).showLogoutDialog).toBe(true)

    ;(wrapper.vm as any).handleLogout()
    expect(wsMocks.disconnect).toHaveBeenCalledWith(true)
    expect(userStore.currentUser).toBeNull()
    expect(authStoreMocks.logout).toHaveBeenCalled()
    expect(replaceSpy).toHaveBeenCalledWith('/login')
  })
})

describe('views/ChatListView.vue', () => {
  const createListRouter = () => {
    return createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/identity', component: { template: '<div />' } },
        { path: '/list', component: ChatListView },
        { path: '/chat/:userId', component: { template: '<div />' } }
      ]
    })
  }

  it('redirects to /identity when current user is missing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = null

    const router = createListRouter()
    await router.push('/list')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    mount(ChatListView, {
      global: {
        plugins: [pinia, router],
        stubs: { ChatSidebar: true }
      }
    })

    await nextTick()

    expect(pushSpy).toHaveBeenCalledWith('/identity')
  })

  it('enters chat and navigates when sidebar emits select', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me' } as any

    const router = createListRouter()
    await router.push('/list')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    const wrapper = mount(ChatListView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatSidebar: {
            name: 'ChatSidebar',
            template: '<div />',
            emits: ['select', 'match-success']
          }
        }
      }
    })

    const user = { id: 'u1', nickname: 'n1' } as any
    wrapper.findComponent({ name: 'ChatSidebar' }).vm.$emit('select', user)
    await nextTick()

    expect(chatMocks.enterChat).toHaveBeenCalledWith(user, true)
    expect(pushSpy).toHaveBeenCalledWith('/chat/u1')
  })

  it('enters chat and navigates when sidebar emits match-success', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me' } as any

    const router = createListRouter()
    await router.push('/list')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    const wrapper = mount(ChatListView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatSidebar: {
            name: 'ChatSidebar',
            template: '<div />',
            emits: ['select', 'match-success']
          }
        }
      }
    })

    const user = { id: 'u2', nickname: 'n2' } as any
    wrapper.findComponent({ name: 'ChatSidebar' }).vm.$emit('match-success', user)
    await nextTick()

    expect(chatMocks.enterChat).toHaveBeenCalledWith(user, false)
    expect(pushSpy).toHaveBeenCalledWith('/chat/u2')
  })
})

describe('views/ChatRoomView.vue', () => {
  it('redirects to /identity when current user is missing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = null

    const router = createTestRouter()
    await router.push('/chat/u1')
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')
    pushSpy.mockClear()

    shallowMount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatHeader: true,
          MessageList: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          MediaPreview: true,
          Toast: true,
          Dialog: true,
          ChatSidebar: true,
          teleport: true
        }
      }
    })

    await nextTick()
    expect(pushSpy).toHaveBeenCalledWith('/identity')
  })

  it('redirects to /list when route userId is not found in store', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'me',
      nickname: 'me',
      sex: '未知',
      color: '',
      created_at: '',
      cookie: '',
      ip: '',
      area: ''
    }

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const router = createTestRouter()
    await router.push('/chat/u404')
    await router.isReady()

    const replaceSpy = vi.spyOn(router, 'replace')
    replaceSpy.mockClear()

    shallowMount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatHeader: true,
          MessageList: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          MediaPreview: true,
          Toast: true,
          Dialog: true,
          ChatSidebar: true,
          teleport: true
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    await Promise.resolve()
    await nextTick()

    // 刷新恢复：先尝试拉取会话列表，仍找不到才回列表页
    expect(chatMocks.loadUsers).toHaveBeenCalled()
    expect(replaceSpy).toHaveBeenCalledWith('/list')
  })

  it('toggles sidebar when ChatHeader emits toggle-sidebar and closes on overlay click', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'me',
      nickname: 'me',
      sex: '未知',
      color: '',
      created_at: '',
      cookie: '',
      ip: '',
      area: ''
    }

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const router = createTestRouter()
    await router.push('/chat')
    await router.isReady()

    const wrapper = shallowMount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatHeader: {
            name: 'ChatHeader',
            template: '<div />',
            emits: ['toggle-sidebar']
          },
          MessageList: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          MediaPreview: true,
          Toast: true,
          Dialog: true,
          ChatSidebar: true,
          teleport: true
        }
      }
    })

    expect(wrapper.findComponent({ name: 'ChatSidebar' }).exists()).toBe(false)

    wrapper.findComponent({ name: 'ChatHeader' }).vm.$emit('toggle-sidebar')
    await nextTick()
    expect(wrapper.findComponent({ name: 'ChatSidebar' }).exists()).toBe(true)

    const overlay = wrapper.find('div.absolute.inset-0.z-40')
    await overlay.trigger('click')
    await nextTick()
    expect(wrapper.findComponent({ name: 'ChatSidebar' }).exists()).toBe(false)
  })

  it('clears currentChatUser on unmount to avoid unread mis-detection on list page', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'me',
      nickname: 'me',
      sex: '未知',
      color: '',
      created_at: '',
      cookie: '',
      ip: '',
      area: ''
    }

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const router = createTestRouter()
    await router.push('/chat/u2')
    await router.isReady()

    const wrapper = shallowMount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatHeader: true,
          MessageList: true,
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          MediaPreview: true,
          Toast: true,
          Dialog: true,
          ChatSidebar: true,
          teleport: true
        }
      }
    })

    expect(chatStore.currentChatUser).not.toBeNull()
    wrapper.unmount()
    expect(chatStore.currentChatUser).toBeNull()
  })

  it('applies dynamic bottom spacing for floating input (no fixed pb-32)', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.upsertUser({ id: 'u1', name: 'U1' } as any)
    chatStore.enterChat(chatStore.getUser('u1') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const router = createTestRouter()
    await router.push('/chat/u1')
    await router.isReady()

    const wrapper = shallowMount(ChatRoomView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          ChatHeader: true,
          MessageList: {
            name: 'MessageList',
            inheritAttrs: false,
            template: '<div class="message-list-stub" v-bind="$attrs"></div>',
            methods: { scrollToBottom: () => {} }
          },
          UploadMenu: true,
          EmojiPanel: true,
          ChatInput: true,
          MediaPreview: true,
          Toast: true,
          Dialog: true,
          ChatSidebar: true,
          teleport: true
        }
      }
    })

    const messageList = wrapper.find('.message-list-stub')
    expect(messageList.exists()).toBe(true)

    const classes = messageList.attributes('class')
    expect(classes).not.toContain('pb-32')
    expect(classes).not.toContain('!pb-32')

    const styleAttr = messageList.attributes('style') || ''
    expect(styleAttr).toContain('padding-top')
    expect(styleAttr).toContain('padding-bottom')
  })
})
