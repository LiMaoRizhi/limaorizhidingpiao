// 纯格式化工具函数（跨页面共用，避免复制粘贴）

const { BASE_URL } = require('./request')

/**
 * 安全提取日期字符串的纯日期部分（前10位）
 * 兼容 "2026-07-21" 和 "2026-07-21T00:00:00+08:00" 两种格式
 * @param {string} str
 * @returns {string} 如 "2026-07-21"
 */
function safeDate(str) {
  if (!str) return ''
  return str.length > 10 ? str.substring(0, 10) : str
}

/**
 * 日期对象 → "YYYY-MM-DD" 字符串
 * @param {Date} d
 * @returns {string} 如 "2026-07-27"
 */
function formatDate(d) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/**
 * 相对 URL → 完整网络 URL
 * 小程序 image 组件需要完整 URL 才能加载网络图片
 * @param {string} url
 * @returns {string}
 */
function normalizeUrl(url) {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  return BASE_URL + url
}

/**
 * 手机号脱敏（中间4位替换为****）
 * @param {string} phone
 * @returns {string} 如 "138****1234"
 */
function maskPhone(phone) {
  if (!phone || phone.length < 7) return phone
  return phone.substring(0, 3) + '****' + phone.substring(phone.length - 4)
}

/**
 * 校验中国大陆手机号
 * @param {string} phone
 * @returns {boolean}
 */
function validatePhone(phone) {
  return /^1[3-9]\d{9}$/.test(phone)
}

/**
 * 速度格式化（m/s → km/h 文本）
 * @param {number} speed m/s
 * @returns {string} 如 "60 km/h" 或 "静止"
 */
function formatSpeed(speed) {
  if (speed > 0) return (speed * 3.6).toFixed(0) + ' km/h'
  return '静止'
}

/**
 * 上报时间格式化 → "MM-DD HH:mm"（含日期，便于判断时效）
 * @param {string} timeStr ISO格式如 "2026-07-27T14:30:00"
 * @returns {string} 如 "07-27 14:30"
 */
function formatReportTime(timeStr) {
  if (!timeStr) return ''
  var s = timeStr.replace('T', ' ')
  if (s.length < 16) return ''
  return s.substring(5, 16)
}

/**
 * 格式化ISO时间为 "YYYY-MM-DD HH:MM:SS"（兼容 "T" 分隔符）
 * @param {string} timeStr ISO格式如 "2026-07-27T14:30:00.395+08:00"
 * @returns {string} 如 "2026-07-27 14:30:00"
 */
function formatDateTime(timeStr) {
  if (!timeStr) return ''
  var s = timeStr.replace('T', ' ')
  return s.length >= 19 ? s.substring(0, 19) : s
}

/**
 * 获取当前时间字符串 "HH:MM"
 * @returns {string} 如 "14:30"
 */
function formatCurrentTime() {
  var now = new Date()
  var h = now.getHours()
  var m = now.getMinutes()
  var hh = h < 10 ? '0' + h : '' + h
  var mm = m < 10 ? '0' + m : '' + m
  return hh + ':' + mm
}

module.exports = {
  safeDate: safeDate,
  formatDate: formatDate,
  normalizeUrl: normalizeUrl,
  maskPhone: maskPhone,
  validatePhone: validatePhone,
  formatSpeed: formatSpeed,
  formatReportTime: formatReportTime,
  formatDateTime: formatDateTime,
  formatCurrentTime: formatCurrentTime
}
