// 只提示刷新；保留聊天输入、登录状态及用户本地数据。
export function watchAppUpdates(onUpdate: (reason: 'version' | 'load-error') => void): () => void {
  let version: string | undefined
  let checking = false
  let stopped = false
  let notified = false
  const currentEntry = document.querySelector<HTMLScriptElement>('script[type="module"][src]')?.getAttribute('src')

  const notify = (reason: 'version' | 'load-error') => {
    if (stopped || notified) return
    notified = true
    onUpdate(reason)
  }
  const check = async () => {
    if (stopped || checking || notified || document.visibilityState === 'hidden') return
    checking = true
    try {
      const response = await fetch(`/version.json?t=${Date.now()}`, { cache: 'no-store' })
      if (!response.ok) return
      const next: unknown = await response.json()
      if (!next || typeof next !== 'object' || !('version' in next) || typeof next.version !== 'string' || !next.version) return
      const entry = 'entry' in next && typeof next.entry === 'string' ? next.entry : ''
      if ((currentEntry && entry && new URL(currentEntry, location.origin).href !== new URL(entry, location.origin).href) || (version && version !== next.version)) {
        notify('version')
      }
      version = next.version
    } catch {
      // 离线、部署切换或开发服务器未提供版本接口时，下次检测再试。
    } finally {
      checking = false
    }
  }
  const checkOnVisible = () => { void check() }
  const onPreloadError = (event: Event) => {
    event.preventDefault()
    notify('load-error')
  }
  window.addEventListener('focus', checkOnVisible)
  window.addEventListener('pageshow', checkOnVisible)
  document.addEventListener('visibilitychange', checkOnVisible)
  window.addEventListener('vite:preloadError', onPreloadError)
  const timer = window.setInterval(checkOnVisible, 60_000)
  void check()

  return () => {
    stopped = true
    window.clearInterval(timer)
    window.removeEventListener('focus', checkOnVisible)
    window.removeEventListener('pageshow', checkOnVisible)
    document.removeEventListener('visibilitychange', checkOnVisible)
    window.removeEventListener('vite:preloadError', onPreloadError)
  }
}
