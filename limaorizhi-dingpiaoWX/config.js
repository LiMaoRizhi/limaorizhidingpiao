// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声

// 小程序环境配置
// 开发环境: https://yiban.yiban.asia （走生产HTTPS域名，微信域名校验要求）
//   如需连本地后端 localhost:8181 调试：
//   微信开发者工具 → 详情 → 本地设置 → 勾选「不校验合法域名」
//   然后将下方 dev.baseURL 改回 http://localhost:8181
// ⚠️ 注意：dev.baseURL 当前指向生产域名，开发操作会直接影响生产数据！
//    涉及写操作的调试（下单/退款/核销等）请改回 localhost 避免误操作
// 生产环境: https://yiban.yiban.asia （必须HTTPS，微信要求）
const config = {
  // 开发环境（当前走生产HTTPS域名，避免微信域名校验失败）
  // 调试写操作时请改回 http://localhost:8181
  dev: {
    baseURL: 'https://yiban.yiban.asia'
  },
  // 生产环境（统一域名，必须HTTPS，微信要求）
  prod: {
    baseURL: 'https://yiban.yiban.asia'
  }
}

// 自动检测环境：开发工具中为 develop，体验版/正式版用生产环境
let env = 'dev'
try {
  const accountInfo = wx.getAccountInfoSync()
  const envVersion = accountInfo.miniProgram.envVersion
  if (envVersion === 'release' || envVersion === 'trial') {
    env = 'prod'
  }
} catch (e) {
  // 降级为开发环境
}

// 生产环境域名校验，防止误用http或占位域名导致请求失败/不安全
if (env === 'prod') {
  const url = config.prod.baseURL
  if (url.indexOf('https://') !== 0) {
    console.error('[安全警告] 生产环境baseURL必须为https，当前为: ' + url + '，请在config.js修改为实际HTTPS域名')
  }
  if (url.indexOf('localhost') !== -1 || url.indexOf('your-domain') !== -1) {
    console.warn('[配置警告] 生产环境baseURL仍为占位域名，请修改为实际域名: ' + url)
  }
}

// 开发环境安全提示：dev.baseURL指向非localhost时警告，防止误操作生产数据
if (env === 'dev') {
  const devUrl = config.dev.baseURL
  if (devUrl.indexOf('localhost') === -1 && devUrl.indexOf('127.0.0.1') === -1) {
    console.warn('[安全警告] 开发环境baseURL指向非本地地址(' + devUrl + ')，您的操作将直接影响生产数据！' +
      ' 调试写操作（下单/退款/核销等）请改回 http://localhost:8181')
  }
}

module.exports = config[env]
