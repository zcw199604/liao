import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('stores/chat', () => {
  it('upsertUser merges updates and computed lists derive from ids', () => {
    const store = useChatStore()

    store.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: 'hi',
      lastTime: '刚刚',
      unreadCount: 0
    })

    store.historyUserIds.push('u1')
    expect(store.historyUsers).toHaveLength(1)
    expect(store.historyUsers[0]?.lastMsg).toBe('hi')

    const refBefore = store.getUser('u1')
    store.upsertUser({ ...store.getUser('u1')!, lastMsg: 'hello' })
    const refAfter = store.getUser('u1')

    expect(refAfter).toBe(refBefore)
    expect(store.historyUsers[0]?.lastMsg).toBe('hello')
  })

  it('removeUser removes from lists and clears currentChatUser', () => {
    const store = useChatStore()
    const user = {
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: true,
      lastMsg: 'hi',
      lastTime: '刚刚',
      unreadCount: 0
    }

    store.upsertUser(user)
    store.historyUserIds.push('u1')
    store.favoriteUserIds.push('u1')
    store.enterChat(user as any)

    store.removeUser('u1')

    expect(store.historyUserIds).not.toContain('u1')
    expect(store.favoriteUserIds).not.toContain('u1')
    expect(store.getUser('u1')).toBeUndefined()
    expect(store.currentChatUser).toBeNull()
  })

  it('continuous match state transitions are consistent', () => {
    const store = useChatStore()

    store.startContinuousMatch(3)
    expect(store.isMatching).toBe(true)
    expect(store.continuousMatchConfig.enabled).toBe(true)
    expect(store.continuousMatchConfig.total).toBe(3)
    expect(store.continuousMatchConfig.current).toBe(1)

    store.incrementMatchCount()
    expect(store.continuousMatchConfig.current).toBe(2)

    store.setCurrentMatchedUser({
      id: 'u9',
      name: 'U9',
      nickname: 'U9',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: 'x',
      lastTime: '刚刚',
      unreadCount: 0
    })
    expect(store.currentMatchedUser?.id).toBe('u9')

    store.cancelContinuousMatch()
    expect(store.continuousMatchConfig.enabled).toBe(false)
    expect(store.isMatching).toBe(false)
    expect(store.currentMatchedUser).toBeNull()
  })
})

describe('stores/message', () => {
  it('addMessage sorts by time and tid, and deduplicates by tid', () => {
    const store = useMessageStore()

    const base = {
      code: 7,
      fromuser: { id: 'me', name: 'me', nickname: 'me', sex: '未知', ip: '' },
      touser: { id: 'u1', name: 'u1', nickname: 'u1', sex: '未知', ip: '' },
      type: 'text',
      isSelf: true,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any

    store.addMessage('u1', { ...base, tid: '2', time: '2026-01-01 00:00:00.000', content: 'b' })
    store.addMessage('u1', { ...base, tid: '1', time: '2026-01-01 00:00:00.000', content: 'a' })

    const messages = store.getMessages('u1')
    expect(messages).toHaveLength(2)
    expect(messages.map(m => m.tid)).toEqual(['1', '2'])

    // same tid should be treated as duplicate
    store.addMessage('u1', { ...base, tid: '1', time: '2026-01-01 00:00:00.000', content: 'dup' })
    expect(store.getMessages('u1')).toHaveLength(2)
    expect(store.firstTidMap['u1']).toBe('1')
  })

  it('deduplicates messages without tid using fallback key', () => {
    const store = useMessageStore()

    const msg = {
      code: 7,
      fromuser: { id: 'a', name: 'a', nickname: 'a', sex: '未知', ip: '' },
      touser: { id: 'b', name: 'b', nickname: 'b', sex: '未知', ip: '' },
      type: 'text',
      content: 'same',
      time: '2026-01-01 00:00:00.000',
      tid: '',
      isSelf: false,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any

    store.addMessage('b', msg)
    store.addMessage('b', { ...msg })
    expect(store.getMessages('b')).toHaveLength(1)
  })

  it('clearHistory removes chat history and firstTid', () => {
    const store = useMessageStore()
    store.addMessage('u1', {
      code: 7,
      fromuser: { id: 'me', name: 'me', nickname: 'me', sex: '未知', ip: '' },
      touser: { id: 'u1', name: 'u1', nickname: 'u1', sex: '未知', ip: '' },
      type: 'text',
      content: 'x',
      time: '2026-01-01 00:00:00.000',
      tid: '1',
      isSelf: true,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any)

    expect(store.getMessages('u1')).toHaveLength(1)
    expect(store.firstTidMap['u1']).toBe('1')

    store.clearHistory('u1')
    expect(store.getMessages('u1')).toHaveLength(0)
    expect(store.firstTidMap['u1']).toBeUndefined()
  })
})

