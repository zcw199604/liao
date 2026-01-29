import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useMessageStore } from '@/stores/message'

const makeUser = (id: string) => ({
  id,
  name: id,
  nickname: id,
  sex: '未知',
  ip: ''
})

const makeMsg = (partial: any) => {
  return {
    code: 7,
    fromuser: makeUser('from'),
    touser: makeUser('to'),
    type: 'text',
    content: '',
    time: '2026-01-01 00:00:00.000',
    tid: '',
    isSelf: false,
    isImage: false,
    isVideo: false,
    isFile: false,
    imageUrl: '',
    videoUrl: '',
    fileUrl: '',
    segments: [],
    ...partial
  } as any
}

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
})

describe('stores/message normalize & optimistic merge', () => {
  it('sorts by numeric tid when parsed time matches', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '10', time: '2026-01-01 00:00', content: 'b', fromuser: makeUser('b') }),
      makeMsg({ tid: '2', time: '2026-01-01 00:00', content: 'a', fromuser: makeUser('a') })
    ])

    expect(store.getMessages('u1').map(m => String(m.tid))).toEqual(['2', '10'])
  })

  it('sorts invalid tid before valid tid when time parsing fails', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '3', time: 'bad-time', content: 'c', fromuser: makeUser('c') }),
      makeMsg({ tid: 'x', time: 'bad-time', content: 'd', fromuser: makeUser('d') })
    ])

    expect(store.getMessages('u1').map(m => String(m.tid))).toEqual(['x', '3'])
  })

  it('covers reverse tid-valid comparator direction (invalid vs valid)', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: 'x', time: 'bad-time', content: 'd', fromuser: makeUser('d') }),
      makeMsg({ tid: '3', time: 'bad-time', content: 'c', fromuser: makeUser('c') })
    ])

    expect(store.getMessages('u1').map(m => String(m.tid))).toEqual(['x', '3'])
  })

  it('sorts by raw time string when tid is invalid', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '', time: 'bad-time', content: 'b', fromuser: makeUser('b') }),
      makeMsg({ tid: '', time: '', content: 'a', fromuser: makeUser('a') })
    ])

    expect(store.getMessages('u1').map(m => String(m.time))).toEqual(['', 'bad-time'])
  })

  it('sorts by fromuser.id when time & tid are equal', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '', time: 'same', content: 'x', fromuser: makeUser('b') }),
      makeMsg({ tid: '', time: 'same', content: 'x', fromuser: makeUser('a') })
    ])

    expect(store.getMessages('u1').map(m => String(m.fromuser?.id))).toEqual(['a', 'b'])
  })

  it('sorts by content when time & fromuser.id are equal', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '', time: 'same', content: 'b', fromuser: makeUser('a') }),
      makeMsg({ tid: '', time: 'same', content: 'a', fromuser: makeUser('a') })
    ])

    expect(store.getMessages('u1').map(m => String(m.content))).toEqual(['a', 'b'])
  })

  it('sorts empty content before non-empty content when all other fields match', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({ tid: '', time: 'same', content: 'a', fromuser: makeUser('a') }),
      makeMsg({ tid: '', time: 'same', content: '', fromuser: makeUser('a') })
    ])

    expect(store.getMessages('u1').map(m => String(m.content))).toEqual(['', 'a'])
  })

  it('deduplicates media messages and keeps higher-rank version', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        tid: '',
        isSelf: true,
        type: 'image',
        content: '[2026/01/a.png]',
        time: '2026-01-01 00:00:00.000',
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }],
        clientId: 'cid-1',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        tid: 't-1',
        isSelf: true,
        type: 'image',
        content: '[2026/01/a.png]',
        time: '2026-01-01 00:00:00.100',
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }],
        sendStatus: 'sent',
        optimistic: false
      })
    ])

    const list = store.getMessages('u1')
    expect(list).toHaveLength(1)
    expect(String(list[0]?.tid)).toBe('t-1')
  })

  it('keeps separate media dedup keys for in/out directions', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        tid: '',
        isSelf: true,
        type: 'image',
        time: '2026-01-01 00:00:00.000',
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }],
        content: '[2026/01/a.png]'
      }),
      makeMsg({
        tid: '',
        isSelf: false,
        type: 'image',
        time: '2026-01-01 00:00:00.000',
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }],
        content: '[2026/01/a.png]'
      })
    ])

    expect(store.getMessages('u1')).toHaveLength(2)
  })

  it('normalizes remote path by stripping query/hash and ignores emoji tokens', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        isSelf: true,
        time: '2026-01-01 00:00:00.000',
        imageUrl: 'http://img:9006/img/Upload/2026/01/a.png?x=1#h',
        tid: '',
        clientId: 'cid-1',
        sendStatus: 'sending'
      }),
      makeMsg({
        isSelf: true,
        time: '2026-01-01 00:00:00.100',
        imageUrl: 'http://img:9006/img/Upload/2026/01/a.png',
        tid: 't-1',
        sendStatus: 'sent'
      }),
      makeMsg({
        time: '2026-01-01 00:00:00.200',
        tid: '',
        content: '[doge]'
      })
    ])

    const list = store.getMessages('u1')
    expect(list).toHaveLength(2)
    expect(list.some(m => String(m.tid) === 't-1')).toBe(true)
    expect(list.some(m => String(m.content) === '[doge]')).toBe(true)
  })

  it('updateMessageByClientId returns false when no match and true when updated', () => {
    const store = useMessageStore()
    expect(store.updateMessageByClientId('u1', 'cid-x', () => {})).toBe(false)

    store.addMessage('u1', makeMsg({ clientId: 'cid-1', content: 'a' }))
    expect(store.updateMessageByClientId('u1', 'cid-x', () => {})).toBe(false)

    const ok = store.updateMessageByClientId('u1', 'cid-1', msg => {
      ;(msg as any).content = 'b'
    })
    expect(ok).toBe(true)
    expect(String(store.getMessages('u1')[0]?.content)).toBe('b')
  })

  it('startOptimisticTimeout marks sending message failed and ignores non-sending', async () => {
	    vi.useFakeTimers()
	    try {
	      const store = useMessageStore()
	      store.addMessage('u1', makeMsg({ clientId: 'cid-1', sendStatus: 'sending', content: 'm1' }))
	      store.addMessage('u1', makeMsg({ clientId: 'cid-2', sendStatus: 'sent', content: 'm2' }))

      store.startOptimisticTimeout('u1', 'cid-1', 1)
      store.startOptimisticTimeout('u1', 'cid-2', 1)

      await vi.advanceTimersByTimeAsync(1)

      const list = store.getMessages('u1') as any[]
      const m1 = list.find(m => m.clientId === 'cid-1')
      const m2 = list.find(m => m.clientId === 'cid-2')
      expect(m1.sendStatus).toBe('failed')
      expect(m1.sendError).toBe('发送超时')
      expect(m2.sendStatus).toBe('sent')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('confirmOutgoingEcho selects best optimistic match and ignores out-of-window echoes', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.000',
        content: 'hello<br>world',
        clientId: 'cid-sending',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.500',
        content: 'hello world',
        clientId: 'cid-failed',
        sendStatus: 'failed',
        optimistic: true
      })
    ])

    const matched = store.confirmOutgoingEcho(
      'u1',
      makeMsg({
        isSelf: true,
        tid: 't-echo',
        time: '2026-01-01 00:00:00.700',
        content: 'hello world',
        type: 'text'
      })
    )
    expect(matched).toBe(true)

    const list = store.getMessages('u1') as any[]
    const updated = list.find(m => m.clientId === 'cid-sending')
    expect(updated.sendStatus).toBe('sent')
    expect(updated.optimistic).toBe(false)
    expect(updated.tid).toBe('t-echo')

    const tooLate = store.confirmOutgoingEcho(
      'u1',
      makeMsg({
        isSelf: true,
        tid: 't-late',
        time: '2026-01-01 00:02:00.000',
        content: 'hello world'
      })
    )
    expect(tooLate).toBe(false)
  })

  it('confirmOutgoingEcho returns false when echoed message is not self', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.000',
        content: 'hello',
        clientId: 'cid-1',
        sendStatus: 'sending',
        optimistic: true
      })
    ])

    const ok = store.confirmOutgoingEcho('u1', makeMsg({ isSelf: false, content: 'hello' }))
    expect(ok).toBe(false)
  })

  it('confirmOutgoingEcho skips non-matching candidates and prefers sending over failed', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.000',
        content: 'hello',
        clientId: 'cid-failed',
        sendStatus: 'failed',
        optimistic: true
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.100',
        content: 'hello',
        clientId: 'cid-sending',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        isSelf: false,
        tid: '',
        time: '2026-01-01 00:00:00.200',
        content: 'hello',
        clientId: 'cid-other',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.300',
        content: 'hello',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.350',
        content: 'hello',
        clientId: 'cid-sent',
        sendStatus: 'sent',
        optimistic: false
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.400',
        content: '[2026/01/a.png]',
        type: 'image',
        isImage: true,
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img/a.png' }],
        clientId: 'cid-img',
        sendStatus: 'sending',
        optimistic: true
      }),
      makeMsg({
        isSelf: true,
        tid: '',
        time: '2026-01-01 00:00:00.450',
        content: 'different',
        clientId: 'cid-diff',
        sendStatus: 'sending',
        optimistic: true
      })
    ])

    const ok = store.confirmOutgoingEcho(
      'u1',
      makeMsg({
        isSelf: true,
        tid: 't-echo',
        time: '2026-01-01 00:00:00.500',
        content: 'hello',
        type: 'text'
      })
    )

    expect(ok).toBe(true)
    const updated = (store.getMessages('u1') as any[]).find(m => m.clientId === 'cid-sending')
    expect(updated.sendStatus).toBe('sent')
    expect(updated.tid).toBe('t-echo')
  })

  it('confirmOutgoingEcho matches media messages by remote path', () => {
    const store = useMessageStore()
    store.setMessages('u1', [
      makeMsg({
        isSelf: true,
        tid: '',
        time: 'bad-time',
        content: '[2026/01/a.png]',
        type: 'image',
        isImage: true,
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }],
        clientId: 'cid-img',
        sendStatus: 'sending',
        optimistic: true
      })
    ])

    const ok = store.confirmOutgoingEcho(
      'u1',
      makeMsg({
        isSelf: true,
        tid: 't-img',
        time: 'bad-time-2',
        content: '[2026/01/a.png]',
        type: 'image',
        isImage: true,
        segments: [{ kind: 'image', path: '2026/01/a.png', url: 'http://img:9006/img/Upload/2026/01/a.png' }]
      })
    )
    expect(ok).toBe(true)

    const updated = (store.getMessages('u1') as any[]).find(m => m.clientId === 'cid-img')
    expect(updated.sendStatus).toBe('sent')
    expect(updated.tid).toBe('t-img')
  })
})
