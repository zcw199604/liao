import type { User } from './user'

export interface Message {
  code: number
  fromuser: User
  touser?: User
  type?: string
  content: string
  time: string
  tid: string
  act?: string
  forceout?: boolean
}

export type MessageSegment =
  | { kind: 'text'; text: string }
  | { kind: 'image' | 'video' | 'file'; path: string; url: string }

export interface ChatMessage extends Message {
  isSelf: boolean
  isImage?: boolean
  isVideo?: boolean
  isFile?: boolean
  imageUrl?: string
  videoUrl?: string
  fileUrl?: string
  // When present, render message as "rich segments" (text + inline media).
  segments?: MessageSegment[]
}
