import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastError = vi.fn()
const toastShow = vi.fn()
const routerPush = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: toastError,
    show: toastShow
  })
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<any>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: routerPush })
  }
})

vi.mock('@/utils/id', async () => {
  const actual = await vi.importActual<any>('@/utils/id')
  return {
    ...actual,
    generateRandomIP: () => '127.0.0.9'
  }
})

vi.mock('@/utils/cookie', () => ({
  generateCookie: () => 'cookie-x'
}))

vi.mock('@/api/system', () => ({
  getConnectionStats: vi.fn(),
  getForceoutUserCount: vi.fn(),
  disconnectAllConnections: vi.fn(),
  clearForceoutUsers: vi.fn(),
  getSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn()
}))

vi.mock('@/api/media', () => ({
  getImgServerAddress: vi.fn(),
  updateImgServerAddress: vi.fn(),
  uploadMedia: vi.fn()
}))

vi.mock('@/api/identity', () => ({
  getIdentityList: vi.fn(),
  createIdentity: vi.fn(),
  deleteIdentity: vi.fn(),
  selectIdentity: vi.fn()
}))

import { useSettings } from '@/composables/useSettings'
import { useUpload } from '@/composables/useUpload'
import { useIdentity } from '@/composables/useIdentity'

import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useUserStore } from '@/stores/user'
import { useIdentityStore } from '@/stores/identity'

import * as systemApi from '@/api/system'
import * as mediaApi from '@/api/media'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('composables/useUpload', () => {
  it('uploads file via upstream and returns mapped UploadedMedia', async () => {
    const mediaStore = useMediaStore()
    const systemConfigStore = useSystemConfigStore()

    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockImplementation(async () => {
      mediaStore.imgServer = 'img.local'
    })
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({
      state: 'OK',
      msg: 'images/2026/01/a.png',
      localFilename: 'a.png'
    } as any)

    const file = new File(['x'], 'a.png', { type: 'image/png' })
    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(file, 'me', 'Me')

    expect(uploaded?.type).toBe('image')
    expect(uploaded?.url).toBe('http://img.local:9006/img/Upload/images/2026/01/a.png')
    expect(mediaStore.uploadedMedia[0]?.url).toBe(uploaded?.url)
  })

  it('returns null when imgServer is still missing after loadImgServer', async () => {
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockImplementation(async () => {
      mediaStore.imgServer = ''
    })
    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.txt', { type: 'text/plain' }), 'me', 'Me')
    expect(uploaded).toBeNull()
  })

  it('loads imgServer on demand and maps video type', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = ''

    vi.spyOn(mediaStore, 'loadImgServer').mockImplementation(async () => {
      mediaStore.imgServer = 'img.local'
    })

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9007')

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({
      state: 'OK',
      msg: 'videos/2026/01/a.mp4',
      localFilename: 'a.mp4'
    } as any)

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.mp4', { type: 'video/mp4' }), 'me', 'Me')

    expect(mediaStore.loadImgServer).toHaveBeenCalled()
    expect(uploaded?.type).toBe('video')
    expect(uploaded?.url).toBe('http://img.local:9007/img/Upload/videos/2026/01/a.mp4')
  })

  it('maps non-image/video MIME types to file', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({
      state: 'OK',
      msg: 'files/2026/01/a.bin',
      localFilename: 'a.bin'
    } as any)

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.bin', { type: 'application/octet-stream' }), 'me', 'Me')
    expect(uploaded?.type).toBe('file')
  })

  it('sets posterUrl when upstream returns video posterUrl', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({
      state: 'OK',
      msg: 'videos/2026/01/a.mp4',
      localFilename: 'a.mp4',
      posterUrl: 'http://img.local/poster.jpg'
    } as any)

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.mp4', { type: 'video/mp4' }), 'me', 'Me')
    expect(uploaded?.type).toBe('video')
    expect(uploaded?.posterUrl).toBe('http://img.local/poster.jpg')
  })

  it('derives posterUrl from posterLocalPath variants when posterUrl is empty', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'resolveImagePort').mockResolvedValue('9006')

    vi.mocked(mediaApi.uploadMedia)
      .mockResolvedValueOnce({
        state: 'OK',
        msg: 'videos/2026/01/a1.mp4',
        localFilename: 'a1.mp4',
        posterUrl: '',
        posterLocalPath: '/upload/poster1.jpg'
      } as any)
      .mockResolvedValueOnce({
        state: 'OK',
        msg: 'videos/2026/01/a2.mp4',
        localFilename: 'a2.mp4',
        posterUrl: '',
        posterLocalPath: '/poster2.jpg'
      } as any)
      .mockResolvedValueOnce({
        state: 'OK',
        msg: 'videos/2026/01/a3.mp4',
        localFilename: 'a3.mp4',
        posterUrl: '',
        posterLocalPath: 'poster3.jpg'
      } as any)

    const { uploadFile } = useUpload()
    const file = new File(['x'], 'a.mp4', { type: 'video/mp4' })

    const u1 = await uploadFile(file, 'me', 'Me')
    expect(u1?.posterUrl).toBe('/upload/poster1.jpg')

    const u2 = await uploadFile(file, 'me', 'Me')
    expect(u2?.posterUrl).toBe('/upload/poster2.jpg')

    const u3 = await uploadFile(file, 'me', 'Me')
    expect(u3?.posterUrl).toBe('/upload/poster3.jpg')
  })

  it('returns null when upstream response is not OK or missing msg', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    const { uploadFile } = useUpload()

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({ state: 'OK', msg: '' } as any)
    expect(await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')).toBeNull()

    vi.mocked(mediaApi.uploadMedia).mockResolvedValue({ state: 'ERR', msg: 'x' } as any)
    expect(await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')).toBeNull()
  })

  it('formats error message from backend response and shows toastError', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    vi.mocked(mediaApi.uploadMedia).mockRejectedValue({
      response: { data: { error: 'bad', localPath: '/tmp/a.png' } }
    })

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')
    expect(uploaded).toBeNull()
    expect(toastError).toHaveBeenCalledWith('bad。文件已保存到本地，可在\"全站图片库\"中重试')
  })

  it('shows backend error message without localPath when provided', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    vi.mocked(mediaApi.uploadMedia).mockRejectedValue({
      response: { data: { error: 'bad2' } }
    })

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')
    expect(uploaded).toBeNull()
    expect(toastError).toHaveBeenCalledWith('bad2')
  })

  it('falls back to Error.message when backend does not provide error details', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    vi.mocked(mediaApi.uploadMedia).mockRejectedValue(new Error('boom'))

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')
    expect(uploaded).toBeNull()
    expect(toastError).toHaveBeenCalledWith('上传失败: boom')
  })

  it('uses default message when neither backend error nor Error.message is present', async () => {
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined as any)

    vi.mocked(mediaApi.uploadMedia).mockRejectedValue({})

    const { uploadFile } = useUpload()
    const uploaded = await uploadFile(new File(['x'], 'a.png', { type: 'image/png' }), 'me', 'Me')
    expect(uploaded).toBeNull()
    expect(toastError).toHaveBeenCalledWith('上传失败，请稍后重试')
  })

  it('getMediaUrl maps relative upload paths', () => {
    const { getMediaUrl } = useUpload()
    expect(getMediaUrl('')).toBe('')
    expect(getMediaUrl('http://x/a.png')).toBe('http://x/a.png')
    expect(getMediaUrl('https://x/a.png')).toBe('https://x/a.png')
    expect(getMediaUrl('/upload/images/a.png')).toContain('/upload/images/a.png')
    expect(getMediaUrl('/images/a.png')).toContain('/upload/images/a.png')
    expect(getMediaUrl('/videos/a.mp4')).toContain('/upload/videos/a.mp4')
    expect(getMediaUrl('/other/a.bin')).toBe('/other/a.bin')
  })
})

