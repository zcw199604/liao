export interface User {
  id: string
  name: string  // 改为name，和后端一致
  nickname: string  // 保留nickname作为别名
  sex: string
  age?: string
  area?: string
  address?: string  // 后端可能用address
  ip: string
  isFavorite?: boolean
  lastMsg?: string
  lastTime?: string
  unreadCount?: number
  lastMessageTime?: string
}

export interface Identity {
  id: string
  name: string  // 后端返回name
  sex: string
  color?: string
  created_at?: string
  createdAt?: string  // 可能的驼峰命名
}

export interface CurrentUser {
  id: string
  name: string
  nickname: string  // 用于显示
  sex: string
  color: string
  created_at: string
  cookie: string
  ip: string
  area: string
}
