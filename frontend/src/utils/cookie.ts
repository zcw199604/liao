// 生成Cookie字符串
export const generateCookie = (userid: string, nickname: string): string => {
  const timestamp = Math.floor(Date.now() / 1000)
  const randomStr = Math.random().toString(36).substring(2, 8)
  return `${userid}_${nickname}_${timestamp}_${randomStr}`
}

// 解析Cookie字符串
export const parseCookie = (cookie: string): { userid: string; nickname: string } | null => {
  const parts = cookie.split('_')
  if (parts.length >= 2) {
    return {
      userid: parts[0] || '',
      nickname: parts[1] || ''
    }
  }
  return null
}
