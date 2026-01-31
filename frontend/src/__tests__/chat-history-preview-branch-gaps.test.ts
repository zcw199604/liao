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

import ChatHistoryPreview from '@/components/chat/ChatHistoryPreview.vue'
import { useIdentityStore } from '@/stores/identity'
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'

import * as chatApi from '@/api/chat'
import * as segments from '@/utils/messageSegments'

const flush = async () => {
  await new Promise<void>(resolve => setTimeout(resolve, 0))
  await nextTick()
}

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type'],
  emits: ['error'],
  template: `<button data-testid="tile" :data-src="src" :data-type="type" @click="$emit('error')"></button>`
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/chat/ChatHistoryPreview.vue (missing branches)', () => {
  it('covers JSON mapping fallbacks, segment file label fallback, and image error keys when tid is empty', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'

    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    // Two messages have empty content: we want to treat the first as image-empty and the second as video-empty.
    let emptyCalls = 0
    vi.mocked(segments.parseMessageSegments).mockImplementation(async (raw: string) => {
      if (raw === 'segimage') {
        return [{ kind: 'image', url: '2026/01/a.png' }] as any
      }
      if (raw === 'segfile') {
        return [{ kind: 'file', url: '2026/01/c.pdf', path: '' }] as any
      }
      if (raw === 'from-content.png') {
        const segs: any[] = []
        ;(segs as any).__marker = 'imgContent'
        return segs as any
      }
      if (raw === 'from-content.mp4') {
        const segs: any[] = []
        ;(segs as any).__marker = 'vidContent'
        return segs as any
      }
      if (raw === '') {
        emptyCalls += 1
        const segs: any[] = []
        ;(segs as any).__marker = emptyCalls === 1 ? 'imgEmpty' : 'vidEmpty'
        return segs as any
      }
      return [] as any
    })

    vi.mocked(segments.getSegmentsMeta).mockImplementation((segs: any[]) => {
      const marker = (segs as any)?.__marker
      if (marker === 'imgContent' || marker === 'imgEmpty') {
        return { hasImage: true, hasVideo: false, hasFile: false, imageUrl: '' } as any
      }
      if (marker === 'vidContent' || marker === 'vidEmpty') {
        return { hasImage: false, hasVideo: true, hasFile: false, videoUrl: '' } as any
      }

      const hasImage = segs.some((s: any) => s?.kind === 'image')
      const hasVideo = segs.some((s: any) => s?.kind === 'video')
      const hasFile = segs.some((s: any) => s?.kind === 'file')
      const imageUrl = segs.find((s: any) => s?.kind === 'image')?.url || ''
      const videoUrl = segs.find((s: any) => s?.kind === 'video')?.url || ''
      const fileUrl = segs.find((s: any) => s?.kind === 'file')?.url || ''
      return { hasImage, hasVideo, hasFile, imageUrl, videoUrl, fileUrl } as any
    })

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [
        // 1) empty Tid/tid -> tid becomes '' -> failedImageIds key falls back to msg.time
        { id: 'u1', Tid: '', content: '', time: '2026-01-01 00:00:00.000' },
        // 2) Tid missing but tid exists -> msgTid uses msg.tid branch; time missing -> msgTime fallback ''
        { id: undefined, tid: 't-lower', content: 'from-content.png', time: undefined },
        // 3) empty content again -> second empty call -> treat as video-empty -> src falls back to ''
        { id: 'me', Tid: 't-vid-empty', content: '', time: '2026-01-01 00:00:02.000' },
        // 4) video fallback uses content when videoUrl is empty
        { id: 'me', Tid: 't-vid-content', content: 'from-content.mp4', time: '2026-01-01 00:00:03.000' },
        // 5) segments image with empty Tid -> failedImageIds key falls back to msg.time for idx
        { id: 'u1', Tid: '', content: 'segimage', time: '2026-01-01 00:00:04.000' },
        // 6) segments file label fallback (seg.path || '文件')
        { id: 'u1', Tid: 't-file-seg', content: 'segfile', time: '2026-01-01 00:00:05.000' }
      ]
    } as any)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: { plugins: [pinia], stubs: { MediaTile: MediaTileStub } }
    })

    await wrapper.setProps({ visible: true })
    for (let i = 0; i < 8; i += 1) {
      await flush()
    }

    // Segment file label fallback should render "文件".
    expect(wrapper.text()).toContain('文件')

    // Fallback image tile (tid empty) -> trigger error -> should show error UI.
    const tiles = wrapper.findAll('[data-testid="tile"]')
    const emptyImageTile = tiles.find(t => t.attributes('data-type') === 'image' && t.attributes('data-src') === '')
    expect(emptyImageTile).toBeTruthy()
    await emptyImageTile!.trigger('click')
    await flush()
    expect(wrapper.text()).toContain('图片加载失败')

    // Segment image tile error should also be handled.
    const segImageTile = tiles.find(t => t.attributes('data-type') === 'image' && t.attributes('data-src') === '2026/01/a.png')
    expect(segImageTile).toBeTruthy()
    await segImageTile!.trigger('click')
    await flush()
    expect(wrapper.text()).toContain('图片加载失败')

    // Cover msgContainer null branch explicitly.
    ;(wrapper.vm as any).msgContainer = null
    ;(wrapper.vm as any).scrollToBottom()
  })

  it('covers XML object response branch and no-ArrayOfMsg branch; visible=false does not reload', async () => {
    const identityStore = useIdentityStore()
    identityStore.saveIdentityCookie('i1', 'cookie-1')

    vi.mocked(chatApi.getMessageHistory)
      .mockResolvedValueOnce({ data: '<ArrayOfMsg><Msg></Msg></ArrayOfMsg>' } as any)
      .mockResolvedValueOnce({ data: '<xml></xml>' } as any)

    const wrapper = mount(ChatHistoryPreview, {
      props: { visible: false, identityId: 'i1', targetUserId: 'u1' },
      global: { stubs: { MediaTile: true } }
    })

    await wrapper.setProps({ visible: true })
    await flush()
    await flush()

    // Toggle to false -> watch branch should not load history.
    await wrapper.setProps({ visible: false })
    await flush()

    await wrapper.setProps({ visible: true })
    await flush()
    await flush()

    expect(wrapper.text()).toContain('暂无聊天记录')
  })
})

