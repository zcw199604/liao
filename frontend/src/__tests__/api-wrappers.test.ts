import { beforeEach, describe, expect, it, vi } from 'vitest'

// These API modules are thin wrappers around request/douyinRequest.
// We mock the request layer so tests assert correct endpoints + payload shapes.
const spies = vi.hoisted(() => ({
  requestGet: vi.fn(),
  requestPost: vi.fn(),
  douyinGet: vi.fn(),
  douyinPost: vi.fn(),
  createFormData: vi.fn((data: any) => ({ __form: data }))
}))

vi.mock('@/api/request', () => ({
  default: {
    get: spies.requestGet,
    post: spies.requestPost
  },
  douyinRequest: {
    get: spies.douyinGet,
    post: spies.douyinPost
  },
  createFormData: spies.createFormData
}))

import * as authApi from '@/api/auth'
import * as chatApi from '@/api/chat'
import * as douyinApi from '@/api/douyin'
import * as favoriteApi from '@/api/favorite'
import * as identityApi from '@/api/identity'
import * as mediaApi from '@/api/media'
import * as mtphotoApi from '@/api/mtphoto'
import * as systemApi from '@/api/system'
import * as videoExtractApi from '@/api/videoExtract'

beforeEach(() => {
  vi.clearAllMocks()
})

