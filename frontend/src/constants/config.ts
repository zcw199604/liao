export const API_BASE = '/api'
export const WS_URL = '/ws'
// 上游媒体服务器默认端口（fixed 模式默认值）
// 上游新版页面 Imgarr=['7001','7004']，图片与视频共用该端口；旧版 9006/8006 已不再监听。
export const IMG_SERVER_IMAGE_PORT = 7001

// 图片服务器地址（运行时从后端获取）
export let IMG_SERVER_ADDRESS = ''

export const setImgServerAddress = (address: string) => {
  IMG_SERVER_ADDRESS = address
}

// 上游接口要求携带的 Referer（与旧版页面地址保持一致）
export const UPSTREAM_REFERER = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'

// WebSocket 断线时的统一提示文案（断线后会自动重连）
export const WS_DISCONNECTED_TIP = '连接已断开，正在重连…'
