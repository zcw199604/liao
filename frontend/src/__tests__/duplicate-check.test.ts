import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import DuplicateCheckModal from '@/components/media/DuplicateCheckModal.vue'

// Mock dependencies
const toastShow = vi.fn()
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

const checkDuplicateMediaMock = vi.fn()
vi.mock('@/api/media', () => ({
  checkDuplicateMedia: (...args: any[]) => checkDuplicateMediaMock(...args)
}))

let mockImgServerValue = 'localhost'
const loadImgServerMock = vi.fn()
vi.mock('@/stores/media', () => ({
  useMediaStore: () => ({
    get imgServer() { return mockImgServerValue },
    loadImgServer: loadImgServerMock
  })
}))

// Mock FileReader
class MockFileReader {
  onload: ((e: any) => void) | null = null
  readAsDataURL(blob: Blob) {
    setTimeout(() => {
      if (this.onload) {
        this.onload({
          target: {
            result: 'data:image/png;base64,mockdata'
          }
        })
      }
    }, 10)
  }
}
globalThis.FileReader = MockFileReader as any

describe('components/media/DuplicateCheckModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    mockImgServerValue = 'localhost'
    HTMLElement.prototype.scrollIntoView = vi.fn()
  })

  it('mounts correctly and calls loadImgServer if needed', () => {
    mockImgServerValue = '' // Set empty to trigger load
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    expect(wrapper.find('h3').text()).toBe('图片查重工具')
    expect(loadImgServerMock).toHaveBeenCalled()
  })

  it('handles file selection and shows preview', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    
    // Simulate file change
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })
    await input.trigger('change')
    
    vi.advanceTimersByTime(20)
    await nextTick()

    // Check preview
    const previewImg = wrapper.find('img.w-full.h-full.object-contain')
    expect(previewImg.exists()).toBe(true)
    expect(previewImg.attributes('src')).toBe('data:image/png;base64,mockdata')
    
    vi.useRealTimers()
  })

  it('calls API with correct parameters when check button is clicked', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    // Select file
    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    vi.advanceTimersByTime(20)
    await nextTick()

    // Mock API response with controlled promise
    let resolveApi: Function
    checkDuplicateMediaMock.mockImplementation(() => new Promise((resolve) => {
      resolveApi = resolve
    }))

    // Click Check button
    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('开始查重'))
    expect(checkBtn?.attributes('disabled')).toBeUndefined()
    
    await checkBtn?.trigger('click')
    // Wait for microtasks but API is still pending
    await nextTick()
    
    expect(wrapper.find('.fa-spinner').exists()).toBe(true) // Loading state

    // Resolve API
    resolveApi!({
      code: 0,
      data: {
        matchType: 'none',
        md5: 'mockmd5',
        thresholdType: 'similarity',
        similarityThreshold: 0.85,
        distanceThreshold: 10,
        limit: 20,
        items: []
      }
    })
    
    // Advance timers/ticks to process resolution
    await nextTick()
    await nextTick()
    await nextTick()
    await nextTick()

    expect(checkDuplicateMediaMock).toHaveBeenCalled()
    const formData = checkDuplicateMediaMock.mock.calls[0]![0] as FormData
    expect(formData.get('file')).toBe(file)
    expect(formData.get('similarityThreshold')).toBe('0.85')
    expect(formData.get('limit')).toBe('20')

    // Check result display
    expect(wrapper.text()).toContain('未发现重复图片')
    expect(wrapper.find('.fa-spinner').exists()).toBe(false)
    
    vi.useRealTimers()
  })

  it('displays duplicate results correctly', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    // Select file
    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    vi.advanceTimersByTime(20)
    await nextTick()

    // Mock API response with duplicates
    checkDuplicateMediaMock.mockResolvedValue({
      code: 0,
      data: {
        matchType: 'phash',
        md5: 'mockmd5',
        pHash: '123456',
        thresholdType: 'similarity',
        similarityThreshold: 0.85,
        distanceThreshold: 10,
        limit: 20,
        items: [
          {
            id: 1,
            filePath: '/img/Upload/1.jpg',
            fileName: '1.jpg',
            md5Hash: 'abc',
            pHash: '123',
            distance: 2,
            similarity: 0.95,
            createdAt: '2026-01-01'
          }
        ]
      }
    })

    // Click Check button
    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('开始查重'))
    await checkBtn?.trigger('click')
    await nextTick()
    await nextTick()

    // Verify results
    expect(wrapper.text()).toContain('pHash 相似命中')
    expect(wrapper.text()).toContain('95.0% 相似')
    expect(wrapper.text()).toContain('1.jpg')
    
    vi.useRealTimers()
  })

  it('shows error toast when API fails', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    // Select file
    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    vi.advanceTimersByTime(20)
    await nextTick()

    // Mock API failure
    checkDuplicateMediaMock.mockResolvedValue({
      code: 500,
      msg: 'Server error'
    })

    // Click Check button
    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('开始查重'))
    await checkBtn?.trigger('click')
    await nextTick()
    await nextTick()

    expect(toastShow).toHaveBeenCalledWith('Server error')
    
    vi.useRealTimers()
  })

  it('updates threshold and limit when sliders change', async () => {
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    const sliders = wrapper.findAll('input[type="range"]')
    const thresholdSlider = sliders[0]!
    const limitSlider = sliders[1]!

    await thresholdSlider.setValue(0.9)
    await limitSlider.setValue(50)

    expect(wrapper.text()).toContain('90%')
    
    // We can't easily check internal state without exposing it, 
    // but we can check if the API is called with new values
    
    vi.useFakeTimers()
    // Select file
    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    vi.advanceTimersByTime(20)
    await nextTick()

    checkDuplicateMediaMock.mockResolvedValue({ code: 0, data: { items: [] } })

    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('开始查重'))
    await checkBtn?.trigger('click')
    await nextTick()
    await nextTick()

    const formData = checkDuplicateMediaMock.mock.calls[0]![0] as FormData
    expect(formData.get('similarityThreshold')).toBe('0.9')
    expect(formData.get('limit')).toBe('50')
    
    vi.useRealTimers()
  })

  it('handles null items from API gracefully', async () => {
    vi.useFakeTimers()
    const wrapper = mount(DuplicateCheckModal, {
      props: { visible: true },
      global: {
        stubs: { teleport: true }
      }
    })

    const file = new File([''], 'test.png', { type: 'image/png' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    vi.advanceTimersByTime(20)
    await nextTick()

    // Mock API response with null items
    checkDuplicateMediaMock.mockResolvedValue({
      code: 0,
      data: {
        matchType: 'none',
        md5: 'mockmd5',
        thresholdType: 'similarity',
        similarityThreshold: 0.85,
        distanceThreshold: 10,
        limit: 20,
        items: null // Simulate backend returning null
      }
    })

    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('开始查重'))
    await checkBtn?.trigger('click')
    await nextTick()
    await nextTick()
    await nextTick()
    await nextTick()

    // Should show "0 similar results" and empty message without crashing
    expect(wrapper.text()).toContain('共找到 0 个相似结果')
    expect(wrapper.text()).toContain('未发现重复图片')
    
    vi.useRealTimers()
  })
})
