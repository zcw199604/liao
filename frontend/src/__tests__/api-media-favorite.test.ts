import { beforeEach, describe, expect, it, vi } from 'vitest'

const requestSpies = vi.hoisted(() => ({
  requestGet: vi.fn(),
  requestPost: vi.fn(),
  createFormData: vi.fn((data: any) => ({ __form: data }))
}))

vi.mock('@/api/request', () => ({
  default: {
    get: requestSpies.requestGet,
    post: requestSpies.requestPost
  },
  createFormData: requestSpies.createFormData
}))

import { addFavorite } from '@/api/favorite'
import { getAllUploadImages, getChatImages } from '@/api/media'

beforeEach(() => {
  vi.clearAllMocks()
})

describe('api/media', () => {
  it('getAllUploadImages merges optional params', () => {
    getAllUploadImages(1, 20)
    expect(requestSpies.requestGet).toHaveBeenCalledWith('/getAllUploadImages', {
      params: { page: 1, pageSize: 20 }
    })

    getAllUploadImages(2, 10, { source: 'local', douyinSecUserId: 'sec' })
    expect(requestSpies.requestGet).toHaveBeenLastCalledWith('/getAllUploadImages', {
      params: { page: 2, pageSize: 10, source: 'local', douyinSecUserId: 'sec' }
    })
  })

  it('getChatImages uses default and custom limit', () => {
    getChatImages('a', 'b')
    expect(requestSpies.requestGet).toHaveBeenCalledWith('/getChatImages', {
      params: { userId1: 'a', userId2: 'b', limit: 20 }
    })

    getChatImages('a', 'b', 99)
    expect(requestSpies.requestGet).toHaveBeenCalledWith('/getChatImages', {
      params: { userId1: 'a', userId2: 'b', limit: 99 }
    })
  })
})

describe('api/favorite', () => {
  it('addFavorite fills optional targetUserName', () => {
    addFavorite('i1', 'u1')
    expect(requestSpies.createFormData).toHaveBeenCalledWith({ identityId: 'i1', targetUserId: 'u1', targetUserName: '' })
    expect(requestSpies.requestPost).toHaveBeenCalledWith(
      '/favorite/add',
      { __form: { identityId: 'i1', targetUserId: 'u1', targetUserName: '' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )

    addFavorite('i1', 'u1', 'Bob')
    expect(requestSpies.createFormData).toHaveBeenCalledWith({ identityId: 'i1', targetUserId: 'u1', targetUserName: 'Bob' })
    expect(requestSpies.requestPost).toHaveBeenCalledWith(
      '/favorite/add',
      { __form: { identityId: 'i1', targetUserId: 'u1', targetUserName: 'Bob' } },
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
    )
  })
})
