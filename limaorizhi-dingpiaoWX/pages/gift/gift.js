// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
const { request } = require('../../utils/request')
const { safeDate } = require('../../utils/format')

Page({
  data: {
    activeTab: 0, // 0=优惠券 1=积分
    tabs: [{ name: '优惠券' }, { name: '积分' }],
    // 优惠券
    couponTab: 0, // 0=未使用 1=已使用 2=已过期
    couponTabs: [{ name: '未使用' }, { name: '已使用' }, { name: '已过期' }],
    couponList: [],
    // 积分
    pointsBalance: 0,
    pointRecords: []
  },

  _firstLoad: true,

  onLoad() {
    this.loadCoupons()
    this.loadPoints()
    this.loadPointRecords()
  },

  onShow() {
    // 首次加载由 onLoad 触发，不重复请求
    if (this._firstLoad) {
      this._firstLoad = false
      return
    }
    // 返回页面时刷新数据
    this.loadCoupons()
    this.loadPoints()
    this.loadPointRecords()
  },

  // 切换主标签
  onMainTabTap(e) {
    const index = e.currentTarget.dataset.index
    if (index === this.data.activeTab) return
    this.setData({ activeTab: index })
  },

  // 切换优惠券子标签
  onCouponTabTap(e) {
    const index = e.currentTarget.dataset.index
    if (index === this.data.couponTab) return
    this.setData({ couponTab: index, couponList: [] })
    this.loadCoupons()
  },

  // 加载优惠券
  loadCoupons() {
    const status = this.data.couponTab === 0 ? '0' : this.data.couponTab === 1 ? '1' : '2'
    request({
      url: '/api/wx/coupons?status=' + status,
      method: 'GET'
    }).then(res => {
      const list = (res.data || []).map(item => {
        return { ...item, expired_at: safeDate(item.expired_at) }
      })
      this.setData({ couponList: list })
    }).catch(() => {
      wx.showToast({ title: '加载失败，请检查网络', icon: 'none' })
    })
  },

  // 加载积分余额
  loadPoints() {
    request({ url: '/api/wx/points', method: 'GET' }).then(res => {
      this.setData({ pointsBalance: (res.data && res.data.balance) || 0 })
    }).catch(() => {
      wx.showToast({ title: '加载失败，请检查网络', icon: 'none' })
    })
  },

  // 加载积分明细（适配后端分页返回格式）
  loadPointRecords() {
    request({ url: '/api/wx/points/records?page=1&page_size=50', method: 'GET' }).then(res => {
      const records = ((res.data && res.data.list) || []).map(item => {
        const ds = item.created_at ? item.created_at.replace('T', ' ') : ''
        const month = parseInt(ds.substring(5, 7)) || 1
        const day = parseInt(ds.substring(8, 10)) || 1
        const time = ds.length >= 16 ? ds.substring(11, 16) : ''
        const dateText = `${month}月${day}日 ${time}`
        return { ...item, dateText }
      })
      this.setData({ pointRecords: records })
    }).catch(() => {
      wx.showToast({ title: '加载失败，请检查网络', icon: 'none' })
    })
  },

  // 去使用（跳转到首页选车次）
  goUse() {
    wx.switchTab({ url: '/pages/home/home' })
  },

  // 分享给好友（分享惊喜礼包活动，好友点开可领优惠券和积分）
  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 优惠券积分等你来领',
      path: '/pages/gift/gift'
    }
  }
})
