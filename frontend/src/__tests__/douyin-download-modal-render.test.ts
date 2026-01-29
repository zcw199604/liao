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
  listDouyinFavoriteUsers: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteAwemes: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ items: [] }),
  listDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ items: [] }),
  applyDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ success: true }),
  applyDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ success: true }),
  reorderDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ success: true }),
  reorderDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ success: true }),
  addDouyinFavoriteUser: vi.fn().mockResolvedValue({}),
  removeDouyinFavoriteUser: vi.fn().mockResolvedValue({ success: true }),
  addDouyinFavoriteAweme: vi.fn().mockResolvedValue({}),
  removeDouyinFavoriteAweme: vi.fn().mockResolvedValue({ success: true }),
  getDouyinDetail: vi.fn(),
  getDouyinAccount: vi.fn(),
  listDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ items: [], cursor: 0, hasMore: false }),
  pullLatestDouyinFavoriteUserAwemes: vi.fn().mockResolvedValue({ success: true, added: 0 }),
  addDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
  updateDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
  removeDouyinFavoriteUserTag: vi.fn().mockResolvedValue({ success: true }),
  addDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
  updateDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
  removeDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({ success: true }),
  importDouyinMedia: vi.fn()
}))

import DouyinDownloadModal from '@/components/media/DouyinDownloadModal.vue'
import { useDouyinStore } from '@/stores/douyin'

const flush = async () => {
  await Promise.resolve()
  await nextTick()
}

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type', 'muted', 'showSkeleton', 'indicatorSize'],
  template: `<div data-testid="media-tile"><slot /></div>`
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

