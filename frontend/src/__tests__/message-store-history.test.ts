import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mediaStoreMock = {
  imgServer: '',
  loadImgServer: vi.fn(async () => {
    mediaStoreMock.imgServer = 'img.local'
  })
}

const systemConfigStoreMock = {
  loaded: false,
  loadSystemConfig: vi.fn(async () => {
    systemConfigStoreMock.loaded = true
  }),
  resolveImagePort: vi.fn(async () => '9006')
}

vi.mock('@/stores/media', () => ({
  useMediaStore: () => mediaStoreMock
}))

vi.mock('@/stores/systemConfig', () => ({
  useSystemConfigStore: () => systemConfigStoreMock
}))

vi.mock('@/api/chat', () => ({
  getMessageHistory: vi.fn()
}))

vi.mock('@/utils/cookie', () => ({
  generateCookie: vi.fn(() => 'cookie-data')
}))

vi.mock('@/utils/messageSegments', () => ({
  parseMessageSegments: vi.fn(),
  getSegmentsMeta: vi.fn()
}))

import { useMessageStore } from '@/stores/message'
import * as chatApi from '@/api/chat'
import * as segments from '@/utils/messageSegments'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  mediaStoreMock.imgServer = ''
  systemConfigStoreMock.loaded = false
})

describe('stores/message loadHistory', () => {
  it('returns 0 and initializes empty history when backend responds with error', async () => {
    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ error: 'boom' } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(0)
    expect(store.getMessages('u1')).toEqual([])
    expect(store.loadingMore).toBe(false)
    expect(store.isLoadingHistory).toBe(false)
  })

  it('maps messages, triggers incremental merge cleanup, and prepends older pages', async () => {
    vi.mocked(segments.parseMessageSegments).mockImplementation(async (raw: string, opts: any) => {
      const content = String(raw || '')
      if (content.includes('[img]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.png')
        return [{ kind: 'image', path: '2026/01/a.png', url }]
      }
      if (content.includes('[vid]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.mp4')
        return [{ kind: 'video', path: '2026/01/a.mp4', url }]
      }
      if (content.includes('[file]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.txt')
        return [{ kind: 'file', path: '2026/01/a.txt', url }]
      }
      return [{ kind: 'text', text: content }]
    })

    vi.mocked(segments.getSegmentsMeta).mockImplementation((segs: any[]) => {
      const hasImage = Array.isArray(segs) && segs.some((s) => s?.kind === 'image')
      const hasVideo = Array.isArray(segs) && segs.some((s) => s?.kind === 'video')
      const hasFile = Array.isArray(segs) && segs.some((s) => s?.kind === 'file')
      const imageUrl = hasImage ? String(segs.find((s) => s?.kind === 'image')?.url || '') : ''
      const videoUrl = hasVideo ? String(segs.find((s) => s?.kind === 'video')?.url || '') : ''
      const fileUrl = hasFile ? String(segs.find((s) => s?.kind === 'file')?.url || '') : ''
      return { hasImage, hasVideo, hasFile, imageUrl, videoUrl, fileUrl } as any
    })

    vi.mocked(chatApi.getMessageHistory)
      .mockResolvedValueOnce({
        code: 0,
        contents_list: [
          { content: 'dup', Tid: '2', time: '2026-01-01 00:00:01.000', id: 'u1', nickname: '' },
          { content: '[img]', Tid: '3', time: '2026-01-01 00:00:02.000', id: 'me', nickname: '' }
        ]
      } as any)
      .mockResolvedValueOnce({
        code: 0,
        contents_list: [
          { content: 'dup', Tid: '4', time: '2026-01-01 00:00:01.100', id: 'u1', nickname: 'U1' },
          { content: 'new', Tid: '5', time: '2026-01-01 00:00:03.000', id: 'u1', nickname: '' }
        ]
      } as any)
      .mockResolvedValueOnce({
        code: 0,
        contents_list: [{ content: 'old', Tid: '1', time: '2026-01-01 00:00:00.000', id: 'u1', nickname: '' }]
      } as any)

    const store = useMessageStore()

    const first = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(first).toBe(2)
    expect(mediaStoreMock.loadImgServer).toHaveBeenCalledTimes(1)
    expect(systemConfigStoreMock.loadSystemConfig).toHaveBeenCalledTimes(1)
    expect(systemConfigStoreMock.resolveImagePort).toHaveBeenCalled()

    // add a temporary local message that should be removed by incremental cleanup
    store.addMessage('u1', {
      code: 7,
      fromuser: { id: 'u1', name: '', nickname: '', sex: '未知', ip: '' },
      touser: undefined,
      type: 'text',
      content: 'dup',
      time: '2026-01-01 00:00:01.050',
      tid: '',
      isSelf: false,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any)

    const inc = await store.loadHistory('me', 'u1', { isFirst: true, incremental: true, myUserName: 'Me' })
    expect(inc).toBeGreaterThanOrEqual(0)

    const afterInc = store.getMessages('u1')
    expect(afterInc.some((m) => String(m.content || '') === 'new')).toBe(true)
    expect(afterInc.some((m) => String(m.tid || '') === '2')).toBe(false)

    const more = await store.loadHistory('me', 'u1', { isFirst: false, firstTid: store.firstTidMap['u1'], myUserName: 'Me' })
    expect(more).toBe(1)
    expect(store.getMessages('u1')[0]?.content).toBe('old')
  })

  it('maps msgTid from Tid/tid and assigns image/video/file types', async () => {
    vi.mocked(segments.parseMessageSegments).mockImplementation(async (raw: string, opts: any) => {
      const content = String(raw || '')
      if (content.includes('[img]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.png')
        return [{ kind: 'image', path: '2026/01/a.png', url }]
      }
      if (content.includes('[vid]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.mp4')
        return [{ kind: 'video', path: '2026/01/a.mp4', url }]
      }
      if (content.includes('[file]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.txt')
        return [{ kind: 'file', path: '2026/01/a.txt', url }]
      }
      return [{ kind: 'text', text: content }]
    })

    vi.mocked(segments.getSegmentsMeta).mockImplementation((segs: any[]) => {
      const hasImage = Array.isArray(segs) && segs.some((s) => s?.kind === 'image')
      const hasVideo = Array.isArray(segs) && segs.some((s) => s?.kind === 'video')
      const hasFile = Array.isArray(segs) && segs.some((s) => s?.kind === 'file')
      const imageUrl = hasImage ? String(segs.find((s) => s?.kind === 'image')?.url || '') : ''
      const videoUrl = hasVideo ? String(segs.find((s) => s?.kind === 'video')?.url || '') : ''
      const fileUrl = hasFile ? String(segs.find((s) => s?.kind === 'file')?.url || '') : ''
      return { hasImage, hasVideo, hasFile, imageUrl, videoUrl, fileUrl } as any
    })

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [
        { content: '[img]', Tid: '1', time: '2026-01-01 00:00:00.000', id: 'me', nickname: '' },
        { content: '[vid]', tid: '2', time: '2026-01-01 00:00:01.000', id: 'u1', nickname: 'U1' },
        { content: '[file]', time: '2026-01-01 00:00:02.000', id: 'u1', nickname: 'U1' }
      ]
    } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(3)

    const msgs = store.getMessages('u1') as any[]
    expect(msgs.some(m => m.type === 'image' && String(m.tid) === '1')).toBe(true)
    expect(msgs.some(m => m.type === 'video' && String(m.tid) === '2')).toBe(true)
    expect(msgs.some(m => m.type === 'file' && String(m.tid) === '')).toBe(true)
  })
})
