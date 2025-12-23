// 格式化时间为相对时间
export const formatTime = (timeStr: string): string => {
  if (!timeStr) return ''

  // 上游常见格式：2025-12-18 03:02:11.721 → 2025-12-18 03:02:11
  const match = timeStr.match(/(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2})/)
  if (match) return `${match[1]} ${match[2]}`

  const date = new Date(timeStr)
  if (Number.isNaN(date.getTime())) return timeStr
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`

  const month = date.getMonth() + 1
  const day = date.getDate()
  const hour = date.getHours().toString().padStart(2, '0')
  const minute = date.getMinutes().toString().padStart(2, '0')

  return `${month}月${day}日 ${hour}:${minute}`
}

// 格式化完整时间
export const formatFullTime = (timeStr: string): string => {
  const date = new Date(timeStr)
  const year = date.getFullYear()
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hour = date.getHours().toString().padStart(2, '0')
  const minute = date.getMinutes().toString().padStart(2, '0')
  const second = date.getSeconds().toString().padStart(2, '0')

  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}
