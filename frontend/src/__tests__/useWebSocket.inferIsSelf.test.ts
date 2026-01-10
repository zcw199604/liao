import { describe, expect, it } from 'vitest'
import { inferWsPrivateMessageIsSelf } from '@/composables/useWebSocket'

describe('composables/useWebSocket inferWsPrivateMessageIsSelf', () => {
  it('marks self when fromUserId matches md5(currentUserId)', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'ab86a1e1ef70dff97959067b723c5c24'
    })).toBe(true)
  })

  it('marks not-self when fromUserId does not match md5(currentUserId)', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'e4774cdda0793f86414e8b9140bb6db4'
    })).toBe(false)
  })

  it('is case-insensitive on fromUserId', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'AB86A1E1EF70DFF97959067B723C5C24'
    })).toBe(true)
  })

  it('returns false when input is missing', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: ''
    })).toBe(false)
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: '',
      fromUserId: 'ab86a1e1ef70dff97959067b723c5c24'
    })).toBe(false)
  })
})