describe('components/media/DouyinDownloadModal.vue (render branches)', () => {
  it('renders many conditional branches across modes and drawers', async () => {
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
    await flush()
    await flush()

    const vm = wrapper.vm as any

    // detail mode: input + advanced + error
    vm.activeMode = 'detail'
    vm.inputText = 'x'
    vm.showAdvanced = true
    vm.cookie = 'c'
    vm.cookieHint = 'Cookie 提示'
    vm.highlightConfig = true
    vm.error = '解析失败'
    await flush()

    // detail result with item states, metas, selection, progress
    vm.detail = {
      key: 'k1',
      detailId: 'a1',
      type: '图集',
      title: 'T',
      coverUrl: 'cov',
      duration: '10',
      width: 100,
      height: 200,
      items: [
        { index: 0, type: 'image', url: 'u0', downloadUrl: '/api/douyin/download?key=k1&index=0', originalFilename: 'a.jpg' },
        { index: 1, type: 'image', url: 'u1', downloadUrl: '/api/douyin/download?key=k1&index=1', originalFilename: 'b.jpg' },
        { index: 2, type: 'video', url: 'u2', downloadUrl: '/api/douyin/download?key=k1&index=2', originalFilename: 'c.mp4' },
        { index: 3, type: 'image', url: 'u3', downloadUrl: '/api/douyin/download?key=k1&index=3', originalFilename: 'd.jpg' },
        { index: 4, type: 'image', url: 'u4', downloadUrl: '/api/douyin/download?key=k1&index=4', originalFilename: 'e.jpg' }
      ]
    }
    vm.itemStateById['k1:0'] = { status: 'idle' }
    vm.itemStateById['k1:1'] = { status: 'importing' }
    vm.itemStateById['k1:2'] = { status: 'imported' }
    vm.itemStateById['k1:3'] = { status: 'exists' }
    vm.itemStateById['k1:4'] = { status: 'error', message: 'x' }
    vm.itemMetaById['k1:0'] = { size: 0, mime: 'image/jpeg' }
    vm.itemMetaById['k1:1'] = { size: 1024, mime: 'image/png' }
    vm.selectionMode = true
    vm.selectedIndices = new Set([0, 1])
    vm.batchImport.running = true
    vm.batchImport.total = 2
    vm.batchImport.done = 1
    vm.batchImport.success = 1
    vm.batchImport.fail = 0
    await flush()

    vm.batchImport.running = false
    vm.batchDownload.running = true
    vm.batchDownload.total = 5
    vm.batchDownload.done = 3
    vm.batchDownload.success = 2
    vm.batchDownload.fail = 1
    await flush()

    vm.batchDownload.running = false
    vm.showPreview = true
    vm.previewUrl = 'u0'
    vm.previewType = 'image'
    vm.previewMediaList = [{ url: 'u0', type: 'image' }]
    await flush()
    vm.showPreview = false
    await flush()

    // account mode: header + list + errors + hasMore
    vm.activeMode = 'account'
    vm.accountInput = 'MS4wLjABuser'
    vm.accountSecUserId = 'SU1'
    vm.accountDisplayName = 'Alice'
    vm.accountSignature = 'sig'
    vm.accountProfileUrl = 'https://x'
    vm.accountItems = [{ detailId: 'aw1', type: 'video', desc: 'd', coverUrl: 'c', coverDownloadUrl: 'cd', key: '', items: [] }]
    vm.accountHasMore = true
    vm.accountError = '获取失败'
    await flush()
    vm.accountError = ''
    vm.accountHasMore = false
    await flush()

    // favorites mode: loading + error + users tab + filters + selection + tag sheet + tag manager + confirm delete
    vm.activeMode = 'favorites'
    vm.favoritesLoading = true
    await flush()
    vm.favoritesLoading = false
    vm.favoritesError = '加载失败'
    await flush()
    vm.favoritesError = ''

    vm.favoritesTab = 'users'
    vm.favoriteUserTags = [{ id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }]
    vm.favoriteUsers = [
      { secUserId: 'SU1', displayName: 'Alice', createTime: 't', updateTime: 't', tagIds: [1] },
      { secUserId: 'SU2', displayName: 'Bob', createTime: 't', updateTime: 't', tagIds: [] }
    ]
    vm.favoriteUserTagFilter = null
    await flush()
    vm.favoriteUserTagFilter = -1
    await flush()
    vm.favoriteUserTagFilter = 1
    await flush()

    vm.favoritesSelectionMode = true
    vm.selectedFavoriteUserIds.add('SU1')
    await flush()

    vm.tagSheetOpen = true
    vm.tagSheetKind = 'users'
    vm.tagSheetMode = 'single'
    vm.tagSheetTargetIds = ['SU1']
    vm.tagSheetError = '保存失败'
    await flush()
    vm.tagSheetOpen = false
    await flush()

    vm.tagManagerOpen = true
    vm.tagManagerKind = 'users'
    vm.editingTagId = 1
    vm.editingTagName = '美食2'
    vm.tagManagerError = 'err'
    await flush()

    vm.confirmDeleteTagOpen = true
    vm.confirmDeleteTag = { id: 1, name: '美食', count: 1, createTime: 't', updateTime: 't' }
    await flush()
    vm.confirmDeleteTagOpen = false
    vm.confirmDeleteTag = null
    vm.tagManagerOpen = false
    vm.editingTagId = null
    await flush()

    // favorites awemes tab + filters
    vm.favoritesTab = 'awemes'
    vm.favoriteAwemeTags = [{ id: 2, name: '教程', count: 1, createTime: 't', updateTime: 't' }]
    vm.favoriteAwemes = [
      { awemeId: 'A1', createTime: 't', updateTime: 't', tagIds: [2] },
      { awemeId: 'A2', createTime: 't', updateTime: 't', tagIds: [] }
    ]
    vm.favoriteAwemeTagFilter = null
    await flush()
    vm.favoriteAwemeTagFilter = -1
    await flush()
    vm.favoriteAwemeTagFilter = 2
    await flush()

    // favorite user detail drawer: empty -> loading -> hasMore -> done
    vm.favoritesTab = 'users'
    vm.favoriteUserDetailOpen = true
    vm.favoriteUserDetailId = 'SU1'
    vm.favoriteUserAvatarError.add('SU1')
    vm.favoriteUserDetailLoading = true
    vm.favoriteUserWorks = []
    vm.favoriteUserWorksLoading = false
    await flush()

    vm.favoriteUserDetailLoading = false
    vm.favoriteUserWorks = [{ detailId: 'awx', type: 'image', desc: '', coverUrl: '', coverDownloadUrl: '', key: '', items: [] }]
    vm.favoriteUserWorksLoading = true
    await flush()

    vm.favoriteUserWorksLoading = false
    vm.favoriteUserWorksHasMore = true
    await flush()

    vm.favoriteUserWorksHasMore = false
    await flush()

    vm.favoriteUserDetailOpen = false
    await flush()

    expect(wrapper.text()).toContain('抖音下载')
    expect(toastShow).not.toHaveBeenCalledWith(expect.stringMatching(/ERROR/i))
  })
})
