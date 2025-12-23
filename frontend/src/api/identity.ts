import request, { createFormData } from './request'
import type { ApiResponse, Identity } from '@/types'

// 获取身份列表
export const getIdentityList = () => {
  return request.get<any, ApiResponse<Identity[]>>('/getIdentityList')
}

// 创建身份（urlencoded）
export const createIdentity = (data: { name: string; sex: string }) => {
  const formData = createFormData(data)
  return request.post<any, ApiResponse>('/createIdentity', formData, {
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    }
  })
}

// 删除身份（urlencoded）
export const deleteIdentity = (id: string) => {
  const formData = createFormData({ id })
  return request.post<any, ApiResponse>('/deleteIdentity', formData, {
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    }
  })
}

// 选择身份（urlencoded）
export const selectIdentity = (id: string) => {
  return request.post<any, ApiResponse>('/selectIdentity', null, {
    params: { id }
  })
}

// 更新身份信息（urlencoded）
export const updateIdentity = (data: { id: string; name: string; sex: string }) => {
  const formData = createFormData(data)
  return request.post<any, ApiResponse>('/updateIdentity', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

// 更新身份ID（urlencoded）
export const updateIdentityId = (data: { oldId: string; newId: string; name: string; sex: string }) => {
  const formData = createFormData(data)
  return request.post<any, ApiResponse>('/updateIdentityId', formData, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}
