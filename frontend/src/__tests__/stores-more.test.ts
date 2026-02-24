import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

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

vi.mock('@/api/mtphoto', () => ({
  getMtPhotoAlbums: vi.fn(),
  getMtPhotoAlbumFiles: vi.fn(),
  getMtPhotoFolderRoot: vi.fn(),
  getMtPhotoFolderContent: vi.fn(),
  getMtPhotoFolderFavorites: vi.fn(),
  upsertMtPhotoFolderFavorite: vi.fn(),
  removeMtPhotoFolderFavorite: vi.fn()
}))

vi.mock('@/api/identity', () => ({
  getIdentityList: vi.fn(),
  createIdentity: vi.fn(),
  deleteIdentity: vi.fn(),
  selectIdentity: vi.fn()
}))

vi.mock('@/api/favorite', () => ({
  listAllFavorites: vi.fn(),
  addFavorite: vi.fn(),
  removeFavorite: vi.fn(),
  removeFavoriteById: vi.fn()
}))

vi.mock('@/api/videoExtract', () => ({
  probeVideo: vi.fn(),
  createVideoExtractTask: vi.fn(),
  getVideoExtractTaskList: vi.fn(),
  getVideoExtractTaskDetail: vi.fn(),
  cancelVideoExtractTask: vi.fn(),
  continueVideoExtractTask: vi.fn(),
  deleteVideoExtractTask: vi.fn()
}))

import { useSystemConfigStore } from '@/stores/systemConfig'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useIdentityStore } from '@/stores/identity'
import { useFavoriteStore } from '@/stores/favorite'
import { useUserStore } from '@/stores/user'
import { useDouyinStore } from '@/stores/douyin'
import { useVideoExtractStore } from '@/stores/videoExtract'

import * as systemApi from '@/api/system'
import * as mediaApi from '@/api/media'
import * as mtphotoApi from '@/api/mtphoto'
import * as identityApi from '@/api/identity'
import * as favoriteApi from '@/api/favorite'
import * as videoExtractApi from '@/api/videoExtract'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('stores/systemConfig', () => {
  it('loads and saves system config, and fixed mode returns fixed port', async () => {
    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)
    vi.mocked(systemApi.updateSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'fixed', imagePortFixed: '9007', imagePortRealMinBytes: 2048 }
    } as any)

    const store = useSystemConfigStore()
    await store.loadSystemConfig()
    expect(store.loaded).toBe(true)
    expect(store.imagePortFixed).toBe('9006')

    const ok = await store.saveSystemConfig({ imagePortFixed: '9007' } as any)
    expect(ok).toBe(true)
    expect(store.imagePortFixed).toBe('9007')

    const port = await store.resolveImagePort('2026/01/a.png', 'img.local')
    expect(port).toBe('9007')
    expect(systemApi.resolveImagePort).not.toHaveBeenCalled()
  })

  it('resolves image port with caching and clears cache when server changes', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))
    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'probe', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)
    vi.mocked(systemApi.resolveImagePort).mockResolvedValue({ code: 0, data: { port: '9010' } } as any)

    const store = useSystemConfigStore()
    const port1 = await store.resolveImagePort('2026/01/a.png', 'img-1')
    expect(port1).toBe('9010')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(1)

    const port2 = await store.resolveImagePort('2026/01/a.png', 'img-1')
    expect(port2).toBe('9010')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(1)

    vi.mocked(systemApi.resolveImagePort).mockResolvedValue({ code: 0, data: { port: '9020' } } as any)
    const port3 = await store.resolveImagePort('2026/01/a.png', 'img-2')
    expect(port3).toBe('9020')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(2)

    vi.useRealTimers()
  })

  it('falls back to fixed port when resolve fails', async () => {
    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'probe', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)
    vi.mocked(systemApi.resolveImagePort).mockRejectedValue(new Error('boom'))

    const store = useSystemConfigStore()
    const port = await store.resolveImagePort('2026/01/a.png')
    expect(port).toBe('9006')
  })

  it('applies positive mtphoto timeline threshold with floor on load/save responses', async () => {
    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: {
        imagePortMode: 'fixed',
        imagePortFixed: '9006',
        imagePortRealMinBytes: 2048,
        mtPhotoTimelineDeferSubfolderThreshold: 12.9
      }
    } as any)
    vi.mocked(systemApi.updateSystemConfig).mockResolvedValue({
      code: 0,
      data: {
        imagePortMode: 'fixed',
        imagePortFixed: '9006',
        imagePortRealMinBytes: 2048,
        mtPhotoTimelineDeferSubfolderThreshold: 7.2
      }
    } as any)

    const store = useSystemConfigStore()
    await store.loadSystemConfig()
    expect(store.mtPhotoTimelineDeferSubfolderThreshold).toBe(12)

    const ok = await store.saveSystemConfig({ mtPhotoTimelineDeferSubfolderThreshold: 7.2 } as any)
    expect(ok).toBe(true)
    expect(store.mtPhotoTimelineDeferSubfolderThreshold).toBe(7)
  })
})

describe('stores/media', () => {
  it('loads imgServer from backend and persists it', async () => {
    vi.mocked(mediaApi.getImgServerAddress).mockResolvedValue({ state: 'OK', msg: { server: 'img.local' } } as any)
    vi.mocked(mediaApi.updateImgServerAddress).mockResolvedValue({ state: 'OK' } as any)

    const store = useMediaStore()
    await store.loadImgServer()
    expect(store.imgServer).toBe('img.local')
    expect(mediaApi.updateImgServerAddress).toHaveBeenCalledWith('img.local')
  })

  it('loads cached images and maps upload urls using systemConfig port resolver', async () => {
    vi.mocked(mediaApi.getCachedImages).mockResolvedValue({
      data: ['/upload/images/2026/01/a.png', '/upload/videos/2026/01/b.mp4', null, '']
    } as any)

    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)

    const store = useMediaStore()
    store.imgServer = 'img.local'

    await store.loadCachedImages('me')
    expect(store.uploadedMedia.length).toBe(2)
    expect(store.uploadedMedia[0]!.url).toContain('http://img.local:9006/img/Upload/2026/01/a.png')
    expect(store.uploadedMedia[1]!.url).toContain('http://img.local:9006/img/Upload/2026/01/b.mp4')
  })

  it('handles cached image api returning a non-array value', async () => {
    vi.mocked(mediaApi.getCachedImages).mockResolvedValue({ data: { not: 'array' } } as any)
    const store = useMediaStore()
    await store.loadCachedImages('me')
    expect(store.uploadedMedia).toEqual([])
  })

  it('loads cached images when backend returns array directly and keeps local urls when imgServer is empty', async () => {
    vi.mocked(mediaApi.getCachedImages).mockResolvedValue([
      '/upload/images/2026/01/a.png',
      '/upload/videos/2026/01/b.mp4'
    ] as any)

    vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
      code: 0,
      data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
    } as any)

    const store = useMediaStore()
    store.imgServer = ''

    await store.loadCachedImages('me')
    expect(store.uploadedMedia).toHaveLength(2)
    expect(store.uploadedMedia[0]!.url).toBe('/upload/images/2026/01/a.png')
    expect(store.uploadedMedia[1]!.url).toBe('/upload/videos/2026/01/b.mp4')
  })

  it('loadCachedImages skips systemConfig load when systemConfigStore is already loaded', async () => {
    vi.mocked(mediaApi.getCachedImages).mockResolvedValue({
      data: ['/upload/images/2026/01/a.png']
    } as any)

    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.imagePortMode = 'fixed' as any
    systemConfigStore.imagePortFixed = '9006'

    const store = useMediaStore()
    store.imgServer = ''

    const loadSpy = vi.spyOn(systemConfigStore, 'loadSystemConfig')
    await store.loadCachedImages('me')
    expect(loadSpy).not.toHaveBeenCalled()
    expect(store.uploadedMedia).toHaveLength(1)
  })

  it('loadAllUploadImages replaces first page, appends next page, and computes totalPages when missing', async () => {
    vi.mocked(mediaApi.getAllUploadImages)
      .mockResolvedValueOnce({
        data: [
          {
            url: 'u1',
            type: 'image',
            localFilename: 'a.png',
            originalFilename: 'A.png',
            fileSize: 1,
            fileType: 'image/png',
            fileExtension: '.png',
            uploadTime: 't1',
            updateTime: 't1'
          }
        ],
        total: 21,
        page: 1,
        pageSize: 20
      } as any)
      .mockResolvedValueOnce({
        data: [
          {
            url: 'u2',
            type: 'video',
            localFilename: 'b.mp4',
            originalFilename: 'B.mp4',
            fileSize: 2,
            fileType: 'video/mp4',
            fileExtension: '.mp4',
            uploadTime: 't2',
            updateTime: 't2'
          }
        ],
        total: 21,
        page: 2,
        pageSize: 20,
        totalPages: 2
      } as any)

    const store = useMediaStore()
    store.allUploadSource = 'douyin'
    store.allUploadDouyinSecUserId = '  sec-1  '

    await store.loadAllUploadImages(1)
    expect(mediaApi.getAllUploadImages).toHaveBeenCalledWith(1, 20, { source: 'douyin', douyinSecUserId: 'sec-1' })
    expect(store.allUploadImages).toHaveLength(1)
    expect(store.allUploadImages[0]!.url).toBe('u1')
    expect(store.allUploadTotal).toBe(21)
    expect(store.allUploadTotalPages).toBe(2)

    await store.loadAllUploadImages(2)
    expect(store.allUploadImages).toHaveLength(2)
    expect(store.allUploadImages[1]!.url).toBe('u2')
    expect(store.allUploadPage).toBe(2)
    expect(store.allUploadTotalPages).toBe(2)
  })

  it('loadAllUploadImages maps douyin author metadata into context.work', async () => {
    vi.mocked(mediaApi.getAllUploadImages).mockResolvedValue({
      data: [
        {
          url: 'u-dy-1',
          type: 'image',
          localFilename: 'dy-1.jpg',
          source: 'douyin',
          douyinSecUserId: 'sec-888',
          douyinDetailId: 'detail-1',
          douyinAuthorUniqueId: 'dy_author_1',
          douyinAuthorName: '作者A'
        }
      ],
      total: 1,
      page: 1,
      pageSize: 20
    } as any)

    const store = useMediaStore()
    await store.loadAllUploadImages(1)

    expect(store.allUploadImages).toHaveLength(1)
    expect(store.allUploadImages[0]?.context?.provider).toBe('douyin')
    expect(store.allUploadImages[0]?.context?.work?.authorSecUserId).toBe('sec-888')
    expect(store.allUploadImages[0]?.context?.work?.detailId).toBe('detail-1')
    expect(store.allUploadImages[0]?.context?.work?.authorUniqueId).toBe('dy_author_1')
    expect(store.allUploadImages[0]?.context?.work?.authorName).toBe('作者A')
  })

  it('loadAllUploadImages keeps douyin context and falls back optional author fields to undefined', async () => {
    vi.mocked(mediaApi.getAllUploadImages).mockResolvedValue({
      data: [
        {
          url: 'u-dy-empty',
          type: 'image',
          localFilename: 'dy-empty.jpg',
          source: 'douyin',
          douyinSecUserId: '   ',
          douyinDetailId: '',
          douyinAuthorUniqueId: '',
          douyinAuthorName: ''
        }
      ],
      total: 1,
      page: 1,
      pageSize: 20
    } as any)

    const store = useMediaStore()
    await store.loadAllUploadImages(1)

    const work = store.allUploadImages[0]?.context?.work
    expect(store.allUploadImages[0]?.context?.provider).toBe('douyin')
    expect(work?.detailId).toBeUndefined()
    expect(work?.authorSecUserId).toBeUndefined()
    expect(work?.authorUniqueId).toBeUndefined()
    expect(work?.authorName).toBeUndefined()
  })

  it('loadAllUploadImages ignores non-array response payloads', async () => {
    vi.mocked(mediaApi.getAllUploadImages).mockResolvedValue({ data: { nope: true } } as any)
    const store = useMediaStore()
    await store.loadAllUploadImages(1)
    expect(store.allUploadImages).toEqual([])
    expect(store.allUploadLoading).toBe(false)
  })



  it('loadCachedImages clears uploadedMedia when api throws', async () => {
    vi.mocked(mediaApi.getCachedImages).mockRejectedValue(new Error('boom'))

    const store = useMediaStore()
    store.uploadedMedia = [{ url: 'x', type: 'image' } as any]
    await store.loadCachedImages('me')

    expect(store.uploadedMedia).toEqual([])
  })

  it('loadCachedImages covers defensive non-array guard with unstable payload getter', async () => {
    let access = 0
    const unstable: any = {}
    Object.defineProperty(unstable, 'data', {
      get() {
        access += 1
        // First access (for Array.isArray check) is array, second access (assignment) is not.
        return access === 1 ? ['/upload/images/2026/01/a.png'] : { not: 'array' }
      }
    })
    vi.mocked(mediaApi.getCachedImages).mockResolvedValue(unstable)

    const store = useMediaStore()
    store.uploadedMedia = [{ url: 'keep', type: 'image' } as any]

    await store.loadCachedImages('me')
    expect(store.uploadedMedia).toEqual([])
  })

  it('removeUploadedMedia and clearUploadedMedia mutate list as expected', () => {
    const store = useMediaStore()
    store.uploadedMedia = [
      { url: 'u1', type: 'image' } as any,
      { url: 'u2', type: 'video' } as any
    ]

    store.removeUploadedMedia('u1')
    expect(store.uploadedMedia.map(i => i.url)).toEqual(['u2'])

    store.clearUploadedMedia()
    expect(store.uploadedMedia).toEqual([])
  })

  it('loadAllUploadImages falls back to default source and pagination fields', async () => {
    vi.mocked(mediaApi.getAllUploadImages).mockResolvedValue({ data: [], total: 0 } as any)
    const store = useMediaStore()
    store.allUploadSource = '' as any
    store.allUploadDouyinSecUserId = ''

    await store.loadAllUploadImages(3)
    expect(mediaApi.getAllUploadImages).toHaveBeenCalledWith(3, 20, { source: 'all', douyinSecUserId: undefined })
    expect(store.allUploadTotal).toBe(0)
    expect(store.allUploadPage).toBe(3)
    expect(store.allUploadPageSize).toBe(20)
    expect(store.allUploadTotalPages).toBe(0)
  })
})

