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

vi.mock('@/api/douyin', () => ({
  // list
  listDouyinFavoriteUsers: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteAwemes: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ items: [], cursor: 0, hasMore: false }),
  listDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ items: [] }),
  // tags
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
  // favorites
  addDouyinFavoriteUser: vi.fn(),
  removeDouyinFavoriteUser: vi.fn().mockResolvedValue({ success: true }),
  upsertDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ success: true }),
  pullLatestDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ success: true }),
  addDouyinFavoriteAweme: vi.fn(),
  removeDouyinFavoriteAweme: vi.fn().mockResolvedValue({ success: true }),
  // detail/account
  getDouyinDetail: vi.fn(),
  getDouyinAccount: vi.fn(),
  importDouyinMedia: vi.fn()
}))

import DouyinDownloadModal from '@/components/media/DouyinDownloadModal.vue'
import * as douyinApi from '@/api/douyin'
import { useDouyinStore } from '@/stores/douyin'
import { useUserStore } from '@/stores/user'
import { useMediaStore } from '@/stores/media'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type', 'muted', 'showSkeleton', 'indicatorSize'],
  template: `<div data-testid="media-tile">
    <slot name="top-left"></slot>
    <slot name="top-right"></slot>
    <slot name="bottom-left"></slot>
    <slot name="bottom-right"></slot>
    <slot name="file"></slot>
    <slot></slot>
  </div>`
}

const MediaTileBadgeStub = {
  name: 'MediaTileBadge',
  template: `<div class="badge"><slot /></div>`
}

const MediaTileSelectMarkStub = {
  name: 'MediaTileSelectMark',
  props: ['checked'],
  template: `<div class="select-mark">{{ checked ? 'checked' : 'unchecked' }}</div>`
}

const MediaPreviewStub = {
  name: 'MediaPreview',
  props: ['visible', 'url', 'type', 'canUpload', 'uploadDisabled', 'uploadLoading', 'uploadText', 'mediaList'],
  emits: ['update:visible', 'upload', 'media-change'],
  template: `<div data-testid="media-preview"></div>`
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())

  localStorage.setItem('douyin_auto_clipboard', '0')
  localStorage.setItem('douyin_auto_resolve_clipboard', '0')
})

