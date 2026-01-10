import { describe, expect, it } from 'vitest'
import { inferWsPrivateMessageIsSelf } from '@/composables/useWebSocket'

describe('composables/useWebSocket inferWsPrivateMessageIsSelf', () => {
  it('marks self when fromUserId matches currentUserId', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'me',
      toUserId: 'u1'
    })).toBe(true)
  })

  it('marks not-self when toUserId matches currentUserId', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'u1',
      toUserId: 'me'
    })).toBe(false)
  })

  it('handles alias sender id by using peerId/toUserId', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'alias-me',
      toUserId: 'peer',
      peerId: 'peer'
    })).toBe(true)
  })

  it('handles alias receiver id by using peerId/fromUserId', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'peer',
      toUserId: 'alias-me',
      peerId: 'peer'
    })).toBe(false)
  })

  it('falls back to nickname when ids are not reliable', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      currentUserNickname: '我',
      fromUserId: 'alias-me',
      toUserId: 'peer',
      fromUserNickname: '我',
      toUserNickname: '对方'
    })).toBe(true)
  })

  it('falls back to known user list to avoid alias treated as new user', () => {
    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'alias-me',
      toUserId: 'peer',
      isKnownUserId: (userId) => userId === 'peer'
    })).toBe(true)

    expect(inferWsPrivateMessageIsSelf({
      currentUserId: 'me',
      fromUserId: 'peer',
      toUserId: 'alias-me',
      isKnownUserId: (userId) => userId === 'peer'
    })).toBe(false)
  })
})

