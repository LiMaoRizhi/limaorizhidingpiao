/**
 * 日志工具 — 生产环境自动静默，开发环境输出到控制台
 * 替代散落在各页面的 console.error / console.warn，统一管理日志输出
 * 用法：
 *   const log = require('../../utils/log')
 *   log.error('加载失败', e)
 *   log.warn('降级处理', e)
 */

// 判断当前环境：develop=开发版, trial=体验版, release=正式版
function getEnv() {
  try {
    // 微信基础库 2.10+ 可通过 __wxConfig 获取环境版本
    if (typeof __wxConfig !== 'undefined') {
      return __wxConfig.envVersion || 'release'
    }
  } catch (e) {}
  return 'release'
}

var isDev = getEnv() === 'develop'

module.exports = {
  error: function () {
    if (isDev) {
      console.error.apply(console, arguments)
    }
  },
  warn: function () {
    if (isDev) {
      console.warn.apply(console, arguments)
    }
  },
  log: function () {
    if (isDev) {
      console.log.apply(console, arguments)
    }
  }
}
