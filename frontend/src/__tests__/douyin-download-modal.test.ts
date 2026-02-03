import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('vuedraggable', () => ({
  default: {
    name: 'draggable',
    props: ['modelValue', 'itemKey', 'disabled', 'handle'],
    emits: ['update:modelValue', 'end'],
    template: `<div data-testid="draggable">
      <div v-for="(el, idx) in modelValue" :key="itemKey ? el[itemKey] : idx">
        <slot name="item" :element="el" :index="idx"></slot>
      </div>
    </div>`
  }
}))

vi.mock('@/api/douyin', async () => {
  const actual = await vi.importActual<any>('@/api/douyin')
  return {
    ...actual,
	    listDouyinFavoriteUsers: vi.fn().mockResolvedValue({ items: [] }),
	    listDouyinFavoriteAwemes: vi.fn().mockResolvedValue({ items: [] }),
	    listDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ items: [], cursor: 0, hasMore: false }),
	    listDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ items: [] }),
	    listDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ items: [] }),
	    upsertDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ success: true, added: 0 }),
	    pullLatestDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ success: true, added: 0, fetched: 0 }),
	    getDouyinAccount: vi.fn().mockResolvedValue({ secUserId: '', tab: 'post', cursor: 0, hasMore: false, items: [] }),
	    addDouyinFavoriteUser: vi.fn().mockResolvedValue({}),
	    addDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
	    updateDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
	    removeDouyinFavoriteUserTag: vi.fn().mockResolvedValue({ success: true }),
	    applyDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ success: true }),
    reorderDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ success: true }),
    addDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
    updateDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
    removeDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({ success: true }),
    applyDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ success: true }),
    reorderDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ success: true }),
    getDouyinDetail: vi.fn(),
    importDouyinMedia: vi.fn(),
    addDouyinFavoriteAweme: vi.fn().mockResolvedValue({})
  }
})

import DouyinDownloadModal from '@/components/media/DouyinDownloadModal.vue'
import * as douyinApi from '@/api/douyin'
import { useDouyinStore } from '@/stores/douyin'

const flushAsync = async () => {
  await new Promise<void>((resolve) => setTimeout(resolve, 0))
  await nextTick()
}

