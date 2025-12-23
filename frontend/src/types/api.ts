export interface ApiResponse<T = any> {
  code: number
  msg?: string
  message?: string
  data?: T
  token?: string
}

export interface ConnectionStats {
  active: number
  upstream: number
  downstream: number
}

export interface PaginationResponse<T> {
  code: number
  data: {
    items: T[]
    total: number
    page: number
    pageSize: number
  }
}
