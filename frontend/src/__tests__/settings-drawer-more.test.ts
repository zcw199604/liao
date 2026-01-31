import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()

const wsMocks = {
  disconnect: vi.fn(),
  connect: vi.fn(),
  send: vi.fn()
}

const settingsMocks = {
  connectionStats: { value: {} },
  forceoutUserCount: { value: 0 },
  disconnectAllLoading: { value: false },
  loadConnectionStats: vi.fn().mockResolvedValue(undefined),
  loadForceoutUserCount: vi.fn().mockResolvedValue(undefined),
  disconnectAll: vi.fn().mockResolvedValue(true),
  clearForceout: vi.fn().mockResolvedValue({ success: true, message: 'ok' })
}

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => wsMocks
}))

vi.mock('@/composables/useSettings', () => ({
  useSettings: () => settingsMocks
}))

vi.mock('@/api/identity', () => ({
  updateIdentityId: vi.fn(),
  updateIdentity: vi.fn()
}))

import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import * as identityApi from '@/api/identity'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

const mountDrawer = (opts: { mode: any; visible?: boolean; beforeMount?: () => void }) => {
  const pinia = createPinia()
  setActivePinia(pinia)
  opts.beforeMount?.()

  return mount(SettingsDrawer, {
    props: { visible: opts.visible ?? true, mode: opts.mode },
    global: {
      plugins: [pinia],
      stubs: {
        teleport: true,
        Dialog: true,
        SystemSettings: true,
        GlobalFavorites: true,
        DuplicateCheckModal: true
      }
    }
  })
}

