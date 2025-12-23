export const API_BASE = '/api'
export const WS_URL = '/ws'
// 上游媒体服务器端口（旧版固定端口策略）
export const IMG_SERVER_IMAGE_PORT = 9006
export const IMG_SERVER_VIDEO_PORT = 8006

// 图片服务器地址（运行时从后端获取）
export let IMG_SERVER_ADDRESS = ''

export const setImgServerAddress = (address: string) => {
  IMG_SERVER_ADDRESS = address
}
