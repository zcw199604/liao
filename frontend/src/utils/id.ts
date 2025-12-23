// 生成Session ID
export const generateSessionId = (): string => {
  return Date.now().toString(36) + Math.random().toString(36).substring(2)
}

// 生成随机IP地址
export const generateRandomIP = (): string => {
  return Array.from({ length: 4 }, () =>
    Math.floor(Math.random() * 256)
  ).join('.')
}

// 生成唯一ID
export const generateUniqueId = (): string => {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`
}

// 生成随机32位十六进制ID（用于 randomOut 等消息）
export const generateRandomHexId = (length: number = 32): string => {
  const bytesLength = Math.ceil(length / 2)
  const bytes = new Uint8Array(bytesLength)

  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < bytes.length; i++) {
      bytes[i] = Math.floor(Math.random() * 256)
    }
  }

  return Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('').slice(0, length)
}
