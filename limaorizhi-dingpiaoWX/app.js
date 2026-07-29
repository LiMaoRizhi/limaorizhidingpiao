// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声

const { request, driverRequest } = require('./utils/request')
const log = require('./utils/log')

App({
  globalData: {
    userInfo: null,
    isLoggedIn: false,
    lastValidateTime: 0,
    // 全局司机定位追踪状态（不随页面销毁而停止）
    driverTracking: {
      tripId: 0,
      isTracking: false,
      reportCount: 0,
      lastReportTime: 0,
      lastLocation: null,
      locationChangeHandler: null,
      trackError: '' // 定位启动失败时的错误提示，空字符串表示正常
    }
  },
  onLaunch() {
    // 检查登录状态
    const token = wx.getStorageSync('user_token')
    const userInfo = wx.getStorageSync('user_info')
    if (token && userInfo) {
      this.globalData.userInfo = userInfo
      this.globalData.isLoggedIn = true
      // 异步验证token有效性（不阻塞启动，token过期时由统一请求封装处理跳转）
      this.validateToken()
    }
  },
  // onShow 回到前台时重新校验登录态，防止后台长期挂起后token过期未感知
  // 节流：距上次校验超过5分钟才再次校验，避免频繁请求
  onShow() {
    const token = wx.getStorageSync('user_token')
    if (!token) return
    const now = Date.now()
    if (now - this.globalData.lastValidateTime > 5 * 60 * 1000) {
      this.validateToken()
    }
  },
  // 全局司机定位追踪
  // 定位逻辑挂在 app 实例上，页面销毁不影响定位持续运行
  // 司机点"开始行程"→ 调用 startDriverTracking → 后台持续上报
  // 司机可自由切换页面去核销，定位不受影响
  // 到站后点"结束行程"→ 调用 stopDriverTracking → 停止定位

  // 开始行程追踪（调用前需先请求后端 PUT /driver/trips/:id/start 改班次状态为已发车）
  startDriverTracking(tripId) {
    const t = this.globalData.driverTracking
    // 已在追踪同一班次，不重复启动
    if (t.isTracking && t.tripId === tripId) return
    // 如在追踪其他班次，先停止
    if (t.isTracking) this.stopDriverTracking()

    t.tripId = tripId
    t.isTracking = true
    t.reportCount = 0
    t.lastReportTime = 0
    t.lastLocation = null
    t.trackError = ''

    // 先获取一次位置 + 首次上报
    wx.getLocation({
      type: 'gcj02',
      success: (res) => {
        // 定位回调时追踪可能已被停止，需二次检查避免竞态
        if (!t.isTracking) return
        this.reportDriverLocation(res)
        this.startContinuousLocation()
      },
      fail: () => {
        if (!t.isTracking) return
        this.startContinuousLocation()
      }
    })
  },

  // 开始持续监听位置变化（优先后台定位，降级前台定位）
  startContinuousLocation() {
    const t = this.globalData.driverTracking
    wx.startLocationUpdateBackground({
      success: () => {
        t.locationChangeHandler = (res) => { this.onDriverLocationChange(res) }
        wx.onLocationChange(t.locationChangeHandler)
      },
      fail: () => {
        // 后台定位不可用，降级为前台持续定位
        wx.startLocationUpdate({
          success: () => {
            t.locationChangeHandler = (res) => { this.onDriverLocationChange(res) }
            wx.onLocationChange(t.locationChangeHandler)
          },
          fail: () => {
            t.trackError = '定位启动失败，请检查位置权限是否开启'
            log.error('定位启动失败，请检查位置权限')
          }
        })
      }
    })
  },

  // 位置变化回调（节流：至少间隔10秒上报一次，减少流量和耗电）
  onDriverLocationChange(res) {
    const t = this.globalData.driverTracking
    if (!t.isTracking) return
    const now = Date.now()
    if (now - t.lastReportTime < 10000) return
    t.lastReportTime = now
    t.lastLocation = res
    this.reportDriverLocation(res)
  },

  // 上报位置到服务器
  reportDriverLocation(res) {
    const t = this.globalData.driverTracking
    driverRequest({
      url: '/api/wx/driver/location',
      method: 'POST',
      data: {
        trip_id: parseInt(t.tripId),
        longitude: res.longitude,
        latitude: res.latitude,
        speed: res.speed || 0,
        heading: res.heading || 0
      }
    }).then(() => {
      t.reportCount++
    }).catch((e) => { log.error('上报位置失败', e) })
  },

  // 停止行程追踪（到站后司机手动结束行程时调用）
  stopDriverTracking() {
    const t = this.globalData.driverTracking
    if (t.locationChangeHandler) {
      wx.offLocationChange(t.locationChangeHandler)
      t.locationChangeHandler = null
    }
    wx.stopLocationUpdate({ fail: () => {} })
    t.isTracking = false
    t.trackError = ''
  },

  // 获取当前追踪状态（供页面读取展示）
  getDriverTrackingStatus() {
    return this.globalData.driverTracking
  },

  // 改用统一请求封装，复用 token 自动注入、1002 跳转登录等逻辑
  validateToken() {
    const token = wx.getStorageSync('user_token')
    if (!token) return
    this.globalData.lastValidateTime = Date.now()
    request({ url: '/api/wx/user', method: 'GET' }).then(data => {
      // token有效，更新缓存
      this.globalData.userInfo = data.data
      wx.setStorageSync('user_info', data.data)
    }).catch(() => {
      // request.js 已处理错误（含 1002 自动跳转登录），此处无需额外处理
      // 网络错误时保留本地缓存状态（离线降级）
    })
  }
})
