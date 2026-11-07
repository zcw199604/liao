import request, { createFormData } from './request'

// 匹配偏好（期望对方性别/年龄区间）
// 说明：目前只有「晚风线路」真正支持；chat2019 线路接口会返回 supported=false，
// 前端据此隐藏入口，避免用户设了却看不到效果。
export interface MatchPreference {
  identityId: string
  // male / female / any
  preferGender: string
  // 0 表示不限
  preferAgeMin: number
  preferAgeMax: number
  preferSameProvince: boolean
}

export interface MatchPreferenceResponse {
  code: number
  data: MatchPreference
  supported: boolean
}

// 读取某个身份的匹配偏好
export const getMatchPreference = (identityId: string) => {
  return request.get<any, MatchPreferenceResponse>('/getMatchPreference', {
    params: { identityId }
  })
}

// 更新匹配偏好（晚风线路会立即下发生效）
export const updateMatchPreference = (pref: MatchPreference) => {
  const formData = createFormData({
    identityId: pref.identityId,
    preferGender: pref.preferGender,
    preferAgeMin: String(pref.preferAgeMin ?? 0),
    preferAgeMax: String(pref.preferAgeMax ?? 0),
    preferSameProvince: pref.preferSameProvince ? '1' : '0'
  })
  return request.post<any, { code: number; data: MatchPreference; msg?: string }>(
    '/updateMatchPreference',
    formData,
    { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }
  )
}