const defineClipboard = (readText: any) => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      readText
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/media/DouyinDownloadModal.vue', () => {
  it('pastes any non-empty clipboard text (no douyin filter)', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const readText = vi.fn().mockResolvedValue('not douyin')
    defineClipboard(readText)

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await nextTick()

    const pasteBtn = wrapper.findAll('button').find((btn) => btn.text().includes('粘贴'))
    expect(pasteBtn).toBeTruthy()

    await pasteBtn!.trigger('click')
    await flushAsync()

    expect(readText).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('not douyin')
    expect(toastShow).toHaveBeenCalledWith('已从剪贴板填充')
  })

  it('shows a hint when clipboard readText is not supported', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {}
    })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await nextTick()

    const pasteBtn = wrapper.findAll('button').find((btn) => btn.text().includes('粘贴'))
    expect(pasteBtn).toBeTruthy()

    await pasteBtn!.trigger('click')
    await flushAsync()

    expect(toastShow).toHaveBeenCalledWith('当前浏览器不支持一键读取剪贴板，请长按输入框手动粘贴')
  })

  it('routes manual paste to account mode when clipboard text looks like a user input', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await nextTick()

    const detailTextarea = wrapper.get('textarea')
    await detailTextarea.trigger('paste', {
      clipboardData: {
        getData: (kind: string) => (kind === 'text/plain' ? 'MS4wLjABuser' : '')
      }
    })
    await flushAsync()

    const activeTextarea = wrapper.get('textarea')
    expect(activeTextarea.attributes('placeholder')).toBe('粘贴抖音用户主页链接/分享文本/sec_uid')
  })

  it('auto-reads clipboard on open and fills any non-empty text', async () => {
    localStorage.setItem('douyin_auto_clipboard', '1')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const readText = vi.fn().mockResolvedValue('not douyin')
    defineClipboard(readText)

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    expect(readText).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('not douyin')
    expect(toastShow).toHaveBeenCalledWith('已从剪贴板读取')
  })

  it('filters favorite users by tag chips', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({
      items: [
        { secUserId: 'u1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [1] },
        { secUserId: 'u2', displayName: 'Bob', createTime: 't', updateTime: 't', tagIds: [] }
      ]
    })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '收藏')!.trigger('click')
    await flushAsync()

    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('Bob')

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '未分类')!.trigger('click')
    await flushAsync()
    expect(wrapper.text()).not.toContain('Alice')
    expect(wrapper.text()).toContain('Bob')

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '美食')!.trigger('click')
    await flushAsync()
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).not.toContain('Bob')

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '全部')!.trigger('click')
    await flushAsync()
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('Bob')
  })

  it('manages favorite user tags via tag manager', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 2, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '收藏')!.trigger('click')
    await flushAsync()

    await wrapper.get('button[title="管理标签"]').trigger('click')
    await flushAsync()

    await wrapper.get('input[placeholder="新建标签名称（仅名称）"]').setValue('新标签')
    await wrapper.findAll('button').find((btn) => btn.text().trim() === '创建')!.trigger('click')
    await flushAsync()
    expect(douyinApi.addDouyinFavoriteUserTag).toHaveBeenCalledWith({ name: '新标签' })
    expect(toastShow).toHaveBeenCalledWith('已创建标签')

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '重命名')!.trigger('click')
    await flushAsync()
    await wrapper.get('input[placeholder="标签名称"]').setValue('美食2')
    await wrapper.findAll('button').find((btn) => btn.text().trim() === '保存')!.trigger('click')
    await flushAsync()
    expect(douyinApi.updateDouyinFavoriteUserTag).toHaveBeenCalledWith({ id: 1, name: '美食2' })
    expect(toastShow).toHaveBeenCalledWith('已更新标签')

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '删除')!.trigger('click')
    await flushAsync()
    await wrapper.findAll('button').find((btn) => btn.text().trim() === '确认删除')!.trigger('click')
    await flushAsync()
    expect(douyinApi.removeDouyinFavoriteUserTag).toHaveBeenCalledWith({ id: 1 })
    expect(toastShow).toHaveBeenCalledWith('已删除标签')
  })

  it('applies favorite user tags in single mode (set)', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({
      items: [{ secUserId: 'u1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [1] }]
    })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })
    ;(douyinApi.applyDouyinFavoriteUserTags as any).mockResolvedValue({ success: true })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '收藏')!.trigger('click')
    await flushAsync()

    await wrapper.findAll('[role="button"]').find((el) => el.text().includes('Alice'))!.trigger('click')
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '编辑')!.trigger('click')
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '保存')!.trigger('click')
    await flushAsync()

    expect(douyinApi.applyDouyinFavoriteUserTags).toHaveBeenCalledWith({
      secUserIds: ['u1'],
      tagIds: [1],
      mode: 'set'
    })
    expect(toastShow).toHaveBeenCalledWith('已更新标签')
  })

  it('applies favorite user tags in batch mode (add)', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({
      items: [{ secUserId: 'u1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [] }]
    })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })
    ;(douyinApi.applyDouyinFavoriteUserTags as any).mockResolvedValue({ success: true })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '收藏')!.trigger('click')
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '批量')!.trigger('click')
    await flushAsync()

    await wrapper.findAll('[role="button"]').find((el) => el.text().includes('Alice'))!.trigger('click')
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '打标签')!.trigger('click')
    await flushAsync()

    const tagButtons = wrapper.findAll('button').filter((btn) => btn.text().trim() === '美食')
    expect(tagButtons.length).toBeGreaterThan(0)
    await tagButtons[tagButtons.length - 1]!.trigger('click')
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '添加')!.trigger('click')
    await flushAsync()

    expect(douyinApi.applyDouyinFavoriteUserTags).toHaveBeenCalledWith({
      secUserIds: ['u1'],
      tagIds: [1],
      mode: 'add'
    })
    expect(toastShow).toHaveBeenCalledWith('已批量添加标签')
  })

  it('reorders favorite user tags and persists order', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const tags = [
      { id: 1, name: '美食', count: 0, createTime: 't', updateTime: 't' },
      { id: 2, name: '教程', count: 0, createTime: 't', updateTime: 't' }
    ]
    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({ items: tags })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })
    ;(douyinApi.reorderDouyinFavoriteUserTags as any).mockResolvedValue({ success: true })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: true,
          MediaTile: true,
          MediaTileBadge: true,
          MediaTileSelectMark: true
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    await wrapper.findAll('button').find((btn) => btn.text().trim() === '收藏')!.trigger('click')
    await flushAsync()

    await wrapper.get('button[title="管理标签"]').trigger('click')
    await flushAsync()

    const drag = wrapper.findComponent({ name: 'draggable' })
    expect(drag.exists()).toBe(true)
    drag.vm.$emit('update:modelValue', [tags[1], tags[0]])
    await flushAsync()
    await (wrapper.vm as any).saveTagManagerOrder()

    expect(douyinApi.reorderDouyinFavoriteUserTags).toHaveBeenCalledWith({ tagIds: [2, 1] })
    expect(toastShow).toHaveBeenCalledWith('已更新顺序')
  })

  it('handleResolve shows hint for empty input and switches to account mode for user input', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: { teleport: true, MediaPreview: true, MediaTile: true, MediaTileBadge: true, MediaTileSelectMark: true }
      }
    })

    const vm = wrapper.vm as any
    vm.inputText = ''
    await vm.handleResolve()
    expect(toastShow).toHaveBeenCalledWith('请输入抖音分享文本/链接/作品ID')

    vm.inputText = 'MS4wLjABuser'
    await vm.handleResolve()
    expect(toastShow).toHaveBeenCalledWith('识别到用户主页链接，请切换到“用户作品”')
    expect(vm.activeMode).toBe('account')
    expect(vm.accountInput).toBe('MS4wLjABuser')
  })

  it('handleResolve populates detail and updates favorite aweme when already favorited', async () => {
    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: (k: string) => (k === 'Content-Length' ? '2048' : k === 'Content-Type' ? 'image/jpeg' : '') }
    } as any)
    ;(globalThis as any).fetch = fetchMock

    localStorage.setItem('authToken', 't')
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: { teleport: true, MediaPreview: true, MediaTile: true, MediaTileBadge: true, MediaTileSelectMark: true }
      }
    })

    const vm = wrapper.vm as any
    // mark as already favorited
    vm.favoriteAwemes = [{ awemeId: 'a1', createTime: 't', updateTime: 't' }]

    ;(douyinApi.getDouyinDetail as any).mockResolvedValue({
      key: 'k1',
      detailId: 'a1',
      type: '图集',
      title: 'T',
      items: [
        { index: 0, type: 'image', url: 'u', downloadUrl: '/api/douyin/download?key=k1&index=0', originalFilename: 'a.jpg' }
      ]
    })
    ;(douyinApi.addDouyinFavoriteAweme as any).mockResolvedValue({ awemeId: 'a1', createTime: 't', updateTime: 't' })

    vm.inputText = 'a1'
    await vm.handleResolve()
    expect(vm.detail?.key).toBe('k1')
    expect(douyinApi.addDouyinFavoriteAweme).toHaveBeenCalled()

    // run prefetch explicitly for determinism
    await vm.prefetchMetas('k1', vm.detail.items)
    expect(fetchMock).toHaveBeenCalled()
    expect(vm.getItemMeta('k1', 0)?.size).toBe(2048)

    localStorage.removeItem('authToken')
    ;(globalThis as any).fetch = originalFetch
  })

  it('handleResolve highlights cookie config when cookie-related error happens', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: { teleport: true, MediaPreview: true, MediaTile: true, MediaTileBadge: true, MediaTileSelectMark: true }
      }
    })

    const vm = wrapper.vm as any
    ;(douyinApi.getDouyinDetail as any).mockRejectedValue(new Error('cookie invalid'))
    vm.inputText = 'a1'
    await vm.handleResolve()
    expect(vm.showAdvanced).toBe(true)
    expect(vm.highlightConfig).toBe(true)
    expect(String(vm.cookieHint || '')).toContain('Cookie')
  })

  it('syncFavoriteUserWorksFromAccount keeps pinnedRank=0 when favoriting user from account mode', async () => {
    localStorage.setItem('douyin_auto_clipboard', '0')
    localStorage.setItem('douyin_auto_resolve_clipboard', '0')

    ;(douyinApi.getDouyinAccount as any).mockResolvedValue({
      secUserId: 'u1',
      tab: 'post',
      cursor: 0,
      hasMore: false,
      items: [
        {
          detailId: 'a1',
          type: 'video',
          desc: '置顶作品',
          coverUrl: 'https://example.com/c1.jpg',
          isPinned: true,
          pinnedRank: 0,
          pinnedAt: '2026-01-01T00:00:00',
          publishAt: '2026-01-01T00:00:00',
          status: 'normal',
          authorUniqueId: 'dy1',
          authorName: '作者A',
          key: 'k1',
          items: [{ index: 0, type: 'video', url: 'https://example.com/v1.mp4', downloadUrl: 'https://example.com/v1.mp4' }]
        }
      ]
    })
    ;(douyinApi.addDouyinFavoriteUser as any).mockResolvedValue({ secUserId: 'u1', tagIds: [] })

    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: { teleport: true, MediaPreview: true, MediaTile: true, MediaTileBadge: true, MediaTileSelectMark: true }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await nextTick()

    const accountBtn = wrapper.findAll('button').find((btn) => btn.text().includes('用户作品'))
    expect(accountBtn).toBeTruthy()
    await accountBtn!.trigger('click')
    await nextTick()

    await wrapper.get('textarea').setValue('u1')

    const fetchBtn = wrapper.findAll('button').find((btn) => btn.text().includes('获取作品'))
    expect(fetchBtn).toBeTruthy()
    await fetchBtn!.trigger('click')
    await flushAsync()

    const favBtn = wrapper.findAll('button').find((btn) => btn.text().includes('收藏用户'))
    expect(favBtn).toBeTruthy()
    await favBtn!.trigger('click')
    await flushAsync()

    expect(douyinApi.upsertDouyinFavoriteUserAwemes).toHaveBeenCalledTimes(1)
    const payload = (douyinApi.upsertDouyinFavoriteUserAwemes as any).mock.calls[0][0]
    expect(payload?.secUserId).toBe('u1')
    expect(payload?.items?.[0]?.isPinned).toBe(true)
    expect(payload?.items?.[0]?.pinnedRank).toBe(0)
  })

  it('batch import and download update counters and show summary', async () => {
    vi.useFakeTimers()
    const originalFetch = (globalThis as any).fetch
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: (k: string) => (k === 'Content-Disposition' ? "attachment; filename*=UTF-8''a.jpg" : '') },
        blob: async () => new Blob(['x'], { type: 'image/jpeg' })
      } as any)
      .mockResolvedValueOnce({
        ok: false,
        status: 403,
        json: async () => ({ error: 'forbidden' })
      } as any)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => '' },
        blob: async () => new Blob(['x'], { type: 'image/jpeg' })
      } as any)
    ;(globalThis as any).fetch = fetchMock

    const originalCreateObjectURL = (URL as any).createObjectURL
    const originalRevokeObjectURL = (URL as any).revokeObjectURL
    if (!(URL as any).createObjectURL) {
      Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn().mockReturnValue('blob:mock') })
    }
    if (!(URL as any).revokeObjectURL) {
      Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })
    }
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    localStorage.setItem('authToken', 't')

    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: { teleport: true, MediaPreview: true, MediaTile: true, MediaTileBadge: true, MediaTileSelectMark: true }
      }
    })

    const vm = wrapper.vm as any
    vm.detail = {
      key: 'k1',
      detailId: 'a1',
      type: '图集',
      title: 'T',
      items: [
        { index: 0, type: 'image', url: 'u0', downloadUrl: '/api/douyin/download?key=k1&index=0', originalFilename: 'a.jpg' },
        { index: 1, type: 'image', url: 'u1', downloadUrl: '/api/douyin/download?key=k1&index=1', originalFilename: 'b.jpg' },
        { index: 2, type: 'image', url: 'u2', downloadUrl: '/api/douyin/download?key=k1&index=2', originalFilename: 'c.jpg' }
      ]
    }

    ;(douyinApi.importDouyinMedia as any)
      .mockResolvedValueOnce({ state: 'OK', localPath: '/tmp/a.jpg', localFilename: 'a.jpg', dedup: false })
      .mockResolvedValueOnce({ state: 'OK', localPath: '/tmp/b.jpg', localFilename: 'b.jpg', dedup: true })
      .mockResolvedValueOnce({ state: 'FAIL', error: 'x' })

    vm.selectionMode = true
    vm.selectedIndices = new Set([0, 1])
    await vm.handleBatchImport()
    expect(vm.batchImport.done).toBe(2)
    expect(vm.batchImport.success).toBe(2)

    vm.selectionMode = false
    vm.selectedIndices = new Set()
    await vm.handleBatchDownload()
    expect(confirmSpy).toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()

    localStorage.removeItem('authToken')
    confirmSpy.mockRestore()
    clickSpy.mockRestore()
    ;(globalThis as any).fetch = originalFetch
    ;(URL as any).createObjectURL = originalCreateObjectURL
    ;(URL as any).revokeObjectURL = originalRevokeObjectURL
    vi.useRealTimers()
  })
})
