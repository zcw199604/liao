// 当前上游线路（provider）的运行时快照。
//
// 为什么不用 localStorage 直读：provider store 的 currentId 有「未选择时回落到后端默认线路」
// 的语义，localStorage 里可能什么都没有，但当前线路其实是 chat2019 / wanfeng 之一。
// 历史、收藏、聊天记录这些接口必须带上真实线路，否则后端会按「身份绑定关系」分流，
// 从晚风切回 chat2019 后刷新页面就会看到晚风侧的列表（自己这边的人全消失）。
let currentProviderId = ''

// setCurrentProviderId 由 provider store 在 currentId 变化时调用。
export const setCurrentProviderId = (id: string) => {
  currentProviderId = String(id || '')
}

// getCurrentProviderId 供请求拦截器读取；空字符串表示线路尚未确定。
export const getCurrentProviderId = (): string => currentProviderId

// 交给后端识别线路的请求头名（与 internal/app/provider_context.go 保持一致）。
export const PROVIDER_HEADER_NAME = 'X-Liao-Provider'
