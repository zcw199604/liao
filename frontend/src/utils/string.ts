// 解析表情符号
export const parseEmoji = (text: string, emojiMap: Record<string, string>): string => {
  if (!text) return ''
  let result = text
  for (const [key, value] of Object.entries(emojiMap)) {
    const imgTag = `<img src="${value}" class="inline-block w-6 h-6" alt="${key}">`
    result = result.replace(new RegExp(escapeRegex(key), 'g'), imgTag)
  }
  return result
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