describe('components/media/DouyinDownloadModal.vue (modes)', () => {
  it('detail mode covers selection UI, item badges/metas, upload menu and favorite toggle', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const mediaStore = useMediaStore()
    const openUploadSpy = vi.spyOn(mediaStore, 'requestOpenUploadMenu')

    ;(douyinApi.addDouyinFavoriteAweme as any).mockResolvedValue({ awemeId: 'a1', createTime: 't', updateTime: 't' })
    ;(douyinApi.removeDouyinFavoriteAweme as any).mockResolvedValue({ success: true })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const userStore = useUserStore()
    userStore.currentUser = null

    const vm = wrapper.vm as any
    vm.activeMode = 'detail'
    vm.detail = {
      key: 'k1',
      detailId: 'a1',
      type: '图集',
      title: 'T',
      items: [
        { index: 0, type: 'image', url: 'u0', downloadUrl: '/api/douyin/download?key=k1&index=0', originalFilename: 'a.jpg' },
        { index: 1, type: 'image', url: 'u1', downloadUrl: '/api/douyin/download?key=k1&index=1', originalFilename: 'b.jpg' },
        { index: 2, type: 'image', url: 'u2', downloadUrl: '/api/douyin/download?key=k1&index=2', originalFilename: 'c.jpg' },
        { index: 3, type: 'image', url: 'u3', downloadUrl: '/api/douyin/download?key=k1&index=3', originalFilename: 'd.jpg' }
      ]
    }

    vm.itemStateById['k1:0'] = { status: 'importing' }
    vm.itemStateById['k1:1'] = { status: 'imported' }
    vm.itemStateById['k1:2'] = { status: 'exists' }
    vm.itemStateById['k1:3'] = { status: 'error', message: 'x' }
    vm.itemMetaById['k1:3'] = { size: 2048, mime: 'image/jpeg' }

    await flushAsync()

    expect(wrapper.text()).toContain('导入中')
    expect(wrapper.text()).toContain('已导入')
    expect(wrapper.text()).toContain('已存在')
    expect(wrapper.text()).toContain('失败')

    // selection mode toggles and affects click behavior
    expect(wrapper.text()).toContain('多选模式')
    await vm.toggleSelectionMode()
    await flushAsync()
    expect(wrapper.text()).toContain('退出多选')

    vm.handleItemClick(0)
    await flushAsync()
    expect(vm.selectedIndices.size).toBe(1)

    // upload menu is only available when current user exists
    userStore.currentUser = { id: 'me', name: 'me', nickname: 'me' } as any
    await flushAsync()
    expect(wrapper.find('button[title="打开聊天页上传菜单"]').exists()).toBe(true)
    await vm.openUploadMenu()
    expect(openUploadSpy).toHaveBeenCalled()
    expect(toastShow).toHaveBeenCalledWith('已打开上传菜单（可在“已上传的文件”中发送）')

    // favorite toggle: add then remove
    expect(vm.favoriteAwemes.length).toBe(0)
    await vm.toggleFavoriteCurrentDetail()
    expect(douyinApi.addDouyinFavoriteAweme).toHaveBeenCalled()
    expect(vm.favoriteAwemes.length).toBe(1)

    await vm.toggleFavoriteCurrentDetail()
    expect(douyinApi.removeDouyinFavoriteAweme).toHaveBeenCalled()
    expect(vm.favoriteAwemes.length).toBe(0)
  })

  it('account mode covers fetch paging/append, cookie error hint, and favorite user toggle', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    // first fetch: hasMore true, cursor 1; second: stop
    const accountMock = vi.mocked(douyinApi.getDouyinAccount as any)
    accountMock
      .mockResolvedValueOnce({
        secUserId: 'SU1',
        displayName: 'Alice',
        signature: 'sig',
        tab: 'post',
        cursor: 1,
        hasMore: true,
        items: [{ detailId: 'aw1', type: 'video', desc: 'd', coverUrl: 'c', coverDownloadUrl: 'cd', key: '', items: [] }]
      })
      .mockResolvedValueOnce({
        secUserId: 'SU1',
        displayName: 'Alice',
        tab: 'post',
        cursor: 0,
        hasMore: false,
        items: [{ detailId: 'aw2', type: 'image', desc: '', coverUrl: '', coverDownloadUrl: '', key: '', items: [] }]
      })

    ;(douyinApi.addDouyinFavoriteUser as any).mockResolvedValue({ secUserId: 'SU1', createTime: 't', updateTime: 't' })
    ;(douyinApi.removeDouyinFavoriteUser as any).mockResolvedValue({ success: true })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    vm.activeMode = 'account'

    vm.accountInput = ''
    await vm.handleFetchAccount()
    expect(toastShow).toHaveBeenCalledWith('请输入抖音用户主页链接/分享文本/sec_uid')

    vm.accountInput = 'https://www.douyin.com/video/123'
    await vm.handleFetchAccount()
    expect(toastShow).toHaveBeenCalledWith('识别到作品链接，请切换到“作品解析”')
    expect(vm.activeMode).toBe('detail')

    vm.activeMode = 'account'
    vm.accountInput = 'MS4wLjABuser'
    await vm.handleFetchAccount()
    expect(vm.accountItems.length).toBe(1)
    expect(vm.accountHasMore).toBe(true)
    expect(vm.accountSecUserId).toBe('SU1')

    await vm.handleFetchMoreAccount()
    expect(vm.accountItems.length).toBe(2)
    expect(vm.accountHasMore).toBe(false)

    // cookie hint path
    accountMock.mockRejectedValueOnce(new Error('cookie invalid'))
    vm.accountInput = 'MS4wLjABuser'
    await vm.handleFetchAccount()
    expect(vm.showAdvanced).toBe(true)
    expect(vm.highlightConfig).toBe(true)
    expect(String(vm.cookieHint || '')).toContain('Cookie')

    // favorite user toggle
    vm.accountSecUserId = ''
    await vm.toggleFavoriteCurrentUser()
    expect(toastShow).toHaveBeenCalledWith('请先获取用户作品')

    vm.accountSecUserId = 'SU1'
    await vm.toggleFavoriteCurrentUser()
    expect(douyinApi.addDouyinFavoriteUser).toHaveBeenCalled()
    expect(vm.favoriteUsers.length).toBeGreaterThan(0)

    await vm.toggleFavoriteCurrentUser()
    expect(douyinApi.removeDouyinFavoriteUser).toHaveBeenCalled()
  })

  it('favorites mode covers refreshFavorites, tag sheet apply for awemes, and tag manager for awemes', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({
      items: [{ secUserId: 'SU1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [1] }]
    })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({
      items: [{ awemeId: 'A1', createTime: 't', updateTime: 't', tagIds: [2] }]
    })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({
      items: [{ id: 2, name: '教程', count: 1, createTime: 't', updateTime: 't' }]
    })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    vm.activeMode = 'favorites'
    vm.favoritesTab = 'awemes'
    vm.favoriteAwemeTagFilter = 999
    vm.selectedFavoriteAwemeIds.add('not-exist')

    await vm.refreshFavorites()
    expect(vm.favoriteAwemeTagFilter).toBe(null)
    expect(vm.selectedFavoriteAwemeIds.size).toBe(0)

    // batch tag sheet: requires selection
    vm.toggleFavoritesSelectionMode()
    vm.toggleSelectFavoriteAweme('A1')
    vm.openBatchTagSheet()
    expect(vm.tagSheetOpen).toBe(true)

    vm.toggleTagInSheet(2)
    await vm.applyTagSheet()
    expect(douyinApi.applyDouyinFavoriteAwemeTags).toHaveBeenCalled()
    expect(vm.tagSheetOpen).toBe(false)

    // tag manager for awemes
    vm.openTagManager('awemes')
    expect(vm.tagManagerOpen).toBe(true)
    vm.tagManagerNameInput = '新标签'
    await vm.handleCreateTag()
    expect(douyinApi.addDouyinFavoriteAwemeTag).toHaveBeenCalledWith({ name: '新标签' })

    vm.startEditTag({ id: 2, name: '教程', count: 1, createTime: 't', updateTime: 't' })
    vm.editingTagName = '教程2'
    await vm.saveEditTag()
    expect(douyinApi.updateDouyinFavoriteAwemeTag).toHaveBeenCalledWith({ id: 2, name: '教程2' })

    vm.askDeleteTag({ id: 2, name: '教程', count: 1, createTime: 't', updateTime: 't' })
    await vm.confirmDeleteTagNow()
    expect(douyinApi.removeDouyinFavoriteAwemeTag).toHaveBeenCalledWith({ id: 2 })
  })

  it('favorite user detail drawer loads works, supports scroll append + pull latest, opens preview and refreshes user info', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({
      items: [{ secUserId: 'SU1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [] }]
    })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })

    const worksMock = vi.mocked(douyinApi.listDouyinFavoriteUserAwemes as any)
    worksMock
      .mockResolvedValueOnce({
        cursor: 1,
        hasMore: true,
        items: [
          { detailId: 'aw1', type: 'video', desc: 'd1', coverUrl: 'c1', coverDownloadUrl: 'cd1', key: '', items: [] },
          { detailId: 'aw2', type: 'image', desc: '', coverUrl: '', coverDownloadUrl: '', key: '', items: [] }
        ]
      })
      .mockResolvedValueOnce({
        cursor: 0,
        hasMore: false,
        items: [{ detailId: 'aw3', type: 'image', desc: 'd3', coverUrl: 'c3', coverDownloadUrl: 'cd3', key: '', items: [] }]
      })
      .mockResolvedValueOnce({
        cursor: 0,
        hasMore: false,
        items: [{ detailId: 'aw4', type: 'image', desc: 'new', coverUrl: 'c4', coverDownloadUrl: 'cd4', key: '', items: [] }]
      })

    ;(douyinApi.pullLatestDouyinFavoriteUserAwemes as any).mockResolvedValue({ success: true, added: 5 })
    ;(douyinApi.getDouyinDetail as any).mockResolvedValue({
      key: 'k1',
      coverUrl: 'cov',
      items: [
        { index: 0, type: 'image', url: 'u0', downloadUrl: '/api/douyin/download?key=k1&index=0', originalFilename: 'a.jpg' },
        { index: 1, type: 'video', url: 'u1', downloadUrl: '/api/douyin/download?key=k1&index=1', originalFilename: 'b.mp4' }
      ]
    })

    ;(douyinApi.getDouyinAccount as any).mockResolvedValue({
      secUserId: 'SU1',
      displayName: 'Alice2',
      signature: 'sig',
      tab: 'post',
      cursor: 0,
      hasMore: false,
      items: []
    })
    ;(douyinApi.addDouyinFavoriteUser as any).mockResolvedValue({
      secUserId: 'SU1',
      displayName: 'Alice2',
      createTime: 't',
      updateTime: 't',
      tagIds: []
    })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    vm.activeMode = 'favorites'
    vm.favoritesTab = 'users'
    await flushAsync()

    expect(vm.favoriteUsers.length).toBe(1)

    // open favorite user detail and load works
    vm.openFavoriteUserDetail(vm.favoriteUsers[0])
    await flushAsync()
    await flushAsync()
    expect(vm.favoriteUserDetailOpen).toBe(true)
    expect(vm.favoriteUserWorks.length).toBe(2)
    expect(wrapper.text()).toContain('已加载 2 个作品')

    // near-bottom scroll triggers append load
    const scrollEl = document.createElement('div')
    Object.defineProperty(scrollEl, 'scrollHeight', { configurable: true, value: 1000 })
    Object.defineProperty(scrollEl, 'clientHeight', { configurable: true, value: 800 })
    scrollEl.scrollTop = 0
    vm.handleFavoriteUserDetailScroll({ target: scrollEl } as any)
    await flushAsync()
    await flushAsync()
    expect(vm.favoriteUserWorks.length).toBe(3)

    // pull latest resets and reloads
    await vm.pullLatestFavoriteUserWorks()
    await flushAsync()
    expect(douyinApi.pullLatestDouyinFavoriteUserAwemes).toHaveBeenCalled()
    expect(vm.favoriteUserWorks.length).toBe(1)

    // open an item without key/items triggers getDouyinDetail and shows preview
    await vm.openFavoriteUserWork(vm.favoriteUserWorks[0])
    await flushAsync()
    expect(douyinApi.getDouyinDetail).toHaveBeenCalled()
    expect(vm.showPreview).toBe(true)

    // refresh selected favorite user updates favoriteUsers
    await vm.refreshSelectedFavoriteUser()
    expect(douyinApi.getDouyinAccount).toHaveBeenCalled()
    expect(douyinApi.addDouyinFavoriteUser).toHaveBeenCalled()
    expect(vm.favoriteUsers[0].displayName).toBe('Alice2')
  })

  it('preview helpers cover fullList fallbacks and non-douyin media-change indexing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any

    // openPreviewFromAccount: previewUrl uses url when downloadUrl missing
    const accountItem = {
      detailId: 'aw1',
      key: 'k1',
      desc: 'd',
      items: [
        { index: 0, type: 'image', url: 'onlyUrl', downloadUrl: '' },
        { index: 1, type: 'image', url: 'u1', downloadUrl: '/api/douyin/download?key=k1&index=1' }
      ]
    }
    vm.accountItems = [
      accountItem,
      { detailId: 'aw2', key: 'k2', desc: 'd2', items: [{ index: 0, type: 'video', url: 'v0', downloadUrl: 'dv0' }] }
    ]
    await flushAsync()
    vm.openPreviewFromAccount(accountItem)
    await flushAsync()
    expect(vm.previewUrl).toBe('onlyUrl')
    expect(vm.previewMediaList.length).toBeGreaterThan(1)

    // fallback: accountItems empty -> buildPreviewMediaList(item.items)
    vm.accountItems = []
    const standalone = { detailId: 'aw3', key: 'k3', desc: 'd3', items: [{ index: 0, type: 'image', url: 'u', downloadUrl: 'd' }] }
    vm.openPreviewFromAccount(standalone)
    await flushAsync()
    expect(vm.previewMediaList.length).toBe(1)

    // openPreviewFromFavoriteUserWork: fullList empty branch
    vm.favoriteUserWorks = []
    const favWork = { detailId: 'awx', key: 'kf', desc: 'x', items: [{ index: 0, type: 'image', url: 'u', downloadUrl: 'd' }] }
    vm.openPreviewFromFavoriteUserWork(favWork)
    await flushAsync()
    expect(vm.previewMediaList.length).toBe(1)

    // non-douyin media-change: picks index from previewContextItems match
    vm.previewContextItems = [{ index: 7, type: 'image', url: 'uX', downloadUrl: 'dX' }]
    vm.previewIndex = 0
    vm.handlePreviewMediaChange({ url: 'dX', type: 'image' })
    expect(vm.previewIndex).toBe(7)
  })


  it('maps grouped live-photo resources to distinct liveVideoIndex by rank', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    const accountItem = {
      detailId: 'aw-live',
      key: 'k-live',
      desc: 'live',
      items: [
        { index: 0, type: 'image', url: 'u0', downloadUrl: '/api/douyin/download?key=k-live&index=0' },
        { index: 1, type: 'image', url: 'u1', downloadUrl: '/api/douyin/download?key=k-live&index=1' },
        { index: 2, type: 'video', url: 'u2', downloadUrl: '/api/douyin/download?key=k-live&index=2' },
        { index: 3, type: 'video', url: 'u3', downloadUrl: '/api/douyin/download?key=k-live&index=3' }
      ]
    }

    vm.accountItems = [accountItem]
    await flushAsync()
    vm.openPreviewFromAccount(accountItem)
    await flushAsync()

    const medias = vm.previewMediaList as any[]
    const img0 = medias.find((m) => m?.type === 'image' && Number(m?.context?.index) === 0)
    const img1 = medias.find((m) => m?.type === 'image' && Number(m?.context?.index) === 1)

    expect(Number(img0?.context?.liveVideoIndex)).toBe(2)
    expect(Number(img1?.context?.liveVideoIndex)).toBe(3)
  })

  it('tag manager and tag sheet error branches populate error fields', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    ;(douyinApi.listDouyinFavoriteUsers as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteAwemes as any).mockResolvedValue({ items: [] })
    ;(douyinApi.listDouyinFavoriteUserTags as any).mockResolvedValue({
      items: [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    })
    ;(douyinApi.listDouyinFavoriteAwemeTags as any).mockResolvedValue({ items: [] })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    vm.activeMode = 'favorites'
    vm.favoritesTab = 'users'
    await flushAsync()

    // saveTagManagerOrder error
    vm.openTagManager('users')
    ;(douyinApi.reorderDouyinFavoriteUserTags as any).mockRejectedValueOnce({ response: { data: { error: 'boom' } } })
    await vm.saveTagManagerOrder()
    expect(String(vm.tagManagerError || '')).toContain('boom')

    // create tag error
    ;(douyinApi.addDouyinFavoriteUserTag as any).mockRejectedValueOnce(new Error('nope'))
    vm.tagManagerNameInput = '新标签'
    await vm.handleCreateTag()
    expect(String(vm.tagManagerError || '')).toContain('nope')

    // edit tag error
    ;(douyinApi.updateDouyinFavoriteUserTag as any).mockRejectedValueOnce(new Error('up'))
    vm.startEditTag({ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' })
    vm.editingTagName = '美食2'
    await vm.saveEditTag()
    expect(String(vm.tagManagerError || '')).toContain('up')

    // delete tag error
    ;(douyinApi.removeDouyinFavoriteUserTag as any).mockRejectedValueOnce(new Error('del'))
    vm.askDeleteTag({ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' })
    await vm.confirmDeleteTagNow()
    expect(String(vm.tagManagerError || '')).toContain('del')

    // tagSheet error
    ;(douyinApi.applyDouyinFavoriteUserTags as any).mockRejectedValueOnce(new Error('sheet'))
    vm.openTagSheet({ kind: 'users', mode: 'single', targetIds: ['SU1'], presetTagIds: [1] })
    await vm.applyTagSheet()
    expect(String(vm.tagSheetError || '')).toContain('sheet')
  })

  it('syncFavoriteUserWorksFromAccount maps optional fields and remove/reparse close detail drawer', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    ;(douyinApi.upsertDouyinFavoriteUserAwemes as any).mockResolvedValue({ success: true })

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.showModal = true
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any

    vm.accountSecUserId = 'SU1'
    vm.accountItems = [
      {
        detailId: 'aw1',
        type: '',
        desc: '',
        coverUrl: '',
        items: [{ url: 'u0' }, { url: '' }, { url: 'u1' }]
      },
      { detailId: 'aw2', type: 'image', desc: 'd', coverUrl: 'c', items: [] }
    ]

    await vm.syncFavoriteUserWorksFromAccount('SU1')
    expect(douyinApi.upsertDouyinFavoriteUserAwemes).toHaveBeenCalled()

    ;(douyinApi.upsertDouyinFavoriteUserAwemes as any).mockRejectedValueOnce(new Error('boom'))
    await vm.syncFavoriteUserWorksFromAccount('SU1')

    // remove favorite user closes drawer when current
    vm.favoriteUsers = [{ secUserId: 'SU1', createTime: 't', updateTime: 't', tagIds: [] }]
    vm.favoriteUserDetailOpen = true
    vm.favoriteUserDetailId = 'SU1'
    await vm.removeFavoriteUser('SU1')
    expect(vm.favoriteUserDetailOpen).toBe(false)

    // reparseFavoriteUser closes drawer too
    ;(douyinApi.getDouyinAccount as any).mockResolvedValueOnce({
      secUserId: 'SU1',
      tab: 'post',
      cursor: 0,
      hasMore: false,
      items: []
    })
    vm.favoriteUserDetailOpen = true
    vm.favoriteUserDetailId = 'SU1'
    await vm.reparseFavoriteUser({ secUserId: 'SU1' })
    expect(vm.activeMode).toBe('account')
    expect(vm.accountInput).toBe('SU1')
    expect(vm.favoriteUserDetailOpen).toBe(false)
  })

  it('applies chat entry favorites/users context and keeps default entry unchanged after close', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(DouyinDownloadModal, {
      global: {
        plugins: [pinia],
        stubs: {
          teleport: true,
          MediaPreview: MediaPreviewStub,
          MediaTile: MediaTileStub,
          MediaTileBadge: MediaTileBadgeStub,
          MediaTileSelectMark: MediaTileSelectMarkStub
        }
      }
    })

    const douyinStore = useDouyinStore()
    douyinStore.open({
      entryMode: 'favorites',
      favoritesTab: 'users'
    })
    await flushAsync()
    await flushAsync()

    const vm = wrapper.vm as any
    expect(vm.activeMode).toBe('favorites')
    expect(vm.favoritesTab).toBe('users')

    vm.close()
    await flushAsync()
    expect(douyinStore.showModal).toBe(false)
    expect(douyinStore.entryMode).toBe('default')
    expect(douyinStore.favoritesTab).toBe('users')

    douyinStore.open()
    await flushAsync()
    await flushAsync()
    expect(vm.activeMode).toBe('detail')
  })

})