describe('stores/mtphoto', () => {
  it('loads albums with favorites entry and filters upstream favorites album', async () => {
    vi.useFakeTimers()
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockResolvedValue({
      data: [
        { id: 1, name: '收藏夹(上游)', cover: 'x', count: 9 },
        { id: 2, name: 'A', cover: 'c', count: 1 }
      ]
    } as any)
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockResolvedValue({ total: 123, data: [], page: 1 } as any)

    const store = useMtPhotoStore()
    await store.open()
    expect(store.showModal).toBe(true)
    expect(store.view).toBe('albums')
    expect(store.albums[0]?.isFavorites).toBe(true)
    expect(store.albums[0]?.mtPhotoAlbumId).toBe(1)
    expect(store.albums.some(a => a.mtPhotoAlbumId === 2)).toBe(true)

    await vi.runAllTicks()
    await vi.runAllTimersAsync()
    expect(store.albums[0]?.count).toBe(123)
    vi.useRealTimers()
  })

  it('handles album load error and records lastError', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockRejectedValue({ message: 'nope' } as any)
    const store = useMtPhotoStore()
    await store.open()
    expect(store.albums).toEqual([])
    expect(store.lastError).toBe('nope')
  })

  it('opens an album and loads more pages', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockResolvedValue({ data: [] } as any)
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles)
      // background favorites count fetch during loadAlbums
      .mockResolvedValueOnce({ data: [], total: 0, page: 1, pageSize: 1, totalPages: 0 } as any)
      .mockResolvedValueOnce({ data: [{ id: 1, md5: 'm1', type: 'image' }], total: 2, page: 1, pageSize: 1, totalPages: 2 } as any)
      .mockResolvedValueOnce({ data: [{ id: 2, md5: 'm2', type: 'image' }], total: 2, page: 2, pageSize: 1, totalPages: 2 } as any)

    const store = useMtPhotoStore()
    await store.open()
    const album = { id: 2, mtPhotoAlbumId: 2, name: 'A', cover: '', count: 0 } as any
    await store.openAlbum(album)
    expect(store.view).toBe('album')
    expect(store.mediaItems.length).toBe(1)
    expect(store.selectedAlbum?.count).toBe(2)

    await store.loadMore()
    expect(store.mediaItems.length).toBe(2)
  })

  it('tolerates non-array album payloads and logs warning when favorites count fails', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockResolvedValue({ data: { nope: true } } as any)
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockRejectedValueOnce(new Error('boom'))

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      const store = useMtPhotoStore()
      await store.open()
      expect(store.albums).toHaveLength(1)
      expect(store.albums[0]?.isFavorites).toBe(true)

      // background favorites count fetch is not awaited by open()
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(warnSpy).toHaveBeenCalled()
    } finally {
      warnSpy.mockRestore()
    }
  })

  it('loadMore returns early for missing album / loading state / last page', async () => {
    const store = useMtPhotoStore()

    await store.loadMore()
    expect(mtphotoApi.getMtPhotoAlbumFiles).not.toHaveBeenCalled()

    store.selectedAlbum = { id: 2, mtPhotoAlbumId: 2, name: 'A', cover: '', count: 0 } as any
    store.mediaLoading = true
    await store.loadMore()
    expect(mtphotoApi.getMtPhotoAlbumFiles).not.toHaveBeenCalled()

    store.mediaLoading = false
    store.mediaPage = 2
    store.mediaTotalPages = 2
    await store.loadMore()
    expect(mtphotoApi.getMtPhotoAlbumFiles).not.toHaveBeenCalled()
  })

  it('openAlbum handles errors on page=1 and later pages', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockRejectedValueOnce({
      response: { data: { error: 'bad' } }
    } as any)

    const store = useMtPhotoStore()
    await store.openAlbum({ id: 2, mtPhotoAlbumId: 2, name: 'A', cover: '', count: 0 } as any)

    expect(store.mediaItems).toEqual([])
    expect(store.lastError).toBe('bad')
    expect(store.mediaLoading).toBe(false)

    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles)
      .mockResolvedValueOnce({ data: [{ id: 1, md5: 'm1', type: 'image' }], total: 2, page: 1, pageSize: 1, totalPages: 2 } as any)
      .mockRejectedValueOnce({ message: 'nope' } as any)

    setActivePinia(createPinia())
    const store2 = useMtPhotoStore()
    await store2.openAlbum({ id: 3, mtPhotoAlbumId: 3, name: 'B', cover: '', count: 0 } as any)
    expect(store2.mediaItems).toHaveLength(1)

    await store2.loadMore()
    expect(store2.mediaItems).toHaveLength(1)
    expect(store2.lastError).toBe('nope')
  })

  it('openAlbum treats non-array file payload as empty list', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockResolvedValue({
      data: { nope: true },
      total: 0,
      page: 1,
      pageSize: 60,
      totalPages: 0
    } as any)

    const store = useMtPhotoStore()
    await store.openAlbum({ id: 2, mtPhotoAlbumId: 2, name: 'A', cover: '', count: 0 } as any)

    expect(store.mediaItems).toEqual([])
    expect(store.mediaTotal).toBe(0)
  })

  it('covers album mapping fallbacks for optional fields and start/end time branches', async () => {
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockResolvedValue({
      data: [
        // upstream favorites album should be filtered out from mapped list
        { id: 1, name: undefined, cover: undefined, count: undefined },
        // regular album with missing fields + start/end time
        { id: 2, name: undefined, cover: undefined, count: undefined, startTime: 's', endTime: 'e' }
      ]
    } as any)
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockResolvedValue({ total: 0, data: [], page: 1 } as any)

    const store = useMtPhotoStore()
    await store.open()

    const a2 = store.albums.find(a => a.mtPhotoAlbumId === 2) as any
    expect(a2).toBeTruthy()
    expect(a2.name).toBe('')
    expect(a2.cover).toBe('')
    expect(a2.count).toBe(0)
    expect(a2.startTime).toBe('s')
    expect(a2.endTime).toBe('e')
  })

  it('covers mtphoto error message fallbacks for albums and files', async () => {
    // loadAlbums: no response.error and no message -> uses default
    vi.mocked(mtphotoApi.getMtPhotoAlbums).mockRejectedValueOnce({} as any)
    const store = useMtPhotoStore()
    await store.open()
    expect(store.lastError).toBe('加载失败')

    // loadAlbumPage: missing fields fall back to provided page and existing pageSize
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockResolvedValueOnce({ data: [], total: 0, totalPages: 0 } as any)
    const store2 = useMtPhotoStore()
    await store2.openAlbum({ id: 2, mtPhotoAlbumId: 2, name: 'A', cover: '', count: 0 } as any)
    expect(store2.mediaPage).toBe(1)
    expect(store2.mediaPageSize).toBe(60)

    // loadAlbumPage catch: no response.error and no message -> uses default
    vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mockRejectedValueOnce({} as any)
    const store3 = useMtPhotoStore()
    await store3.openAlbum({ id: 3, mtPhotoAlbumId: 3, name: 'B', cover: '', count: 0 } as any)
    expect(store3.lastError).toBe('加载失败')
  })

  it('switchMode folders covers root/favorites loading and favorite filter/sort/group branches', async () => {
    vi.useFakeTimers()
    try {
      const systemConfigStore = useSystemConfigStore()
      systemConfigStore.loaded = true
      systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 2

      vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValue({
        path: '/root',
        folderList: [{ id: 10, name: '目录A', path: '/root/A', cover: 'c1,c2', s_cover: 's1', subFolderNum: 1 }],
        fileList: [{ id: 1, md5: 'm1', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)

      vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValue({
        items: [
          { id: 1, folderId: 10, folderName: '目录A', folderPath: '/root/A', tags: ['tag1', 'tag2', 'tag1'], note: 'n1', updateTime: '2026-01-01' },
          { id: 2, folderId: 11, folderName: '目录B', folderPath: '/root/B', tags: ['tag2'], note: '', createTime: '2026-01-02' },
          { id: 3, folderId: 12, folderName: '目录C', folderPath: '/root/C', tags: [] }
        ]
      } as any)

      const store = useMtPhotoStore()
      await store.switchMode('folders')

      expect(store.mode).toBe('folders')
      expect(store.view).toBe('folders')
      expect(store.folderList).toHaveLength(1)
      expect(store.folderFiles).toHaveLength(1)
      expect(store.folderFavorites).toHaveLength(3)
      expect(store.allUniqueTags).toEqual(['tag1', 'tag2'])

      store.favoriteFilterMode = 'all'
      store.setFavoriteFilterKeyword('tag1 tag2', { immediate: true })
      expect(store.filteredFolderFavorites).toHaveLength(1)

      store.setFavoriteFilterKeyword('tag2', { debounceMs: 50 })
      expect(store.favoriteFilterKeyword).toBe('tag1 tag2')
      vi.advanceTimersByTime(50)
      expect(store.favoriteFilterKeyword).toBe('tag2')
      expect(store.filteredFolderFavorites).toHaveLength(2)

      store.setFavoriteFilterKeyword('', { immediate: true })

      store.favoriteSortBy = 'name'
      store.favoriteSortOrder = 'asc'
      expect(store.sortedFolderFavorites[0]?.folderId).toBe(10)

      store.favoriteSortBy = 'tagCount'
      store.favoriteSortOrder = 'desc'
      expect(store.sortedFolderFavorites[0]?.folderId).toBe(10)

      store.favoriteGroupBy = 'tag'
      const groupKeys = store.groupedFolderFavorites.map(g => g.key)
      expect(groupKeys).toContain('tag1')
      expect(groupKeys).toContain('tag2')
      expect(groupKeys).toContain('未标记')

      store.startEditFavorite(store.folderFavorites[0] as any)
      expect(store.favoriteEditingFolderId).toBe(10)
      expect(store.favoriteDraftTags).toContain('tag1')

      store.resetFavoriteFilter()
      expect(store.favoriteFilterKeyword).toBe('')
      expect(store.favoriteFilterMode).toBe('any')
      expect(store.favoriteSortBy).toBe('updatedAt')
      expect(store.favoriteSortOrder).toBe('desc')
      expect(store.favoriteGroupBy).toBe('none')

      store.cancelEditFavorite()
      expect(store.favoriteEditingFolderId).toBeNull()

      await store.switchMode('folders') // nextMode === mode 早退分支
      await store.switchMode('albums')
      expect(store.mode).toBe('albums')
      expect(store.view).toBe('albums')
    } finally {
      vi.useRealTimers()
    }
  })

  it('openFavoriteFolder + loadFolderTimeline covers deferred/success/failure branches', async () => {
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 1

    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({
        path: '/root/F20',
        folderList: [{ id: 201, name: '子目录1' }, { id: 202, name: '子目录2' }],
        fileList: [{ id: 1, md5: 'm1', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/F20',
        folderList: [],
        fileList: [{ id: 2, md5: 'm2', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)
      .mockRejectedValueOnce({ response: { data: { error: 'timeline boom' } } } as any)

    const store = useMtPhotoStore()
    const ok = await store.openFavoriteFolder({
      id: 1,
      folderId: 20,
      folderName: '收藏目录20',
      folderPath: '/root/F20',
      coverMd5: 'cover-20',
      tags: []
    } as any)

    expect(ok).toBe(true)
    expect(store.folderCurrentId).toBe(20)
    expect(store.folderTimelineDeferred).toBe(true)
    expect(store.folderFiles).toEqual([])
    expect(store.folderHistory).toHaveLength(2)

    const loaded = await store.loadFolderTimeline()
    expect(loaded).toBe(true)
    expect(store.folderTimelineDeferred).toBe(false)
    expect(store.folderFiles).toHaveLength(1)

    store.folderTimelineDeferred = true
    const loadedFail = await store.loadFolderTimeline()
    expect(loadedFail).toBe(false)
    expect(store.lastError).toBe('timeline boom')
  })

  it('loadFolderMore covers early returns, append success and error branches', async () => {
    const store = useMtPhotoStore()

    await store.loadFolderMore()
    expect(mtphotoApi.getMtPhotoFolderContent).not.toHaveBeenCalled()

    store.folderCurrentId = 7
    store.folderLoading = true
    await store.loadFolderMore()
    expect(mtphotoApi.getMtPhotoFolderContent).not.toHaveBeenCalled()

    store.folderLoading = false
    store.folderTimelineDeferred = true
    await store.loadFolderMore()
    expect(mtphotoApi.getMtPhotoFolderContent).not.toHaveBeenCalled()

    store.folderTimelineDeferred = false
    store.folderPage = 2
    store.folderTotalPages = 2
    await store.loadFolderMore()
    expect(mtphotoApi.getMtPhotoFolderContent).not.toHaveBeenCalled()

    store.folderFiles = [{ id: 1, md5: 'm1', type: 'image' } as any]
    store.folderPage = 1
    store.folderTotalPages = 3
    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockResolvedValueOnce({
      path: '/root/F7',
      folderList: [],
      fileList: [{ id: 2, md5: 'm2', fileType: 'JPG' }],
      total: 2,
      page: 2,
      pageSize: 60,
      totalPages: 3
    } as any)
    await store.loadFolderMore()
    expect(store.folderFiles).toHaveLength(2)
    expect(store.folderPage).toBe(2)

    store.folderPage = 2
    store.folderTotalPages = 4
    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockRejectedValueOnce({ message: 'more boom' } as any)
    await store.loadFolderMore()
    expect(store.lastError).toBe('more boom')
  })

  it('openFolder/backFolder covers history navigation branches', async () => {
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 10

    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({
        path: '/root/A',
        folderList: [],
        fileList: [{ id: 1, md5: 'm1', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/A/B',
        folderList: [],
        fileList: [{ id: 2, md5: 'm2', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/A',
        folderList: new Array(11).fill(null).map((_, idx) => ({ id: 300 + idx, name: `S${idx}` })),
        fileList: [{ id: 3, md5: 'm3', fileType: 'JPG' }],
        total: 1,
        page: 1,
        pageSize: 60,
        totalPages: 1
      } as any)

    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValue({
      path: '',
      folderList: [],
      fileList: [],
      total: 0,
      page: 1,
      pageSize: 60,
      totalPages: 0
    } as any)

    const store = useMtPhotoStore()
    await store.openFolder({ id: 101, name: '目录A', cover: 'c1,c2', sCover: null, subFolderNum: 0 } as any)
    await store.openFolder({ id: 102, name: '目录B', cover: 'c3,c4', sCover: 's-cover', subFolderNum: 0 } as any)
    expect(store.folderHistory).toHaveLength(3)

    store.folderLoading = true
    await store.backFolder()
    expect(store.folderCurrentId).toBe(102)

    store.folderLoading = false
    await store.backFolder()
    expect(store.folderCurrentId).toBe(101)
    expect(store.folderTimelineDeferred).toBe(true)

    await store.backFolder()
    expect(store.folderCurrentId).toBeNull()
    expect(store.folderHistory).toHaveLength(1)

    await store.backFolder()
    expect(store.folderCurrentId).toBeNull()
    expect(store.folderHistory).toHaveLength(1)
  })

  it('loadFolderFavorites/openFavoriteFolder covers cancel-edit, invalid and failure branches', async () => {
    const store = useMtPhotoStore()
    store.favoriteEditingFolderId = 999

    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({
      items: [
        { id: 1, folderId: 1, folderName: 'A', folderPath: '/A', tags: ['x'] },
        { id: 2, folderId: 0, folderName: 'invalid', folderPath: '/invalid', tags: [] }
      ]
    } as any)

    await store.loadFolderFavorites({ tagKeyword: 'x' } as any)
    expect(store.folderFavorites).toHaveLength(1)
    expect(store.favoriteEditingFolderId).toBeNull()

    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockRejectedValueOnce({ message: 'fav boom' } as any)
    await store.loadFolderFavorites()
    expect(store.folderFavorites).toEqual([])
    expect(store.lastError).toBe('fav boom')

    expect(await store.openFavoriteFolder(null as any)).toBe(false)

    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockRejectedValueOnce({ message: 'open boom' } as any)
    expect(
      await store.openFavoriteFolder({ id: 2, folderId: 22, folderName: 'F22', folderPath: '/F22', tags: [] } as any)
    ).toBe(false)
  })

  it('upsert/remove favorite and upsertCurrent cover validation/success/fallback/error branches', async () => {
    const store = useMtPhotoStore()

    expect(await store.upsertFolderFavorite({ folderId: 0, folderName: 'X', folderPath: '/X' } as any)).toBe(false)
    expect(store.lastError).toBe('folderId 参数非法')

    expect(await store.upsertFolderFavorite({ folderId: 1, folderName: 'X', folderPath: '   ' } as any)).toBe(false)
    expect(store.lastError).toBe('folderPath 不能为空')

    vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({
      success: true,
      item: { id: 1, folderId: 1, folderName: 'F1', folderPath: '/F1', coverMd5: 'c1', tags: ['t1'], note: 'n1' }
    } as any)
    expect(await store.upsertFolderFavorite({ folderId: 1, folderName: 'F1', folderPath: '/F1', tags: ['t1'] })).toBe(true)
    expect(store.folderFavorites[0]?.folderId).toBe(1)

    store.folderCurrentId = 1
    vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({
      success: true,
      item: { id: 2, folderId: 1, folderName: 'F1-NEW', folderPath: '/F1', coverMd5: 'c2', tags: ['t2'], note: 'n2' }
    } as any)
    expect(await store.upsertFolderFavorite({ folderId: 1, folderName: 'F1-NEW', folderPath: '/F1', tags: ['t2'] })).toBe(true)

    vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({ success: true, item: null } as any)
    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({
      items: [{ id: 3, folderId: 3, folderName: 'F3', folderPath: '/F3', tags: [] }]
    } as any)
    expect(await store.upsertFolderFavorite({ folderId: 3, folderName: 'F3', folderPath: '/F3' })).toBe(true)
    expect(mtphotoApi.getMtPhotoFolderFavorites).toHaveBeenCalled()

    vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockRejectedValueOnce({ response: { data: { error: 'save boom' } } } as any)
    expect(await store.upsertFolderFavorite({ folderId: 4, folderName: 'F4', folderPath: '/F4' })).toBe(false)
    expect(store.lastError).toBe('save boom')

    store.folderCurrentId = null
    expect(await store.upsertCurrentFolderFavorite({ tags: ['x'] })).toBe(false)

    store.folderCurrentId = 5
    store.folderCurrentName = ''
    store.folderPath = '/root/F5'
    store.folderCurrentCoverMd5 = ''
    vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({
      success: true,
      item: { id: 5, folderId: 5, folderName: 'F5', folderPath: '/root/F5', coverMd5: 'c5', tags: ['a', 'b'], note: 'memo' }
    } as any)
    expect(await store.upsertCurrentFolderFavorite({ tags: ['a', 'b'], note: 'memo' })).toBe(true)
    expect(store.favoriteDraftTags).toBe('a, b')
    expect(store.favoriteDraftNote).toBe('memo')

    expect(await store.removeFolderFavorite(0)).toBe(false)

    store.favoriteEditingFolderId = 5
    vi.mocked(mtphotoApi.removeMtPhotoFolderFavorite).mockResolvedValueOnce({ success: true } as any)
    expect(await store.removeFolderFavorite(5)).toBe(true)
    expect(store.favoriteEditingFolderId).toBeNull()

    vi.mocked(mtphotoApi.removeMtPhotoFolderFavorite).mockResolvedValueOnce({ success: false } as any)
    expect(await store.removeFolderFavorite(9)).toBe(false)

    vi.mocked(mtphotoApi.removeMtPhotoFolderFavorite).mockRejectedValueOnce({ message: 'rm boom' } as any)
    expect(await store.removeFolderFavorite(10)).toBe(false)
    expect(store.lastError).toBe('rm boom')
  })

  it('covers currentFolderFavorite + updatedAt sort fallback branches', () => {
    const store = useMtPhotoStore()

    // currentFolderFavorite: folderCurrentId 为空
    expect(store.currentFolderFavorite).toBeNull()

    store.folderFavorites = [
      { id: 1, folderId: 1, folderName: 'A', folderPath: '/A', tags: [], updateTime: 'bad-time' },
      { id: 2, folderId: 2, folderName: 'B', folderPath: '/B', tags: [], createTime: '2026-01-02 00:00:00' },
      { id: 3, folderId: 3, folderName: 'C', folderPath: '/C', tags: [], createTime: '' }
    ] as any

    // currentFolderFavorite: find(...) || null 的 null 分支
    store.folderCurrentId = 999
    expect(store.currentFolderFavorite).toBeNull()

    // currentFolderFavorite: 命中分支
    store.folderCurrentId = 2
    expect(store.currentFolderFavorite?.folderId).toBe(2)

    // updatedAt 排序：覆盖 toTimestamp 的 NaN -> 0 分支 + value||'' 分支
    store.favoriteSortBy = 'updatedAt'
    store.favoriteSortOrder = 'asc'
    const sorted = store.sortedFolderFavorites
    expect(sorted.map(item => item.folderId)).toEqual([1, 3, 2])
  })

  it('covers loadFolderRoot/loadFolderMore pagination fallbacks and folderName fallback', async () => {
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 10

    const store = useMtPhotoStore()

    // applyFolderContent append=false: page/pageSize/total/totalPages 缺省分支
    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: [],
      fileList: []
    } as any)
    await store.loadFolderRoot()
    expect(store.folderPage).toBe(1)
    expect(store.folderPageSize).toBe(60)
    expect(store.folderTotal).toBe(0)
    expect(store.folderTotalPages).toBe(0)

    // applyFolderContent append=true: page 缺省走 folderPage+1；total 缺省走已有 folderTotal
    store.folderCurrentId = 33
    store.folderPage = 1
    store.folderPageSize = 2
    store.folderTotal = 5
    store.folderTotalPages = 0
    store.folderTimelineDeferred = false
    store.folderFiles = []

    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockResolvedValueOnce({
      path: '/root/33',
      folderList: [],
      fileList: [{ id: 1, md5: 'm1', fileType: 'JPG' }]
    } as any)
    await store.loadFolderMore()
    expect(store.folderPage).toBe(2)
    expect(store.folderPageSize).toBe(2)
    expect(store.folderTotal).toBe(5)
    expect(store.folderTotalPages).toBe(3)
    expect(store.folderFiles).toHaveLength(1)

    // loadFolderByID: options.folderName 为空 + res.path 为空时回退到 “目录 {id}”
    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({
        path: '',
        folderList: [],
        fileList: []
      } as any)
      .mockResolvedValueOnce({
        path: '',
        folderList: [],
        fileList: []
      } as any)

    const opened = await store.openFavoriteFolder({
      id: 9,
      folderId: 77,
      folderName: '',
      folderPath: '/root/77',
      tags: []
    } as any)
    expect(opened).toBe(true)
    expect(store.folderCurrentName).toBe('目录 77')
  })

  it('covers loadFolderRoot error fallback message branch', async () => {
    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockRejectedValueOnce({} as any)

    const store = useMtPhotoStore()
    await store.loadFolderRoot()

    expect(store.lastError).toBe('加载失败')
    expect(store.folderLoading).toBe(false)
  })

  it('covers mtphoto folder/media/favorite mapping guard branches', async () => {
    const store = useMtPhotoStore()

    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: [
        null,
        { id: 0 },
        {
          id: 7,
          name: '',
          path: '',
          cover: '',
          s_cover: 'sc-7',
          subFolderNum: 'bad',
          subFileNum: '3',
          fileType: '',
          trashNum: 'bad'
        }
      ],
      fileList: [
        null,
        { id: 0, md5: 'invalid-id' },
        { id: 2, MD5: 'm2', fileType: 'mov', width: 0, height: -1, duration: 'bad', status: 'bad' },
        { id: 3, md5: 'm3', type: 'other', fileType: 'mp4', fileName: 'f3', size: 123, tokenAt: 't3', day: 'd3', width: 10, height: 20, duration: 30, status: 1 }
      ],
      page: 0,
      pageSize: 0
    } as any)

    await store.loadFolderRoot()
    expect(store.folderList).toHaveLength(1)
    expect(store.folderList[0]?.name).toBe('目录 7')
    expect(store.folderList[0]?.path).toBeUndefined()
    expect(store.folderList[0]?.sCover).toBe('sc-7')
    expect(store.folderList[0]?.subFolderNum).toBeUndefined()
    expect(store.folderList[0]?.subFileNum).toBe(3)
    expect(store.folderList[0]?.fileType).toBeUndefined()
    expect(store.folderList[0]?.trashNum).toBeUndefined()

    expect(store.folderFiles).toHaveLength(2)
    expect(store.folderFiles[0]?.type).toBe('video') // 来自 fileType=mov 推断
    expect(store.folderFiles[0]?.duration).toBeNull()
    expect(store.folderFiles[1]?.type).toBe('image') // raw.type=other 分支
    expect(store.folderFiles[1]?.fileName).toBe('f3')
    expect(store.folderFiles[1]?.size).toBe('123')
    expect(store.folderFiles[1]?.tokenAt).toBe('t3')
    expect(store.folderFiles[1]?.day).toBe('d3')

    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({
      items: [
        null,
        { folderId: 0, folderName: 'invalid' },
        { id: '', folderId: 9, folderName: '', folderPath: '/F9', coverMd5: null, tags: 'bad', note: null, createTime: 0, updateTime: '' },
        { id: 10, folderId: 10, folderName: 'F10', folderPath: '/F10', tags: ['x', 'x', '  ', 'y'], note: 'n10' }
      ]
    } as any)
    await store.loadFolderFavorites()
    expect(store.folderFavorites).toHaveLength(2)
    expect(store.folderFavorites[0]?.folderName).toBe('目录 9')
    expect(store.folderFavorites[0]?.tags).toEqual([])
    expect(store.folderFavorites[1]?.tags).toEqual(['x', 'y'])
  })

  it('covers timeline threshold clamp/default and loadFolderTimeline early-return branches', async () => {
    const store = useMtPhotoStore()
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true

    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValue({
      path: '',
      folderList: [],
      fileList: []
    } as any)
    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValue({ items: [] } as any)

    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 0
    await store.switchMode('folders')
    expect(store.folderTimelineThreshold).toBe(10)

    await store.switchMode('albums')
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 999
    await store.switchMode('folders')
    expect(store.folderTimelineThreshold).toBe(500)

    store.folderLoading = true
    expect(await store.loadFolderTimeline()).toBe(false)

    store.folderLoading = false
    store.folderCurrentId = null
    expect(await store.loadFolderTimeline()).toBe(false)

    store.folderCurrentId = 1
    store.folderTimelineDeferred = false
    expect(await store.loadFolderTimeline()).toBe(false)

    store.folderTimelineDeferred = true
    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockRejectedValueOnce({} as any)
    expect(await store.loadFolderTimeline()).toBe(false)
    expect(store.lastError).toBe('加载失败')
  })

  it('covers folder history dedup/invalid-id and loadFolderByID fallback timeline path', async () => {
    const store = useMtPhotoStore()
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 10

    const callCountBefore = vi.mocked(mtphotoApi.getMtPhotoFolderContent).mock.calls.length
    await store.openFolder({ id: 0, name: 'invalid', cover: '', sCover: null, subFolderNum: 0 } as any)
    expect(vi.mocked(mtphotoApi.getMtPhotoFolderContent).mock.calls.length).toBe(callCountBefore)

    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({
        path: '/root/A',
        folderList: [],
        fileList: [{ id: 1, md5: 'm1', fileType: 'JPG' }],
        page: 1,
        pageSize: 60,
        total: 1,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/A',
        folderList: [],
        fileList: [{ id: 2, md5: 'm2', fileType: 'JPG' }],
        page: 1,
        pageSize: 60,
        total: 1,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/B',
        folderList: [],
        fileList: [{ id: 3, md5: 'm3', fileType: 'JPG' }],
        page: 1,
        pageSize: 60,
        total: 1,
        totalPages: 1
      } as any)
      .mockResolvedValueOnce({
        path: '/root/B',
        folderList: [],
        fileList: [{ id: 4, md5: 'm4', fileType: 'JPG' }],
        page: 1,
        pageSize: 60,
        total: 1,
        totalPages: 1
      } as any)

    await store.openFolder({ id: 11, name: 'A', cover: '', sCover: null, subFolderNum: undefined } as any)
    await store.openFolder({ id: 11, name: 'A', cover: '', sCover: null, subFolderNum: undefined } as any)
    expect(store.folderHistory).toHaveLength(2) // idx>=0 分支：不重复增长

    await store.openFolder({ id: 12, name: 'B', cover: '', sCover: null, subFolderNum: undefined } as any)
    expect(store.folderHistory).toHaveLength(3)
    expect(store.folderTimelineDeferred).toBe(false)

    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockRejectedValueOnce({} as any)
    expect(await store.openFavoriteFolder({ id: 1, folderId: 99, folderName: 'F99', folderPath: '/F99', tags: [] } as any)).toBe(false)
    expect(store.lastError).toBe('加载失败')
  })

  it('covers favorite filter timer clear and upsert/remove remaining fallback branches', async () => {
    vi.useFakeTimers()
    try {
      const store = useMtPhotoStore()

      store.setFavoriteFilterKeyword('first', { debounceMs: 100 })
      store.setFavoriteFilterKeyword('second', { debounceMs: 100 }) // clearTimeout 分支
      vi.advanceTimersByTime(100)
      expect(store.favoriteFilterKeyword).toBe('second')

      store.startEditFavorite({ folderId: 0, tags: [], note: '' } as any)
      expect(store.favoriteEditingFolderId).toBeNull()

      vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({
        success: true,
        item: { id: 30, folderId: 30, folderName: '目录 30', folderPath: '/F30', coverMd5: '', tags: [], note: '' }
      } as any)
      expect(await store.upsertFolderFavorite({ folderId: 30, folderName: '', folderPath: '/F30', tags: 'bad' as any })).toBe(true)
      expect(vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mock.calls.at(-1)?.[0]).toEqual({
        folderId: 30,
        folderName: '目录 30',
        folderPath: '/F30',
        coverMd5: undefined,
        tags: [],
        note: ''
      })

      store.folderCurrentId = 31
      store.folderCurrentCoverMd5 = 'keep-cover'
      vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({
        success: true,
        item: { id: 31, folderId: 31, folderName: 'F31', folderPath: '/F31', coverMd5: '', tags: [], note: '' }
      } as any)
      expect(await store.upsertFolderFavorite({ folderId: 31, folderName: 'F31', folderPath: '/F31' })).toBe(true)
      expect(store.folderCurrentCoverMd5).toBe('keep-cover')

      vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockRejectedValueOnce({} as any)
      expect(await store.upsertFolderFavorite({ folderId: 32, folderName: 'F32', folderPath: '/F32' })).toBe(false)
      expect(store.lastError).toBe('保存失败')

      store.folderCurrentId = 40
      store.folderCurrentName = 'F40'
      store.folderPath = '/F40'
      vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({ success: false, item: null } as any)
      vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({ items: [] } as any)
      expect(await store.upsertCurrentFolderFavorite({})).toBe(false) // ok=false 分支

      store.folderCurrentId = 41
      store.folderCurrentName = ''
      store.folderPath = '/F41'
      store.favoriteDraftTags = 'will-reset'
      store.favoriteDraftNote = 'will-reset'
      vi.mocked(mtphotoApi.upsertMtPhotoFolderFavorite).mockResolvedValueOnce({ success: true, item: null } as any)
      vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({ items: [] } as any)
      expect(await store.upsertCurrentFolderFavorite({})).toBe(true)
      expect(store.favoriteDraftTags).toBe('')
      expect(store.favoriteDraftNote).toBe('')

      store.folderFavorites = [{ id: 1, folderId: 50, folderName: 'F50', folderPath: '/F50', tags: [] }] as any
      store.favoriteEditingFolderId = 999
      vi.mocked(mtphotoApi.removeMtPhotoFolderFavorite).mockResolvedValueOnce({ success: true } as any)
      expect(await store.removeFolderFavorite(50)).toBe(true)
      expect(store.favoriteEditingFolderId).toBe(999)

      vi.mocked(mtphotoApi.removeMtPhotoFolderFavorite).mockRejectedValueOnce({} as any)
      expect(await store.removeFolderFavorite(51)).toBe(false)
      expect(store.lastError).toBe('移除失败')
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers mapping fallback branches and non-array payload guards', async () => {
    const store = useMtPhotoStore()

    // 1) folderList/fileList 非数组，覆盖 mapFolderNodes/mapMediaItems 的早返回分支
    store.folderPageSize = 0
    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: { not: 'array' },
      fileList: { not: 'array' },
      page: 0,
      pageSize: 'abc'
    } as any)
    await store.loadFolderRoot()
    expect(store.folderList).toEqual([])
    expect(store.folderFiles).toEqual([])
    expect(store.folderPage).toBe(1)
    expect(store.folderPageSize).toBe(60)

    // 2) pageSize 回退到 60 的分支（res.pageSize/folderPageSize 都为 falsy）
    store.folderPageSize = 0
    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: [],
      fileList: [],
      page: 1
    } as any)
    await store.loadFolderRoot()
    expect(store.folderPageSize).toBe(60)

    // 3) 覆盖 folder/media/favorite 映射中的多个兜底与 nullish 分支
    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: [
        {
          id: 7,
          name: undefined, // 触发 name 的 nullish 回退
          path: '',
          cover: '',
          s_cover: undefined,
          subFolderNum: 'bad',
          subFileNum: '3',
          fileType: 'JPG', // 触发 fileType 真分支
          trashNum: '2' // 触发 trashNum 真分支
        }
      ],
      fileList: [
        { id: 1 }, // 触发 md5 缺失分支（raw.md5/raw.MD5 都缺失）
        { id: 2, MD5: 'm2' }, // 触发 inferMediaType(fileType undefined)
        { id: 3, md5: 'm3', type: 'video' } // 触发 type==='video' 分支
      ]
    } as any)
    await store.loadFolderRoot()
    expect(store.folderList[0]?.name).toBe('目录 7')
    expect(store.folderList[0]?.fileType).toBe('JPG')
    expect(store.folderList[0]?.trashNum).toBe(2)
    expect(store.folderFiles.map(v => v.md5)).toEqual(['m2', 'm3'])
    expect(store.folderFiles[0]?.type).toBe('image')
    expect(store.folderFiles[1]?.type).toBe('video')

    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({
      items: [
        {
          id: 1,
          folderId: 1,
          folderName: undefined, // 触发 folderName nullish 回退
          folderPath: undefined, // 触发 folderPath nullish 回退
          tags: [null, 'x'], // 触发 normalizeTags 的 raw??'' 分支
          note: undefined
        }
      ]
    } as any)
    await store.loadFolderFavorites()
    expect(store.folderFavorites[0]?.folderName).toBe('目录 1')
    expect(store.folderFavorites[0]?.folderPath).toBe('')
    expect(store.folderFavorites[0]?.tags).toEqual(['x'])
  })

  it('covers filter/group/reset/backToAlbums/openAlbum-null branches', async () => {
    vi.useFakeTimers()
    try {
      const store = useMtPhotoStore()

      // groupedFolderFavorites: favoriteGroupBy !== 'tag' 分支
      store.favoriteGroupBy = 'none'
      store.folderFavorites = [{ id: 1, folderId: 1, folderName: 'A', folderPath: '/A', tags: ['x'] }] as any
      expect(store.groupedFolderFavorites[0]?.key).toBe('全部')

      // filteredFolderFavorites: tokens.length === 0 分支
      store.setFavoriteFilterKeyword(',', { immediate: true })
      expect(store.filteredFolderFavorites).toHaveLength(1)

      // filteredFolderFavorites: favoriteFilterMode === 'all' 分支
      store.favoriteFilterMode = 'all'
      store.setFavoriteFilterKeyword('x y', { immediate: true })
      store.folderFavorites = [{ id: 2, folderId: 2, folderName: 'B', folderPath: '/B', tags: ['x', 'y'] }] as any
      expect(store.filteredFolderFavorites).toHaveLength(1)

      // allUniqueTags: String(rawTag || '') 的右侧分支
      store.folderFavorites = [{ id: 3, folderId: 3, folderName: 'C', folderPath: '/C', tags: ['', 'z'] }] as any
      expect(store.allUniqueTags).toEqual(['z'])

      // resetFavoriteViewState: clearTimeout 分支（close() 会调用）
      store.setFavoriteFilterKeyword('pending', { debounceMs: 100 })
      store.close()
      vi.advanceTimersByTime(100)
      expect(store.favoriteFilterKeyword).toBe('')

      // backToAlbums 覆盖 446-448
      store.selectedAlbum = { id: 9, mtPhotoAlbumId: 9, name: 'A', cover: '', count: 1 } as any
      store.mediaItems = [{ id: 1, md5: 'm1', type: 'image' } as any]
      store.backToAlbums()
      expect(store.view).toBe('albums')
      expect(store.selectedAlbum).toBeNull()
      expect(store.mediaItems).toEqual([])

      // loadAlbumPage 中 selectedAlbum 为空的早返回分支（通过 openAlbum(null) 触发）
      const before = vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mock.calls.length
      await store.openAlbum(null as any)
      expect(vi.mocked(mtphotoApi.getMtPhotoAlbumFiles).mock.calls.length).toBe(before)
    } finally {
      vi.useRealTimers()
    }
  })

  it('covers remaining error-fallback branches for folder/favorite operations', async () => {
    const store = useMtPhotoStore()
    const systemConfigStore = useSystemConfigStore()
    systemConfigStore.loaded = true
    systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold = 10

    // normalizeFolderName: parts 为空时回退 fallback（path='///'）
    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({ path: '///', folderList: [], fileList: [] } as any)
      .mockResolvedValueOnce({ path: '///', folderList: [], fileList: [] } as any)
    const openFavOk = await store.openFavoriteFolder({
      id: 1,
      folderId: 88,
      folderName: '',
      folderPath: '/F88',
      tags: []
    } as any)
    expect(openFavOk).toBe(true)
    expect(store.folderCurrentName).toBe('目录 88')

    // firstCoverMD5: cover 为 undefined 触发 String(cover ?? '') 的右侧分支
    vi.mocked(mtphotoApi.getMtPhotoFolderContent)
      .mockResolvedValueOnce({ path: '/A', folderList: [], fileList: [] } as any)
      .mockResolvedValueOnce({ path: '/A', folderList: [], fileList: [] } as any)
    await store.openFolder({ id: 90, name: 'A', cover: undefined, sCover: '', subFolderNum: undefined } as any)

    // loadFolderMore catch：error/message 都没有时回落“加载失败”
    store.folderCurrentId = 90
    store.folderTimelineDeferred = false
    store.folderPage = 1
    store.folderTotalPages = 2
    vi.mocked(mtphotoApi.getMtPhotoFolderContent).mockRejectedValueOnce({} as any)
    await store.loadFolderMore()
    expect(store.lastError).toBe('加载失败')

    // loadFolderFavorites：items 非数组 + catch 默认错误
    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockResolvedValueOnce({ items: {} } as any)
    await store.loadFolderFavorites()
    expect(store.folderFavorites).toEqual([])

    vi.mocked(mtphotoApi.getMtPhotoFolderFavorites).mockRejectedValueOnce({} as any)
    await store.loadFolderFavorites()
    expect(store.lastError).toBe('加载失败')

    // upsertFolderFavorite: folderPath 为 undefined 触发 String(payload.folderPath || '') 右侧分支
    expect(await store.upsertFolderFavorite({ folderId: 123, folderName: 'F123', folderPath: undefined as any })).toBe(false)
    expect(store.lastError).toBe('folderPath 不能为空')

    // upsertCurrentFolderFavorite: folderPath.value 为空触发 String(folderPath.value || '') 右侧分支
    store.folderCurrentId = 124
    store.folderCurrentName = 'F124'
    store.folderPath = ''
    expect(await store.upsertCurrentFolderFavorite({ tags: ['a'] })).toBe(false)
    expect(store.lastError).toBe('folderPath 不能为空')

    // startEditFavorite: note 缺省触发 note || '' 的右侧分支
    store.startEditFavorite({ folderId: 124, tags: ['a'] } as any)
    expect(store.favoriteDraftNote).toBe('')
  })

  it('covers folder favorites any-mode filtering fallback and invalid page fallback to 1', async () => {
    const store = useMtPhotoStore()

    store.folderFavorites = [
      { id: 1, folderId: 1, folderName: 'A', folderPath: '/A', tags: ['', 'tag1'] },
      { id: 2, folderId: 2, folderName: 'B', folderPath: '/B', tags: ['tag2'] }
    ] as any
    store.favoriteFilterMode = 'any'
    store.setFavoriteFilterKeyword('tag1', { immediate: true })
    expect(store.filteredFolderFavorites.map(v => v.folderId)).toEqual([1])

    vi.mocked(mtphotoApi.getMtPhotoFolderRoot).mockResolvedValueOnce({
      path: '/root',
      folderList: [],
      fileList: [],
      page: 'abc',
      pageSize: 60,
      total: 0,
      totalPages: 0
    } as any)
    await store.loadFolderRoot()
    expect(store.folderPage).toBe(1)
  })
})

describe('stores/identity + user', () => {
  it('persists identity cookies and loads list', async () => {
    vi.mocked(identityApi.getIdentityList).mockResolvedValue({ code: 0, data: [{ id: 'i1', name: 'A', sex: '男' }] } as any)
    const store = useIdentityStore()

    store.saveIdentityCookie('', 'c')
    store.saveIdentityCookie('i1', '')
    expect(store.getIdentityCookie('i1')).toBe('')

    store.saveIdentityCookie('i1', 'cookie-1')
    expect(store.getIdentityCookie('i1')).toBe('cookie-1')
    expect(JSON.parse(localStorage.getItem('identityCookies') || '{}').i1).toBe('cookie-1')

    await store.loadList()
    expect(store.identityList).toHaveLength(1)
  })

  it('creates/deletes identities and returns boolean status', async () => {
    vi.mocked(identityApi.createIdentity).mockResolvedValue({ code: 0 } as any)
    vi.mocked(identityApi.deleteIdentity).mockResolvedValue({ code: 0 } as any)
    vi.mocked(identityApi.getIdentityList).mockResolvedValue({ code: 0, data: [] } as any)

    const store = useIdentityStore()
    expect(await store.createIdentity({ name: 'A', sex: '男' })).toBe(true)
    expect(await store.deleteIdentity('x')).toBe(true)

    vi.mocked(identityApi.createIdentity).mockResolvedValue({ code: 1 } as any)
    expect(await store.createIdentity({ name: 'A', sex: '男' })).toBe(false)

    vi.mocked(identityApi.deleteIdentity).mockResolvedValueOnce({ code: 1 } as any)
    expect(await store.deleteIdentity('x2')).toBe(false)
  })

  it('user store edit flow and cookie saving', () => {
    const identityStore = useIdentityStore()
    const cookieSpy = vi.spyOn(identityStore, 'saveIdentityCookie')

    const userStore = useUserStore()
    userStore.setCurrentUser({ id: 'i1', name: 'A', nickname: 'A', cookie: 'c' } as any)
    expect(cookieSpy).toHaveBeenCalledWith('i1', 'c')

    userStore.startEdit()
    expect(userStore.editMode).toBe(true)
    userStore.editUserInfo.nickname = 'B'
    userStore.saveEdit()
    expect(userStore.currentUser?.nickname).toBe('B')
    expect(userStore.editMode).toBe(false)

    userStore.cancelEdit()
    expect(userStore.editUserInfo).toEqual({})

    userStore.clearCurrentUser()
    expect(userStore.currentUser).toBeNull()
  })
})

describe('stores/favorite', () => {
  it('groups favorites and supports add/remove', async () => {
    vi.mocked(favoriteApi.listAllFavorites).mockResolvedValue({
      code: 0,
      data: [
        { id: 1, identityId: 'i1', targetUserId: 'u1', targetUserName: 'U1' },
        { id: 2, identityId: 'i1', targetUserId: 'u2', targetUserName: 'U2' },
        { id: 3, identityId: 'i2', targetUserId: 'u3', targetUserName: 'U3' }
      ]
    } as any)

    const store = useFavoriteStore()
    await store.loadAllFavorites()
    expect(Object.keys(store.groupedFavorites)).toEqual(['i1', 'i2'])
    expect(store.groupedFavorites.i1).toHaveLength(2)

    vi.mocked(favoriteApi.addFavorite).mockResolvedValue({ code: 0 } as any)
    expect(await store.addFavorite('i1', 'u9', 'U9')).toBe(true)

    vi.mocked(favoriteApi.removeFavorite).mockResolvedValue({ code: 0 } as any)
    expect(await store.removeFavorite('i1', 'u2')).toBe(true)
    expect(store.isFavorite('i1', 'u2')).toBe(false)

    vi.mocked(favoriteApi.removeFavoriteById).mockResolvedValue({ code: 0 } as any)
    expect(await store.removeFavoriteById(1)).toBe(true)
  })
})

describe('stores/douyin', () => {
  it('open respects existing draft, supports account/favorite jump options, and close resets state', () => {
    const store = useDouyinStore()
    store.open('x')
    expect(store.showModal).toBe(true)
    expect(store.draftInput).toBe('x')
    expect(store.entryMode).toBe('default')
    expect(store.favoritesTab).toBe('users')
    expect(store.targetMode).toBe('detail')
    expect(store.accountSecUserId).toBe('')
    expect(store.autoFetchAccount).toBe(false)
    expect(store.favoriteSecUserId).toBe('')
    expect(store.autoOpenFavoriteUserDetail).toBe(false)
    expect(store.favoriteSecUserId).toBe('')
    expect(store.autoOpenFavoriteUserDetail).toBe(false)

    store.open('y')
    expect(store.draftInput).toBe('x')

    store.open({
      entryMode: 'favorites',
      favoritesTab: 'awemes',
      targetMode: 'account',
      accountSecUserId: ' sec-2 ',
      autoFetchAccount: true,
      favoriteSecUserId: ' sec-fav-1 ',
      autoOpenFavoriteUserDetail: true
    })
    expect(store.entryMode).toBe('favorites')
    expect(store.favoritesTab).toBe('awemes')
    expect(store.targetMode).toBe('account')
    expect(store.accountSecUserId).toBe('sec-2')
    expect(store.autoFetchAccount).toBe(true)
    expect(store.favoriteSecUserId).toBe('sec-fav-1')
    expect(store.autoOpenFavoriteUserDetail).toBe(true)

    store.close()
    expect(store.showModal).toBe(false)
    expect(store.draftInput).toBe('')
    expect(store.entryMode).toBe('default')
    expect(store.favoritesTab).toBe('users')
    expect(store.targetMode).toBe('detail')
    expect(store.accountSecUserId).toBe('')
    expect(store.autoFetchAccount).toBe(false)
    expect(store.favoriteSecUserId).toBe('')
    expect(store.autoOpenFavoriteUserDetail).toBe(false)
  })
})

describe('stores/videoExtract', () => {
  it('openCreateFromMedia returns false for non-video media', async () => {
    const store = useVideoExtractStore()
    const ok = await store.openCreateFromMedia({ type: 'image', url: '/upload/images/a.jpg' } as any)
    expect(ok).toBe(false)
    expect(store.showCreateModal).toBe(false)
  })

  it('openCreateFromMedia handles blank url and mtPhoto detection for /api urls', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)

    const store = useVideoExtractStore()

    // Blank url -> mediaUrl should be undefined and label empty.
    const okBlank = await store.openCreateFromMedia({ type: 'video' } as any, 'u1')
    expect(okBlank).toBe(true)
    expect(store.createSource?.mediaUrl).toBeUndefined()
    expect(store.createSourceLabel).toBe('')

    // /api url + md5 -> mtPhoto source.
    const okApi = await store.openCreateFromMedia({ type: 'video', url: '/api/download/xxx', md5: 'm2' } as any, 'u2')
    expect(okApi).toBe(true)
    expect(store.createSource?.sourceType).toBe('mtPhoto')
    expect(store.createSourceLabel).toBe('mtPhoto:m2')
  })

  it('createSourceLabel covers mtPhoto without md5 and upload without localPath', () => {
    const store = useVideoExtractStore()

    // No createSource -> empty label.
    expect(store.createSourceLabel).toBe('')

    store.createSource = { sourceType: 'mtPhoto' } as any
    expect(store.createSourceLabel).toBe('mtPhoto:')

    store.createSource = { sourceType: 'upload' } as any
    expect(store.createSourceLabel).toBe('')
  })

  it('fetchProbe returns early when createSource is missing and uses default message when thrown error has no message', async () => {
    const store = useVideoExtractStore()
    await store.fetchProbe()
    expect(videoExtractApi.probeVideo).not.toHaveBeenCalled()

    store.createSource = { sourceType: 'upload', localPath: '/videos/a.mp4' } as any
    vi.mocked(videoExtractApi.probeVideo).mockRejectedValue({})
    await store.fetchProbe()
    expect(store.probeError).toBe('探测失败')
  })

  it('openCreateFromMedia selects mtPhoto source when md5 and /lsp url are present', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: { durationSec: 10, width: 1, height: 1 } } as any)
    const store = useVideoExtractStore()

    const ok = await store.openCreateFromMedia({ type: 'video', url: '/lsp/a', md5: 'm1' } as any, 'u1')
    expect(ok).toBe(true)
    expect(store.createSource?.sourceType).toBe('mtPhoto')
    expect(store.createSource?.md5).toBe('m1')
    expect(store.showCreateModal).toBe(true)
    expect(store.probe?.durationSec).toBe(10)
  })

  it('openCreateFromMedia selects upload source and uses originalFilename as label', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)
    const store = useVideoExtractStore()

    const ok = await store.openCreateFromMedia(
      { type: 'video', url: '/upload/videos/a.mp4', originalFilename: 'Nice.mp4' } as any,
      'u1'
    )
    expect(ok).toBe(true)
    expect(store.createSource?.sourceType).toBe('upload')
    expect(store.createSource?.localPath).toBe('/videos/a.mp4')
    expect(store.createSource?.userId).toBe('u1')
    expect(store.createSourceLabel).toBe('Nice.mp4')
  })

  it('createSourceLabel falls back to localPath when displayName is missing', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)
    const store = useVideoExtractStore()

    const ok = await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any, 'u1')
    expect(ok).toBe(true)
    expect(store.createSourceLabel).toBe('/videos/a.mp4')
  })

  it('fetchProbe sets probeError when backend returns error', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 1, msg: 'bad' } as any)
    const store = useVideoExtractStore()
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any)
    expect(store.probe).toBeNull()
    expect(store.probeError).toBe('bad')
  })

  it('fetchProbe uses default error message when backend returns empty message', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 1 } as any)
    const store = useVideoExtractStore()
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any)
    expect(store.probeError).toBe('探测失败')
  })

  it('fetchProbe sets probeError when probe throws', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockRejectedValue(new Error('boom'))
    const store = useVideoExtractStore()
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any)
    expect(store.probe).toBeNull()
    expect(store.probeError).toBe('boom')
  })

  it('createTask throws without source and returns taskId on success', async () => {
    const store = useVideoExtractStore()
    await expect(store.createTask({ mode: 'keyframe', maxFrames: 1, outputFormat: 'jpg' } as any)).rejects.toThrow('缺少视频来源')

    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any, 'u1')

    vi.mocked(videoExtractApi.createVideoExtractTask).mockResolvedValue({ code: 0, data: { taskId: 't1' } } as any)
    const created = await store.createTask({ mode: 'keyframe', maxFrames: 1, outputFormat: 'jpg' } as any)
    expect(created.taskId).toBe('t1')
  })

  it('createTask throws when backend returns error', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)
    const store = useVideoExtractStore()
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any, 'u1')

    vi.mocked(videoExtractApi.createVideoExtractTask).mockResolvedValue({ code: 1, msg: 'bad' } as any)
    await expect(store.createTask({ mode: 'keyframe', maxFrames: 1, outputFormat: 'jpg' } as any)).rejects.toThrow('bad')
  })

  it('createTask error message falls back to res.message and then default', async () => {
    vi.mocked(videoExtractApi.probeVideo).mockResolvedValue({ code: 0, data: {} } as any)
    const store = useVideoExtractStore()
    await store.openCreateFromMedia({ type: 'video', url: '/upload/videos/a.mp4' } as any, 'u1')

    vi.mocked(videoExtractApi.createVideoExtractTask).mockResolvedValue({ code: 1, message: 'bad2' } as any)
    await expect(store.createTask({ mode: 'keyframe', maxFrames: 1, outputFormat: 'jpg' } as any)).rejects.toThrow('bad2')

    vi.mocked(videoExtractApi.createVideoExtractTask).mockResolvedValue({ code: 1 } as any)
    await expect(store.createTask({ mode: 'keyframe', maxFrames: 1, outputFormat: 'jpg' } as any)).rejects.toThrow('创建任务失败')
  })

  it('loadTasks handles non-array items and falls back to provided page and defaults', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: null } } as any)
    const store = useVideoExtractStore()
    await store.loadTasks(3)

    expect(store.tasks).toEqual([])
    expect(store.listTotal).toBe(0)
    expect(store.listPage).toBe(3)
  })

  it('cancelTask returns early when id is empty', async () => {
    const store = useVideoExtractStore()
    await store.cancelTask('  ')
    expect(videoExtractApi.cancelVideoExtractTask).not.toHaveBeenCalled()
  })

  it('cancelTask calls refreshTaskDetail and loadTasks when id is valid', async () => {
    vi.mocked(videoExtractApi.cancelVideoExtractTask).mockResolvedValue({ code: 0 } as any)
    vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
      code: 0,
      data: { task: { taskId: 't1', status: 'SUCCESS' }, frames: { items: [], nextCursor: 0, hasMore: false } }
    } as any)
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: [], total: 0, page: 1, pageSize: 20 } } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    await store.cancelTask('t1')
    expect(videoExtractApi.cancelVideoExtractTask).toHaveBeenCalledWith('t1')
    expect(videoExtractApi.getVideoExtractTaskDetail).toHaveBeenCalled()
    expect(videoExtractApi.getVideoExtractTaskList).toHaveBeenCalled()
  })

  it('openTaskCenter without taskId only loads list', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: [], total: 0, page: 1, pageSize: 20 } } as any)
    const store = useVideoExtractStore()
    await store.openTaskCenter()
    expect(store.showTaskModal).toBe(true)
    expect(store.selectedTaskId).toBe('')
  })

  it('refreshTaskDetail returns early when backend response has no task', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({ code: 0, data: { frames: { items: [] } } } as any)
    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    await store.refreshTaskDetail(true)
    expect(store.selectedTask).toBeNull()
    expect(store.detailLoading).toBe(false)
  })

  it('refreshTaskDetail returns early when selectedTaskId is empty', async () => {
    const store = useVideoExtractStore()
    store.selectedTaskId = ''
    await store.refreshTaskDetail()
    expect(videoExtractApi.getVideoExtractTaskDetail).not.toHaveBeenCalled()
  })

  it('refreshTaskDetail tolerates missing frames page', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
      code: 0,
      data: { task: { taskId: 't1', status: 'SUCCESS' }, frames: null }
    } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    await store.refreshTaskDetail(true)
    expect(store.selectedTask?.taskId).toBe('t1')
  })

  it('refreshTaskDetail merges frames when loading more and loadMoreFrames respects hasMore', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
      code: 0,
      data: {
        task: { taskId: 't1', status: 'RUNNING' },
        frames: { items: [{ url: 'b' }], nextCursor: 2, hasMore: false }
      }
    } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    store.frames = { items: [{ url: 'a' }], nextCursor: 1, hasMore: true } as any
    await store.refreshTaskDetail(false)

    expect(store.frames.items).toHaveLength(2)
    expect(store.frames.items.map((i: any) => i.url)).toEqual(['a', 'b'])
    expect(store.frames.nextCursor).toBe(2)

    vi.clearAllMocks()
    store.frames.hasMore = false
    await store.loadMoreFrames()
    expect(videoExtractApi.getVideoExtractTaskDetail).not.toHaveBeenCalled()
  })

  it('loadMoreFrames calls refreshTaskDetail when hasMore is true', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
      code: 0,
      data: {
        task: { taskId: 't1', status: 'RUNNING' },
        frames: { items: [], nextCursor: 1, hasMore: false }
      }
    } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    store.frames = { items: [], nextCursor: 0, hasMore: true } as any

    await store.loadMoreFrames()
    expect(videoExtractApi.getVideoExtractTaskDetail).toHaveBeenCalledTimes(1)
  })

  it('loadMoreFrames returns early when selectedTaskId is empty', async () => {
    const store = useVideoExtractStore()
    store.selectedTaskId = ''
    await store.loadMoreFrames()
    expect(videoExtractApi.getVideoExtractTaskDetail).not.toHaveBeenCalled()
  })

  it('cancelTask returns early when taskId is empty', async () => {
    const store = useVideoExtractStore()
    await store.cancelTask('')
    expect(videoExtractApi.cancelVideoExtractTask).not.toHaveBeenCalled()
  })

  it('openTaskDetail tolerates empty taskId (covers taskId fallback branch) and does not call backend', async () => {
    vi.useFakeTimers()
    try {
      const store = useVideoExtractStore()
      await store.openTaskDetail('')
      expect(store.selectedTaskId).toBe('')
      expect(videoExtractApi.getVideoExtractTaskDetail).not.toHaveBeenCalled()
      store.stopPolling()
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('deleteTask clears selection when deleting current task', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: [], total: 0, page: 1, pageSize: 20 } } as any)
    vi.mocked(videoExtractApi.deleteVideoExtractTask).mockResolvedValue({ code: 0 } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    store.selectedTask = { taskId: 't1', status: 'RUNNING' } as any
    store.frames = { items: [{ url: 'a' }], nextCursor: 1, hasMore: true } as any
    store.polling = true

    await store.deleteTask({ taskId: 't1', deleteFiles: false })

    expect(store.selectedTaskId).toBe('')
    expect(store.selectedTask).toBeNull()
    expect(store.frames.items).toHaveLength(0)
    expect(store.polling).toBe(false)
  })

  it('deleteTask keeps selection when deleting a different task', async () => {
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: [], total: 0, page: 1, pageSize: 20 } } as any)
    vi.mocked(videoExtractApi.deleteVideoExtractTask).mockResolvedValue({ code: 0 } as any)

    const store = useVideoExtractStore()
    store.selectedTaskId = 't1'
    store.selectedTask = { taskId: 't1', status: 'RUNNING' } as any
    await store.deleteTask({ taskId: 't2', deleteFiles: false })

    expect(store.selectedTaskId).toBe('t1')
    expect(store.selectedTask?.taskId).toBe('t1')
  })

  it('polling tick returns early when polling is turned off before tick fires', async () => {
    vi.useFakeTimers()
    try {
      const store = useVideoExtractStore()
      store.selectedTask = { taskId: 't1', status: 'RUNNING' } as any
      store.selectedTaskId = 't1'

      store.startPolling()
      store.polling = false

      await vi.advanceTimersByTimeAsync(600)
      store.stopPolling()
    } finally {
      vi.useRealTimers()
    }
  })

  it('polling uses 5000ms interval when document is hidden', async () => {
    vi.useFakeTimers()
    const original = Object.getOwnPropertyDescriptor(document, 'visibilityState')
    try {
      Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })

      vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
        code: 0,
        data: { task: { taskId: 't1', status: 'RUNNING' }, frames: { items: [], nextCursor: 0, hasMore: false } }
      } as any)

      const store = useVideoExtractStore()
      store.selectedTaskId = 't1'
      store.selectedTask = { taskId: 't1', status: 'RUNNING' } as any

      const setTimeoutSpy = vi.spyOn(globalThis, 'setTimeout')
      store.startPolling()

      await vi.advanceTimersByTimeAsync(600)

      // one of the scheduled timers should be for the 5000ms interval path
      expect(setTimeoutSpy.mock.calls.some(call => Number(call[1]) === 5000)).toBe(true)

      store.stopPolling()
      setTimeoutSpy.mockRestore()
    } finally {
      if (original) {
        Object.defineProperty(document, 'visibilityState', original)
      } else {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete (document as any).visibilityState
      }
      vi.useRealTimers()
    }
  })

  it('polling handles document undefined branch when scheduling tick', async () => {
    vi.useFakeTimers()
    const originalDocument = (globalThis as any).document
    try {
      Object.defineProperty(globalThis, 'document', { value: undefined, configurable: true })
      vi.mocked(videoExtractApi.getVideoExtractTaskDetail).mockResolvedValue({
        code: 0,
        data: { task: { taskId: 't1', status: 'RUNNING' }, frames: { items: [], nextCursor: 0, hasMore: false } }
      } as any)

      const store = useVideoExtractStore()
      store.selectedTaskId = 't1'
      store.selectedTask = { taskId: 't1', status: 'RUNNING' } as any
      store.startPolling()

      await vi.advanceTimersByTimeAsync(600)
      expect(videoExtractApi.getVideoExtractTaskDetail).toHaveBeenCalled()
      store.stopPolling()
    } finally {
      Object.defineProperty(globalThis, 'document', { value: originalDocument, configurable: true })
      vi.useRealTimers()
    }
  })

  it('polls task detail while running and stops when not running', async () => {
    vi.useFakeTimers()
    vi.mocked(videoExtractApi.getVideoExtractTaskList).mockResolvedValue({ data: { items: [], total: 0, page: 1, pageSize: 20 } } as any)

    vi.mocked(videoExtractApi.getVideoExtractTaskDetail)
      .mockResolvedValueOnce({
        code: 0,
        data: { task: { taskId: 't1', status: 'RUNNING' }, frames: { items: [], nextCursor: 0, hasMore: false } }
      } as any)
      .mockResolvedValueOnce({
        code: 0,
        data: { task: { taskId: 't1', status: 'SUCCESS' }, frames: { items: [], nextCursor: 0, hasMore: false } }
      } as any)

    const store = useVideoExtractStore()
    await store.openTaskCenter('t1')
    expect(store.showTaskModal).toBe(true)
    expect(store.selectedTaskId).toBe('t1')
    expect(store.polling).toBe(true)

    await vi.advanceTimersByTimeAsync(600)
    await vi.advanceTimersByTimeAsync(1500)

    expect(videoExtractApi.getVideoExtractTaskDetail).toHaveBeenCalled()
    expect(store.polling).toBe(false)
    vi.useRealTimers()
  })
})
