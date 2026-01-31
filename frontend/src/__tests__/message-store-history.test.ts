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
import { generateCookie } from '@/utils/cookie'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  mediaStoreMock.imgServer = ''
  mediaStoreMock.loadImgServer = vi.fn(async () => {
    mediaStoreMock.imgServer = 'img.local'
  })
  systemConfigStoreMock.loaded = false
  systemConfigStoreMock.loadSystemConfig = vi.fn(async () => {
    systemConfigStoreMock.loaded = true
  })
  systemConfigStoreMock.resolveImagePort = vi.fn(async () => '9006')
})

describe('stores/message loadHistory', () => {
  it('uses default options when omitted (isFirst=true, firstTid=0, myUserName=User)', async () => {
    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ error: 'boom' } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1')
    expect(n).toBe(0)

    expect(vi.mocked(generateCookie)).toHaveBeenCalledWith('me', 'User')
    expect(vi.mocked(chatApi.getMessageHistory).mock.calls[0]?.[2]).toBe('1')
    expect(vi.mocked(chatApi.getMessageHistory).mock.calls[0]?.[3]).toBe('0')
  })

  it('returns 0 and initializes empty history when backend responds with error', async () => {
    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ error: 'boom' } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(0)
    expect(store.getMessages('u1')).toEqual([])
    expect(store.loadingMore).toBe(false)
    expect(store.isLoadingHistory).toBe(false)
  })

  it('does not overwrite existing history when backend responds with error', async () => {
    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ error: 'boom' } as any)

    const store = useMessageStore()
    store.setMessages('u1', [
      {
        code: 7,
        fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        touser: undefined,
        type: 'text',
        content: 'keep',
        time: '2026-01-01 00:00:00.000',
        tid: '1',
        isSelf: false,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: ''
      } as any
    ])

    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(0)
    expect(store.getMessages('u1')[0]?.content).toBe('keep')
  })

  it('initializes empty history on non-array payload and does not clear when entry already exists', async () => {
    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({ code: 0, contents_list: null } as any)

    const store = useMessageStore()
    const n1 = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n1).toBe(0)
    expect(store.getMessages('u1')).toEqual([])

    store.addMessage('u1', {
      code: 7,
      fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
      touser: undefined,
      type: 'text',
      content: 'keep2',
      time: '2026-01-01 00:00:00.000',
      tid: '1',
      isSelf: false,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any)

    const n2 = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n2).toBe(0)
    expect(store.getMessages('u1').some((m) => String(m.content) === 'keep2')).toBe(true)
  })

  it('skips loading system config when already loaded and covers resolveMediaUrl imgServer-missing branch', async () => {
    systemConfigStoreMock.loaded = true
    systemConfigStoreMock.loadSystemConfig = vi.fn(async () => {
      systemConfigStoreMock.loaded = true
    })

    // Keep imgServer missing even after loadImgServer attempt.
    mediaStoreMock.imgServer = ''
    mediaStoreMock.loadImgServer = vi.fn(async () => {
      // no-op
    })

    vi.mocked(segments.parseMessageSegments).mockImplementation(async (raw: string, opts: any) => {
      const content = String(raw || '')
      if (content.includes('[img]')) {
        const url = await opts.resolveMediaUrl('2026/01/a.png')
        if (!url) return [{ kind: 'text', text: content }]
        return [{ kind: 'image', path: '2026/01/a.png', url }]
      }
      return [{ kind: 'text', text: content }]
    })

    vi.mocked(segments.getSegmentsMeta).mockImplementation((segs: any[]) => {
      const hasImage = Array.isArray(segs) && segs.some((s) => s?.kind === 'image' && s?.url)
      return { hasImage, hasVideo: false, hasFile: false, imageUrl: hasImage ? segs[0].url : '', videoUrl: '', fileUrl: '' } as any
    })

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ content: '[img]', Tid: '1', time: '2026-01-01 00:00:00.000', id: 'u1', nickname: '' }]
    } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(1)

    // Already loaded -> should not call loadSystemConfig.
    expect(systemConfigStoreMock.loadSystemConfig).not.toHaveBeenCalled()
    // Missing imgServer -> resolveMediaUrl returns '', no resolveImagePort call.
    expect(systemConfigStoreMock.resolveImagePort).not.toHaveBeenCalled()
  })

  it('does not set firstTidMap when all mapped messages have no tid', async () => {
    vi.mocked(segments.parseMessageSegments).mockResolvedValue([{ kind: 'text', text: 'x' }] as any)
    vi.mocked(segments.getSegmentsMeta).mockReturnValue({ hasImage: false, hasVideo: false, hasFile: false, imageUrl: '', videoUrl: '', fileUrl: '' } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ content: 'x', time: '2026-01-01 00:00:00.000', id: 'u1', nickname: '' }]
    } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(1)
    expect(store.firstTidMap['u1']).toBeUndefined()
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

  it('maps missing fields and exercises fallback branches for content/tid/time/id/nickname', async () => {
    vi.mocked(segments.parseMessageSegments).mockResolvedValue([{ kind: 'text', text: 'x' }] as any)
    vi.mocked(segments.getSegmentsMeta).mockReturnValue({ hasImage: false, hasVideo: false, hasFile: false, imageUrl: '', videoUrl: '', fileUrl: '' } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [
        // nickname falsy + isSelf=true -> fallback to myUserName
        { content: 'x', Tid: '1', time: '2026-01-01 00:00:00.000', id: 'me', nickname: '' },
        // tid path (Tid missing) + time missing + isSelf=false -> nickname fallback to ''
        { content: 'x2', tid: '2', time: undefined, id: 'u1', nickname: '' },
        // id missing + content/time missing + nickname provided -> uses nickname branch, tid fallback to ''
        { content: undefined, Tid: '', time: undefined, id: undefined, nickname: 'Nick' }
      ]
    } as any)

    const store = useMessageStore()
    const n = await store.loadHistory('me', 'u1', { isFirst: true, myUserName: 'Me' })
    expect(n).toBe(3)

    const msgs = store.getMessages('u1') as any[]
    // Covers msgTime fallback -> '' should exist.
    expect(msgs.some(m => String(m.time) === '')).toBe(true)
    // Covers msgTid fallback -> '' should exist.
    expect(msgs.some(m => String(m.tid) === '')).toBe(true)
    // Covers isSelf=false when msg.id === UserToID.
    expect(msgs.some(m => m.isSelf === false)).toBe(true)
    // Covers nickname fallback to myUserName.
    expect(msgs.some(m => String(m.fromuser?.name) === 'Me')).toBe(true)
    // Covers nickname direct value.
    expect(msgs.some(m => String(m.fromuser?.name) === 'Nick')).toBe(true)
  })

  it('incremental cleanup uses parseMessageTime fallback (null -> 0) when times are invalid', async () => {
    vi.mocked(segments.parseMessageSegments).mockResolvedValue([{ kind: 'text', text: 'dup' }] as any)
    vi.mocked(segments.getSegmentsMeta).mockReturnValue({ hasImage: false, hasVideo: false, hasFile: false, imageUrl: '', videoUrl: '', fileUrl: '' } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ content: 'dup', Tid: '1', time: 'bad-time-new', id: 'u1', nickname: '' }]
    } as any)

    const store = useMessageStore()
    // Seed an invalid-time local message that should be removed as a duplicate of mapped data.
    store.setMessages('u1', [
      {
        code: 7,
        fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        touser: undefined,
        type: 'text',
        content: 'dup',
        time: 'bad-time-old',
        tid: '',
        isSelf: false,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: ''
      } as any
    ])

    const n = await store.loadHistory('me', 'u1', { isFirst: true, incremental: true, myUserName: 'Me' })
    // Incremental mode returns an approximate "new messages appended" count,
    // so replacing a local duplicate without changing list length yields 0.
    expect(n).toBe(0)
    // The optimistic/local version should be removed during cleanup, leaving only Tid=1.
    const msgs = store.getMessages('u1') as any[]
    expect(msgs).toHaveLength(1)
    expect(String(msgs[0]?.tid)).toBe('1')
  })

  it('incremental cleanup compares by content when remote path is empty and content is missing', async () => {
    vi.mocked(segments.parseMessageSegments).mockResolvedValue([{ kind: 'text', text: '' }] as any)
    vi.mocked(segments.getSegmentsMeta).mockReturnValue({ hasImage: false, hasVideo: false, hasFile: false, imageUrl: '', videoUrl: '', fileUrl: '' } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ content: undefined, Tid: '', time: '2026-01-01 00:00:01.100', id: 'u1', nickname: '' }]
    } as any)

    const store = useMessageStore()
    // Seed an existing message with empty content and no media path.
    store.setMessages('u1', [
      {
        code: 7,
        fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        touser: undefined,
        type: 'text',
        content: undefined,
        time: '2026-01-01 00:00:01.050',
        tid: '',
        isSelf: false,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: ''
      } as any
    ])

    const n = await store.loadHistory('me', 'u1', { isFirst: true, incremental: true, myUserName: 'Me' })
    expect(n).toBeGreaterThanOrEqual(0)
    // No numeric tid -> should not set firstTidMap in incremental merge branch.
    expect(store.firstTidMap['u1']).toBeUndefined()
  })

  it('isFirst=false path does not set firstTidMap when all tids are empty', async () => {
    vi.mocked(segments.parseMessageSegments).mockResolvedValue([{ kind: 'text', text: 'x' }] as any)
    vi.mocked(segments.getSegmentsMeta).mockReturnValue({ hasImage: false, hasVideo: false, hasFile: false, imageUrl: '', videoUrl: '', fileUrl: '' } as any)

    vi.mocked(chatApi.getMessageHistory).mockResolvedValue({
      code: 0,
      contents_list: [{ content: 'old', Tid: '', time: '2026-01-01 00:00:00.000', id: 'u1', nickname: '' }]
    } as any)

    const store = useMessageStore()
    store.setMessages('u1', [
      {
        code: 7,
        fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        touser: undefined,
        type: 'text',
        content: 'new',
        time: '2026-01-01 00:00:01.000',
        tid: '',
        isSelf: false,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: ''
      } as any
    ])

    const n = await store.loadHistory('me', 'u1', { isFirst: false, firstTid: '0', myUserName: 'Me' })
    expect(n).toBe(1)
    expect(store.firstTidMap['u1']).toBeUndefined()
  })
})
