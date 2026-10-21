// 上游媒体端口回退：
// 上游把媒体服务端口写在自己的页面里（旧版图片 9006 / 视频 8006，新版统一 7001/7004），
// 而 getImgServer 只返回 host、不返回端口。一旦上游换端口，配置里写死的端口就会导致裂图。
// 这里在“加载失败”时按候选端口自动重试，并记住真正可用的端口，后续直接使用。
//
// learnedMediaPort 使用 ref：在模板里通过 getMediaUrl 读取时会建立响应式依赖，
// 一旦探测到可用端口，所有媒体地址会自动切换到该端口并重新加载。

import { ref } from 'vue'

// 候选端口顺序：优先上游当前端口，再回退历史端口。
export const MEDIA_PORT_CANDIDATES = [
  '7001', '7004',
  '9006', '9005', '9003', '9002', '9001',
  '8006', '8005', '8003', '8002', '8001'
] as const

export interface UpstreamMediaUrl {
  // 协议 + 主机 + 端口，例如 http://1.2.3.4:7001
  origin: string
  protocol: string
  host: string
  port: string
  // 端口之后的部分，例如 /img/Upload/2026/01/a.png
  rest: string
}

// 解析形如 http://host:port/img/Upload/... 的上游媒体地址；
// 不是这种结构（本地 /upload、同源代理、第三方图床等）时返回 null，避免误改端口。
export const parseUpstreamMediaUrl = (input: string): UpstreamMediaUrl | null => {
  const raw = String(input || '').trim()
  if (!raw) return null

  const match = raw.match(/^(https?:\/\/)([^/?#]+)(\/img\/Upload\/.*)$/i)
  if (!match) return null

  // 正则分组保证存在，这里直接取用（避免无意义的分支）
  const protocol = match[1] as string
  const hostPort = match[2] as string
  const rest = match[3] as string

  // 拆 host 与 port，兼容 IPv6 字面量（[::1]:7001）
  let host = hostPort
  let port = ''
  if (hostPort.startsWith('[')) {
    const close = hostPort.indexOf(']')
    if (close < 0) return null
    host = hostPort.slice(0, close + 1)
    const after = hostPort.slice(close + 1)
    if (after.startsWith(':')) port = after.slice(1)
  } else {
    const idx = hostPort.lastIndexOf(':')
    if (idx >= 0) {
      host = hostPort.slice(0, idx)
      port = hostPort.slice(idx + 1)
    }
  }

  if (!host) return null
  return { origin: `${protocol}${hostPort}`, protocol, host, port, rest }
}

// 实测可用的端口（模块级共享，跨组件/跨会话生效）
export const learnedMediaPort = ref('')

// 已验证过端口的主机：同一主机下再失败，说明是资源本身的问题（404/删除），
// 不应再遍历端口，避免一张不存在的图触发十几次请求。
export const learnedMediaHost = ref('')

export const learnMediaPort = (port: string, host?: string): void => {
  const p = String(port || '').trim()
  if (!p) return
  learnedMediaPort.value = p
  learnedMediaHost.value = String(host || '').trim()
}

export const resetLearnedMediaPort = (): void => {
  learnedMediaPort.value = ''
  learnedMediaHost.value = ''
}

// 把上游媒体地址的端口替换为指定端口；非上游地址原样返回。
export const withMediaPort = (input: string, port: string): string => {
  const parsed = parseUpstreamMediaUrl(input)
  const p = String(port || '').trim()
  if (!parsed || !p || parsed.port === p) return input
  return `${parsed.protocol}${parsed.host}:${p}${parsed.rest}`
}

// 若已探测到可用端口，把地址切换到该端口；否则原样返回。
// 只对同一主机生效：不同主机的媒体服务端口未必相同，不能盲目套用。
export const applyLearnedMediaPort = (input: string): string => {
  const port = learnedMediaPort.value
  if (!port) return input
  const parsed = parseUpstreamMediaUrl(input)
  if (!parsed) return input
  const host = learnedMediaHost.value
  if (host && parsed.host !== host) return input
  return withMediaPort(input, port)
}

// 给定当前地址与已尝试过的端口，返回下一个候选地址；没有可用候选时返回 null。
// triedPorts 会被就地更新，调用方无需重复处理。
export const nextMediaUrlCandidate = (input: string, triedPorts: Set<string>): string | null => {
  const parsed = parseUpstreamMediaUrl(input)
  if (!parsed) return null

  const learned = learnedMediaPort.value
  const learnedHost = learnedMediaHost.value

  // 该主机的端口已被验证过：当前请求本身失败（404/资源被删），换端口没有意义
  if (learned && learnedHost && parsed.host === learnedHost && parsed.port === learned) {
    return null
  }

  if (parsed.port) triedPorts.add(parsed.port)

  // 换了主机时不要套用其他主机学到的端口
  const learnedHostApplies = learnedHost === '' || parsed.host === learnedHost
  const canUseLearned = !!learned && learnedHostApplies
  const ordered: string[] = canUseLearned && !triedPorts.has(learned)
    ? [learned, ...MEDIA_PORT_CANDIDATES]
    : [...MEDIA_PORT_CANDIDATES]

  for (const port of ordered) {
    if (triedPorts.has(port)) continue
    triedPorts.add(port)
    return `${parsed.protocol}${parsed.host}:${port}${parsed.rest}`
  }

  return null
}
