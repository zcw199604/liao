import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()
const startMatchMock = vi.fn()
const startContinuousMatchMock = vi.fn()
const cancelMatchMock = vi.fn()
const handleAutoMatchMock = vi.fn()

vi.mock('@/constants/emoji', () => ({
  emojiMap: {
    '[ok]': 'https://example.com/ok.png'
  }
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => ({
    getMediaUrl: (input: string) => input
  })
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => ({
    startMatch: startMatchMock,
    startContinuousMatch: startContinuousMatchMock,
    cancelMatch: cancelMatchMock,
    handleAutoMatch: handleAutoMatchMock
  })
}))

import ChatInput from '@/components/chat/ChatInput.vue'
import EmojiPanel from '@/components/chat/EmojiPanel.vue'
import UploadMenu from '@/components/chat/UploadMenu.vue'
import MessageList from '@/components/chat/MessageList.vue'
import MatchButton from '@/components/chat/MatchButton.vue'

import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())

  if (!(HTMLElement.prototype as any).scrollTo) {
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      value: () => {}
    })
  }

  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      writeText: vi.fn().mockResolvedValue(undefined)
    }
  })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('components/chat/ChatInput.vue', () => {
  it('emits update:modelValue and typing start/end with debounce', async () => {
    vi.useFakeTimers()

    const wrapper = mount(ChatInput, {
      props: { modelValue: '', disabled: false, wsConnected: true }
    })

    await wrapper.get('textarea').setValue('hi')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['hi'])
    expect(wrapper.emitted('typingStart')).toBeTruthy()

    await wrapper.get('textarea').setValue('hi!')
    expect(wrapper.emitted('typingStart')?.length).toBe(1)

    vi.advanceTimersByTime(2999)
    await nextTick()
    expect(wrapper.emitted('typingEnd')).toBeFalsy()

    vi.advanceTimersByTime(1)
    await nextTick()
    expect(wrapper.emitted('typingEnd')).toBeTruthy()
  })

  it('sends on Enter / Ctrl+Enter but not Shift+Enter', async () => {
    const wrapper = mount(ChatInput, {
      props: { modelValue: 'x', disabled: false, wsConnected: true }
    })

    await wrapper.get('textarea').trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('send')).toBeTruthy()

    await wrapper.get('textarea').trigger('keydown', { key: 'Enter', ctrlKey: true })
    expect(wrapper.emitted('send')?.length).toBe(2)

    await wrapper.get('textarea').trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(wrapper.emitted('send')?.length).toBe(2)
  })

  it('disables match button when wsConnected=false', () => {
    const wrapper = mount(ChatInput, {
      props: { modelValue: 'x', disabled: false, wsConnected: false }
    })

    expect(wrapper.get('button[aria-label=\"匹配\"]').attributes('disabled')).toBeDefined()
  })
})

