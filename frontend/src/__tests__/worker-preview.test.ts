import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

const harness = vi.hoisted(() => ({
  routes: [] as any[],
  post: vi.fn(),
  delete: vi.fn(),
  back: vi.fn(),
  replace: vi.fn()
}))

vi.mock('@/api/request', () => ({
  default: { post: harness.post, delete: harness.delete }
}))
vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(),
  createRouter: vi.fn((options: any) => {
    harness.routes = options.routes
    return { beforeEach: vi.fn() }
  }),
  useRouter: () => ({ back: harness.back, replace: harness.replace })
}))

import '@/router'

async function mountPreview() {
  const route = harness.routes.find(route => route.path === '/worker-preview')
  expect(route, '提供登录后可用的媒体预览入口').toBeDefined()
  expect(route.meta).toEqual({ requiresAuth: true })
  const { default: component } = await route.component()
  return mount(component)
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
  harness.delete.mockResolvedValue({ code: 0 })
  harness.post.mockResolvedValue({ code: 0, data: { url: '/worker-preview/preview' } })
})

describe('Worker 媒体预览', () => {
  it('建立会话后才加载同源页面，不把账号密码或 token 放入页面地址', async () => {
    let resolve!: (value: unknown) => void
    harness.post.mockImplementationOnce(() => new Promise(done => { resolve = done }))
    const wrapper = await mountPreview()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('正在打开')
    expect(harness.post).toHaveBeenCalledWith('/workerPreview/session')
    resolve({ code: 0, data: { url: '/worker-preview/preview' } })
    await flushPromises()
    expect(wrapper.get('iframe').attributes('src')).toBe('/worker-preview/preview')
    expect(wrapper.get('iframe').attributes('title')).toBe('X 媒体预览')
    expect(wrapper.html()).not.toMatch(/password|username|token=/i)
    wrapper.unmount()
    expect(harness.delete).not.toHaveBeenCalled()
  })

  it('会话失败显示可读错误，重试后恢复嵌入页面', async () => {
    harness.post.mockRejectedValueOnce({ response: { data: { msg: '媒体预览暂不可用' } } })
    const wrapper = await mountPreview()
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('媒体预览暂不可用')
    await wrapper.get('[data-testid="worker-preview-retry"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })

  it('刷新重新建立会话并重新挂载 iframe', async () => {
    const wrapper = await mountPreview()
    await flushPromises()
    const firstFrame = wrapper.get('iframe').element
    await wrapper.get('iframe').trigger('load')
    await wrapper.get('[data-testid="worker-preview-refresh"]').trigger('click')
    await flushPromises()
    expect(harness.post).toHaveBeenCalledTimes(2)
    expect(wrapper.get('iframe').element).not.toBe(firstFrame)
    wrapper.unmount()
  })

  it('拒绝外站或带凭据的地址，避免把它作为嵌入入口', async () => {
    harness.post.mockResolvedValueOnce({ code: 0, data: { url: 'https://other.example/preview?token=secret' } })
    const wrapper = await mountPreview()
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('预览地址无效')
    expect(wrapper.html()).not.toContain('secret')
    wrapper.unmount()
  })

  it('退出 liao 时用退出前的 token 清除预览会话，随即清空本地登录态', () => {
    localStorage.setItem('authToken', 'liao-token-before-logout')
    const store = useAuthStore()
    store.isAuthenticated = true
    store.logout()
    expect(harness.delete).toHaveBeenCalledWith('/workerPreview/session', {
      headers: { Authorization: 'Bearer liao-token-before-logout' }
    })
    expect(store.token).toBe('')
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem('authToken')).toBeNull()
  })

  it('会话清理失败不影响 liao 退出', async () => {
    localStorage.setItem('authToken', 'liao-token')
    harness.delete.mockRejectedValueOnce(new Error('offline'))
    const store = useAuthStore()
    store.logout()
    await flushPromises()
    expect(store.isAuthenticated).toBe(false)
    expect(store.token).toBe('')
    expect(localStorage.getItem('authToken')).toBeNull()
  })
})
