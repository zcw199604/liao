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

vi.mock('@/api/douyin', async () => {
  const actual = await vi.importActual<any>('@/api/douyin')
  return {
    ...actual,
    listDouyinFavoriteUsers: vi.fn().mockResolvedValue({ items: [] }),
    listDouyinFavoriteAwemes: vi.fn().mockResolvedValue({ items: [] }),
    listDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ items: [] }),
    listDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ items: [] }),
    addDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
    updateDouyinFavoriteUserTag: vi.fn().mockResolvedValue({}),
    removeDouyinFavoriteUserTag: vi.fn().mockResolvedValue({ success: true }),
    applyDouyinFavoriteUserTags: vi.fn().mockResolvedValue({ success: true }),
    addDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
    updateDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({}),
    removeDouyinFavoriteAwemeTag: vi.fn().mockResolvedValue({ success: true }),
    applyDouyinFavoriteAwemeTags: vi.fn().mockResolvedValue({ success: true })
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
})
