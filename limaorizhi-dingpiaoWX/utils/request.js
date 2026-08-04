
// 从配置文件读取 BASE_URL，支持多环境切换
const { baseURL: BASE_URL } = require('../config.js')
const log = require('./log')

function getUserToken() {
  return wx.getStorageSync('user_token') || ''
}

function getDriverToken() {
  return wx.getStorageSync('driver_token') || ''
}

// 防止Token失效时多次跳转登录页（用户端和司机端独立控制）
let isUserRedirecting = false
let isDriverRedirecting = false

// 记录最近一次登录成功的时间戳
// 登录成功后5秒内的1002响应视为旧请求残留，跳过重定向避免登录死循环
// 5月16号上线第一天就遇到登录死循环，token一过期就跳登录页，跳完又报1002又跳，转圈圈
let lastLoginAt = 0

// 设置登录成功时间（供 login.js 调用）
function markLoginSuccess() {
  lastLoginAt = Date.now()
}

// 检查当前是否在登录页（避免在登录页收到1002时重复跳转）
function isOnLoginPage() {
  var pages = getCurrentPages()
  if (!pages || pages.length === 0) return false
  var current = pages[pages.length - 1]
  return current && current.route === 'pages/login/login'
}

// 处理用户端Token失效（1002）：三重防护避免登录死循环
// 1. 无token（未登录）不跳转，仅拒绝请求
// 2. 当前已在登录页不跳转（避免自跳转）
// 3. 登录成功后5秒内的1002视为旧请求残留，跳过重定向
// 6月18号又改过一次，之前只在登录页判断，没考虑刚登录完旧请求还在飞的竞态
function handleUserTokenExpired(token, data, reject) {
  if (token) {
    if (lastLoginAt > 0 && Date.now() - lastLoginAt < 5000) {
      reject(data)
      return
    }
    if (isOnLoginPage()) {
      reject(data)
      return
    }
    wx.removeStorageSync('user_token')
    wx.removeStorageSync('user_info')
    // 同步重置全局登录态，避免依赖 globalData 的页面误判为已登录
    const app = getApp()
    if (app && app.globalData) {
      app.globalData.userInfo = null
      app.globalData.isLoggedIn = false
    }
    if (!isUserRedirecting) {
      isUserRedirecting = true
      wx.reLaunch({
        url: '/pages/login/login',
        complete: () => { isUserRedirecting = false }
      })
    }
  }
  reject(data)
}

// 统一请求函数（用户端）
function request(options) {
  return new Promise((resolve, reject) => {
    const token = getUserToken()
    wx.request({
      url: BASE_URL + options.url,
      method: options.method || 'GET',
      data: options.data || {},
      timeout: options.timeout || 15000,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : '',
        ...(options.header || {}) // 支持自定义 header（与 driverRequest 一致）
      },
      success(res) {
        const data = res.data || {}
        if (res.statusCode === 200 && data.code === 0) {
          resolve(data)
        } else if (data.code === 1002) {
          handleUserTokenExpired(token, data, reject)
        } else {
          // silent 模式不弹 toast，由调用方自行处理错误展示
          if (!options.silent) {
            wx.showToast({
              title: data.message || '请求失败',
              icon: 'none'
            })
          }
          reject(data)
        }
      },
      fail(err) {
        if (!options.silent) {
          wx.showToast({ title: '网络错误', icon: 'none' })
        }
        reject(err)
      }
    })
  })
}

// 文件上传函数（用户端，封装 wx.uploadFile）
function upload(options) {
  return new Promise((resolve, reject) => {
    const token = getUserToken()
    wx.uploadFile({
      url: BASE_URL + options.url,
      filePath: options.filePath,
      name: options.name || 'file',
      formData: options.formData || {},
      header: {
        'Authorization': token ? 'Bearer ' + token : ''
      },
      success(res) {
        let data = {}
        try { data = res.data ? JSON.parse(res.data) : {} } catch (e) { log.warn('upload响应非JSON', e) }
        if (res.statusCode === 200 && data.code === 0) {
          resolve(data)
        } else if (data.code === 1002) {
          handleUserTokenExpired(token, data, reject)
        } else {
          wx.showToast({
            title: data.message || '上传失败',
            icon: 'none'
          })
          reject(data)
        }
      },
      fail(err) {
        wx.showToast({ title: '网络错误', icon: 'none' })
        reject(err)
      }
    })
  })
}

// 司机端请求函数
function driverRequest(options) {
  return new Promise((resolve, reject) => {
    const token = getDriverToken()
    wx.request({
      url: BASE_URL + options.url,
      method: options.method || 'GET',
      data: options.data || {},
      timeout: 15000,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : '',
        ...(options.header || {}) // 支持自定义 header（如登录时传 X-User-Token）
      },
      success(res) {
        const data = res.data || {}
        if (res.statusCode === 200 && data.code === 0) {
          resolve(data)
        } else if (data.code === 1002) {
          wx.removeStorageSync('driver_token')
          wx.removeStorageSync('driver_info')
          if (!isDriverRedirecting) {
            isDriverRedirecting = true
            wx.reLaunch({
              url: '/pages/verify/verify',
              complete: () => { isDriverRedirecting = false }
            })
          }
          reject(data)
        } else {
          // silent 模式不弹 toast，由调用方自行处理错误展示（如核销结果对话框）
          if (!options.silent) {
            wx.showToast({
              title: data.message || '请求失败',
              icon: 'none'
            })
          }
          reject(data)
        }
      },
      fail(err) {
        if (!options.silent) {
          wx.showToast({ title: '网络错误', icon: 'none' })
        }
        reject(err)
      }
    })
  })
}

// 看看登录木有
function checkLogin() {
  const token = getUserToken()
  if (!token) {
    wx.navigateTo({ url: '/pages/login/login' })
    return false
  }
  return true
}

module.exports = { request, upload, driverRequest, BASE_URL, checkLogin, markLoginSuccess }
