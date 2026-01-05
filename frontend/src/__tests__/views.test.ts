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
  startMatch: vi.fn()
}

const wsMocks = {
  connect: vi.fn(),
  setScrollToBottom: vi.fn()
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
  loginLoading: false
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
          Toast: true
        }
      }
    })

    await Promise.resolve()
    await nextTick()

    identityMocks.loadList.mockClear()

    const buttons = wrapper.findAll('button')
    await buttons[0]?.trigger('click')

    await Promise.resolve()
    await nextTick()

    expect(identityMocks.quickCreate).toHaveBeenCalledOnce()
    expect(toastShow).toHaveBeenCalledWith('创建成功')
    expect(identityMocks.loadList).toHaveBeenCalledOnce()
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

    await Promise.resolve()
    await nextTick()

    expect(pushSpy).toHaveBeenCalledWith('/list')
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
})
