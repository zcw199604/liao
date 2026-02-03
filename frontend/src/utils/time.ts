const safariSafeDate = (timeStr: string): string => {
  // Only replace the leading "YYYY-MM-DD" part, so we don't break timezone offsets like "-08:00".
  return timeStr.replace(/^(\d{4})-(\d{2})-(\d{2})(?=[T ]|$)/, '$1/$2/$3')
}

// 格式化时间为相对时间
export const formatTime = (timeStr: string): string => {
  if (!timeStr) return ''

  let date = new Date(timeStr)
  // 兼容 iOS/Safari 的日期格式 (yyyy-MM-dd)
  if (Number.isNaN(date.getTime())) {
    date = new Date(safariSafeDate(timeStr))
  }
  if (Number.isNaN(date.getTime())) return timeStr

  const now = new Date()
  
  // 获取当天0点时间戳
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  // 获取昨天0点时间戳
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  // 获取今年1月1日0点时间戳
  const thisYear = new Date(now.getFullYear(), 0, 1)

  const targetTime = date.getTime()
  
  // 1. 今天：显示 HH:mm
  if (targetTime >= today.getTime()) {
    const hour = date.getHours().toString().padStart(2, '0')
    const minute = date.getMinutes().toString().padStart(2, '0')
    return `${hour}:${minute}`
  }

  // 2. 昨天：显示 昨天
  if (targetTime >= yesterday.getTime()) {
    return '昨天'
  }

  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')

  // 3. 今年其他时间：显示 MM/DD
  if (targetTime >= thisYear.getTime()) {
    return `${month}/${day}`
  }

  // 4. 非今年：显示 YY/MM/DD
  const year = date.getFullYear().toString().slice(-2)
  return `${year}/${month}/${day}`
}

// 格式化完整时间
export const formatFullTime = (timeStr: string): string => {
  if (!timeStr) return ''

  let date = new Date(timeStr)
  // 兼容 iOS/Safari 的日期格式 (yyyy-MM-dd)
  if (Number.isNaN(date.getTime())) {
    date = new Date(safariSafeDate(timeStr))
  }
  if (Number.isNaN(date.getTime())) return timeStr

  const year = date.getFullYear()
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hour = date.getHours().toString().padStart(2, '0')
  const minute = date.getMinutes().toString().padStart(2, '0')
  const second = date.getSeconds().toString().padStart(2, '0')

  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}