describe('components/chat/EmojiPanel.vue', () => {
  it('emits select when clicking an emoji, and closes when clicking x', async () => {
    const wrapper = mount(EmojiPanel, { props: { visible: true } })

    await wrapper.get('div.grid > div').trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual(['[ok]'])

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('renders nothing when visible=false', () => {
    const wrapper = mount(EmojiPanel, { props: { visible: false } })
    expect(wrapper.text()).toBe('')
    expect(wrapper.find('div.grid').exists()).toBe(false)
  })
})

describe('components/chat/UploadMenu.vue', () => {
  it('emits send and action events', async () => {
    const media = { url: 'http://x/a.png', type: 'image' } as any
    const wrapper = mount(UploadMenu, {
      props: {
        visible: true,
        uploadedMedia: [media],
        canOpenChatHistory: true
      }
    })

    await wrapper.get('div.w-16.h-16').trigger('click')
    expect(wrapper.emitted('send')?.[0]).toEqual([media])

    const buttons = wrapper.findAll('button')
    await buttons[0]?.trigger('click')
    expect(wrapper.emitted('uploadFile')).toBeTruthy()

    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('openChatHistory')).toBeTruthy()

    await buttons[2]?.trigger('click')
    expect(wrapper.emitted('openAllUploads')).toBeTruthy()

    await buttons[3]?.trigger('click')
    expect(wrapper.emitted('openMtPhoto')).toBeTruthy()

    await buttons[4]?.trigger('click')
    expect(wrapper.emitted('openDouyinFavoriteAuthors')).toBeTruthy()
  })
})

describe('components/chat/MessageList.vue', () => {
  it('copies text on dblclick and shows toast', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const clipboard = (navigator as any).clipboard
    expect(clipboard).toBeTruthy()

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          {
            content: 'hello',
            time: new Date(2026, 0, 5, 9, 5, 0).toISOString(),
            tid: '1',
            isSelf: false,
            isImage: false,
            isVideo: false,
            fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia] }
    })

    await wrapper.get('span[title=\"双击或长按复制\"]').trigger('dblclick')
    expect(clipboard.writeText).toHaveBeenCalledWith('hello')
    expect(toastShow).toHaveBeenCalledWith('已复制')
  })

  it('opens message menu on contextmenu/long-press and supports copy and retry', async () => {
    vi.useFakeTimers()
    const pinia = createPinia()
    setActivePinia(pinia)
    const clipboard = (navigator as any).clipboard

    const failed = {
      content: 'oops',
      time: '',
      tid: '',
      clientId: 'c1',
      isSelf: true,
      isImage: false,
      isVideo: false,
      sendStatus: 'failed',
      fromuser: { id: 'me', nickname: 'me', name: 'me', sex: '未知', ip: '' }
    }
    const image = {
      content: '',
      time: '',
      tid: 'img-1',
      isSelf: false,
      isImage: true,
      isVideo: false,
      imageUrl: 'http://x/a.png',
      fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
    }
    const wrapper = mount(MessageList, {
      props: { messages: [image, failed] as any, isTyping: false, loadingMore: false, canLoadMore: true },
      global: { plugins: [pinia] },
      attachTo: document.body
    })
    await wrapper.vm.$nextTick()

    const bubbles = wrapper.findAll('.msg-bubble')
    // 纯图片消息没有可用操作：不拦截浏览器默认右键菜单
    const imgEvent = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 })
    bubbles[0]!.element.dispatchEvent(imgEvent)
    expect(imgEvent.defaultPrevented).toBe(false)
    expect(document.querySelector('[data-testid="message-menu"]')).toBeNull()

    // 右键失败消息：出现复制 + 重新发送
    const evt = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 9999, clientY: 9999 })
    bubbles[1]!.element.dispatchEvent(evt)
    await wrapper.vm.$nextTick()
    expect(evt.defaultPrevented).toBe(true)
    ;(document.querySelector('[data-testid="message-menu-retry"]') as HTMLElement).click()
    expect(wrapper.emitted('retry')?.[0]).toEqual([failed])
    await wrapper.vm.$nextTick()
    expect(document.querySelector('[data-testid="message-menu"]')).toBeNull()

    // 长按打开，复制文本
    bubbles[1]!.element.dispatchEvent(new TouchEvent('touchstart', {
      touches: [{ clientX: 5, clientY: 5 } as any]
    }))
    vi.advanceTimersByTime(500)
    await wrapper.vm.$nextTick()
    ;(document.querySelector('[data-testid="message-menu-copy"]') as HTMLElement).click()
    await Promise.resolve()
    expect(clipboard.writeText).toHaveBeenCalledWith('oops')

    // 移动手指取消长按
    bubbles[1]!.element.dispatchEvent(new TouchEvent('touchstart', { touches: [{ clientX: 5, clientY: 5 } as any] }))
    bubbles[1]!.element.dispatchEvent(new TouchEvent('touchmove', { touches: [{ clientX: 50, clientY: 5 } as any] }))
    vi.advanceTimersByTime(600)
    await wrapper.vm.$nextTick()
    expect(document.querySelector('[data-testid="message-menu"]')).toBeNull()

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('shows new message badge when not at bottom and new messages arrive', async () => {
    vi.useFakeTimers()

    const pinia = createPinia()
    setActivePinia(pinia)
    const messageStore = useMessageStore()
    messageStore.isLoadingHistory = false

    const wrapper = mount(MessageList, {
      props: {
        messages: [
          {
            content: 'm1',
            time: new Date(2026, 0, 5, 9, 5, 0).toISOString(),
            tid: '1',
            isSelf: false,
            isImage: false,
            isVideo: false,
            fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
          }
        ] as any,
        isTyping: false,
        loadingMore: false,
        canLoadMore: true
      },
      global: { plugins: [pinia] }
    })

    const chatBox = wrapper.get('.chat-area').element as HTMLElement
    Object.defineProperty(chatBox, 'scrollHeight', { configurable: true, value: 1000 })
    Object.defineProperty(chatBox, 'clientHeight', { configurable: true, value: 500 })
    Object.defineProperty(chatBox, 'scrollTop', { configurable: true, value: 0, writable: true })

    await wrapper.get('.chat-area').trigger('scroll')
    vi.advanceTimersByTime(100)
    await nextTick()

    await wrapper.setProps({
      messages: [
        ...(wrapper.props('messages') as any[]),
        {
          content: 'm2',
          time: new Date(2026, 0, 5, 9, 6, 0).toISOString(),
          tid: '2',
          isSelf: false,
          isImage: false,
          isVideo: false,
          fromuser: { id: 'u1', nickname: 'u1', name: 'u1', sex: '未知', ip: '' }
        }
      ] as any
    })
    await nextTick()

    expect(wrapper.text()).toContain('新消息')

    await wrapper.get('button[title=\"有新消息\"]').trigger('click')
    await nextTick()
    expect(wrapper.text()).not.toContain('新消息')
  })
})

