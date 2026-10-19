import { afterEach, describe, expect, it, vi } from 'vitest'
import { hasNativeEdgeSwipeBack } from '@/utils/device'

const overridden = new Set<string>()
// jsdom 的 navigator 属性定义在原型上，这里在实例上覆盖，用完删除即可恢复
const setNavigator = (props: Record<string, unknown>) => {
  for (const [key, value] of Object.entries(props)) {
    Object.defineProperty(navigator, key, { configurable: true, get: () => value })
    overridden.add(key)
  }
}

afterEach(() => {
  vi.restoreAllMocks()
  for (const key of overridden) delete (navigator as any)[key]
  overridden.clear()
})

describe('utils/device', () => {
  it('returns false for non-iOS browsers', () => {
    setNavigator({ userAgent: 'Mozilla/5.0 (Windows NT 10.0)', platform: 'Win32', maxTouchPoints: 0 })
    expect(hasNativeEdgeSwipeBack()).toBe(false)
  })

  it('returns true for iPhone Safari and iPadOS desktop-mode Safari', () => {
    setNavigator({ userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)', platform: 'iPhone', maxTouchPoints: 5 })
    expect(hasNativeEdgeSwipeBack()).toBe(true)

    setNavigator({ userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', platform: 'MacIntel', maxTouchPoints: 5 })
    expect(hasNativeEdgeSwipeBack()).toBe(true)
  })

  it('returns false for iOS home-screen web apps (standalone)', () => {
    setNavigator({ userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)', platform: 'iPhone', maxTouchPoints: 5 })
    setNavigator({ standalone: true })
    expect(hasNativeEdgeSwipeBack()).toBe(false)

    setNavigator({ standalone: false })
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: true } as any)
    expect(hasNativeEdgeSwipeBack()).toBe(false)
  })
})