describe('components/settings/SettingsDrawer.vue (more branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders header title branches for each mode', async () => {
    const setUser = () => {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
    }

    const w1 = mountDrawer({ mode: 'identity', beforeMount: setUser })
    expect(w1.text()).toContain('身份信息')

    const w2 = mountDrawer({ mode: 'system', beforeMount: setUser })
    await flushAsync()
    expect(w2.text()).toContain('系统设置')

    const w3 = mountDrawer({ mode: 'media', beforeMount: setUser })
    await flushAsync()
    expect(w3.text()).toContain('图片管理')

    const w4 = mountDrawer({ mode: 'favorites', beforeMount: setUser })
    expect(w4.text()).toContain('全局收藏')
  })

  it('watch(visible) refreshes system stats on system mode and loads media list on media mode', async () => {
    // system mode triggers refreshSystem()
    const wSystem = mountDrawer({
      mode: 'system',
      visible: false,
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await wSystem.setProps({ visible: true })
    await flushAsync()
    expect(settingsMocks.loadConnectionStats).toHaveBeenCalled()
    expect(settingsMocks.loadForceoutUserCount).toHaveBeenCalled()

    // media mode triggers mediaStore.loadAllUploadImages(1)
    const wMedia = mountDrawer({
      mode: 'media',
      visible: false,
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const mediaStore = useMediaStore()
        vi.spyOn(mediaStore, 'loadAllUploadImages').mockResolvedValue(undefined as any)
      }
    })
    await wMedia.setProps({ visible: true })
    await flushAsync()
    expect(useMediaStore().loadAllUploadImages).toHaveBeenCalledWith(1)
  })

  it('saveUserInfo: idChanged success updates user, reconnects, and closes drawer', async () => {
    vi.mocked(identityApi.updateIdentityId).mockResolvedValueOnce({ code: 0, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const chatStore = useChatStore()
        vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined)
        vi.spyOn(chatStore, 'loadFavoriteUsers').mockResolvedValue(undefined)

        const messageStore = useMessageStore()
        vi.spyOn(messageStore, 'resetAll')
      }
    })
    await flushAsync()

    const editBtn = wrapper.findAll('button').find((b) => b.text().includes('编辑'))
    expect(editBtn).toBeTruthy()
    await editBtn!.trigger('click')
    await flushAsync()

    await wrapper.get('input[placeholder="输入用户ID"]').setValue('me2')

    const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    await flushAsync()
    await flushAsync()

    expect(identityApi.updateIdentityId).toHaveBeenCalledWith(
      expect.objectContaining({ oldId: 'me', newId: 'me2', name: 'Me', sex: '男' })
    )
    expect(wsMocks.disconnect).toHaveBeenCalledWith(true)
    expect(useMessageStore().resetAll).toHaveBeenCalled()
    expect(wsMocks.connect).toHaveBeenCalled()
    expect(useChatStore().loadHistoryUsers).toHaveBeenCalledWith('me2', 'Me')
    expect(useChatStore().loadFavoriteUsers).toHaveBeenCalledWith('me2', 'Me')
    expect(toastShow).toHaveBeenCalledWith('用户ID已更新，正在重新连接...')
    expect(wrapper.emitted('update:visible')?.some((e) => e[0] === false)).toBe(true)
    expect(useUserStore().currentUser?.id).toBe('me2')
  })

  it('saveUserInfo: idChanged failure shows error and does not reconnect', async () => {
    vi.mocked(identityApi.updateIdentityId).mockResolvedValueOnce({ code: 1, msg: 'bad' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()

    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()
    await wrapper.get('input[placeholder="输入用户ID"]').setValue('me2')

    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('更新失败: bad')
    expect(wsMocks.disconnect).not.toHaveBeenCalled()
    expect(wsMocks.connect).not.toHaveBeenCalled()
  })

  it('saveUserInfo: nameChanged sends chgname when ws is connected', async () => {
    vi.mocked(identityApi.updateIdentity).mockResolvedValueOnce({ code: 0, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const chatStore = useChatStore()
        chatStore.wsConnected = true
      }
    })
    await flushAsync()
    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()

    await wrapper.get('input[placeholder="输入用户名"]').setValue('Me2')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(identityApi.updateIdentity).toHaveBeenCalledWith(expect.objectContaining({ id: 'me', name: 'Me2', sex: '男' }))
    expect(wsMocks.send).toHaveBeenCalledWith(expect.objectContaining({ act: 'chgname', id: 'me', msg: 'Me2' }))
    // sex did not change -> modinfo should not be sent
    expect(wsMocks.send).not.toHaveBeenCalledWith(expect.objectContaining({ act: 'modinfo' }))
    expect(toastShow).toHaveBeenCalledWith('信息已保存')
    expect(useUserStore().currentUser?.name).toBe('Me2')
  })

  it('saveUserInfo: sexChanged sends modinfo when ws is connected', async () => {
    vi.mocked(identityApi.updateIdentity).mockResolvedValueOnce({ code: 0, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const chatStore = useChatStore()
        chatStore.wsConnected = true
      }
    })
    await flushAsync()
    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()

    await wrapper.get('select').setValue('女')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(wsMocks.send).toHaveBeenCalledWith(expect.objectContaining({ act: 'modinfo', id: 'me', userSex: '女' }))
    expect(wsMocks.send).not.toHaveBeenCalledWith(expect.objectContaining({ act: 'chgname' }))
    expect(useUserStore().currentUser?.sex).toBe('女')
  })

  it('saveUserInfo: ws disconnected path shows warning and does not send', async () => {
    vi.mocked(identityApi.updateIdentity).mockResolvedValueOnce({ code: 0, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const chatStore = useChatStore()
        chatStore.wsConnected = false
      }
    })
    await flushAsync()
    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()

    await wrapper.get('input[placeholder="输入用户名"]').setValue('Me2')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('数据库已更新，但WebSocket未连接，无法同步聊天状态')
    expect(wsMocks.send).not.toHaveBeenCalled()
  })

  it('saveUserInfo: catches errors and resets saving flag', async () => {
    vi.mocked(identityApi.updateIdentity).mockRejectedValueOnce(new Error('boom'))

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()
    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()

    await wrapper.get('input[placeholder="输入用户名"]').setValue('Me2')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('保存失败: boom')
    expect((wrapper.vm as any).saving).toBe(false)
  })

  it('startEdit/saveUserInfo return early when currentUser is missing', async () => {
    const wrapper = mountDrawer({ mode: 'identity' })
    await flushAsync()

    // startEdit early return: editMode stays false
    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()
    expect((wrapper.vm as any).editMode).toBe(false)

    // saveUserInfo early return should be safe (no throw)
    await (wrapper.vm as any).saveUserInfo()
    expect((wrapper.vm as any).saving).toBe(false)
  })

  it('saveUserInfo: no changes branch shows toast and exits editMode', async () => {
    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()

    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()

    // No change -> click save
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('没有任何修改')
    expect((wrapper.vm as any).editMode).toBe(false)
  })

  it('saveUserInfo: idChanged failure uses default msg when res.msg is empty', async () => {
    vi.mocked(identityApi.updateIdentityId).mockResolvedValueOnce({ code: 1, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()

    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()
    await wrapper.get('input[placeholder=\"输入用户ID\"]').setValue('me2')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('更新失败: 未知错误')
    expect(wsMocks.disconnect).not.toHaveBeenCalled()
  })

  it('saveUserInfo: updateIdentity failure uses default msg when res.msg is empty', async () => {
    vi.mocked(identityApi.updateIdentity).mockResolvedValueOnce({ code: 1, msg: '' } as any)

    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any

        const chatStore = useChatStore()
        chatStore.wsConnected = true
      }
    })
    await flushAsync()

    await wrapper.findAll('button').find((b) => b.text().includes('编辑'))!.trigger('click')
    await flushAsync()
    await wrapper.get('input[placeholder=\"输入用户名\"]').setValue('Me2')
    await wrapper.findAll('button').find((b) => b.text().includes('保存'))!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('更新失败: 未知错误')
    expect(wsMocks.send).not.toHaveBeenCalled()
  })

  it('covers doDisconnectAll branches and watch(!visible) early return', async () => {
    const wrapper = mountDrawer({
      mode: 'identity',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: '', nickname: '', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()

    // currentUser.name is empty -> avatar falls back to '?'
    expect(wrapper.text()).toContain('?')

    // confirmDisconnectAll toggles dialog flag
    ;(wrapper.vm as any).confirmDisconnectAll()
    expect((wrapper.vm as any).showDisconnectAllDialog).toBe(true)

    settingsMocks.disconnectAll.mockResolvedValueOnce(true)
    await (wrapper.vm as any).doDisconnectAll()
    expect(toastShow).toHaveBeenCalledWith('已断开所有连接')

    settingsMocks.disconnectAll.mockResolvedValueOnce(false)
    await (wrapper.vm as any).doDisconnectAll()
    expect(toastShow).toHaveBeenCalledWith('操作失败')

    // confirmClearForceout + doClearForceout shows message
    ;(wrapper.vm as any).confirmClearForceout()
    expect((wrapper.vm as any).showClearForceoutDialog).toBe(true)
    settingsMocks.clearForceout.mockResolvedValueOnce({ success: false, message: 'no' })
    await (wrapper.vm as any).doClearForceout()
    expect(toastShow).toHaveBeenCalledWith('no')

    // Toggle visible false to trigger watch(!v) early return
    await wrapper.setProps({ visible: false })
    await flushAsync()
  })

  it('renders GlobalFavorites when mode=favorites', async () => {
    const wrapper = mountDrawer({
      mode: 'favorites',
      beforeMount: () => {
        const userStore = useUserStore()
        userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '', cookie: 'c' } as any
      }
    })
    await flushAsync()
    // stub GlobalFavorites should exist in DOM
    expect(wrapper.html()).toContain('global-favorites-stub')
  })
})
