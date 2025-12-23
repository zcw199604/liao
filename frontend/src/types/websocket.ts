export interface WebSocketMessage {
  code: number
  content?: string
  fromuser?: any
  touser?: any
  act?: string
  time?: string
  tid?: string
  type?: string
  forceout?: boolean
}

export interface SignMessage {
  act: string
  cookie: string
  userInfo: {
    id: string
    nickname: string
    sex: string
  }
}
