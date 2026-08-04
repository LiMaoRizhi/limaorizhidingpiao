const { request, markLoginSuccess } = require('../../utils/request')

Page({
  data: {
    loginLoading: false,
    agreed: false
  },

  onLoad() {
    // 预获取 wx.login code，点击按钮时直接用，不用等
    this.preFetchLoginCode()
  },

  // 预获取 wx.login code（记录时间戳，超过4分钟视为过期）
  preFetchLoginCode() {
    wx.login({
      success: (res) => {
        this.loginCode = res.code
        this.loginCodeTime = Date.now()
      },
      fail: () => {
        // 预获取失败没关系，点击时再获取
      }
    })
  },

  // 协议勾选状态切一下
  toggleAgreement() {
    this.setData({ agreed: !this.data.agreed })
  },

  // 未勾选协议时点击"登录"按钮，提示用户先勾选同意
  onLoginTap() {
    wx.showToast({ title: '请先阅读并勾选同意用户协议', icon: 'none' })
  },

  // 去协议页
  goAgreement(e) {
    const type = e.currentTarget.dataset.type || 'user_agreement'
    wx.navigateTo({ url: `/pages/agreement/agreement?type=${type}` })
  },

  // 微信手机号一键登录
  async handleLogin(e) {
    // 未勾选协议，不允许登录
    if (!this.data.agreed) {
      wx.showToast({ title: '请先阅读并同意用户协议和隐私政策', icon: 'none' })
      return
    }

    const phoneCode = e.detail.code || ''
    const errMsg = e.detail.errMsg || ''

    // 用户明确拒绝授权手机号 → 不继续
    if (errMsg.indexOf('deny') !== -1 || errMsg.indexOf('user deny') !== -1) {
      wx.showToast({ title: '需要授权手机号才能登录', icon: 'none' })
      return
    }

    // phoneCode 为空时：开发环境允许继续（后端给 mock 手机号），生产环境必须拦截
    if (!phoneCode) {
      let isProd = false
      try {
        const info = wx.getAccountInfoSync()
        isProd = info.miniProgram.envVersion === 'release' || info.miniProgram.envVersion === 'trial'
      } catch (e) {}
      if (isProd) {
        wx.showToast({ title: '请授权手机号才能登录', icon: 'none' })
        return
      }
      // 开发环境：phoneCode 为空时仍继续，后端会给 mock 手机号
    }
    this.setData({ loginLoading: true })

    try {
      // 获取 login code：优先用预获取的，超过4分钟（wx.login code有效期5分钟）视为过期
      let loginCode = this.loginCode
      if (!loginCode || (Date.now() - (this.loginCodeTime || 0)) > 4 * 60 * 1000) {
        const loginRes = await new Promise((resolve, reject) => {
          wx.login({ success: resolve, fail: reject })
        })
        loginCode = loginRes.code
      }

      // 一步到位：login_code + phone_code 发后端
      // 清除旧 token，避免携带过期 token 触发 handleUserTokenExpired 与登录操作冲突
      wx.removeStorageSync('user_token')
      const res = await request({
        url: '/api/wx/phone-login',
        method: 'POST',
        data: {
          login_code: loginCode,
          phone_code: phoneCode
        }
      })

      // 登录态存起来
      wx.setStorageSync('user_token', res.data.token)
      wx.setStorageSync('user_info', res.data.user)
      const app = getApp()
      app.globalData.userInfo = res.data.user
      app.globalData.isLoggedIn = true
      // 标记登录成功时间，防止旧请求的1002响应触发重定向死循环
      markLoginSuccess()

      wx.showToast({ title: '登录成功', icon: 'success' })
      setTimeout(() => {
        const pages = getCurrentPages()
        if (pages.length > 1) {
          wx.navigateBack()
        } else {
          wx.switchTab({ url: '/pages/home/home' })
        }
      }, 800)
    } catch (err) {
      // request 工具会显示错误 toast，这里兜底 wx.login 失败和其他异常
      if (err && err.errMsg && err.errMsg.indexOf('login') !== -1) {
        wx.showToast({ title: '微信登录失败，请重试', icon: 'none' })
      } else if (err && err.errMsg && err.errMsg.indexOf('request') !== -1) {
        wx.showToast({ title: '网络错误，请重试', icon: 'none' })
      } else if (err && err.code && err.code !== 1002) {
        // request 工具已显示了 toast，这里不重复提示
        // 但如果 err 没有 message 字段（非标准响应），兜底提示
        if (!err.message) {
          wx.showToast({ title: '登录失败，请重试', icon: 'none' })
        }
      }
      // 1002 错误：登录接口是公开接口，不应返回1002
      // 如果出现说明后端路由配置异常，提示用户
      if (err && err.code === 1002) {
        wx.showToast({ title: '登录服务异常，请稍后重试', icon: 'none' })
      }
    } finally {
      this.setData({ loginLoading: false })
      // 重新获取 login code 供下次使用（code 有有效期）
      this.preFetchLoginCode()
    }
  }
})
