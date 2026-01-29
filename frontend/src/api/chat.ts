import request, { createFormData } from './request'
import type { ApiResponse, User } from '@/types'

// 获取历史用户列表（POST请求，urlencoded）
export const getHistoryUserList = (myUserID: string, cookieData: string, referer: string, userAgent: string) => {
  const formData = createFormData({
    myUserID,
    vipcode: '',
    serverPort: '1001',
    cookieData,
    referer,
    userAgent
  })
  return request.post<any, any>('/getHistoryUserList', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 获取收藏用户列表（POST请求，urlencoded）
export const getFavoriteUserList = (myUserID: string, cookieData: string, referer: string, userAgent: string) => {
  const formData = createFormData({
    myUserID,
    vipcode: '',
    serverPort: '1001',
    cookieData,
    referer,
    userAgent
  })
  return request.post<any, any>('/getFavoriteUserList', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 添加收藏（urlencoded）
export const toggleFavorite = (myUserID: string, UserToID: string, cookieData: string, referer: string, userAgent: string) => {
  const formData = createFormData({
    myUserID,
    UserToID,
    vipcode: '',
    serverPort: '1001',
    cookieData,
    referer,
    userAgent
  })
  return request.post<any, ApiResponse>('/toggleFavorite', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 取消收藏（urlencoded）
export const cancelFavorite = (myUserID: string, UserToID: string, cookieData: string, referer: string, userAgent: string) => {
  const formData = createFormData({
    myUserID,
    UserToID,
    vipcode: '',
    serverPort: '1001',
    cookieData,
    referer,
    userAgent
  })
  return request.post<any, ApiResponse>('/cancelFavorite', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 获取聊天历史（POST请求，urlencoded）
// 后端参数：isFirst(1=首次,0=加载更多)、firstTid(首次=0，加载更多传最早Tid)
export const getMessageHistory = (
  myUserID: string,
  UserToID: string,
  isFirst: '0' | '1',
  firstTid: string,
  cookieData: string,
  referer: string,
  userAgent: string
) => {
  const formData = createFormData({
    myUserID,
    UserToID,
    isFirst,
    firstTid,
    vipcode: '',
    serverPort: '1001',
    cookieData,
    referer,
    userAgent
  })
  return request.post<any, any>('/getMessageHistory', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 上报访问记录（urlencoded）
export const reportReferrer = (data: {
  referrerUrl: string
  currUrl: string
  userid: string
  cookieData: string
  referer: string
  userAgent: string
}) => {
  const formData = createFormData(data)
  return request.post<any, any>('/reportReferrer', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 获取图片服务器地址
export const getImgServer = () => {
  return request.get<any, any>('/getImgServer')
}

// 更新图片服务器地址（urlencoded）
export const updateImgServer = (server: string) => {
  const formData = createFormData({ server })
  return request.post<any, ApiResponse>('/updateImgServer', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 获取缓存图片
export const getCachedImages = (userid: string) => {
  return request.get<any, any>('/getCachedImages', { params: { userid } })
}

// 删除上游用户
export const deleteUser = (myUserId: string, userToId: string) => {
  const formData = createFormData({
    myUserId,
    userToId
  })
  return request.post<any, ApiResponse>('/deleteUpstreamUser', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 批量删除上游用户（JSON）
export const batchDeleteUsers = (myUserId: string, userToIds: string[]) => {
  return request.post<any, ApiResponse>('/batchDeleteUpstreamUsers', { myUserId, userToIds })
}
