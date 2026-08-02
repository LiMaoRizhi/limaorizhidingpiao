// 管理后台首页（仅管理员可见的入口聚合页）
const { request } = require('../../utils/request')

Page({
  data: {
    adminName: '管理员',
    adminRole: 0,
    modules: [
      { key: 'dashboard', name: '统计看板', icon: '/images/icons/admin-dashboard.svg', action: 'dashboard', ready: true },
      { key: 'orders', name: '订单管理', icon: '/images/icons/admin-orders.svg', action: 'orders', ready: true },
      { key: 'ticket', name: '售票管理', icon: '/images/icons/admin-ticket.svg', action: 'ticket', ready: true, desc: '站点/线路/班次/车辆' },
      { key: 'trips', name: '班次管理', icon: '/images/icons/admin-trips.svg', action: 'trips', ready: true },
      { key: 'vehicles', name: '车辆管理', icon: '/images/icons/bus-front.svg', action: 'vehicles', ready: true },
      { key: 'drivers', name: '司机管理', icon: '/images/icons/verify.svg', action: 'drivers', ready: true },
      { key: 'users', name: '用户管理', icon: '/images/icons/default-avatar.svg', action: 'users', ready: true },
      { key: 'marketing', name: '营销管理', icon: '/images/icons/admin-marketing.svg', action: 'marketing', ready: true, desc: '优惠券/积分' },
      { key: 'track', name: '车辆轨迹', icon: '/images/icons/trip-track.svg', action: 'track', ready: true },
      { key: 'ai', name: '数字员工', icon: '/images/icons/admin-ai.svg', action: 'ai', ready: true }
    ]
  },

  onLoad() {
    // 防御性校验：非管理员不允许多停留（入口本身已隐藏，此处防直达）
    const userInfo = wx.getStorageSync('user_info') || {}
    if (!userInfo.is_admin) {
      wx.showToast({ title: '无管理权限', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 800)
      return
    }
    this.setData({
      adminName: userInfo.admin_real_name || userInfo.nickname || '管理员',
      adminRole: userInfo.admin_role || 0
    })
    this.loadProfile()
  },

  // 拉取管理员信息（含最新 role，与后端二次校验一致）
  loadProfile() {
    request({ url: '/api/wx/admin/profile', method: 'GET', silent: true }).then(res => {
      const p = res.data || {}
      if (p.real_name || p.username) {
        this.setData({
          adminName: p.real_name || p.username || this.data.adminName,
          adminRole: p.role || this.data.adminRole
        })
      }
    }).catch(() => {})
  },

  onModuleTap(e) {
    const item = e.currentTarget.dataset.item
    if (!item.ready) {
      wx.showToast({ title: '功能开发中', icon: 'none' })
      return
    }
    const urlMap = {
      dashboard: '/pages/admin-dashboard/admin-dashboard',
      orders: '/pages/admin-orders/admin-orders',
      ticket: '/pages/admin-ticket/admin-ticket',
      trips: '/pages/admin-trips/admin-trips',
      vehicles: '/pages/admin-vehicles/admin-vehicles',
      drivers: '/pages/admin-drivers/admin-drivers',
      users: '/pages/admin-users/admin-users',
      marketing: '/pages/admin-marketing/admin-marketing',
      track: '/pages/admin-track/admin-track',
      ai: '/pages/admin-ai/admin-ai'
    }
    const url = urlMap[item.action]
    if (url) {
      wx.navigateTo({ url })
    }
  }
})
