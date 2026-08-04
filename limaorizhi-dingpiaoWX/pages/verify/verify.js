var log = require('../../utils/log')

const { driverRequest: request } = require('../../utils/request')
const { formatArrivalTime } = require('../../utils/order-helper')
const { validatePhone } = require('../../utils/format')

Page({
  data: {
    isLogin: false,
    driverName: '',
    phone: '',
    password: '',
    loginLoading: false,
    trips: [],
    showManual: false,
    manualOrderNo: '',
    manualTripId: 0,
    verifyLoading: false,
    showResult: false,
    resultType: 'success',
    resultIcon: '\u2713',
    resultTitle: '',
    verifyResult: null,
    showPassengers: false,
    passengerList: [],
    passengerStats: null,
    focusedField: ''
  },

  onLoad() {
    const token = wx.getStorageSync('driver_token')
    const info = wx.getStorageSync('driver_info')
    if (token && info) {
      this.setData({ isLogin: true, driverName: info.name })
      this.loadTrips()
    }
  },

  // 空方法：用于阻止事件冒泡和滚动穿透
  noop() {},

  onFieldFocus(e) {
    this.setData({ focusedField: e.currentTarget.dataset.field })
  },
  onFieldBlur() {
    this.setData({ focusedField: '' })
  },

  onPhoneInput(e) {
    this.setData({ phone: e.detail.value })
  },

  onPasswordInput(e) {
    this.setData({ password: e.detail.value })
  },

  // 司机登录
  async handleLogin() {
    const { phone, password } = this.data
    if (!phone || !password) {
      wx.showToast({ title: '请输入手机号和密码', icon: 'none' })
      return
    }
    if (!validatePhone(phone)) {
      wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
      return
    }
    this.setData({ loginLoading: true })
    try {
      const res = await request({
        url: '/api/wx/driver/login',
        method: 'POST',
        data: { phone, password },
        // 登录时带上小程序用户token，后端会把 Driver.Phone 同步到 User.Phone，
        // 建立“司机-小程序用户”持久关联，让“我的”页 is_driver 稳定为 true
        header: { 'X-User-Token': wx.getStorageSync('user_token') || '' }
      })
      wx.setStorageSync('driver_token', res.data.token)
      wx.setStorageSync('driver_info', res.data.driver)
      this.setData({
        isLogin: true,
        driverName: res.data.driver.name,
        loginLoading: false,
        password: '' // 登录成功后清除密码
      })
      this.loadTrips()
    } catch (e) {
      this.setData({ loginLoading: false, password: '' }) // 登录失败也清除密码
    }
  },

  // 退出
  async handleLogout() {
    try {
      await request({ url: '/api/wx/driver/logout', method: 'POST' })
    } catch (e) {
      // 即使后端调用失败也继续清除本地
    }
    wx.removeStorageSync('driver_token')
    wx.removeStorageSync('driver_info')
    this.setData({
      isLogin: false,
      driverName: '',
      trips: [],
      phone: '',
      password: ''
    })
  },

  // 拉今个的班次
  async loadTrips() {
    try {
      const res = await request({
        url: '/api/wx/driver/trips',
        method: 'GET'
      })
      // 预计算每条班次的跨天到达文案
      const trips = (res.data || []).map(t => ({
        ...t,
        arrival_text: formatArrivalTime(t.arrival_time, t.arrival_day_offset)
      }))
      this.setData({ trips })
    } catch (e) {
      log.error('加载今日班次失败', e)
      wx.showToast({ title: '加载班次失败，请下拉重试', icon: 'none' })
    }
  },

  // 扫码核销
  // 扫码仅保留 qrCode，防止条形码被轻松伪造
  // 识别新版带签名凭证（LIMO|开头），验签后传给后端
  async handleScan(e) {
    const tripId = e.currentTarget.dataset.id
    wx.scanCode({
      scanType: ['qrCode'],
      success: async (res) => {
        const raw = res.result || ''
        let orderNo = ''
        let verifyToken = ''
        if (raw.startsWith('LIMO|')) {
          // 新版带签名凭证：LIMO|{orderNo}|{expireTs}|{sig}
          const parts = raw.split('|')
          if (parts.length !== 4 || !/^(DP|HY)/.test(parts[1])) {
            wx.showToast({ title: '二维码格式错误', icon: 'none' })
            return
          }
          orderNo = parts[1]
          verifyToken = raw
        } else if (raw.startsWith('DP') || raw.startsWith('HY')) {
          // 兼容旧版二维码（仅订单号，后端会记录“未验签”审计）
          orderNo = raw
        } else {
          wx.showToast({ title: '无效的二维码，请重新扫码', icon: 'none' })
          return
        }
        this.doVerify(orderNo, tripId, false, verifyToken)
      },
      fail: () => {
        wx.showToast({ title: '扫码取消', icon: 'none' })
      }
    })
  },

  // 手动输入弹窗（绑到某个班次上）
  showManualInput(e) {
    const tripId = e.currentTarget.dataset.id
    this.setData({
      showManual: true,
      manualOrderNo: '',
      manualTripId: tripId
    })
  },

  // 全局手动输入弹窗（不绑班次，走verify-by-no）
  showGlobalManualInput() {
    this.setData({
      showManual: true,
      manualOrderNo: '',
      manualTripId: 0
    })
  },

  closeManual() {
    this.setData({ showManual: false })
  },

  onManualInput(e) {
    this.setData({ manualOrderNo: e.detail.value })
  },

  // 手动输入核销
  // 增加安全提示，推荐使用扫码核销
  // 无论是否绑定班次，手动输入都计入后端"未验签每小时5次"限流，统一弹提示
  async handleManualVerify() {
    const { manualOrderNo, manualTripId } = this.data
    if (!manualOrderNo) {
      wx.showToast({ title: '请输入订单号', icon: 'none' })
      return
    }
    // 手动输入场景订单号格式校验（DP/HY + 8位日期 + 8位hex）
    if (!/^(DP|HY)[0-9]{8}[0-9a-f]{8}$/.test(manualOrderNo)) {
      wx.showToast({ title: '订单号格式不正确', icon: 'none' })
      return
    }
    wx.showModal({
      title: '安全提示',
      content: '手动输入核销每小时限5次，推荐使用扫码核销更安全。确认继续手动核销？',
      confirmText: '继续核销',
      cancelText: '去扫码',
      success: (res) => {
        if (res.confirm) {
          this.executeManualVerify(manualOrderNo, manualTripId)
        }
      }
    })
  },

  // 手动核销
  executeManualVerify(manualOrderNo, manualTripId) {
    this.setData({ verifyLoading: true })
    // 如果有tripId，使用绑定班次的核销接口；否则使用按订单号核销接口
    // 手动输入场景不传 verify_token（后端会记录“未验签”审计）
    if (manualTripId) {
      this.doVerify(manualOrderNo, manualTripId, true, '')
    } else {
      this.doVerifyByNo(manualOrderNo, '')
    }
  },

  // 按订单号核销（不需要班次ID）
  // verifyToken 可选：扫码场景传入完整凭证，手动输入场景传空字符串
  async doVerifyByNo(orderNo, verifyToken) {
    try {
      const res = await request({
        url: '/api/wx/driver/verify-by-no',
        method: 'POST',
        data: { order_no: orderNo, verify_token: verifyToken || '' },
        silent: true // 核销错误由结果对话框统一展示，不重复弹 toast
      })
      this.setData({
        showResult: true,
        resultType: 'success',
        resultIcon: '\u2713',
        resultTitle: '核销成功',
        verifyResult: res.data,
        verifyLoading: false,
        showManual: false
      })
    } catch (e) {
      this.setData({
        showResult: true,
        resultType: 'fail',
        resultIcon: '\u2717',
        resultTitle: (e && e.message) || '核销失败',
        verifyResult: null,
        verifyLoading: false,
        showManual: false
      })
    }
  },

  // 核销（扫码/手动都走这）
  // verifyToken 可选：扫码场景传入完整凭证，手动输入场景传空字符串
  async doVerify(orderNo, tripId, isManual, verifyToken) {
    if (!isManual) {
      this.setData({ verifyLoading: true })
    }
    try {
      const res = await request({
        url: '/api/wx/driver/verify',
        method: 'POST',
        data: { order_no: orderNo, trip_id: tripId, verify_token: verifyToken || '' },
        silent: true // 核销错误由结果对话框统一展示，不重复弹 toast
      })
      this.setData({
        showResult: true,
        resultType: 'success',
        resultIcon: '\u2713',
        resultTitle: '核销成功',
        verifyResult: res.data,
        verifyLoading: false,
        showManual: false
      })
    } catch (e) {
      this.setData({
        showResult: true,
        resultType: 'fail',
        resultIcon: '\u2717',
        resultTitle: (e && e.message) || '核销失败',
        verifyResult: null,
        verifyLoading: false,
        showManual: false
      })
    }
  },

  closeResult() {
    this.setData({ showResult: false, verifyResult: null })
  },

  // 查看班次乘客详情
  async showTripDetail(e) {
    const tripId = e.currentTarget.dataset.id
    wx.showLoading({ title: '加载中' })
    try {
      const res = await request({
        url: `/api/wx/driver/trips/${tripId}/passengers`,
        method: 'GET'
      })
      wx.hideLoading()
      this.setData({
        showPassengers: true,
        passengerList: res.data.passengers || [],
        passengerStats: res.data.stats
      })
    } catch (e) {
      wx.hideLoading()
      wx.showToast({ title: '加载乘客名单失败，请重试', icon: 'none' })
    }
  },

  // 拨打乘客电话
  callPassenger(e) {
    const phone = e.currentTarget.dataset.phone
    if (!phone) {
      wx.showToast({ title: '该乘客无联系电话', icon: 'none' })
      return
    }
    wx.makePhoneCall({ phoneNumber: phone })
  },

  closePassengerDialog() {
    this.setData({ showPassengers: false })
  },

  // 开始行程（跳转到行程追踪页面，自动定位上报）
  handleStartTrack(e) {
    const tripId = e.currentTarget.dataset.id
    wx.navigateTo({
      url: `/pages/driver-track/driver-track?id=${tripId}`
    })
  }
})
