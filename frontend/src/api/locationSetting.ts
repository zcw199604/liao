import request, { createFormData } from './request'

// 位置（城市）公开设置
// 说明：晚风线路按访客 IP 归属地给身份打城市，匹配成功时会把城市下发给对面；
// 我方默认「不公开」，用户可在这里改成公开。chat2019 线路不在我方控制范围内，
// 接口会返回 supported=false（该线路等同固定不公开），前端据此禁用开关。
export interface LocationSetting {
  identityId: string
  // false = 不公开（默认），true = 公开
  showCity: boolean
}

export interface LocationSettingResponse {
  code: number
  data: LocationSetting
  supported: boolean
}

// 读取某个身份的位置公开设置
export const getLocationSetting = (identityId: string) => {
  return request.get<any, LocationSettingResponse>('/getLocationSetting', {
    params: { identityId }
  })
}

// 更新位置公开设置（晚风线路已连接时立即下发，否则下次连接生效）
export const updateLocationSetting = (setting: Pick<LocationSetting, 'identityId' | 'showCity'>) => {
  const formData = createFormData({
    identityId: setting.identityId,
    showCity: setting.showCity ? '1' : '0'
  })
  return request.post<any, { code: number; data: LocationSetting; msg?: string }>(
    '/updateLocationSetting',
    formData,
    { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
  )
}
