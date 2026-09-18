import { beforeEach, describe, expect, it, vi } from 'vitest'
import { shallowMount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import MtPhotoAlbumModal from '@/components/media/MtPhotoAlbumModal.vue'
import { useMtPhotoStore } from '@/stores/mtphoto'

const api = vi.hoisted(() => ({ create: vi.fn(), toast: vi.fn() }))
vi.mock('@/api/mtphoto', () => ({ createMtPhotoFolderAlbum: api.create }))
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ show: api.toast }) }))

function setup() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useMtPhotoStore()
  store.showModal = true
  store.mode = 'folders'
  store.view = 'folders'
  store.folderPath = '/photos'
  store.folderList = [{ id: 12, name: '旅行', path: '/photos/旅行' }, { id: 34, name: '家人' }]
  vi.spyOn(store, 'loadAlbums').mockResolvedValue(undefined)
  const wrapper = shallowMount(MtPhotoAlbumModal, { global: { plugins: [pinia], stubs: { teleport: true } } })
  return { store, wrapper }
}

describe('创建文件夹相册', () => {
  beforeEach(() => vi.clearAllMocks())

  it('keeps selection across navigation and creates one album with a trimmed name', async () => {
    const { store, wrapper } = setup()
    await wrapper.get('[aria-label="选择文件夹 旅行"]').setValue(true)
    store.folderCurrentId = 34
    store.folderCurrentName = '家人'
    store.folderPath = '/photos/家人'
    store.folderList = [{ id: 56, name: '聚会' }]
    await nextTick()
    await wrapper.get('[aria-label="选择文件夹 聚会"]').setValue(true)
    expect(wrapper.text()).toContain('已选 2 个文件夹')
    await wrapper.get('[aria-label="相册名称"]').setValue('  回忆  ')
    let resolve: (v: any) => void = () => {}
    api.create.mockImplementation(() => new Promise(r => { resolve = r }))
    await wrapper.get('[data-testid="create-folder-album"]').trigger('click')
    expect(api.create).toHaveBeenCalledWith({ name: '回忆', folders: [{ id: 12, path: '/photos/旅行' }, { id: 56, path: '/photos/家人/聚会' }] })
    expect(wrapper.get('[data-testid="create-folder-album"]').attributes('disabled')).toBeDefined()
    resolve({ success: true, album: { id: 99, name: '回忆' } })
    await flushPromises()
    expect(store.loadAlbums).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toContain('已选 2 个文件夹')
    expect(api.toast).toHaveBeenCalledWith('文件夹相册已创建')
    wrapper.unmount()
  })

  it('can select the current folder and retains the draft on a failure before creation', async () => {
    const { store, wrapper } = setup()
    store.folderCurrentId = 34
    store.folderCurrentName = '家人'
    store.folderPath = '/photos/家人'
    await nextTick()
    await wrapper.get('[data-testid="select-current-folder"]').trigger('click')
    expect((wrapper.get('[aria-label="相册名称"]').element as HTMLInputElement).value).toBe('家人')
    api.create.mockRejectedValue({ response: { data: { error: '无权限' } } })
    await wrapper.get('[data-testid="create-folder-album"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('无权限')
    expect(wrapper.text()).toContain('已选 1 个文件夹')
    wrapper.unmount()
  })

  it('shows partial creation separately and clears selection to prevent accidental duplicates', async () => {
    const { store, wrapper } = setup()
    await wrapper.get('[aria-label="选择文件夹 家人"]').setValue(true)
    api.create.mockRejectedValue({ response: { data: { album: { id: 99 }, error: '相册已创建（ID 99），同步失败' } } })
    await wrapper.get('[data-testid="create-folder-album"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('相册已创建（ID 99），同步失败')
    expect(store.loadAlbums).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toContain('已选 1 个文件夹')
    expect(api.toast).not.toHaveBeenCalledWith('文件夹相册已创建')
    wrapper.unmount()
  })

  it('clears selections on close and does not submit an empty name', async () => {
    const { store, wrapper } = setup()
    await wrapper.get('[aria-label="选择文件夹 旅行"]').setValue(true)
    await wrapper.get('[aria-label="相册名称"]').setValue('   ')
    expect(wrapper.get('[data-testid="create-folder-album"]').attributes('disabled')).toBeDefined()
    store.showModal = false
    await nextTick()
    store.showModal = true
    await nextTick()
    expect(wrapper.text()).not.toContain('已选 1 个文件夹')
    expect(api.create).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('selects only visible search results without duplicating existing selections', async () => {
    const { wrapper } = setup()
    await wrapper.get('[aria-label="选择文件夹 旅行"]').setValue(true)
    await wrapper.get('input[placeholder="搜索目录..."]').setValue('家人')
    const selectAll = wrapper.findAll('button').find(button => button.text() === '全选当前列表')!
    await selectAll.trigger('click')
    await selectAll.trigger('click')
    expect(wrapper.text()).toContain('已选 2 个文件夹')
    await wrapper.get('[aria-label="取消选择 /photos/旅行"]').trigger('click')
    expect(wrapper.text()).toContain('已选 1 个文件夹')
    wrapper.unmount()
  })
})
