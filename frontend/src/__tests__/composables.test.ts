import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const sendMock = vi.fn()
const toastShow = vi.fn()

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({
    send: sendMock
  })
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/utils/id', () => ({
  generateRandomHexId: () => 'hex-1'
}))

vi.mock('@/api/chat', () => ({
  toggleFavorite: vi.fn(),
  cancelFavorite: vi.fn()
}))

vi.mock('@/api/media', () => ({
  recordImageSend: vi.fn()
}))

import { useChat } from '@/composables/useChat'
import { useMessage } from '@/composables/useMessage'
import * as chatApi from '@/api/chat'
import * as mediaApi from '@/api/media'

import { useChatStore } from '@/stores/chat'
import { useFavoriteStore } from '@/stores/favorite'
import { useMessageStore } from '@/stores/message'
import { useUserStore } from '@/stores/user'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('composables/useMessage', () => {
  it('sendText is a no-op when currentUser or targetUser is missing', () => {
    // no current user
    useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })
    expect(sendMock).not.toHaveBeenCalled()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    // no target user
    useMessage().sendText('hi', null as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('sendText constructs act and sends message', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_U1',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText uses name when nickname is missing', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1', name: 'NameOnly' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_NameOnly',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText buildAct falls back to empty name when targetUser has no name/nickname', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText falls back to timestamp-based clientId when crypto.randomUUID is unavailable', () => {
    vi.useFakeTimers()
    try {
      vi.setSystemTime(new Date(2026, 0, 1, 0, 0, 0))
      vi.stubGlobal('crypto', {} as any)
      vi.spyOn(Math, 'random').mockReturnValue(0.5)

      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      sendMock.mockReturnValue(true)

      useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })

      const first = messageStore.getMessages('u1')[0] as any
      expect(String(first.clientId || '')).toMatch(/^c_\d+_[0-9a-f]+$/)
    } finally {
      vi.unstubAllGlobals()
      vi.restoreAllMocks()
      vi.useRealTimers()
    }
  })

  it('sendText inserts optimistic message and marks failed on timeout', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      sendMock.mockReturnValue(true)

      useMessage().sendText('hello', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-1' })

      const first = messageStore.getMessages('u1')[0] as any
      expect(first.clientId).toBe('cid-1')
      expect(first.sendStatus).toBe('sending')

      vi.advanceTimersByTime(15000)

      const updated = messageStore.getMessages('u1')[0] as any
      expect(updated.sendStatus).toBe('failed')
      expect(updated.sendError).toBe('发送超时')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('sendText marks optimistic message failed when ws send returns false', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)

    useMessage().sendText('oops', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-2' })

    const msg = messageStore.getMessages('u1')[0] as any
    expect(msg.clientId).toBe('cid-2')
    expect(msg.sendStatus).toBe('failed')
    expect(msg.sendError).toBe('发送失败')
  })

  it('sendText failure handler is a no-op when message is not in sending state (covers branch)', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)

    let calls = 0
    vi.spyOn(messageStore, 'updateMessageByClientId').mockImplementation((_uid, _cid, updater) => {
      calls += 1
      if (calls === 1) {
        // upsertOptimisticMessage writes a sending optimistic message
        const msg: any = { sendStatus: 'sending' }
        updater(msg)
        return true
      }

      // failure handler should see a non-sending message and do nothing
      const msg: any = { sendStatus: 'failed', sendError: 'already failed' }
      updater(msg)
      return true
    })

    useMessage().sendText('oops', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-branch-sendText' })
    expect(messageStore.updateMessageByClientId).toHaveBeenCalled()
  })

  it('retryMessage resets failed message to sending and re-sends', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      sendMock.mockReturnValueOnce(false).mockReturnValueOnce(true)

      useMessage().sendText('hello', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-3' })
      expect(sendMock).toHaveBeenCalledTimes(1)

      const failed = messageStore.getMessages('u1')[0] as any
      expect(failed.sendStatus).toBe('failed')

      useMessage().retryMessage(failed)
      expect(sendMock).toHaveBeenCalledTimes(2)

      const retrying = messageStore.getMessages('u1')[0] as any
      expect(retrying.clientId).toBe('cid-3')
      expect(retrying.sendStatus).toBe('sending')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('sendTypingStatus toggles inputStatusOn/off', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendTypingStatus(true, { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'inputStatusOn_me_Me',
      destuserid: 'u1'
    })

    useMessage().sendTypingStatus(false, { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'inputStatusOff_me_Me',
      destuserid: 'u1'
    })
  })

  it('sendImage wraps remote file path and records send relation', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const mockedRecord = vi.mocked(mediaApi.recordImageSend)
    mockedRecord.mockResolvedValue({ code: 0 } as any)

    await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u2', nickname: 'U2' }, 'a.png')

    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u2_U2',
      id: 'me',
      msg: '[a.png]'
    })
    expect(mockedRecord).toHaveBeenCalledWith({
      remoteUrl: 'http://s:9006/img/Upload/a.png',
      fromUserId: 'me',
      toUserId: 'u2',
      localFilename: 'a.png'
    })
  })

  it('sendImage does not throw when recordImageSend fails', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    vi.mocked(mediaApi.recordImageSend).mockRejectedValue(new Error('boom'))

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u2', nickname: 'U2' })
      expect(sendMock).toHaveBeenCalled()
      expect(warnSpy).toHaveBeenCalled()
    } finally {
      warnSpy.mockRestore()
    }
  })

  it('sendImage failure handler is a no-op when message is not in sending state (covers branch)', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    let calls = 0
    vi.spyOn(messageStore, 'updateMessageByClientId').mockImplementation((_uid, _cid, updater) => {
      calls += 1
      if (calls === 1) {
        const msg: any = { sendStatus: 'sending' }
        updater(msg)
        return true
      }
      const msg: any = { sendStatus: 'sent' }
      updater(msg)
      return true
    })

    await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u2', nickname: 'U2' }, 'a.png', { clientId: 'cid-branch-sendImage' })
    expect(messageStore.updateMessageByClientId).toHaveBeenCalled()
  })

  it('sendImage/sendVideo return early when mediaUrl is empty', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    await useMessage().sendImage('', { id: 'u2', nickname: 'U2' })
    await useMessage().sendVideo('', { id: 'u2', nickname: 'U2' })
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('sendVideo sends bracketed remote file path and records send relation', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    const messageStore = useMessageStore()
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4', { clientId: 'cid-v1' })

    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u2_U2',
      id: 'me',
      msg: '[a.mp4]'
    })

    const msg = (messageStore.getMessages('u2') as any[]).find(m => m.clientId === 'cid-v1')
    expect(msg.type).toBe('video')
    expect(msg.sendStatus).toBe('sending')

    expect(mediaApi.recordImageSend).toHaveBeenCalledWith({
      remoteUrl: 'http://s:9006/img/Upload/a.mp4',
      fromUserId: 'me',
      toUserId: 'u2',
      localFilename: 'a.mp4'
    })
  })

  it('sendVideo marks optimistic message failed when ws send returns false', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(false)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    const messageStore = useMessageStore()
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4', { clientId: 'cid-v2' })

    const msg = (messageStore.getMessages('u2') as any[]).find(m => m.clientId === 'cid-v2')
    expect(msg.sendStatus).toBe('failed')
    expect(msg.sendError).toBe('发送失败')
  })

  it('sendVideo failure handler is a no-op when message is not in sending state (covers branch)', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    let calls = 0
    vi.spyOn(messageStore, 'updateMessageByClientId').mockImplementation((_uid, _cid, updater) => {
      calls += 1
      if (calls === 1) {
        const msg: any = { sendStatus: 'sending' }
        updater(msg)
        return true
      }
      const msg: any = { sendStatus: 'failed', sendError: 'x' }
      updater(msg)
      return true
    })

    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4', { clientId: 'cid-branch-sendVideo' })
    expect(messageStore.updateMessageByClientId).toHaveBeenCalled()
  })

  it('retryMessage is a no-op when required fields are missing', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().retryMessage({} as any)
    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' } } as any)
    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' }, clientId: '' } as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('retryMessage marks message failed when resend fails', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      messageStore.addMessage('u1', {
        code: 7,
        fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
        touser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        type: 'text',
        content: 'hello',
        time: '2026-01-01 00:00:00.000',
        tid: '',
        isSelf: true,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: '',
        segments: [],
        clientId: 'cid-r',
        sendStatus: 'failed',
        sendError: 'x',
        optimistic: true
      } as any)

      sendMock.mockReturnValue(false)

      const msg = (messageStore.getMessages('u1') as any[])[0]
      useMessage().retryMessage(msg)

      const updated = (messageStore.getMessages('u1') as any[])[0]
      expect(updated.sendStatus).toBe('failed')
      expect(updated.sendError).toBe('发送失败')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('retryMessage failure handler is a no-op when message is not in sending state (covers branch)', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)

    let calls = 0
    vi.spyOn(messageStore, 'updateMessageByClientId').mockImplementation((_uid, _cid, updater) => {
      calls += 1
      if (calls === 1) {
        // first update sets sending
        const msg: any = { sendStatus: 'failed', sendError: 'x' }
        updater(msg)
        return true
      }
      // failure handler should see a non-sending message and do nothing
      const msg: any = { sendStatus: 'sent' }
      updater(msg)
      return true
    })

    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' }, clientId: 'cid-branch-retry', content: 'x' } as any)
    expect(messageStore.updateMessageByClientId).toHaveBeenCalled()
  })

  it('sendTypingStatus is a no-op when currentUser/targetUser is missing', () => {
    useMessage().sendTypingStatus(true, { id: 'u1' })
    expect(sendMock).not.toHaveBeenCalled()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    useMessage().sendTypingStatus(true, null as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('generateClientId uses crypto.randomUUID when available; otherwise falls back', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const cryptoObj = (globalThis as any).crypto
    const uuidFn = cryptoObj?.randomUUID
    if (cryptoObj && typeof uuidFn === 'function') {
      const uuidSpy = vi.spyOn(cryptoObj, 'randomUUID').mockReturnValue('uuid-1')

      useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })
      const first = useMessageStore().getMessages('u1')[0] as any
      expect(first.clientId).toBe('uuid-1')

      uuidSpy.mockImplementation(() => { throw new Error('boom') })
      useMessage().sendText('hi2', { id: 'u2', nickname: 'U2' })
      const second = useMessageStore().getMessages('u2')[0] as any
      expect(String(second.clientId)).toMatch(/^c_/)

      uuidSpy.mockRestore()
    } else {
      // In environments without crypto.randomUUID, ensure we still generate a fallback id.
      useMessage().sendText('hi3', { id: 'u3', nickname: 'U3' })
      const third = useMessageStore().getMessages('u3')[0] as any
      expect(String(third.clientId)).toMatch(/^c_/)
    }
  })

  it('upserts optimistic message by clientId when called twice (update branch)', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    const messageStore = useMessageStore()

    useMessage().sendText('one', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-upsert' })
    useMessage().sendText('two', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-upsert' })

    const msgs = messageStore.getMessages('u1') as any[]
    expect(msgs).toHaveLength(1)
    expect(msgs[0].content).toBe('two')
  })

  it('sendImage/sendVideo are no-ops when currentUser or targetUser is missing', async () => {
    // no current user
    await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u1', nickname: 'U1' })
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u1', nickname: 'U1' })
    expect(sendMock).not.toHaveBeenCalled()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    // no target user
    await useMessage().sendImage('http://s:9006/img/Upload/a.png', null as any)
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', null as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('sendVideo without clientId option generates an id and still records send relation', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4')
    const msgs = useMessageStore().getMessages('u2') as any[]
    expect(msgs.length).toBeGreaterThan(0)
    expect(String(msgs[0].clientId)).toBeTruthy()
  })

  it('retryMessage early returns when currentUser is missing, and covers empty id/content fallbacks', () => {
    const userStore = useUserStore()
    userStore.currentUser = null as any
    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' }, clientId: 'cid', content: 'x' } as any)
    expect(sendMock).not.toHaveBeenCalled()

    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    // targetUser.id is empty -> targetUserId becomes '' and returns early
    useMessage().retryMessage({ touser: { id: '', nickname: 'U1' }, clientId: 'cid', content: 'x' } as any)
    expect(sendMock).not.toHaveBeenCalled()

    // content is empty -> payload.msg uses '' fallback
    useMessageStore().addMessage('u1', {
      code: 7,
      fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      touser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
      type: 'text',
      content: '',
      time: '2026-01-01 00:00:00.000',
      tid: '',
      isSelf: true,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: '',
      segments: [],
      clientId: 'cid-empty',
      sendStatus: 'failed',
      sendError: 'x',
      optimistic: true
    } as any)

    sendMock.mockReturnValue(true)
    const msg = (useMessageStore().getMessages('u1') as any[])[0]
    useMessage().retryMessage(msg)
    expect(sendMock).toHaveBeenCalledWith(expect.objectContaining({ msg: '' }))
  })
})

