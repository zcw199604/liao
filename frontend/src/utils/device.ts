// 设备/环境探测工具

// iOS 浏览器（非主屏 Web App 模式）自带左边缘右滑返回手势，
// 此时页面自定义的边缘返回会与系统手势叠加，导致一次滑动返回两层
export const hasNativeEdgeSwipeBack = (): boolean => {
  const ua = navigator.userAgent || ''
  const isIOS = /iP(hone|ad|od)/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  if (!isIOS) return false
  const standalone =
    (navigator as Navigator & { standalone?: boolean }).standalone === true ||
    (typeof window.matchMedia === 'function' && window.matchMedia('(display-mode: standalone)').matches)
  return !standalone
}
