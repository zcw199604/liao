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

const flush = async () => {
  await new Promise<void>(resolve => setTimeout(resolve, 0))
  await nextTick()
}

const deferred = <T,>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type'],
  emits: ['error', 'click'],
  template: `<button data-testid="tile" :data-type="type" :data-src="src" @click="$emit('error')"></button>`
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/chat/ChatHistoryPreview.vue (more branches)', () => {
  it('does not call history api when identityId/targetUserId is missing', async () => {
    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: '', targetUserId: 'u1' },
      global: { stubs: { MediaTile: true } }
    })

    await wrapper.setProps({ visible: true })
    await flush()

    expect(vi.mocked(chatApi.getMessageHistory)).not.toHaveBeenCalled()
  })

  it('shows loading while request is pending and then renders empty state', async () => {
    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    const d = deferred<any>()
    vi.mocked(chatApi.getMessageHistory).mockReturnValueOnce(d.promise)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: { stubs: { MediaTile: true } }
    })

    await wrapper.setProps({ visible: true })
    await nextTick()
    expect(wrapper.text()).toContain('正在加载历史记录')

    d.resolve({ code: 0, contents_list: [] })
    await flush()
    await flush()
    expect(wrapper.text()).toContain('暂无聊天记录')
  })

  it('covers resolveMediaUrl branches, segment kinds, and fallback (isImage/isVideo/isFile) rendering', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'

    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [
        // The component reverses the list; keep this order so our mocks map predictably.
        // 1) segments empty, meta image -> fallback image
        { id: 'me', Tid: 't-img', content: 'IMG', time: '2026-01-01 00:00:03.000' },
        // 2) segments empty, meta file but no fileUrl/content -> fallback text "文件"
        { id: 'u1', tid: 't-file', content: '', time: '2026-01-01 00:00:02.000' },
        // 3) segments empty, meta video -> fallback video
        { id: 'me', Tid: '', content: 'VIDEO', time: '2026-01-01 00:00:01.000' },
        // 4) message with segments: text/image/video/file
        { id: 'u1', Tid: 't-seg', content: 'seg', time: '2026-01-01 00:00:00.000' }
      ]
    } as any)

    const resolveResults: string[] = []
    vi.mocked(messageSegments.parseMessageSegments).mockImplementation(async (raw: string, opts: any) => {
      if (raw === 'seg') {
        resolveResults.push(await opts.resolveMediaUrl('2026/01/a.png'))
        return [
          { kind: 'text', text: 'hello' },
          { kind: 'image', url: '2026/01/a.png' },
          { kind: 'video', url: '2026/01/b.mp4' },
          { kind: 'file', url: '2026/01/c.pdf', path: 'c.pdf' }
        ] as any
      }

      // Use an attached marker so getSegmentsMeta can return a stable meta even when
      // multiple async calls resolve out of order.
      const empty: any[] = []
      if (raw === '') {
        ;(empty as any).__marker = 'file'
        return empty as any
      }
      if (raw === 'VIDEO') {
        resolveResults.push(await opts.resolveMediaUrl('2026/01/v.mp4'))
        ;(empty as any).__marker = 'video'
        return empty as any
      }
      if (raw === 'IMG') {
        ;(empty as any).__marker = 'image'
        return empty as any
      }
      return empty as any
    })

    vi.mocked(messageSegments.getSegmentsMeta).mockImplementation((segments: any[]) => {
      const marker = (segments as any)?.__marker
      if (marker === 'file') {
        return { hasImage: false, hasVideo: false, hasFile: true, fileUrl: '' } as any
      }
      if (marker === 'video') {
        return { hasImage: false, hasVideo: true, hasFile: false, videoUrl: '2026/01/v.mp4' } as any
      }
      if (marker === 'image') {
        return { hasImage: true, hasVideo: false, hasFile: false, imageUrl: '2026/01/fallback.png' } as any
      }

      const hasImage = segments.some((s: any) => s?.kind === 'image')
      const hasVideo = segments.some((s: any) => s?.kind === 'video')
      const hasFile = segments.some((s: any) => s?.kind === 'file')
      const imageUrl = segments.find((s: any) => s?.kind === 'image')?.url
      const videoUrl = segments.find((s: any) => s?.kind === 'video')?.url
      const fileUrl = segments.find((s: any) => s?.kind === 'file')?.url
      return { hasImage, hasVideo, hasFile, imageUrl, videoUrl, fileUrl } as any
    })

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: {
        plugins: [pinia],
        stubs: { MediaTile: MediaTileStub }
      }
    })

    await wrapper.setProps({ visible: true })
    for (let i = 0; i < 6; i += 1) {
      await flush()
    }

    // targetUserName is missing -> header falls back to targetUserId
    expect(wrapper.text()).toContain('u1')

    // resolveMediaUrl uses systemConfigStore.resolveImagePort and formats upstream url
    expect(resolveResults.some(u => u.includes('http://img.local:9006/img/Upload/'))).toBe(true)

    // segment kinds: file segment renders as a link with its path
    expect(wrapper.text()).toContain('c.pdf')

    // fallback file branch: fileUrl/content empty -> fallback label
    expect(wrapper.text()).toContain('文件')

    // Trigger image error for a specific MediaTile (segment image), then it should render the fallback error UI.
    const tiles = wrapper.findAll('[data-testid="tile"]')
    const segImageTile = tiles.find(t => t.attributes('data-type') === 'image' && t.attributes('data-src') === '2026/01/a.png')
    expect(segImageTile).toBeTruthy()
    await segImageTile!.trigger('click')
    await flush()
    expect(wrapper.text()).toContain('图片加载失败')
  })

  it('resolveMediaUrl returns empty string when imgServer is unavailable', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    const mediaStore = useMediaStore()
    mediaStore.imgServer = ''

    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    const resolveSpy = vi.spyOn(systemConfigStore, 'resolveImagePort')

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ id: 'u1', Tid: 't1', content: 'x', time: 't' }]
    } as any)

    const resolveResults: string[] = []
    vi.mocked(messageSegments.parseMessageSegments).mockImplementationOnce(async (_raw: string, opts: any) => {
      resolveResults.push(await opts.resolveMediaUrl('2026/01/a.png'))
      return [{ kind: 'text', text: 'hi' }] as any
    })
    vi.mocked(messageSegments.getSegmentsMeta).mockReturnValueOnce({ hasImage: false, hasVideo: false, hasFile: false } as any)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: { plugins: [pinia], stubs: { MediaTile: true } }
    })

    await wrapper.setProps({ visible: true })
    for (let i = 0; i < 4; i += 1) {
      await flush()
    }

    expect(resolveResults).toEqual([''])
    expect(resolveSpy).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