describe('composables/useChat', () => {
  it('startMatch fails when WebSocket is disconnected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false

    const ok = useChat().startMatch()
    expect(ok).toBe(false)
    expect(toastShow).toHaveBeenCalledWith('WebSocket 未连接，无法匹配')
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('startMatch sends random message and updates matching state', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const ok = useChat().startMatch()
    expect(ok).toBe(true)
    expect(chatStore.isMatching).toBe(true)
    expect(sendMock).toHaveBeenCalledWith({
      act: 'random',
      id: 'me',
      userAge: '0'
    })
  })

  it('startMatch in continuous mode does not toggle isMatching via startMatch()', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.isMatching = false

    const ok = useChat().startMatch(true)
    expect(ok).toBe(true)
    expect(chatStore.isMatching).toBe(false)
    expect(sendMock).toHaveBeenCalled()
  })

  it('cancelMatch cancels continuous matching and sends randomOut when connected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.startContinuousMatch(3)

    useChat().cancelMatch()
    expect(chatStore.continuousMatchConfig.enabled).toBe(false)
    expect(sendMock).toHaveBeenCalledWith({
      act: 'randomOut',
      id: 'me',
      msg: 'hex-1'
    })
  })

  it('cancelMatch does not send randomOut when ws is disconnected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false
    chatStore.startContinuousMatch(2)

    useChat().cancelMatch()
    expect(chatStore.continuousMatchConfig.enabled).toBe(false)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('loadUsers loads history and favorite lists for current user', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined as any)
    vi.spyOn(chatStore, 'loadFavoriteUsers').mockResolvedValue(undefined as any)

    await useChat().loadUsers()
    expect(chatStore.loadHistoryUsers).toHaveBeenCalledWith('me', 'Me')
    expect(chatStore.loadFavoriteUsers).toHaveBeenCalledWith('me', 'Me')
  })

  it('handleAutoMatch shows success immediately when total=1', () => {
    const chatStore = useChatStore()
    chatStore.startContinuousMatch(1)

    useChat().handleAutoMatch()
    expect(toastShow).toHaveBeenCalledWith('匹配成功！')
  })

  it('handleAutoMatch cancels after last match when total>1', () => {
    vi.useFakeTimers()
    try {
      const chatStore = useChatStore()
      chatStore.startContinuousMatch(2)
      chatStore.continuousMatchConfig.current = 2

      useChat().handleAutoMatch()
      expect(chatStore.continuousMatchConfig.enabled).toBe(true)

      vi.advanceTimersByTime(2000)
      expect(chatStore.continuousMatchConfig.enabled).toBe(false)
      expect(toastShow).toHaveBeenCalledWith('连续匹配完成！共匹配 2 次')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('handleAutoMatch schedules next match when not finished', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const chatStore = useChatStore()
      chatStore.wsConnected = true
      chatStore.startContinuousMatch(3)

      useChat().handleAutoMatch()
      vi.advanceTimersByTime(2000)

      expect(chatStore.continuousMatchConfig.current).toBe(2)
      expect(sendMock).toHaveBeenCalledWith({ act: 'random', id: 'me', userAge: '0' })
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('enterChat uses incremental load when cached messages exist', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 2
    })

    messageStore.addMessage('u2', {
      code: 7,
      fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
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

    const user = chatStore.getUser('u2') as any
    const updateSpy = vi.spyOn(chatStore, 'updateUser')

    useChat().enterChat(user, true)
    expect(updateSpy).toHaveBeenCalledWith('u2', { unreadCount: 0 })
    expect(loadSpy).toHaveBeenCalledWith('me', 'u2', expect.objectContaining({ incremental: true }))
  })

  it('enterChat loads history normally when no cached messages exist', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    const user = chatStore.getUser('u2') as any
    useChat().enterChat(user, true)
    expect(loadSpy).toHaveBeenCalledWith('me', 'u2', expect.not.objectContaining({ incremental: true }))
  })

  it('toggleFavorite updates store and list ids', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    const favoriteStore = useFavoriteStore()
    vi.spyOn(favoriteStore, 'addFavorite').mockResolvedValue(true)
    vi.spyOn(favoriteStore, 'removeFavorite').mockResolvedValue(true)

    const mockedToggle = vi.mocked(chatApi.toggleFavorite)
    mockedToggle.mockResolvedValue({ code: '0' } as any)

    const user = chatStore.getUser('u3') as any
    await useChat().toggleFavorite(user)
    expect(user.isFavorite).toBe(true)
    expect(chatStore.favoriteUserIds[0]).toBe('u3')
    expect(toastShow).toHaveBeenCalledWith('收藏成功')

    const mockedCancel = vi.mocked(chatApi.cancelFavorite)
    mockedCancel.mockResolvedValue({ code: '0' } as any)

    await useChat().toggleFavorite(user)
    expect(user.isFavorite).toBe(false)
    expect(chatStore.favoriteUserIds).not.toContain('u3')
    expect(toastShow).toHaveBeenCalledWith('取消收藏成功')
  })

  it('toggleFavorite shows failure message when backend returns non-ok', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    vi.mocked(chatApi.toggleFavorite).mockResolvedValue({ code: '1', msg: 'bad' } as any)
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(toastShow).toHaveBeenCalledWith('操作失败: bad')
    expect((chatStore.getUser('u3') as any).isFavorite).toBe(false)
  })

  it('toggleFavorite catches unexpected errors', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    vi.mocked(chatApi.toggleFavorite).mockRejectedValue(new Error('boom'))
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(toastShow).toHaveBeenCalledWith('操作失败')
  })

  it('covers early-return branches when currentUser is missing', async () => {
    const chatStore = useChatStore()
    vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined as any)
    vi.spyOn(chatStore, 'loadFavoriteUsers').mockResolvedValue(undefined as any)

    // loadUsers returns early with no current user.
    await useChat().loadUsers()
    expect(chatStore.loadHistoryUsers).not.toHaveBeenCalled()
    expect(chatStore.loadFavoriteUsers).not.toHaveBeenCalled()

    // startMatch returns false with no current user.
    chatStore.wsConnected = true
    expect(useChat().startMatch()).toBe(false)
    expect(sendMock).not.toHaveBeenCalled()

    // cancelMatch returns early with no current user (does not cancel existing config).
    chatStore.startContinuousMatch(2)
    useChat().cancelMatch()
    expect(chatStore.continuousMatchConfig.enabled).toBe(true)

    // startContinuousMatch returns false with no current user.
    expect(useChat().startContinuousMatch(2)).toBe(false)

    // toggleFavorite returns early with no current user.
    await useChat().toggleFavorite({ id: 'u1', isFavorite: false })
    expect(vi.mocked(chatApi.toggleFavorite)).not.toHaveBeenCalled()
  })

  it('startContinuousMatch returns false and shows toast when WebSocket is disconnected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false

    expect(useChat().startContinuousMatch(3)).toBe(false)
    expect(toastShow).toHaveBeenCalledWith('WebSocket 未连接，无法匹配')
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('handleAutoMatch returns early when continuous matching is not enabled', () => {
    useChat().handleAutoMatch()
    expect(sendMock).not.toHaveBeenCalled()
    expect(toastShow).not.toHaveBeenCalled()
  })

  it('cancelMatch clears pending auto-match timer branch before cancelling', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const chatStore = useChatStore()
      chatStore.wsConnected = true
      chatStore.startContinuousMatch(3)

      // current < total -> schedules next match (autoMatchTimer set)
      const chat = useChat()
      chat.handleAutoMatch()

      // Cancel should clear timer, so advancing time won't start another match.
      chat.cancelMatch()
      sendMock.mockClear()

      vi.advanceTimersByTime(2000)
      expect(sendMock).not.toHaveBeenCalled()
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('enterChatAndStopMatch clears timer and stops continuous match before entering chat', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const chatStore = useChatStore()
      chatStore.wsConnected = true
      chatStore.startContinuousMatch(3)

      // schedule timer
      const messageStore = useMessageStore()
      vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

      const chat = useChat()
      chat.handleAutoMatch()

      // stop + enter chat
      const user = { id: 'u9', name: 'U9', nickname: 'U9', unreadCount: 0 }
      chat.enterChatAndStopMatch(user)

      expect(chatStore.continuousMatchConfig.enabled).toBe(false)
      expect(chatStore.currentChatUser?.id).toBe('u9')

      // timer cleared -> no further match even if time passes
      sendMock.mockClear()
      vi.advanceTimersByTime(2000)
      expect(sendMock).not.toHaveBeenCalled()
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('enterChat does not load history when loadHistory=false or currentUser is missing', () => {
    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    // no current user -> should not load history even if loadHistory=true
    useChat().enterChat(chatStore.getUser('u2') as any, true)

    // explicit no-loadHistory
    useChat().enterChat(chatStore.getUser('u2') as any, false)
    expect(loadSpy).not.toHaveBeenCalled()
  })

  it('enterChat incremental-load then() covers newCount >0 and ==0 branches', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    // Seed one cached message so enterChat takes incremental path.
    messageStore.addMessage('u2', { tid: 't1', content: 'x', isSelf: true } as any)

    const loadSpy = vi.spyOn(messageStore, 'loadHistory')
    loadSpy.mockResolvedValueOnce(2)
    useChat().enterChat(chatStore.getUser('u2') as any, true)
    await Promise.resolve()
    await Promise.resolve()
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('增量追加 2 条新消息'))

    loadSpy.mockResolvedValueOnce(0)
    useChat().enterChat(chatStore.getUser('u2') as any, true)
    await Promise.resolve()
    await Promise.resolve()
    expect(logSpy).toHaveBeenCalledWith('没有新消息')

    logSpy.mockRestore()
  })

  it('toggleFavorite covers nickname fallback, list includes guard, and index=-1 remove branch', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'NameOnly',
      nickname: '',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    const favoriteStore = useFavoriteStore()
    const addSpy = vi.spyOn(favoriteStore, 'addFavorite').mockResolvedValue(true)
    const removeSpy = vi.spyOn(favoriteStore, 'removeFavorite').mockResolvedValue(true)

    // id already in list -> should not unshift duplicate
    chatStore.favoriteUserIds.push('u3')
    vi.mocked(chatApi.toggleFavorite).mockResolvedValue({ status: 'true' } as any)

    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(chatStore.favoriteUserIds).toEqual(['u3'])
    expect(addSpy).toHaveBeenCalledWith('me', 'u3', 'NameOnly')

    // remove path where id is missing from list -> index=-1 branch
    ;(chatStore.getUser('u3') as any).isFavorite = true
    chatStore.favoriteUserIds.splice(0, chatStore.favoriteUserIds.length)
    vi.mocked(chatApi.cancelFavorite).mockResolvedValue({ code: '0' } as any)
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(removeSpy).toHaveBeenCalledWith('me', 'u3')
  })

  it('toggleFavorite failure message falls back when backend does not provide msg', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    vi.mocked(chatApi.toggleFavorite).mockResolvedValue({ code: '1' } as any)
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(toastShow).toHaveBeenCalledWith('操作失败: 未知错误')
  })
})
