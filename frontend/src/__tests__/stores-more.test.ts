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
  getMtPhotoAlbumFiles: vi.fn()
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

  it('loadAllUploadImages ignores non-array response payloads', async () => {
    vi.mocked(mediaApi.getAllUploadImages).mockResolvedValue({ data: { nope: true } } as any)
    const store = useMediaStore()
    await store.loadAllUploadImages(1)
    expect(store.allUploadImages).toEqual([])
    expect(store.allUploadLoading).toBe(false)
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
  it('open respects existing draft and close resets state', () => {
    const store = useDouyinStore()
    store.open('x')
    expect(store.showModal).toBe(true)
    expect(store.draftInput).toBe('x')

    store.open('y')
    expect(store.draftInput).toBe('x')

    store.close()
    expect(store.showModal).toBe(false)
    expect(store.draftInput).toBe('')
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

  it('loadMoreFrames returns early when selectedTaskId is empty', async () => {
    const store = useVideoExtractStore()
    store.selectedTaskId = ''
    await store.loadMoreFrames()
    expect(videoExtractApi.getVideoExtractTaskDetail).not.toHaveBeenCalled()
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