describe('api/auth', () => {
  it('login posts urlencoded accessCode', () => {
    authApi.login('code-1')
    expect(spies.createFormData).toHaveBeenCalledWith({ accessCode: 'code-1' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/auth/login',
      { __form: { accessCode: 'code-1' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )
  })

  it('verifyToken calls GET /auth/verify', () => {
    authApi.verifyToken()
    expect(spies.requestGet).toHaveBeenCalledWith('/auth/verify')
  })
})

describe('api/chat', () => {
  it('covers all chat endpoints and payload shapes', () => {
    chatApi.getHistoryUserList('me', 'cookie', 'ref', 'ua')
    expect(spies.createFormData).toHaveBeenCalledWith({
      myUserID: 'me',
      vipcode: '',
      serverPort: '1001',
      cookieData: 'cookie',
      referer: 'ref',
      userAgent: 'ua'
    })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/getHistoryUserList',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.getFavoriteUserList('me', 'cookie', 'ref', 'ua')
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/getFavoriteUserList',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.toggleFavorite('me', 'u2', 'cookie', 'ref', 'ua')
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/toggleFavorite',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.cancelFavorite('me', 'u2', 'cookie', 'ref', 'ua')
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/cancelFavorite',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.getMessageHistory('me', 'u2', '1', '0', 'cookie', 'ref', 'ua')
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/getMessageHistory',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.reportReferrer({
      referrerUrl: 'r1',
      currUrl: 'c1',
      userid: 'me',
      cookieData: 'cookie',
      referer: 'ref',
      userAgent: 'ua'
    })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/reportReferrer',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.getImgServer()
    expect(spies.requestGet).toHaveBeenCalledWith('/getImgServer')

    chatApi.updateImgServer('http://img')
    expect(spies.createFormData).toHaveBeenCalledWith({ server: 'http://img' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/updateImgServer',
      { __form: { server: 'http://img' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.getCachedImages('me')
    expect(spies.requestGet).toHaveBeenCalledWith('/getCachedImages', { params: { userid: 'me' } })

    chatApi.deleteUser('me', 'u2')
    expect(spies.createFormData).toHaveBeenCalledWith({ myUserId: 'me', userToId: 'u2' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/deleteUpstreamUser',
      { __form: { myUserId: 'me', userToId: 'u2' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    chatApi.batchDeleteUsers('me', ['u1', 'u2'])
    expect(spies.requestPost).toHaveBeenCalledWith('/batchDeleteUpstreamUsers', { myUserId: 'me', userToIds: ['u1', 'u2'] })
  })
})

describe('api/identity', () => {
  it('covers all identity endpoints', () => {
    identityApi.getIdentityList()
    expect(spies.requestGet).toHaveBeenCalledWith('/getIdentityList')

    identityApi.createIdentity({ name: 'A', sex: 'M' })
    expect(spies.createFormData).toHaveBeenCalledWith({ name: 'A', sex: 'M' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/createIdentity',
      { __form: { name: 'A', sex: 'M' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    identityApi.deleteIdentity('id-1')
    expect(spies.createFormData).toHaveBeenCalledWith({ id: 'id-1' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/deleteIdentity',
      { __form: { id: 'id-1' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    identityApi.selectIdentity('id-2')
    expect(spies.requestPost).toHaveBeenCalledWith('/selectIdentity', null, { params: { id: 'id-2' } })

    identityApi.updateIdentity({ id: 'id-3', name: 'B', sex: 'F' })
    expect(spies.createFormData).toHaveBeenCalledWith({ id: 'id-3', name: 'B', sex: 'F' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/updateIdentity',
      { __form: { id: 'id-3', name: 'B', sex: 'F' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    identityApi.updateIdentityId({ oldId: 'old', newId: 'new', name: 'C', sex: 'M' })
    expect(spies.createFormData).toHaveBeenCalledWith({ oldId: 'old', newId: 'new', name: 'C', sex: 'M' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/updateIdentityId',
      { __form: { oldId: 'old', newId: 'new', name: 'C', sex: 'M' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )
  })
})

describe('api/system', () => {
  it('covers all system endpoints', () => {
    systemApi.getConnectionStats()
    expect(spies.requestGet).toHaveBeenCalledWith('/getConnectionStats')

    systemApi.disconnectAllConnections()
    expect(spies.requestPost).toHaveBeenCalledWith('/disconnectAllConnections')

    systemApi.getForceoutUserCount()
    expect(spies.requestGet).toHaveBeenCalledWith('/getForceoutUserCount')

    systemApi.clearForceoutUsers()
    expect(spies.requestPost).toHaveBeenCalledWith('/clearForceoutUsers')

    systemApi.getSystemConfig()
    expect(spies.requestGet).toHaveBeenCalledWith('/getSystemConfig')

    systemApi.updateSystemConfig({ imgServer: 'http://img' } as any)
    expect(spies.requestPost).toHaveBeenCalledWith('/updateSystemConfig', { imgServer: 'http://img' })

    systemApi.resolveImagePort('/x/y.jpg')
    expect(spies.requestPost).toHaveBeenCalledWith('/resolveImagePort', { path: '/x/y.jpg' })
  })
})

describe('api/favorite', () => {
  it('covers favorite CRUD wrappers', () => {
    favoriteApi.removeFavorite('i1', 'u1')
    expect(spies.createFormData).toHaveBeenCalledWith({ identityId: 'i1', targetUserId: 'u1' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/favorite/remove',
      { __form: { identityId: 'i1', targetUserId: 'u1' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    favoriteApi.removeFavoriteById(123)
    expect(spies.createFormData).toHaveBeenCalledWith({ id: 123 })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/favorite/removeById',
      { __form: { id: 123 } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    favoriteApi.listAllFavorites()
    expect(spies.requestGet).toHaveBeenCalledWith('/favorite/listAll')

    favoriteApi.checkFavorite('i2', 'u2')
    expect(spies.requestGet).toHaveBeenCalledWith('/favorite/check', { params: { identityId: 'i2', targetUserId: 'u2' } })
  })
})

describe('api/media', () => {
  it('covers media upload/delete wrappers', () => {
    const fd = new FormData()

    mediaApi.uploadMedia(fd)
    expect(spies.requestPost).toHaveBeenCalledWith('/uploadMedia', fd)

    mediaApi.checkDuplicateMedia(fd)
    expect(spies.requestPost).toHaveBeenCalledWith('/checkDuplicateMedia', fd)

    mediaApi.getImgServerAddress()
    expect(spies.requestGet).toHaveBeenCalledWith('/getImgServer')

    mediaApi.updateImgServerAddress('http://img2')
    expect(spies.createFormData).toHaveBeenCalledWith({ server: 'http://img2' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/updateImgServer',
      { __form: { server: 'http://img2' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    mediaApi.deleteMedia('/tmp/a.jpg', 'me')
    expect(spies.createFormData).toHaveBeenCalledWith({ localPath: '/tmp/a.jpg', userId: 'me' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/deleteMedia',
      { __form: { localPath: '/tmp/a.jpg', userId: 'me' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    mediaApi.batchDeleteMedia('me', ['/a.jpg', '/b.jpg'])
    expect(spies.requestPost).toHaveBeenCalledWith('/batchDeleteMedia', { userId: 'me', localPaths: ['/a.jpg', '/b.jpg'] })

    mediaApi.getCachedImages('me')
    expect(spies.requestGet).toHaveBeenCalledWith('/getCachedImages', { params: { userid: 'me' } })

    mediaApi.reuploadHistoryImage({ userId: 'me', localPath: '/x.png', cookieData: 'c', referer: 'r', userAgent: 'ua' })
    expect(spies.createFormData).toHaveBeenCalledWith({
      userId: 'me',
      localPath: '/x.png',
      cookieData: 'c',
      referer: 'r',
      userAgent: 'ua'
    })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/reuploadHistoryImage',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    mediaApi.recordImageSend({ remoteUrl: 'http://r', fromUserId: 'a', toUserId: 'b', localFilename: 'x.jpg' })
    expect(spies.createFormData).toHaveBeenCalledWith({ remoteUrl: 'http://r', fromUserId: 'a', toUserId: 'b', localFilename: 'x.jpg' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/recordImageSend',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )
  })
})

describe('api/mtphoto', () => {
  it('covers mtphoto wrappers', () => {
    mtphotoApi.getMtPhotoAlbums()
    expect(spies.requestGet).toHaveBeenCalledWith('/getMtPhotoAlbums')

    mtphotoApi.getMtPhotoAlbumFiles(1, 2, 3)
    expect(spies.requestGet).toHaveBeenCalledWith('/getMtPhotoAlbumFiles', { params: { albumId: 1, page: 2, pageSize: 3 } })

    mtphotoApi.resolveMtPhotoFilePath('md5-1')
    expect(spies.requestGet).toHaveBeenCalledWith('/resolveMtPhotoFilePath', { params: { md5: 'md5-1' } })

    mtphotoApi.importMtPhotoMedia({ userid: 'me', md5: 'md5-2', cookieData: 'c', referer: 'r', userAgent: 'ua' })
    expect(spies.createFormData).toHaveBeenCalledWith({ userid: 'me', md5: 'md5-2', cookieData: 'c', referer: 'r', userAgent: 'ua' })
    expect(spies.requestPost).toHaveBeenCalledWith(
      '/importMtPhotoMedia',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )
  })
})

describe('api/videoExtract', () => {
  it('covers video extract endpoints', () => {
    const fd = new FormData()

    videoExtractApi.uploadVideoExtractInput(fd)
    expect(spies.requestPost).toHaveBeenCalledWith('/uploadVideoExtractInput', fd)

    videoExtractApi.cleanupVideoExtractInput('/tmp/v.mp4')
    expect(spies.requestPost).toHaveBeenCalledWith('/cleanupVideoExtractInput', { localPath: '/tmp/v.mp4' })

    videoExtractApi.probeVideo({ sourceType: 'upload', localPath: '/tmp/v.mp4' })
    expect(spies.requestGet).toHaveBeenCalledWith('/probeVideo', { params: { sourceType: 'upload', localPath: '/tmp/v.mp4' } })

    videoExtractApi.createVideoExtractTask({ a: 1 })
    expect(spies.requestPost).toHaveBeenCalledWith('/createVideoExtractTask', { a: 1 })

    videoExtractApi.getVideoExtractTaskList(1, 20)
    expect(spies.requestGet).toHaveBeenCalledWith('/getVideoExtractTaskList', { params: { page: 1, pageSize: 20 } })

    videoExtractApi.getVideoExtractTaskDetail({ taskId: 't1', cursor: 10, pageSize: 5 })
    expect(spies.requestGet).toHaveBeenCalledWith('/getVideoExtractTaskDetail', { params: { taskId: 't1', cursor: 10, pageSize: 5 } })

    videoExtractApi.cancelVideoExtractTask('t2')
    expect(spies.requestPost).toHaveBeenCalledWith('/cancelVideoExtractTask', { taskId: 't2' })

    videoExtractApi.continueVideoExtractTask({ taskId: 't3', endSec: 9, maxFrames: 99 })
    expect(spies.requestPost).toHaveBeenCalledWith('/continueVideoExtractTask', { taskId: 't3', endSec: 9, maxFrames: 99 })

    videoExtractApi.deleteVideoExtractTask({ taskId: 't4', deleteFiles: true })
    expect(spies.requestPost).toHaveBeenCalledWith('/deleteVideoExtractTask', { taskId: 't4', deleteFiles: true })
  })
})

describe('api/douyin', () => {
  it('covers all douyin wrappers', () => {
    douyinApi.getDouyinDetail({ input: 'x', cookie: 'c' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/detail', { input: 'x', cookie: 'c' })

    douyinApi.getDouyinAccount({ input: 'x', tab: 'favorite', cursor: 1, count: 20 })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/account', { input: 'x', tab: 'favorite', cursor: 1, count: 20 })

    douyinApi.importDouyinMedia({ userid: 'me', key: 'k', index: 1, cookieData: 'c', referer: 'r', userAgent: 'ua' })
    expect(spies.createFormData).toHaveBeenCalledWith({ userid: 'me', key: 'k', index: 1, cookieData: 'c', referer: 'r', userAgent: 'ua' })
    expect(spies.douyinPost).toHaveBeenCalledWith(
      '/douyin/import',
      { __form: expect.any(Object) },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    // Favorite users
    douyinApi.listDouyinFavoriteUsers()
    expect(spies.douyinGet).toHaveBeenCalledWith('/douyin/favoriteUser/list')

    douyinApi.addDouyinFavoriteUser({ secUserId: 'sec', displayName: 'n' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/add', { secUserId: 'sec', displayName: 'n' })

    douyinApi.removeDouyinFavoriteUser({ secUserId: 'sec' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/remove', { secUserId: 'sec' })

    douyinApi.listDouyinFavoriteUserAwemes({ secUserId: 'sec', cursor: 0, count: 12 })
    expect(spies.douyinGet).toHaveBeenCalledWith('/douyin/favoriteUser/aweme/list', { params: { secUserId: 'sec', cursor: 0, count: 12 } })

    douyinApi.upsertDouyinFavoriteUserAwemes({ secUserId: 'sec', items: [{ awemeId: 'a1' }] })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/aweme/upsert', { secUserId: 'sec', items: [{ awemeId: 'a1' }] })

    douyinApi.pullLatestDouyinFavoriteUserAwemes({ secUserId: 'sec', cookie: 'c', count: 1 })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/aweme/pullLatest', { secUserId: 'sec', cookie: 'c', count: 1 })

    // Favorite awemes
    douyinApi.listDouyinFavoriteAwemes()
    expect(spies.douyinGet).toHaveBeenCalledWith('/douyin/favoriteAweme/list')

    douyinApi.addDouyinFavoriteAweme({ awemeId: 'a1', desc: 'd' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/add', { awemeId: 'a1', desc: 'd' })

    douyinApi.removeDouyinFavoriteAweme({ awemeId: 'a1' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/remove', { awemeId: 'a1' })

    // Tags (user)
    douyinApi.listDouyinFavoriteUserTags()
    expect(spies.douyinGet).toHaveBeenCalledWith('/douyin/favoriteUser/tag/list')

    douyinApi.addDouyinFavoriteUserTag({ name: 't1' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/tag/add', { name: 't1' })

    douyinApi.updateDouyinFavoriteUserTag({ id: 1, name: 't2' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/tag/update', { id: 1, name: 't2' })

    douyinApi.removeDouyinFavoriteUserTag({ id: 2 })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/tag/remove', { id: 2 })

    douyinApi.applyDouyinFavoriteUserTags({ secUserIds: ['sec'], tagIds: [1], mode: 'add' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/tag/apply', { secUserIds: ['sec'], tagIds: [1], mode: 'add' })

    douyinApi.reorderDouyinFavoriteUserTags({ tagIds: [2, 1] })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteUser/tag/reorder', { tagIds: [2, 1] })

    // Tags (aweme)
    douyinApi.listDouyinFavoriteAwemeTags()
    expect(spies.douyinGet).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/list')

    douyinApi.addDouyinFavoriteAwemeTag({ name: 'a1' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/add', { name: 'a1' })

    douyinApi.updateDouyinFavoriteAwemeTag({ id: 1, name: 'a2' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/update', { id: 1, name: 'a2' })

    douyinApi.removeDouyinFavoriteAwemeTag({ id: 2 })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/remove', { id: 2 })

    douyinApi.applyDouyinFavoriteAwemeTags({ awemeIds: ['a1'], tagIds: [1], mode: 'set' })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/apply', { awemeIds: ['a1'], tagIds: [1], mode: 'set' })

    douyinApi.reorderDouyinFavoriteAwemeTags({ tagIds: [1, 2] })
    expect(spies.douyinPost).toHaveBeenCalledWith('/douyin/favoriteAweme/tag/reorder', { tagIds: [1, 2] })
  })
})

