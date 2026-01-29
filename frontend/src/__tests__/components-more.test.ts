import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()
const routerPush = vi.fn()
const cancelMatch = vi.fn()
const enterChatAndStopMatch = vi.fn()
const startContinuousMatch = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

vi.mock('@/composables/useChat', () => ({
  useChat: () => ({
    cancelMatch,
    enterChatAndStopMatch,
    startContinuousMatch
  })
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<any>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: routerPush })
  }
})

import ChatHeader from '@/components/chat/ChatHeader.vue'
import CreateDialog from '@/components/identity/CreateDialog.vue'
import DraggableBadge from '@/components/common/DraggableBadge.vue'
import MediaTileSelectMark from '@/components/common/MediaTileSelectMark.vue'
import IdentityList from '@/components/identity/IdentityList.vue'
import ChatMedia from '@/components/chat/ChatMedia.vue'
import MatchOverlay from '@/components/chat/MatchOverlay.vue'
import PullToRefresh from '@/components/common/PullToRefresh.vue'

import { useChatStore } from '@/stores/chat'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
})

describe('components/chat/ChatHeader.vue', () => {
  it('renders fallback user name and connection status, emits events', async () => {
    const wrapper = mount(ChatHeader, {
      props: { user: null, connected: false }
    })

    expect(wrapper.text()).toContain('未知用户')
    expect(wrapper.text()).toContain('离线')

    await wrapper.get('button[aria-label="显示列表"]').trigger('click')
    await wrapper.get('button[aria-label="返回"]').trigger('click')
    await wrapper.get('button[title="清空并重新加载聊天记录"]').trigger('click')
    await wrapper.findAll('button').find(b => b.find('i').classes().includes('fa-star'))!.trigger('click')

    expect(wrapper.emitted('toggleSidebar')).toHaveLength(1)
    expect(wrapper.emitted('back')).toHaveLength(1)
    expect(wrapper.emitted('clearAndReload')).toHaveLength(1)
    expect(wrapper.emitted('toggleFavorite')).toHaveLength(1)
  })

  it('shows sex/age/address and favorite icon when user data is present', () => {
    const wrapper = mount(ChatHeader, {
      props: {
        connected: true,
        user: {
          id: 'u1',
          nickname: 'A',
          name: 'A',
          sex: '男',
          age: '18',
          address: 'CN',
          ip: '',
          area: 'CN',
          isFavorite: true,
          lastMsg: '',
          lastTime: '',
          unreadCount: 0
        } as any
      }
    })

    expect(wrapper.text()).toContain('在线')
    expect(wrapper.text()).toContain('18')
    expect(wrapper.text()).toContain('CN')
    expect(wrapper.find('i.fa-star.text-yellow-500').exists()).toBe(true)
  })
})

describe('components/identity/CreateDialog.vue', () => {
  it('emits created and closes when confirming with valid name', async () => {
    const wrapper = mount(CreateDialog, { props: { visible: true } })
    const createBtn = wrapper.findAll('button').find(b => b.text().trim() === '创建')!
    expect(createBtn.attributes('disabled')).toBeDefined()

    await wrapper.get('input[placeholder="输入名字"]').setValue('  Alice  ')
    expect(createBtn.attributes('disabled')).toBeUndefined()

    await createBtn.trigger('click')
    expect(wrapper.emitted('created')?.[0]).toEqual([{ name: '  Alice  ', sex: '男' }])
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
  })

  it('resets form when closing', async () => {
    const wrapper = mount(CreateDialog, { props: { visible: true } })
    await wrapper.get('input[placeholder="输入名字"]').setValue('A')
    await wrapper.setProps({ visible: false })
    await nextTick()
    await wrapper.setProps({ visible: true })
    await nextTick()
    expect((wrapper.get('input[placeholder="输入名字"]').element as HTMLInputElement).value).toBe('')
  })
})

describe('components/common/DraggableBadge.vue', () => {
  it('emits clear when dragged beyond threshold', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DraggableBadge, { props: { count: 3 } })
    await wrapper.trigger('mousedown', { clientX: 0, clientY: 0 })
    await window.dispatchEvent(new MouseEvent('mousemove', { clientX: 100, clientY: 0 }))
    await window.dispatchEvent(new MouseEvent('mouseup'))

    expect(wrapper.emitted('clear')).toHaveLength(1)
    vi.useRealTimers()
  })

  it('bounces back when threshold is not reached', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DraggableBadge, { props: { count: 1 } })
    await wrapper.trigger('mousedown', { clientX: 0, clientY: 0 })
    await window.dispatchEvent(new MouseEvent('mousemove', { clientX: 10, clientY: 0 }))
    await window.dispatchEvent(new MouseEvent('mouseup'))

    expect(wrapper.emitted('clear')).toBeUndefined()
    vi.advanceTimersByTime(350)
    vi.useRealTimers()
  })
})

