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

import SystemSettings from '@/components/settings/SystemSettings.vue'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useThemeStore } from '@/stores/theme'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('components/settings/SystemSettings.vue', () => {
  it('loads system config on mount and applies draft values (real mode shows min-bytes input)', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'loadSystemConfig').mockImplementation(async () => {
      systemConfigStore.imagePortMode = 'real'
      systemConfigStore.imagePortFixed = '9008'
      systemConfigStore.imagePortRealMinBytes = 4096
    })

    const themeStore = useThemeStore()
    themeStore.setPreference('auto')

    const wrapper = mount(SystemSettings, {
      props: {
        stats: { active: 3, upstream: 1, downstream: 2 } as any,
        forceoutCount: 0,
        loading: false
      },
      global: { plugins: [pinia] }
    })

    await flushAsync()
    await flushAsync()

    expect(systemConfigStore.loadSystemConfig).toHaveBeenCalledTimes(1)

    const modeSelect = wrapper.get('select')
    expect((modeSelect.element as HTMLSelectElement).value).toBe('real')

    const fixedPortInput = wrapper.get('input[placeholder="9006"]')
    expect((fixedPortInput.element as HTMLInputElement).value).toBe('9008')

    const minBytesInput = wrapper.get('input[placeholder="2048"]')
    expect((minBytesInput.element as HTMLInputElement).value).toBe('4096')

    expect(wrapper.text()).toContain('跟随系统')
  })

  it('theme buttons delegate to themeStore.setPreference', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'loadSystemConfig').mockResolvedValue(undefined)

    const themeStore = useThemeStore()
    const setPrefSpy = vi.spyOn(themeStore, 'setPreference')

    const wrapper = mount(SystemSettings, {
      props: { stats: {} as any, forceoutCount: 0, loading: false },
      global: { plugins: [pinia] }
    })

    await flushAsync()

    await wrapper.get('button[title="跟随系统"]').trigger('click')
    await wrapper.get('button[title="浅色"]').trigger('click')
    await wrapper.get('button[title="深色"]').trigger('click')

    expect(setPrefSpy).toHaveBeenCalledWith('auto')
    expect(setPrefSpy).toHaveBeenCalledWith('light')
    expect(setPrefSpy).toHaveBeenCalledWith('dark')
  })

  it('saves image port config and shows toast on success/failure', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'loadSystemConfig').mockResolvedValue(undefined)
    const saveSpy = vi.spyOn(systemConfigStore, 'saveSystemConfig').mockResolvedValue(true)

    const wrapper = mount(SystemSettings, {
      props: { stats: {} as any, forceoutCount: 0, loading: false },
      global: { plugins: [pinia] }
    })

    await flushAsync()

    await wrapper.get('select').setValue('probe')
    await wrapper.get('input[placeholder="9006"]').setValue('9010')

    const saveBtn = wrapper.findAll('button').find(btn => btn.text().includes('保存'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    await flushAsync()

    expect(saveSpy).toHaveBeenCalledWith({
      imagePortMode: 'probe',
      imagePortFixed: '9010',
      imagePortRealMinBytes: 2048
    })
    expect(toastShow).toHaveBeenCalledWith('已保存')

    saveSpy.mockResolvedValue(false)
    await saveBtn!.trigger('click')
    await flushAsync()
    expect(toastShow).toHaveBeenCalledWith('保存失败')
  })

  it('emits refresh/disconnectAll/clearForceout and renders forceout empty-state', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const systemConfigStore = useSystemConfigStore()
    vi.spyOn(systemConfigStore, 'loadSystemConfig').mockResolvedValue(undefined)

    const wrapper = mount(SystemSettings, {
      props: { stats: {} as any, forceoutCount: 2, loading: false },
      global: { plugins: [pinia] }
    })

    await flushAsync()

    const refreshBtn = wrapper.findAll('button').find(btn => btn.text().includes('刷新'))
    expect(refreshBtn).toBeTruthy()
    await refreshBtn!.trigger('click')
    expect(wrapper.emitted('refresh')?.length).toBe(1)

    const disconnectBtn = wrapper.findAll('button').find(btn => btn.text().includes('断开所有连接'))
    expect(disconnectBtn).toBeTruthy()
    await disconnectBtn!.trigger('click')
    expect(wrapper.emitted('disconnectAll')?.length).toBe(1)

    const clearBtn = wrapper.findAll('button').find(btn => btn.text().includes('清除所有禁止连接的用户'))
    expect(clearBtn).toBeTruthy()
    await clearBtn!.trigger('click')
    expect(wrapper.emitted('clearForceout')?.length).toBe(1)

    wrapper.unmount()

    const wrapper2 = mount(SystemSettings, {
      props: { stats: {} as any, forceoutCount: 0, loading: true },
      global: { plugins: [pinia] }
    })
    await flushAsync()
    expect(wrapper2.text()).toContain('当前没有被禁止的用户')

    const disconnectBtn2 = wrapper2.findAll('button').find(btn => btn.text().includes('断开'))
    expect(disconnectBtn2?.attributes('disabled')).toBeDefined()
    expect(disconnectBtn2?.text()).toContain('断开中')
  })
})
