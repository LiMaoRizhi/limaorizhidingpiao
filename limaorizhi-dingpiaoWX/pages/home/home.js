var log = require('../../utils/log')
const { request, BASE_URL } = require('../../utils/request')
const stationPickerMixin = require('../../utils/station-picker-mixin')
const { formatArrivalTime } = require('../../utils/order-helper')
const { formatDate } = require('../../utils/format')

Page({
  data: {
    // 装修布局：[{type:'banner',visible:true},...]，仅含 visible=true 的组件
    layoutList: [],
    bannerList: [],
    couponList: [],
    noticeText: '',
    routeList: [],
    allTrips: [],
    tripDate: '',
    stations: [],
    showStationPicker: false,
    stationPickerType: '',
    selectedFromStation: null,
    selectedToStation: null,
    // 站点弹窗搜索联想
    stationSearchValue: '',
    filteredStations: [],
    // 班次列表排序：direct=直达优先 time=发车时间 price=价格 duration=耗时
    sortBy: 'direct',
    stationSearchFocused: false,
    stationScrollHeight: '300px', // 站点列表scroll-view高度，JS动态计算
    ripples: [], // 水波纹动画数据
    showRefreshToast: false, // 水波纹刷新提示
    _firstLoad: true
  },

  onLoad() {
    // 获取状态栏高度，弹窗占满屏幕时顶部留出状态栏+导航栏空间
    try {
      const sysInfo = wx.getSystemInfoSync()
      this.setData({
        // 弹窗80vh，减去header(50px)+search-box(60px)+搜索框margin(10px*2)≈120px
        stationScrollHeight: Math.floor(sysInfo.windowHeight * 0.8 - 120) + 'px'
      })
    } catch (e) { log.warn('获取系统信息失败', e) }
    const today = formatDate(new Date())
    this.setData({ tripDate: today })
    // 先加载布局配置，再按配置加载对应组件数据
    this.loadLayout()
    this.loadStations()
    this.loadTrips()
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 })
    }
    // 首次加载由 onLoad 触发，不重复请求
    if (this._firstLoad) {
      this._firstLoad = false
      return
    }
    // 后续 onShow：自动刷新班次列表 + 站点列表 + 优惠券领取状态
    // 用户从订单页/我的切回首页时，能实时看到后端新上架的班次/新增的站点
    // 6月18号加的，之前每次切回首页都重新请求一遍，流量浪费严重
    // 站点列表也需刷新：否则后端新增站点后，不杀掉小程序重进就永远看不到
    this.loadTrips()
    this.loadStations()
    if (this.data.layoutList.some(c => c.type === 'coupon')) {
      this.loadCoupons()
    }
  },

  onHide() {
    clearTimeout(this._rippleTimer)
    clearTimeout(this._toastTimer)
  },

  // 下拉刷新：拉取最新班次列表
  // 5月16号刚上线时忘了加下拉刷新，用户反馈说看不到新班次才补上的
  onPullDownRefresh() {
    this.loadTrips()
    if (this.data.layoutList.some(c => c.type === 'coupon')) {
      this.loadCoupons()
    }
  },

  // 点击刷新按钮：水波纹效果 + 重新加载
  handleRefresh(e) {
    wx.vibrateShort({ type: 'light' })
    const ripple = { x: 30, y: 15 }
    this.setData({ ripples: [...this.data.ripples, ripple] })
    this._rippleTimer = setTimeout(() => { this.setData({ ripples: [] }) }, 800)
    this.setData({ showRefreshToast: true })
    this._toastTimer = setTimeout(() => { this.setData({ showRefreshToast: false }) }, 1500)
    this.loadTrips()
    if (this.data.layoutList.some(c => c.type === 'coupon')) {
      this.loadCoupons()
    }
  },

  onUnload() {
    clearTimeout(this._rippleTimer)
    clearTimeout(this._toastTimer)
  },

  // 拉首页装修布局
  loadLayout() {
    request({ url: '/api/homepage/layout', method: 'GET' }).then(res => {
      const list = res.data || []
      this.setData({ layoutList: list })
      const hasBanner = list.some(c => c.type === 'banner')
      const hasCoupon = list.some(c => c.type === 'coupon')
      const hasNotice = list.some(c => c.type === 'notice')
      if (hasBanner) this.loadBanners()
      if (hasCoupon) this.loadCoupons()
      if (hasNotice) this.loadNotice()
    }).catch((e) => {
      log.error('加载首页布局失败', e)
      // 加载失败使用默认布局
      this.setData({
        layoutList: [
          { type: 'banner', title: '轮播图', visible: true },
          { type: 'coupon', title: '优惠券展示', visible: true },
          { type: 'notice', title: '公告通知', visible: true },
          { type: 'search', title: '搜索筛选', visible: true },
          { type: 'trips', title: '车次列表', visible: true }
        ]
      })
      this.loadBanners()
      this.loadCoupons()
      this.loadNotice()
    })
  },

  // 拉公告
  loadNotice() {
    request({ url: '/api/wx/config', method: 'GET' }).then(res => {
      const notice = (res.data && res.data.notice) || ''
      this.setData({ noticeText: notice })
    }).catch((e) => { log.error('加载公告失败', e) })
  },

  // 拉轮播图
  loadBanners() {
    request({ url: '/api/banners', method: 'GET' }).then(res => {
      const banners = (res.data || []).map(item => ({
        id: item.id,
        title: item.title,
        titleColor: item.title_color || '',
        titleEffect: item.title_effect || 0,
        image_url: item.image_url && item.image_url.startsWith('http') ? item.image_url : BASE_URL + item.image_url
      }))
      this.setData({ bannerList: banners })
    }).catch((e) => { log.error('加载轮播图失败', e) })
  },

  // 拉首页优惠券
  loadCoupons() {
    const token = wx.getStorageSync('user_token')
    // 已登录用户调用认证接口（返回 claimed 标记），未登录用公开接口
    const url = token ? '/api/wx/coupons/available' : '/api/homepage/coupons'
    request({ url, method: 'GET' }).then(res => {
      const coupons = (res.data || []).map(item => {
        let label = ''
        let valueText = ''
        if (item.type === 1) {
          label = '满减券'
          valueText = '¥' + item.discount_value
        } else if (item.type === 2) {
          label = '折扣券'
          valueText = item.discount_value + '折'
        } else {
          label = '抵用券'
          valueText = '¥' + item.discount_value
        }
        let desc = ''
        if (item.min_spend > 0) {
          desc = '满' + item.min_spend + '元可用'
        } else {
          desc = '无门槛'
        }
        return { id: item.id, name: item.name, label, valueText, desc, claimed: !!item.claimed }
      })
      this.setData({ couponList: coupons })
    }).catch((e) => { log.error('加载优惠券失败', e) })
  },

  // 领取优惠券
  claimCoupon(e) {
    const token = wx.getStorageSync('user_token')
    if (!token) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }
    const coupon = e.currentTarget.dataset.coupon
    if (coupon.claimed) {
      wx.showToast({ title: '您已领取过该优惠券', icon: 'none' })
      return
    }
    request({
      url: '/api/wx/coupons/claim',
      method: 'POST',
      data: { coupon_id: coupon.id }
    }).then(() => {
      wx.showToast({ title: '领取成功', icon: 'success' })
      const list = this.data.couponList.map(c =>
        c.id === coupon.id ? { ...c, claimed: true } : c
      )
      this.setData({ couponList: list })
    }).catch((e) => {
      log.error('领取优惠券失败', e)
    })
  },

  loadStations() {
    request({ url: '/api/wx/stations', method: 'GET' }).then(res => {
      const stations = res.data || []
      this.setData({ stations: stations, filteredStations: stations })
    }).catch((e) => { log.error('加载站点失败', e) })
  },

  // 车次数据格式化一下
  formatTrip(item) {
    const fromName = item.route && item.route.from_station ? item.route.from_station.name : ''
    const toName = item.route && item.route.to_station ? item.route.to_station.name : ''
    const durationMin = item.route ? item.route.duration_minutes : 0
    const durationStr = durationMin >= 60 ? `约${Math.floor(durationMin/60)}小时${durationMin%60 > 0 ? durationMin%60 + '分钟' : ''}` : `约${durationMin}分钟`
    // 充足时显示发车时间，紧张时显示余票，无票留空（底部"售罄"按钮体现）
    const seatsText = item.available_seats === 0 ? '' : item.available_seats <= 5 ? `余${item.available_seats}座` : (item.departure_time + ' 发车')
    // 跨天到达时间展示（如“次日08:00”）
    const arrivalText = formatArrivalTime(item.arrival_time, item.arrival_day_offset)
    // 余票状态等级：available=充足(绿) tight=紧张(橙) soldout=售罄(红)
    let seatsStatus = 'available'
    if (item.available_seats === 0) {
      seatsStatus = 'soldout'
    } else if (item.available_seats <= 5) {
      seatsStatus = 'tight'
    }
    const rss = (item.route && item.route.route_stations) ? item.route.route_stations : []
    const stationCount = rss.length
    // 票价：有站点序列时一律用区间价，无站点时回退到 base_price
    const { selectedFromStation, selectedToStation } = this.data
    let priceText = (parseFloat(item.base_price) || 0).toFixed(0)
    let viaStations = []
    let stopCountBetween = 0
    let isDirect = false
    let fromSelStationId = 0
    let toSelStationId = 0
    if (stationCount > 0) {
      // 默认全程票价 = 末站price - 首站price
      let fromRS = rss[0]
      let toRS = rss[rss.length - 1]
      // 选了上下车站则用区间价
      if (selectedFromStation && selectedToStation) {
        const f = rss.find(s => s.station_id === selectedFromStation.id)
        const t = rss.find(s => s.station_id === selectedToStation.id)
        if (f && t) { fromRS = f; toRS = t }
      }
      // 实际票价 = max(起步价, 累计差额)，起步价为0时不启用
      let fare = Math.max(0, toRS.price - fromRS.price)
      const minFare = item.route && item.route.min_fare ? (parseFloat(item.route.min_fare) || 0) : 0
      if (minFare > 0 && fare < minFare) fare = minFare
      priceText = fare.toFixed(0)
      // 计算途经站点名与经停数（供列表展示与直达优先排序）
      const fromIdx = rss.findIndex(s => s === fromRS)
      const toIdx = rss.findIndex(s => s === toRS)
      if (fromIdx >= 0 && toIdx >= 0 && fromIdx < toIdx) {
        const segment = rss.slice(fromIdx, toIdx + 1)
        viaStations = segment.map(s => (s.station && s.station.name) || '')
        stopCountBetween = toIdx - fromIdx - 1
        isDirect = stopCountBetween === 0
        fromSelStationId = fromRS.station_id
        toSelStationId = toRS.station_id
      }
    }
    return {
      id: item.id,
      from: fromName,
      to: toName,
      // 所有途经站点名（含首末站），用于搜索匹配
      stationNames: rss.map(s => (s.station && s.station.name) || '').join('|'),
      time: item.departure_time,
      arrivalText: arrivalText,
      duration: durationStr,
      durationMin: durationMin,
      price: priceText,
      priceIsInterval: !!(selectedFromStation && selectedToStation && stationCount > 0),
      seats: seatsText,
      seatsStatus: seatsStatus,
      availableSeats: item.available_seats,
      stationCount,
      viaStations: viaStations,
      stopCountBetween: stopCountBetween,
      isDirect: isDirect,
      fromSelStationId: fromSelStationId,
      toSelStationId: toSelStationId,
      raw: item
    }
  },

  // 加载车次列表（返回Promise，供下拉刷新和按钮刷新控制状态）
  loadTrips() {
    const { tripDate, selectedFromStation, selectedToStation } = this.data
    let url = `/api/wx/trips?trip_date=${tripDate}`
    if (selectedFromStation) {
      url += `&from_station_id=${selectedFromStation.id}`
    }
    if (selectedToStation) {
      url += `&to_station_id=${selectedToStation.id}`
    }
    return request({ url, method: 'GET' }).then(res => {
      const trips = (res.data || []).map(item => this.formatTrip(item))
      const sorted = this.sortTrips(trips, this.data.sortBy)
      this.setData({ allTrips: trips, routeList: sorted })
    }).catch((e) => {
      log.error('加载车次列表失败', e)
      this.setData({ allTrips: [], routeList: [] })
    }).finally(() => {
      wx.stopPullDownRefresh()
    })
  },

  onDateChange(e) {
    this.setData({ tripDate: e.detail.value })
    this.loadTrips()
  },

  // 站点选择弹窗公共方法（来自 station-picker-mixin）
  showFromPicker: stationPickerMixin.showFromPicker,
  showToPicker: stationPickerMixin.showToPicker,
  closeStationPicker: stationPickerMixin.closeStationPicker,
  onStationMaskTap: stationPickerMixin.onStationMaskTap,
  noop: stationPickerMixin.noop,
  onStationSearch: stationPickerMixin.onStationSearch,
  onStationSearchConfirm: stationPickerMixin.onStationSearchConfirm,
  onStationSearchFocus: stationPickerMixin.onStationSearchFocus,
  onStationSearchBlur: stationPickerMixin.onStationSearchBlur,
  clearStationSearch: stationPickerMixin.clearStationSearch,

  // 选择站点（页面自有业务回调）
  onStationTap(e) {
    const station = e.currentTarget.dataset.station
    if (this.data.stationPickerType === 'from') {
      this.setData({ selectedFromStation: station, showStationPicker: false, stationSearchValue: '', filteredStations: this.data.stations, stationSearchFocused: false })
    } else {
      this.setData({ selectedToStation: station, showStationPicker: false, stationSearchValue: '', filteredStations: this.data.stations, stationSearchFocused: false })
    }
    this.loadTrips()
  },

  // 班次列表排序切换
  onSortChange(e) {
    const sortBy = e.currentTarget.dataset.sort
    this.setData({ sortBy: sortBy })
    const sorted = this.sortTrips(this.data.allTrips, sortBy)
    this.setData({ routeList: sorted })
  },

  // 排序工具方法：直达优先 / 发车时间 / 价格 / 耗时
  sortTrips(list, sortBy) {
    const arr = (list || []).slice()
    if (sortBy === 'direct') {
      // 直达优先（直达在前），同级按发车时间升序
      arr.sort((a, b) => {
        if (a.isDirect !== b.isDirect) return a.isDirect ? -1 : 1
        if (a.stopCountBetween !== b.stopCountBetween) return a.stopCountBetween - b.stopCountBetween
        return (a.time || '').localeCompare(b.time || '')
      })
    } else if (sortBy === 'time') {
      arr.sort((a, b) => (a.time || '').localeCompare(b.time || ''))
    } else if (sortBy === 'price') {
      arr.sort((a, b) => (parseFloat(a.price) || 0) - (parseFloat(b.price) || 0))
    } else if (sortBy === 'duration') {
      arr.sort((a, b) => (a.durationMin || 0) - (b.durationMin || 0))
    }
    return arr
  },

  // 交换出发和到达
  swapStations() {
    const { selectedFromStation, selectedToStation } = this.data
    this.setData({
      selectedFromStation: selectedToStation,
      selectedToStation: selectedFromStation
    })
    this.loadTrips()
  },

  // 清除筛选
  clearFilter() {
    this.setData({
      selectedFromStation: null,
      selectedToStation: null
    })
    this.loadTrips()
  },

  // 点击车次 - 去订票
  onTripTap(e) {
    const trip = e.currentTarget.dataset.trip
    if (!trip || !trip.availableSeats) {
      wx.showToast({ title: '该班次已无票', icon: 'none' })
      return
    }
    let url = `/pages/trip-detail/trip-detail?id=${trip.id}`
    if (this.data.selectedFromStation) {
      url += `&from_sid=${this.data.selectedFromStation.id}`
    }
    if (this.data.selectedToStation) {
      url += `&to_sid=${this.data.selectedToStation.id}`
    }
    wx.navigateTo({ url })
  },

  // 分享给好友
  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行',
      path: '/pages/home/home'
    }
  },

  onShareTimeline() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行'
    }
  }
})
