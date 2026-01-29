import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/chat', () => ({
  getMessageHistory: vi.fn()
}))

vi.mock('@/utils/messageSegments', () => ({
  parseMessageSegments: vi.fn(),
  getSegmentsMeta: vi.fn()
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => ({
    getMediaUrl: (input: string) => input
  })
}))

vi.mock('@/api/system', () => ({
  getSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn()
}))

vi.mock('@/api/media', () => ({
  getImgServerAddress: vi.fn(),
  updateImgServerAddress: vi.fn(),
  getCachedImages: vi.fn(),
  getAllUploadImages: vi.fn()
}))

import ChatHistoryPreview from '@/components/chat/ChatHistoryPreview.vue'
import { useIdentityStore } from '@/stores/identity'
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'

import * as chatApi from '@/api/chat'
import * as messageSegments from '@/utils/messageSegments'
import * as systemApi from '@/api/system'
import * as mediaApi from '@/api/media'

const flush = async () => {
  await new Promise<void>(resolve => setTimeout(resolve, 0))
  await nextTick()
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/chat/ChatHistoryPreview.vue', () => {
  it('shows error when identity cookie is missing', async () => {
    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' },
      global: {
        stubs: {
          MediaTile: true
        }
      }
    })

    await wrapper.setProps({ visible: true })
    await flush()

    expect(wrapper.text()).toContain('未找到该身份的登录凭证')
  })

  it('loads JSON history, renders segments, and marks failed images on MediaTile error', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    const mediaStore = useMediaStore()
    vi.mocked(mediaApi.getImgServerAddress).mockResolvedValue({ state: 'OK', msg: { server: 'img.local' } } as any)
    vi.mocked(mediaApi.updateImgServerAddress).mockResolvedValue({ state: 'OK' } as any)

    const systemConfigStore = useSystemConfigStore()
    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [
        { id: 'u1', Tid: 't1', content: 'img', time: '2026-01-01 00:00:00.000' },
        { id: 'me', Tid: 't2', content: 'text', time: '2026-01-01 00:00:01.000' }
      ]
    } as any)

    vi.mocked(messageSegments.parseMessageSegments)
      .mockResolvedValueOnce([{ kind: 'image', url: '2026/01/a.png' } as any])
      .mockResolvedValueOnce([{ kind: 'text', text: 'hello' } as any])
    vi.mocked(messageSegments.getSegmentsMeta)
      .mockReturnValueOnce({ hasImage: true, hasVideo: false, hasFile: false, imageUrl: '2026/01/a.png' } as any)
      .mockReturnValueOnce({ hasImage: false, hasVideo: false, hasFile: false } as any)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' },
      global: {
        plugins: [pinia],
        stubs: {
          MediaTile: {
            name: 'MediaTile',
            props: ['src', 'type'],
            emits: ['error'],
            template: `<button data-testid="tile" @click="$emit('error')"></button>`
          }
        }
      }
    })

    await wrapper.setProps({ visible: true })
    for (let i = 0; i < 6; i += 1) {
      await flush()
    }

    expect(mediaStore.imgServer).toBe('img.local')
    expect(systemConfigStore.loaded).toBe(true)
    expect(wrapper.text()).toContain('历史消息预览')
    expect(wrapper.text()).toContain('U1')

    await wrapper.get('[data-testid=\"tile\"]').trigger('click')
    await flush()
    expect(wrapper.text()).toContain('图片加载失败')

    await wrapper.findAll('button').find(b => b.text().trim() === '关闭')!.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)

    await wrapper.findAll('button').find(b => b.text().includes('切换身份'))!.trigger('click')
    expect(wrapper.emitted('switch')).toHaveLength(1)
  })

  it('loads legacy XML history and maps [img]/[video] messages', async () => {
    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue(
      `<ArrayOfMsg>
        <Msg><From>i1</From><Body>[img]/upload/images/2026/01/a.png[/img]</Body><Time>t1</Time></Msg>
        <Msg><From>u1</From><Body>[video]/upload/videos/2026/01/b.mp4[/video]</Body><Time>t2</Time></Msg>
      </ArrayOfMsg>`
    )

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' },
      global: {
        stubs: {
          MediaTile: {
            props: ['src', 'type'],
            template: `<div data-testid="tile"></div>`
          }
        }
      }
    })

    await wrapper.setProps({ visible: true })
    await flush()
    await flush()

    expect(wrapper.findAll('[data-testid="tile"]').length).toBeGreaterThan(0)
  })

  it('shows generic error when response format is unexpected', async () => {
    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ code: 1 } as any)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: { stubs: { MediaTile: true } }
    })
    await wrapper.setProps({ visible: true })
    await flush()
    expect(wrapper.text()).toContain('获取消息失败')
  })
})