describe('components/chat/MatchButton.vue', () => {
  it('short press triggers startMatch and toast', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    startContinuousMatchMock.mockReturnValue(true)

    const wrapper = mount(MatchButton, {
      global: { plugins: [pinia] }
    })

    const button = wrapper.get('button')
    await button.trigger('click')

    expect(startContinuousMatchMock).toHaveBeenCalledWith(1)
    expect(toastShow).toHaveBeenCalledWith('正在匹配...')

    wrapper.unmount()
  })

  it('opens continuous match menu from desktop dropdown and triggers startContinuousMatch', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    startContinuousMatchMock.mockReturnValue(true)

    const wrapper = mount(MatchButton, {
      global: { plugins: [pinia] }
    })

    await wrapper.get('button[aria-label="连续匹配选项"]').trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('连续匹配 3 次')

    const option = wrapper
      .findAll('button')
      .find(btn => btn.text().includes('连续匹配 3 次'))
    expect(option).toBeTruthy()
    await option!.trigger('click')
    expect(startContinuousMatchMock).toHaveBeenCalledWith(3)
    expect(toastShow).toHaveBeenCalledWith('开始连续匹配 3 次...')

    document.dispatchEvent(new MouseEvent('click'))
    await nextTick()
    expect(wrapper.text()).not.toContain('连续匹配 3 次')

    wrapper.unmount()
  })

  it('cancel button triggers cancelMatch and toast', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = true
    chatStore.continuousMatchConfig.enabled = false

    const wrapper = mount(MatchButton, {
      global: { plugins: [pinia] }
    })

    await wrapper.get('button').trigger('click')
    expect(cancelMatchMock).toHaveBeenCalledOnce()
    expect(toastShow).toHaveBeenCalledWith('已取消匹配')

    wrapper.unmount()
  })

  it('handles match-auto-check event', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })
    window.dispatchEvent(new CustomEvent('match-auto-check'))
    await nextTick()

    expect(handleAutoMatchMock).toHaveBeenCalled()

    wrapper.unmount()
  })

  it('short press does not show toast when startContinuousMatch returns false', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    startContinuousMatchMock.mockReturnValue(false)

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })
    const button = wrapper.get('button')
    await button.trigger('click')

    expect(startContinuousMatchMock).toHaveBeenCalledWith(1)
    expect(toastShow).not.toHaveBeenCalledWith('正在匹配...')

    wrapper.unmount()
  })

  it('touchcancel cancels long press timer and does not open menu', async () => {
    vi.useFakeTimers()

    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })
    const button = wrapper.get('button')

    await button.trigger('touchstart')
    await button.trigger('touchcancel')
    vi.advanceTimersByTime(300)
    await nextTick()

    expect(wrapper.text()).not.toContain('连续匹配 3 次')
    expect(startContinuousMatchMock).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('touch events support short press and long press continuous menu', async () => {
    vi.useFakeTimers()

    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    startContinuousMatchMock.mockReturnValue(true)

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })
    const button = wrapper.get('button')

    await button.trigger('touchstart')
    await button.trigger('touchend')

    expect(startContinuousMatchMock).toHaveBeenCalledWith(1)
    expect(toastShow).toHaveBeenCalledWith('正在匹配...')

    startContinuousMatchMock.mockClear()
    toastShow.mockClear()

    await button.trigger('touchstart')
    vi.advanceTimersByTime(300)
    await nextTick()

    expect(wrapper.text()).toContain('连续匹配 3 次')

    await button.trigger('touchend')
    await nextTick()
    expect(startContinuousMatchMock).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('renders continuous match cancel label and progress when enabled and total>1', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = true
    chatStore.continuousMatchConfig.enabled = true
    chatStore.continuousMatchConfig.total = 3
    chatStore.continuousMatchConfig.current = 2

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })
    expect(wrapper.text()).toContain('取消连续匹配')
    expect(wrapper.text()).toContain('第 2/3 次')

    await wrapper.get('button').trigger('click')
    expect(cancelMatchMock).toHaveBeenCalledOnce()
    expect(toastShow).toHaveBeenCalledWith('已取消匹配')

    wrapper.unmount()
  })

  it('continuous menu hides and does not toast when startContinuousMatch fails', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = false

    startContinuousMatchMock.mockReturnValue(false)

    const wrapper = mount(MatchButton, { global: { plugins: [pinia] } })

    await wrapper.get('button[aria-label="连续匹配选项"]').trigger('click')
    await nextTick()

    const option = wrapper
      .findAll('button')
      .find(btn => btn.text().includes('连续匹配 5 次'))
    expect(option).toBeTruthy()
    await option!.trigger('click')

    expect(startContinuousMatchMock).toHaveBeenCalledWith(5)
    expect(toastShow).not.toHaveBeenCalledWith('开始连续匹配 5 次...')
    expect(wrapper.text()).not.toContain('连续匹配 5 次')

    wrapper.unmount()
  })
})
