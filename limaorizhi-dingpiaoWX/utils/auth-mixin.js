// 登录退出/注销账号工具（mine.js / profile.js 共用）

const { request } = require('./request')

/**
 * 清除本地登录状态（退出登录/注销账号共用）
 * 清除 storage 中的 token 和用户信息，重置 globalData
 */
function clearLocalAuth() {
  wx.removeStorageSync('user_token')
  wx.removeStorageSync('user_info')
  const app = getApp()
  app.globalData.userInfo = null
  app.globalData.isLoggedIn = false
}

/**
 * 退出登录：调后端登出API → 清除本地
 * 后端调用失败时仍然清除本地（保证用户端一定能退出）
 * @returns {Promise}
 */
function logout() {
  return request({ url: '/api/wx/logout', method: 'POST' }).catch(function () {
    // 即使后端调用失败也继续清除本地
  }).then(function () {
    clearLocalAuth()
  })
}

/**
 * 注销账号：调后端删除账号API → 清除本地
 * @returns {Promise} 成功时resolve，失败时reject（调用方需catch处理错误提示）
 */
function deleteAccount() {
  return request({ url: '/api/wx/account', method: 'DELETE' }).then(function () {
    clearLocalAuth()
  })
}

module.exports = {
  clearLocalAuth: clearLocalAuth,
  logout: logout,
  deleteAccount: deleteAccount
}
