// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')
const { request, upload } = require('../../utils/request')
const { normalizeUrl, maskPhone } = require('../../utils/format')
const { logout, deleteAccount } = require('../../utils/auth-mixin')

// 系统配置缓存（5分钟内复用，减少每次 onShow 的 API 调用）
var _configCache = { data: null, time: 0 }

Page({
  data: {
    userInfo: {
      nickName: '未登录',
      phone: '点击登录',
      avatarUrl: '',
      isLogin: false
    },
    orderStats: {
      pending_pay: 0,
      pending_travel: 0,
      completed: 0,
      refunded: 0
    },
    orderList: [
      { icon: "/images/icons/pending-pay.svg", name: "待付款", count: 0, status: '0', cls: "pay" },
      { icon: "/images/icons/pending-travel.svg", name: "待出行", count: 0, status: '1', cls: "travel" },
      { icon: "/images/icons/completed.svg", name: "已完成", count: 0, status: '2', cls: "done" },
      { icon: "/images/icons/refund.svg", name: "退款", count: 0, status: '3', cls: "refund" }
    ],
    menuList: [
      { icon: "/images/icons/passenger-info.svg", name: "乘客实名", action: "passenger" },
      { icon: "/images/icons/surprise-gift.svg", name: "惊喜礼包", action: "gift" },
      { icon: "/images/icons/trip-track.svg", name: "历史脚步", action: "orders" },
      { icon: "/images/icons/cargo.svg", name: "货物托运", action: "cargo" },
      { icon: "/images/icons/wechat-service.svg", name: "微信客服", openType: "contact" },
      { icon: "/images/icons/phone-hotline.svg", name: "电话热线", action: "phone" }
      // 司机核销入口仅司机可见，由 filterMenuByDriver 根据 isDriver 动态控制
    ],
    config: {},
    menuLayoutType: 'list',
    logoutPosition: 'mine_bottom',
    phoneMasked: true,
    maskedPhone: '',
    isDriver: false
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 2 })
    }
    this.checkLoginStatus()
    this.loadConfig()
    // loadMineLayout 由 checkLoginStatus→loadUserInfo 链调用，避免重复请求
    // 未登录时 loadUserInfo 不会被调用，需手动加载布局
    if (!wx.getStorageSync('user_token')) {
      this.loadMineLayout()
    }
  },

  // 加载我的页装修布局（订单分类 + 功能菜单的显隐与顺序）
  loadMineLayout() {
    Promise.all([
      request({ url: '/api/design/page-layout', method: 'GET', data: { page: 'mine_order_grid' } }),
      request({ url: '/api/design/page-layout', method: 'GET', data: { page: 'mine_menu' } })
    ]).then(([gridRes, menuRes]) => {
      const gridLayout = gridRes.data || []
      const menuLayout = menuRes.data || []
      const orderGridMap = {
        pending_pay: { icon: "/images/icons/pending-pay.svg", name: "待付款", status: '0', cls: "pay" },
        pending_travel: { icon: "/images/icons/pending-travel.svg", name: "待出行", status: '1', cls: "travel" },
        completed: { icon: "/images/icons/completed.svg", name: "已完成", status: '2', cls: "done" },
        refund: { icon: "/images/icons/refund.svg", name: "退款", status: '3', cls: "refund" }
      }
      const menuMap = {
        passenger: { icon: "/images/icons/passenger-info.svg", name: "乘客实名", action: "passenger" },
        gift: { icon: "/images/icons/surprise-gift.svg", name: "惊喜礼包", action: "gift" },
        orders: { icon: "/images/icons/trip-track.svg", name: "历史脚步", action: "orders" },
        cargo: { icon: "/images/icons/cargo.svg", name: "货物托运", action: "cargo" },
        wechat_service: { icon: "/images/icons/wechat-service.svg", name: "微信客服", openType: "contact" },
        phone: { icon: "/images/icons/phone-hotline.svg", name: "电话热线", action: "phone" },
        verify: { icon: "/images/icons/verify.svg", name: "司机核销", action: "verify" }
      }
      // 合并订单分类：按装修顺序+显隐，保留已有 count
      const newOrderList = gridLayout.filter(c => c.visible).map(c => {
        const base = orderGridMap[c.type]
        if (!base) return null
        const old = this.data.orderList.find(o => o.status === base.status)
        return { ...base, count: old ? old.count : 0 }
      }).filter(Boolean)
      // 合并功能菜单：按装修顺序+显隐
      let newMenuList = menuLayout.filter(c => c.visible).map(c => menuMap[c.type]).filter(Boolean)
      newMenuList = this.filterMenuByDriver(newMenuList)
      if (newOrderList.length > 0) this.setData({ orderList: newOrderList })
      if (newMenuList.length > 0) this.setData({ menuList: newMenuList })
    }).catch(() => {})
  },

  // 司机核销入口仅司机可见：isDriver=true 确保存在（追加到末尾），isDriver=false 移除
  filterMenuByDriver(menuList) {
    let list = [...menuList]
    if (this.data.isDriver) {
      if (!list.find(item => item.action === 'verify')) {
        list.push({ icon: "/images/icons/verify.svg", name: "司机核销", action: "verify" })
      }
    } else {
      list = list.filter(item => item.action !== 'verify')
    }
    return list
  },

  // 立即根据 isDriver 更新当前菜单
  applyDriverMenu() {
    this.setData({ menuList: this.filterMenuByDriver(this.data.menuList) })
  },

  // 检查登录状态
  checkLoginStatus() {
    const token = wx.getStorageSync('user_token')
    const userInfo = wx.getStorageSync('user_info')
    if (token && userInfo) {
      const phone = userInfo.phone || '未绑定'
      this.setData({
        userInfo: {
          nickName: userInfo.nickname || '用户',
          phone: phone,
          avatarUrl: normalizeUrl(userInfo.avatar_url),
          isLogin: true
        },
        isDriver: !!userInfo.is_driver || !!wx.getStorageSync('driver_token'),
        phoneMasked: true,
        maskedPhone: maskPhone(phone)
      })
      this.applyDriverMenu()
      this.loadOrderStats()
      this.loadUserInfo()
    } else {
      this.setData({
        userInfo: {
          nickName: '未登录',
          phone: '点击登录',
          avatarUrl: '',
          isLogin: false
        },
        isDriver: !!wx.getStorageSync('driver_token'),
        phoneMasked: true,
        maskedPhone: '',
        orderList: this.data.orderList.map(item => ({ ...item, count: 0 }))
      })
      this.applyDriverMenu()
      // 未登录时也加载布局（登录时由 loadUserInfo 调用）
      this.loadMineLayout()
    }
  },

  // 加载用户信息
  loadUserInfo() {
    request({ url: '/api/wx/user', method: 'GET' }).then(res => {
      const user = res.data
      wx.setStorageSync('user_info', user)
      const phone = user.phone || '未绑定'
      this.setData({
        userInfo: {
          nickName: user.nickname || '用户',
          phone: phone,
          avatarUrl: normalizeUrl(user.avatar_url),
          isLogin: true
        },
        isDriver: !!user.is_driver || !!wx.getStorageSync('driver_token'),
        phoneMasked: true,
        maskedPhone: maskPhone(phone)
      })
      this.applyDriverMenu()
      this.loadMineLayout()
    }).catch((e) => { log.error('加载用户信息失败', e) })
  },

  // 切换手机号脱敏/明文
  togglePhoneMask() {
    this.setData({ phoneMasked: !this.data.phoneMasked })
  },

  // 加载订单统计
  loadOrderStats() {
    request({ url: '/api/wx/orders/stats', method: 'GET' }).then(res => {
      const stats = res.data
      const orderList = this.data.orderList.map(item => {
        let count = 0
        if (item.status === '0') count = stats.pending_pay || 0
        if (item.status === '1') count = stats.pending_travel || 0
        if (item.status === '2') count = stats.completed || 0
        if (item.status === '3') count = stats.refunded || 0
        return { ...item, count }
      })
      this.setData({ orderStats: stats, orderList })
    }).catch((e) => { log.error('加载订单统计失败', e) })
  },

  // 加载系统配置（5分钟缓存，减少每次onShow的API调用）
  loadConfig() {
    if (_configCache.data && Date.now() - _configCache.time < 300000) {
      const config = _configCache.data
      this.setData({
        config,
        menuLayoutType: config.mine_menu_layout_type === 'grid' ? 'grid' : 'list',
        logoutPosition: config.logout_position || 'mine_bottom'
      })
      return
    }
    request({ url: '/api/wx/config', method: 'GET' }).then(res => {
      const config = res.data || {}
      _configCache.data = config
      _configCache.time = Date.now()
      this.setData({
        config,
        menuLayoutType: config.mine_menu_layout_type === 'grid' ? 'grid' : 'list',
        logoutPosition: config.logout_position || 'mine_bottom'
      })
    }).catch((e) => { log.error('加载系统配置失败', e) })
  },

  // 点击用户卡片（已登录跳个人中心，未登录跳转登录）
  onUserTap() {
    if (!this.data.userInfo.isLogin) {
      wx.navigateTo({ url: '/pages/login/login' })
    } else {
      wx.navigateTo({ url: '/pages/profile/profile' })
    }
  },

  // 点击头像上传新头像
  onAvatarTap() {
    if (!this.data.userInfo.isLogin) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }
    wx.chooseMedia({
      count: 1,
      mediaType: ['image'],
      sourceType: ['album', 'camera'],
      sizeType: ['compressed'],
      success: (res) => {
        const tempFilePath = res.tempFiles[0].tempFilePath
        wx.showLoading({ title: '上传中...', mask: true })
        upload({ url: '/api/wx/upload', filePath: tempFilePath, name: 'file' }).then((upRes) => {
          const avatarUrl = upRes.data.url
          return request({ url: '/api/wx/user', method: 'PUT', data: { avatar_url: avatarUrl } })
        }).then((res) => {
          const user = res.data
          wx.setStorageSync('user_info', user)
          this.setData({
            'userInfo.avatarUrl': normalizeUrl(user.avatar_url),
            'userInfo.nickName': user.nickname || this.data.userInfo.nickName
          })
          wx.hideLoading()
          wx.showToast({ title: '头像更新成功', icon: 'success' })
        }).catch(() => {
          // request.js 失败时已调用 wx.showToast，showLoading 已被自动关闭，无需再调 hideLoading
        })
      }
    })
  },

  // 点击昵称编辑
  onNicknameTap() {
    if (!this.data.userInfo.isLogin) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }
    wx.showModal({
      title: '修改昵称',
      editable: true,
      placeholderText: '请输入新昵称',
      content: this.data.userInfo.nickName,
      success: (res) => {
        if (res.confirm) {
          const nickname = (res.content || '').trim()
          if (!nickname) {
            wx.showToast({ title: '昵称不能为空', icon: 'none' })
            return
          }
          request({ url: '/api/wx/user', method: 'PUT', data: { nickname } }).then((res) => {
            const user = res.data
            wx.setStorageSync('user_info', user)
            this.setData({ 'userInfo.nickName': user.nickname })
            wx.showToast({ title: '昵称修改成功', icon: 'success' })
          }).catch(() => {})
        }
      }
    })
  },

  // 退出登录
  handleLogout() {
    wx.showModal({
      title: '提示',
      content: '确认退出登录？',
      success: async (res) => {
        if (res.confirm) {
          await logout()
          this.setData({
            userInfo: {
              nickName: '未登录',
              phone: '点击登录',
              avatarUrl: '',
              isLogin: false
            },
            phoneMasked: true,
            maskedPhone: '',
            orderList: this.data.orderList.map(item => ({ ...item, count: 0 }))
          })
          wx.showToast({ title: '已退出', icon: 'success' })
        }
      }
    })
  },

  // 注销账号（F2：清除所有个人数据）
  handleDeleteAccount() {
    wx.showModal({
      title: '注销账号',
      content: '注销后将清除您的全部个人信息（乘客信息、优惠券、积分等），且不可恢复。确定要注销吗？',
      confirmText: '确认注销',
      cancelText: '再想想',
      confirmColor: '#e64340',
      success: async (res) => {
        if (!res.confirm) return
        // 二次确认，防止误操作
        wx.showModal({
          title: '最后确认',
          content: '此操作不可逆！您的账号将被永久注销，无法再登录。确定继续？',
          confirmText: '确认注销',
          cancelText: '取消',
          confirmColor: '#e64340',
          success: async (res2) => {
            if (!res2.confirm) return
            wx.showLoading({ title: '注销中...', mask: true })
            try {
              await deleteAccount()
              this.setData({
                userInfo: {
                  nickName: '未登录',
                  phone: '点击登录',
                  avatarUrl: '',
                  isLogin: false
                },
                phoneMasked: true,
                maskedPhone: '',
                orderList: this.data.orderList.map(item => ({ ...item, count: 0 }))
              })
              wx.hideLoading()
              wx.showToast({ title: '账号已注销', icon: 'success' })
            } catch (e) {
              // request.js 已展示错误提示，这里只需关闭 loading
              wx.hideLoading()
            }
          }
        })
      }
    })
  },

  // 点击订单分类
  onOrderTap(e) {
    if (!this.data.userInfo.isLogin) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }
    const status = e.currentTarget.dataset.status
    const url = status
      ? `/pages/order-list/order-list?status=${status}`
      : '/pages/order-list/order-list'
    wx.navigateTo({ url })
  },

  // 点击菜单
  onMenuTap(e) {
    const item = e.currentTarget.dataset.item
    if (item.action === 'verify') {
      wx.navigateTo({ url: '/pages/verify/verify' })
    } else if (item.action === 'passenger') {
      if (!this.data.userInfo.isLogin) {
        wx.navigateTo({ url: '/pages/login/login' })
        return
      }
      wx.navigateTo({ url: '/pages/passenger/passenger' })
    } else if (item.action === 'orders') {
      if (!this.data.userInfo.isLogin) {
        wx.navigateTo({ url: '/pages/login/login' })
        return
      }
      wx.navigateTo({ url: '/pages/order-list/order-list' })
    } else if (item.action === 'phone') {
      const phone = this.data.config.customer_service_phone
      if (phone) {
        wx.showModal({
          title: '电话热线',
          content: `是否拨打客服电话 ${phone}？`,
          confirmText: '确认',
          cancelText: '取消',
          confirmColor: '#1a1a1a',
          cancelColor: '#999',
          success: (res) => {
            if (res.confirm) {
              wx.makePhoneCall({ phoneNumber: phone })
            }
          }
        })
      } else {
        wx.showToast({ title: '暂未配置客服电话', icon: 'none' })
      }
    } else if (item.action === 'cargo') {
      if (!this.data.userInfo.isLogin) {
        wx.navigateTo({ url: '/pages/login/login' })
        return
      }
      wx.navigateTo({ url: '/pages/cargo-create/cargo-create' })
    } else if (item.action === 'gift') {
      if (!this.data.userInfo.isLogin) {
        wx.navigateTo({ url: '/pages/login/login' })
        return
      }
      wx.navigateTo({ url: '/pages/gift/gift' })
    }
  },

  // 分享给好友
  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行',
      path: '/pages/home/home'
    }
  },

  // 分享到朋友圈
  onShareTimeline() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行'
    }
  }
})