describe('components/common/MediaTileSelectMark.vue', () => {
  it('computes classes by size/tone and emits click only when interactive', async () => {
    const wrapper = mount(MediaTileSelectMark, {
      props: { checked: false, interactive: false, size: 'sm', tone: 'purple' }
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()

    const wrapper2 = mount(MediaTileSelectMark, {
      props: { checked: true, interactive: true, size: 'lg', tone: 'emerald' }
    })
    await wrapper2.trigger('click')
    expect(wrapper2.emitted('click')).toHaveLength(1)
    expect(wrapper2.find('i.fa-check').exists()).toBe(true)

    // default branches: size=md + tone=indigo
    const wrapper3 = mount(MediaTileSelectMark, {
      props: { checked: true, interactive: true }
    })
    expect(wrapper3.classes()).toContain('w-10')
    expect(wrapper3.html()).toContain('bg-indigo-500/80')
  })
})

describe('components/identity/IdentityList.vue', () => {
  it('renders empty state when identities is empty', () => {
    const wrapper = mount(IdentityList, { props: { identities: [] } })
    expect(wrapper.text()).toContain('还没有身份')
  })

  it('emits select and delete events', async () => {
    const identities = [
      { id: 'aaaaaaaa', name: 'Alice', sex: '女', created_at: 't' },
      { id: 'bbbbbbbb', name: 'Bob', sex: '男', createdAt: 't2' }
    ] as any
    const wrapper = mount(IdentityList, { props: { identities } })

    await wrapper.findAll('.identity-card')[0]!.trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([identities[0]])

    await wrapper.findAll('button')[0]!.trigger('click')
    expect(wrapper.emitted('delete')?.[0]).toEqual([identities[0]])
  })
})

describe('components/chat/ChatMedia.vue', () => {
  it('emits preview when previewable and defaults aspect ratio by type', async () => {
    const MediaTileStub = {
      name: 'MediaTile',
      props: ['src', 'type', 'aspectRatio', 'alt', 'showSkeleton', 'controls'],
      emits: ['click', 'layout'],
      template: `<button data-testid="tile" @click="$emit('click')"></button>`
    }
    const wrapper = mount(ChatMedia, {
      props: { type: 'image', src: 'x', previewable: true },
      global: {
        stubs: {
          MediaTile: MediaTileStub
        }
      }
    })

    await wrapper.get('[data-testid="tile"]').trigger('click')
    expect(wrapper.emitted('preview')?.[0]).toEqual(['x', 'image'])

    expect(wrapper.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBeCloseTo(4 / 3)
  })

  it('does not emit preview when previewable=false', async () => {
    const MediaTileStub = {
      name: 'MediaTile',
      props: ['src', 'type', 'aspectRatio', 'alt', 'showSkeleton', 'controls'],
      template: `<button data-testid="tile"></button>`
    }
    const wrapper = mount(ChatMedia, {
      props: { type: 'video', src: 'v', previewable: false },
      global: {
        stubs: {
          MediaTile: MediaTileStub
        }
      }
    })

    await wrapper.get('[data-testid="tile"]').trigger('click')
    expect(wrapper.emitted('preview')).toBeUndefined()

    // invalid aspectRatio falls back to 16/9 for video
    expect(wrapper.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBeCloseTo(16 / 9)
    expect(wrapper.getComponent({ name: 'MediaTile' }).props('controls')).toBe(true)
  })

  it('uses provided aspectRatio when valid and falls back for invalid values', () => {
    const MediaTileStub = {
      name: 'MediaTile',
      props: ['aspectRatio'],
      template: `<div></div>`
    }

    const wrapper = mount(ChatMedia, {
      props: { type: 'image', src: 'x', aspectRatio: 2 },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect(wrapper.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBe(2)

    const wrapper2 = mount(ChatMedia, {
      props: { type: 'image', src: 'x', aspectRatio: 0 },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect(wrapper2.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBeCloseTo(4 / 3)

    const wrapper3 = mount(ChatMedia, {
      props: { type: 'video', src: 'x', aspectRatio: Infinity },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect(wrapper3.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBeCloseTo(16 / 9)

    const wrapper4 = mount(ChatMedia, {
      props: { type: 'video', src: 'x', aspectRatio: -1 },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect(wrapper4.getComponent({ name: 'MediaTile' }).props('aspectRatio')).toBeCloseTo(16 / 9)
  })
})

describe('components/chat/MatchOverlay.vue', () => {
  it('shows matching overlay and cancels match', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.isMatching = true
    chatStore.currentMatchedUser = null

    const wrapper = mount(MatchOverlay, {
      global: { plugins: [pinia], stubs: { teleport: true } }
    })

    expect(wrapper.text()).toContain('正在寻找有缘人')
    await wrapper.get('button[type="button"]').trigger('click')
    expect(cancelMatch).toHaveBeenCalledTimes(1)
    expect(toastShow).toHaveBeenCalledWith('已取消匹配')
  })

  it('enters chat when matched user exists and supports continue match in single mode', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const chatStore = useChatStore()
    chatStore.startContinuousMatch(1)
    chatStore.setCurrentMatchedUser({
      id: 'u1',
      name: 'U1',
      nickname: 'U1',
      sex: '男',
      age: '0',
      area: 'CN',
      address: 'CN',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)

    const wrapper = mount(MatchOverlay, {
      global: { plugins: [pinia], stubs: { teleport: true } }
    })

    await wrapper.findAll('button').find(b => b.text().includes('进入聊天'))!.trigger('click')
    expect(enterChatAndStopMatch).toHaveBeenCalled()
    expect(routerPush).toHaveBeenCalledWith('/chat/u1')

    await wrapper.findAll('button').find(b => b.text().includes('继续匹配'))!.trigger('click')
    expect(startContinuousMatch).toHaveBeenCalledWith(1)
  })
})

describe('components/common/PullToRefresh.vue', () => {
  it('triggers refresh when pulled beyond threshold and then resets', async () => {
    vi.useFakeTimers()
    const onRefresh = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(PullToRefresh, {
      props: { onRefresh, threshold: 60 },
      slots: {
        default: `<div class="overflow-y-auto" style="overflow-y:auto;height:100px"></div>`
      }
    })

    const scrollDiv = wrapper.get('.overflow-y-auto').element as HTMLElement
    scrollDiv.scrollTop = 0

    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    await wrapper.trigger('touchmove', { touches: [{ clientX: 0, clientY: 200 }] })
    expect(wrapper.text()).toContain('释放立即刷新')

    await wrapper.trigger('touchend')
    await Promise.resolve()
    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('正在刷新')

    vi.advanceTimersByTime(400)
    await nextTick()
    expect(wrapper.text()).toContain('下拉刷新')
    vi.useRealTimers()
  })

  it('cancels pull when horizontal swipe is dominant', async () => {
    const onRefresh = vi.fn().mockResolvedValue(undefined)
    const wrapper = mount(PullToRefresh, {
      props: { onRefresh },
      slots: {
        default: `<div class="overflow-y-auto" style="overflow-y:auto;height:100px"></div>`
      }
    })

    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    await wrapper.trigger('touchmove', { touches: [{ clientX: 100, clientY: 10 }] })
    await wrapper.trigger('touchend')
    expect(onRefresh).not.toHaveBeenCalled()
  })

  it('finds nested scroll container via querySelector and still refreshes', async () => {
    vi.useFakeTimers()
    const onRefresh = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(PullToRefresh, {
      props: { onRefresh, threshold: 60 },
      slots: {
        // firstElementChild is not scrollable; inner .overflow-y-auto is
        default: `<div><div class="overflow-y-auto" style="overflow-y:auto;height:100px"></div></div>`
      }
    })

    const inner = wrapper.get('.overflow-y-auto').element as HTMLElement
    inner.scrollTop = 0

    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    await wrapper.trigger('touchmove', { touches: [{ clientX: 0, clientY: 200 }] })
    await wrapper.trigger('touchend')
    await Promise.resolve()
    expect(onRefresh).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(400)
    await nextTick()
    vi.useRealTimers()
  })

  it('cancels pull when scrollTop becomes > 0 during pull and ignores when refreshing', async () => {
    vi.useFakeTimers()
    const onRefresh = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(PullToRefresh, {
      props: { onRefresh, threshold: 60 },
      slots: {
        default: `<div class="overflow-y-auto" style="overflow-y:auto;height:100px"></div>`
      }
    })

    const scrollDiv = wrapper.get('.overflow-y-auto').element as HTMLElement
    scrollDiv.scrollTop = 0

    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    scrollDiv.scrollTop = 10
    await wrapper.trigger('touchmove', { touches: [{ clientX: 0, clientY: 50 }] })
    await wrapper.trigger('touchend')
    await Promise.resolve()
    expect(onRefresh).not.toHaveBeenCalled()

    // status=refreshing -> touch handlers do nothing
    ;(wrapper.vm as any).status = 'refreshing'
    await wrapper.trigger('touchstart', { touches: [{ clientX: 0, clientY: 0 }] })
    await wrapper.trigger('touchmove', { touches: [] })
    await wrapper.trigger('touchend')
    expect(onRefresh).not.toHaveBeenCalled()

    vi.useRealTimers()
  })
})
