// 转义 HTML 特殊字符：消息内容来自匿名对端，必须先转义再交给 v-html
export const escapeHtml = (text: string): string => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// 识别 http(s) 链接；末尾的常见标点不算作链接的一部分
const URL_PATTERN = /https?:\/\/[^\s<>"'，。！？；：、（）【】「」]+/g
const URL_TRAILING_PUNCT = /[.,!?;:)\]}'"]+$/

// 按 emojiMap 缓存合并后的匹配正则（长 key 优先，避免前缀冲突）
const emojiPatternCache = new WeakMap<Record<string, string>, RegExp | null>()
const getEmojiPattern = (emojiMap: Record<string, string>): RegExp | null => {
  if (emojiPatternCache.has(emojiMap)) return emojiPatternCache.get(emojiMap) ?? null
  const keys = Object.keys(emojiMap).filter(Boolean).sort((a, b) => b.length - a.length)
  const pattern = keys.length ? new RegExp(keys.map(escapeRegex).join('|'), 'g') : null
  emojiPatternCache.set(emojiMap, pattern)
  return pattern
}

// 普通文本片段：转义 HTML 并把表情 key 替换为图片
const renderTextWithEmoji = (text: string, emojiMap: Record<string, string>): string => {
  const pattern = getEmojiPattern(emojiMap)
  if (!pattern) return escapeHtml(text)
  let result = ''
  let last = 0
  pattern.lastIndex = 0
  for (let m = pattern.exec(text); m; m = pattern.exec(text)) {
    result += escapeHtml(text.slice(last, m.index))
    const key = m[0]
    result += `<img src="${escapeHtml(emojiMap[key] as string)}" class="inline-block w-6 h-6" alt="${escapeHtml(key)}">`
    last = m.index + key.length
  }
  return result + escapeHtml(text.slice(last))
}

// 渲染消息文本为安全 HTML：转义 → 链接可点击 → 表情替换为图片
export const parseEmoji = (text: string, emojiMap: Record<string, string>): string => {
  if (!text) return ''
  let result = ''
  let last = 0
  URL_PATTERN.lastIndex = 0
  for (let m = URL_PATTERN.exec(text); m; m = URL_PATTERN.exec(text)) {
    // URL 以 http 开头，去掉末尾标点后仍非空
    const url = m[0].replace(URL_TRAILING_PUNCT, '')
    result += renderTextWithEmoji(text.slice(last, m.index), emojiMap)
    const safeUrl = escapeHtml(url)
    result += `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer" class="underline break-all">${safeUrl}</a>`
    last = m.index + url.length
  }
  return result + renderTextWithEmoji(text.slice(last), emojiMap)
}

// 转义正则表达式特殊字符
export const escapeRegex = (str: string): string => {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 截断文本
export const truncate = (text: string, maxLength: number): string => {
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

// 生成随机字符串
export const randomString = (length: number): string => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}