describe('composables/useSettings', () => {
  it('loads connection stats and forceout user count', async () => {
    vi.mocked(systemApi.getConnectionStats).mockResolvedValue({ code: 0, data: { active: 1, upstream: 2, downstream: 3 } } as any)
    vi.mocked(systemApi.getForceoutUserCount).mockResolvedValue({ code: 0, data: 9 } as any)

    const settings = useSettings()
    await settings.loadConnectionStats()
    await settings.loadForceoutUserCount()

    expect(settings.connectionStats.value.active).toBe(1)
    expect(settings.forceoutUserCount.value).toBe(9)
  })

  it('disconnectAll toggles loading and returns boolean based on code', async () => {
    vi.mocked(systemApi.disconnectAllConnections).mockResolvedValue({ code: 0 } as any)
    vi.mocked(systemApi.getConnectionStats).mockResolvedValue({ code: 0, data: { active: 0, upstream: 0, downstream: 0 } } as any)

    const settings = useSettings()
    const ok = await settings.disconnectAll()
    expect(ok).toBe(true)
    expect(settings.disconnectAllLoading.value).toBe(false)

    vi.mocked(systemApi.disconnectAllConnections).mockResolvedValue({ code: 1 } as any)
    const ok2 = await settings.disconnectAll()
    expect(ok2).toBe(false)
  })

  it('clearForceout returns success and failure responses', async () => {
    vi.mocked(systemApi.clearForceoutUsers).mockResolvedValue({ code: 0, msg: 'ok' } as any)
    vi.mocked(systemApi.getForceoutUserCount).mockResolvedValue({ code: 0, data: 0 } as any)

    const settings = useSettings()
    const ok = await settings.clearForceout()
    expect(ok).toEqual({ success: true, message: 'ok' })

    vi.mocked(systemApi.clearForceoutUsers).mockResolvedValue({ code: 1, msg: 'no' } as any)
    const bad = await settings.clearForceout()
    expect(bad).toEqual({ success: false, message: 'no' })

    vi.mocked(systemApi.clearForceoutUsers).mockRejectedValue(new Error('boom'))
    const err = await settings.clearForceout()
    expect(err.success).toBe(false)
  })
})

  describe('composables/useIdentity', () => {
    it('select sets currentUser and navigates to /list', async () => {
      const identityStore = useIdentityStore()
      const selectSpy = vi.spyOn(identityStore, 'selectIdentity').mockResolvedValue(undefined as any)

    const userStore = useUserStore()
    expect(userStore.currentUser).toBeNull()

    await useIdentity().select({ id: 'i1', name: 'A', sex: '男', created_at: 't' })

    expect(userStore.currentUser?.id).toBe('i1')
    expect(userStore.currentUser?.cookie).toBe('cookie-x')
    expect(userStore.currentUser?.ip).toBe('127.0.0.9')
      expect(selectSpy).toHaveBeenCalledWith('i1')
      expect(routerPush).toHaveBeenCalledWith('/list')
    })



    it('loadList delegates to identityStore.loadList', async () => {
      const identityStore = useIdentityStore()
      const loadSpy = vi.spyOn(identityStore, 'loadList').mockResolvedValue(undefined as any)

      await useIdentity().loadList()
      expect(loadSpy).toHaveBeenCalledTimes(1)
    })

    it('deleteIdentity delegates to identityStore.deleteIdentity', async () => {
      const identityStore = useIdentityStore()
      const deleteSpy = vi.spyOn(identityStore, 'deleteIdentity').mockResolvedValue(true as any)

      const ok = await useIdentity().deleteIdentity('i-delete')
      expect(ok).toBe(true)
      expect(deleteSpy).toHaveBeenCalledWith('i-delete')
    })

    it('quickCreate generates name/sex and calls createIdentity', async () => {
      const identityStore = useIdentityStore()
      const createSpy = vi.spyOn(identityStore, 'createIdentity').mockResolvedValue(true as any)

      vi.spyOn(Math, 'random')
        .mockReturnValueOnce(0.1234) // name
        .mockReturnValueOnce(0.9) // sex -> 男

      const ok = await useIdentity().quickCreate()
      expect(ok).toBe(true)
      expect(createSpy).toHaveBeenCalled()

      vi.mocked(Math.random).mockRestore()
    })

    it('quickCreate can generate 女 based on random', async () => {
      const identityStore = useIdentityStore()
      const createSpy = vi.spyOn(identityStore, 'createIdentity').mockResolvedValue(true as any)

      vi.spyOn(Math, 'random')
        .mockReturnValueOnce(0.1234) // name
        .mockReturnValueOnce(0.1) // sex -> 女

      const ok = await useIdentity().quickCreate()
      expect(ok).toBe(true)
      expect(createSpy).toHaveBeenCalledWith(expect.objectContaining({ sex: '女' }))

      vi.mocked(Math.random).mockRestore()
    })

    it('select falls back to default name and createdAt', async () => {
      const identityStore = useIdentityStore()
      const selectSpy = vi.spyOn(identityStore, 'selectIdentity').mockResolvedValue(undefined as any)

      const userStore = useUserStore()
      await useIdentity().select({ id: 'i2', sex: '女', createdAt: 't2' })

      expect(userStore.currentUser?.name).toBe('User')
      expect(userStore.currentUser?.created_at).toBe('t2')
      expect(selectSpy).toHaveBeenCalledWith('i2')
      expect(routerPush).toHaveBeenCalledWith('/list')
    })

    it('select uses empty created_at when identity has no timestamps', async () => {
      const identityStore = useIdentityStore()
      const selectSpy = vi.spyOn(identityStore, 'selectIdentity').mockResolvedValue(undefined as any)

      const userStore = useUserStore()
      await useIdentity().select({ id: 'i3', name: 'C', sex: '男' })

      expect(userStore.currentUser?.created_at).toBe('')
      expect(selectSpy).toHaveBeenCalledWith('i3')
      expect(routerPush).toHaveBeenCalledWith('/list')
    })
  })
