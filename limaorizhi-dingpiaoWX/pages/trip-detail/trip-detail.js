// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')
const { request } = require('../../utils/request')
const { matchStation } = require('../../utils/pinyin')
const { formatArrivalTime } = require('../../utils/order-helper')
const { startPayment } = require('../../utils/pay')
const { safeDate, validatePhone } = require('../../utils/format')

function maskIdCardNo(no) {
  if (!no || no.length < 8) return no || ''
  return no.substring(0, 3) + '****' + no.substring(no.length - 4)
}

Page({
  data: {
    tripId: 0,
    trip: null,
    passengers: [],
    contactName: '',
    contactPhone: '',
    contactFocused: '',
    orderLoading: false,
    isLogin: false,
    coupons: [],
    couponList: [],
    selectedCoupon: null,
    showCouponPicker: false,
    // 区间站点
    routeStations: [],
    fromStationId: 0,
    toStationId: 0,
    intervalPrice: '0.00',
    intervalAvailableSeats: -1, // 区间可用余座（-1=加载中）
    intervalSeatsText: '', // 区间余座文案
    finalPrice: '0.00', // 优惠后总价（合计展示用）
    showStationPicker: false,
    stationPickerType: '',
    stationSearchValue: '',
    filteredRouteStations: [],
    fromStationName: '',
    toStationName: '',
    // 从首页传入的预设站点
    preFromStationId: 0,
    preToStationId: 0,
    // 站点搜索框聚焦态（控制边框高亮）
    searchFocused: false,
    stationScrollHeight: '300px', // 站点列表scroll-view高度，JS动态计算
    // 车辆位置地图预览
    hasVehicleLocation: false,
    vehicleMarkers: [],
    vehiclePolyline: [],
    vehicleIncludePoints: [],
    vehicleMapLng: 116.40,
    vehicleMapLat: 39.90,
    trackPlaceholderText: '班次尚未发车，暂无位置信息',
    vehicleLocation: null,
    mapReady: false,
    currentPassedOrder: 0, // 司机手动标记的已过站序号
    effectivePassedOrder: 0 // 有效过站序号（GPS优先+手动标记取max，驱动进度条）
  },

  onLoad(options) {
    const tripId = options.id
    // 获取屏幕高度，动态计算站点列表 scroll-view 高度
    try {
      const sysInfo = wx.getSystemInfoSync()
      this.setData({
        stationScrollHeight: Math.floor(sysInfo.windowHeight * 0.8 - 120) + 'px'
      })
    } catch (e) { log.warn('获取系统信息失败', e) }
    this.setData({
      tripId: tripId,
      preFromStationId: options.from_sid ? parseInt(options.from_sid) : 0,
      preToStationId: options.to_sid ? parseInt(options.to_sid) : 0
    })
    this.loadTripDetail(tripId)
    
    const token = wx.getStorageSync('user_token')
    if (token) {
      this.setData({ isLogin: true })
      this.loadSelectedPassengers()
      this.loadCoupons()
      this.loadContactInfo()
    }
  },

  // 从服务器获取最新用户信息并预填联系人
  loadContactInfo() {
    request({ url: '/api/wx/user', method: 'GET' }).then(res => {
      const user = res.data
      if (user) {
        this.setData({
          contactName: user.nickname || '',
          contactPhone: user.phone || ''
        })
        // 更新本地缓存
        wx.setStorageSync('user_info', user)
      }
    }).catch((e) => {
      log.error('获取用户信息失败', e)
      // 降级使用本地缓存
      const userInfo = wx.getStorageSync('user_info')
      if (userInfo) {
        this.setData({
          contactName: userInfo.nickname || '',
          contactPhone: userInfo.phone || ''
        })
      }
    })
  },

  // 页面再次显示时刷新登录态并读取选中乘客和优惠券
  onShow() {
    const token = wx.getStorageSync('user_token')
    if (token && !this.data.isLogin) {
      // 用户从登录页返回，更新登录态并加载数据
      this.setData({ isLogin: true })
      this.loadSelectedPassengers()
      this.loadCoupons()
      this.loadContactInfo()
    } else if (this.data.isLogin) {
      // 已登录，刷新选中乘客和优惠券（如从乘客选择页返回）
      this.loadSelectedPassengers()
      this.loadCoupons()
    }
  },

  onUnload() {
    clearTimeout(this._mapReadyTimer)
    clearTimeout(this._orderTimer)
  },

  // 站点搜索input聚焦/失焦：控制搜索框边框高亮
  onStationSearchFocus(e) {
    this.setData({ searchFocused: true })
  },
  onStationSearchBlur() {
    this.setData({ searchFocused: false })
  },

  // 空方法：供 catchtap/catchtouchmove 阻止冒泡和滚动穿透使用
  noop() {},

  // 计算优惠折扣金额（basePrice=原价）
  calcDiscount(basePrice) {
    const c = this.data.selectedCoupon
    if (!c) return 0
    if (c.min_spend > 0 && basePrice < c.min_spend) return 0 // 不满足门槛
    switch (c.type) {
      case 1: return c.discount_value                          // 满减券
      case 2: return basePrice * (1 - c.discount_value / 10)    // 折扣券
      case 3: return c.discount_value                            // 固定金额券
    }
    return 0
  },

  // 计算优惠后总价并更新显示
  calcTotalPrice() {
    const basePrice = (parseFloat(this.data.intervalPrice) || 0) * this.data.passengers.length
    const discount = this.calcDiscount(basePrice)
    const final = Math.max(0, basePrice - discount)
    const updateData = {
      finalPrice: final.toFixed(2),
      originPrice: basePrice.toFixed(2),
      discountAmount: discount.toFixed(2)
    }
    // 已选优惠券但订单金额不再满足门槛时自动取消选择，避免传给后端报错
    if (this.data.selectedCoupon && this.data.selectedCoupon.min_spend > 0 && basePrice < this.data.selectedCoupon.min_spend) {
      updateData.selectedCoupon = null
    }
    this.setData(updateData)
  },

  // 加载班次详情
  loadTripDetail(id) {
    request({ url: `/api/wx/trips/${id}`, method: 'GET' }).then(res => {
      const trip = res.data.trip || res.data // 兼容旧格式
      const effectivePassedOrder = res.data.effective_passed_order || trip.current_passed_order || 0
      const fromName = trip.route && trip.route.from_station ? trip.route.from_station.name : ''
      const toName = trip.route && trip.route.to_station ? trip.route.to_station.name : ''
      const durationMin = trip.route ? trip.route.duration_minutes : 0
      const durationStr = durationMin >= 60 ? `约${Math.floor(durationMin/60)}小时${durationMin%60 > 0 ? durationMin%60 + '分钟' : ''}` : `约${durationMin}分钟`

      // 解析站点序列
      const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []
      // 预设上下车站：优先用首页传入的，否则默认首末站
      let fromSid = this.data.preFromStationId
      let toSid = this.data.preToStationId
      if (routeStations.length > 0) {
        if (!fromSid || !routeStations.find(s => s.station_id === fromSid)) {
          fromSid = routeStations[0].station_id
        }
        if (!toSid || !routeStations.find(s => s.station_id === toSid)) {
          toSid = routeStations[routeStations.length - 1].station_id
        }
      }
      this.setData({
        trip: {
          ...trip,
          fromName,
          toName,
          durationStr,
          arrivalText: formatArrivalTime(trip.arrival_time, trip.arrival_day_offset),
          seatsText: trip.available_seats === 0 ? '无票' : trip.available_seats <= 5 ? `余${trip.available_seats}座` : '有票'
        },
        routeStations,
        fromStationId: fromSid,
        toStationId: toSid,
        currentPassedOrder: trip.current_passed_order || 0,
        effectivePassedOrder
      }, () => {
        this.calcIntervalPrice()
        // 立即用站点坐标构建地图（不依赖班次状态或登录）
        this.buildStationMap()
        // 延迟渲染地图原生组件，避免与页面初始化抢占资源
        this._mapReadyTimer = setTimeout(() => { this.setData({ mapReady: true }) }, 600)
        // 班次已发车时叠加车辆实时位置
        if (this.data.trip && this.data.trip.status === 2) {
          this.loadVehicleLocation(trip.id)
        }
      })
    }).catch((e) => { log.error('加载班次详情失败', e) })
  },

  // 计算区间票价
  calcIntervalPrice() {
    const { routeStations, fromStationId, toStationId } = this.data
    if (!routeStations.length || !fromStationId || !toStationId) return
    const fromRS = routeStations.find(s => s.station_id === fromStationId)
    const toRS = routeStations.find(s => s.station_id === toStationId)
    if (!fromRS || !toRS) return
    // 实际票价 = max(起步价, 累计差额)，起步价为0时不启用
    let price = Math.max(0, toRS.price - fromRS.price)
    const minFare = this.data.trip && this.data.trip.route && this.data.trip.route.min_fare ? (parseFloat(this.data.trip.route.min_fare) || 0) : 0
    if (minFare > 0 && price < minFare) price = minFare
    const fromName = (fromRS.station && fromRS.station.name) || ''
    const toName = (toRS.station && toRS.station.name) || ''
    this.setData({
      intervalPrice: price.toFixed(2),
      fromStationName: fromName,
      toStationName: toName,
      intervalAvailableSeats: -1,
      intervalSeatsText: '查询中...'
    })
    this.calcTotalPrice()
    this.loadIntervalSeats()
  },

  // 查询区间实时余票
  loadIntervalSeats() {
    const { tripId, fromStationId, toStationId } = this.data
    if (!tripId || !fromStationId || !toStationId) return
    request({
      url: `/api/wx/trips/${tripId}/available-seats?from_station_id=${fromStationId}&to_station_id=${toStationId}`,
      method: 'GET'
    }).then(res => {
      const avail = res.data.available_seats
      const total = res.data.total_seats
      let seatsText = ''
      if (avail === 0) {
        seatsText = '无票'
      } else if (avail <= 5) {
        seatsText = `余${avail}座`
      } else {
        seatsText = '有票'
      }
      this.setData({
        intervalAvailableSeats: avail,
        intervalSeatsText: seatsText
      })
    }).catch((e) => {
      log.error('查询区间余票失败', e)
      this.setData({
        intervalAvailableSeats: -1,
        intervalSeatsText: ''
      })
    })
  },

  // 站点选择
  showFromStationPicker() {
    this.setData({ showStationPicker: true, stationPickerType: 'from', stationSearchValue: '', filteredRouteStations: this.data.routeStations })
  },
  showToStationPicker() {
    this.setData({ showStationPicker: true, stationPickerType: 'to', stationSearchValue: '', filteredRouteStations: this.data.routeStations })
  },
  closeStationPicker() {
    wx.hideKeyboard()
    this.setData({ showStationPicker: false, stationSearchValue: '', searchFocused: false })
  },
  // 站点搜索：支持中文包含/全拼包含/拼音首字母包含
  onStationSearchInput(e) {
    const kw = e.detail.value
    let filtered = this.data.routeStations
    if (kw && kw.trim()) {
      filtered = this.data.routeStations.filter(rs => rs.station && matchStation(rs.station, kw))
    }
    this.setData({ stationSearchValue: kw, filteredRouteStations: filtered })
  },
  // 站点搜索确认（点击键盘"搜索"按钮）：收起键盘，让弹窗恢复全高，方便查看结果
  onStationSearchConfirm() {
    wx.hideKeyboard()
  },
  onStationSelect(e) {
    const stationId = e.currentTarget.dataset.sid
    const { stationPickerType, routeStations, fromStationId, toStationId } = this.data
    if (stationPickerType === 'from') {
      // 不能等于下车站，且 stop_order 要小于下车站
      const toRS = routeStations.find(s => s.station_id === toStationId)
      const newRS = routeStations.find(s => s.station_id === stationId)
      if (toRS && newRS && newRS.stop_order >= toRS.stop_order) {
        wx.showToast({ title: '上车站必须在下车站之前', icon: 'none' })
        return
      }
      this.setData({ fromStationId: stationId, showStationPicker: false, searchFocused: false })
    } else {
      const fromRS = routeStations.find(s => s.station_id === fromStationId)
      const newRS = routeStations.find(s => s.station_id === stationId)
      if (fromRS && newRS && newRS.stop_order <= fromRS.stop_order) {
        wx.showToast({ title: '下车站必须在上车站之后', icon: 'none' })
        return
      }
      this.setData({ toStationId: stationId, showStationPicker: false, searchFocused: false })
    }
    this.calcIntervalPrice()
  },

  // 从 storage 读取乘客实名页勾选的乘客
  loadSelectedPassengers() {
    const selected = wx.getStorageSync('trip_selected_passengers')
    if (!selected || !selected.length) return
    // 合并到已有乘客列表（去重）
    const existing = this.data.passengers
    const newPassengers = [...existing]
    for (const p of selected) {
      if (newPassengers.find(item => item.id === p.id)) continue
      newPassengers.push({
        name: p.name,
        idCardNo: p.id_card_no,
        maskedIdCardNo: maskIdCardNo(p.id_card_no),
        id: p.id,
        uid: 'p' + Date.now() + Math.floor(Math.random() * 1000)
      })
    }
    this.setData({ passengers: newPassengers })
    this.calcTotalPrice()
    // 清除 storage，避免重复读取
    wx.removeStorageSync('trip_selected_passengers')
  },

  // 加载可用优惠券
  loadCoupons() {
    request({ url: '/api/wx/coupons', method: 'GET' }).then(res => {
      this.setData({ coupons: res.data || [] })
    }).catch((e) => { log.error('加载优惠券失败', e) })
  },

  // 显示优惠券选择器（预计算每张券的可用状态）
  showCouponPicker() {
    // 未选乘客时按1人预览，避免 basePrice=0 导致优惠券门槛判断错误
    const personCount = Math.max(this.data.passengers.length, 1)
    const basePrice = (parseFloat(this.data.intervalPrice) || 0) * personCount
    const couponList = this.data.coupons.map(c => {
      const notUsable = c.min_spend > 0 && basePrice < c.min_spend
      const expiredAt = safeDate(c.expired_at)
      return { ...c, not_usable: notUsable, diff_amount: notUsable ? (c.min_spend - basePrice).toFixed(2) : '', expired_at: expiredAt }
    })
    this.setData({ showCouponPicker: true, couponList })
  },

  // 关闭优惠券选择器
  closeCouponPicker() {
    this.setData({ showCouponPicker: false })
  },

  // 选择优惠券（校验最低消费门槛）
  selectCoupon(e) {
    const coupon = e.currentTarget.dataset.coupon
    // 第一道拦截：检查预计算的 not_usable 标记
    if (coupon.not_usable) {
      wx.showToast({ title: `差¥${coupon.diff_amount}元可用，不满足该券使用条件`, icon: 'none' })
      return
    }
    // 第二道拦截：实时校验门槛（防止弹窗打开后乘客数变化）
    const personCount = Math.max(this.data.passengers.length, 1)
    const basePrice = (parseFloat(this.data.intervalPrice) || 0) * personCount
    if (coupon.min_spend > 0 && basePrice < coupon.min_spend) {
      const diff = (coupon.min_spend - basePrice).toFixed(2)
      wx.showToast({ title: `订单金额差¥${diff}元，不满足该券使用条件`, icon: 'none' })
      return
    }
    this.setData({ selectedCoupon: coupon, showCouponPicker: false })
    this.calcTotalPrice()
  },

  // 清除已选优惠券（主页面的“不使用优惠券”链接）
  clearCoupon() {
    this.setData({ selectedCoupon: null })
    this.calcTotalPrice()
  },

  // 弹窗内点击“不使用优惠券”：清除选择并关闭弹窗
  clearCouponInPicker() {
    this.setData({ selectedCoupon: null, showCouponPicker: false })
    this.calcTotalPrice()
  },

  // 添加乘客：跳转到乘客实名页（勾选模式）
  addPassenger() {
    wx.navigateTo({ url: '/pages/passenger/passenger?selectMode=1' })
  },

  // 删除乘客
  removePassenger(e) {
    const idx = e.currentTarget.dataset.idx
    const passengers = this.data.passengers.filter((_, i) => i !== idx)
    this.setData({ passengers })
    this.calcTotalPrice()
  },

  // 联系人输入
  onContactNameInput(e) {
    this.setData({ contactName: e.detail.value })
  },

  onContactPhoneInput(e) {
    this.setData({ contactPhone: e.detail.value })
  },

  // 联系人输入框聚焦/失焦（控制边框变蓝）
  onContactFocus(e) {
    this.setData({ contactFocused: e.currentTarget.dataset.field })
  },
  onContactBlur() {
    this.setData({ contactFocused: '' })
  },

  // 点击到站进度跳转到站进度页面（查看完整站点时间线）
  goToProgressDetail() {
    const { tripId } = this.data
    if (!tripId) return
    wx.navigateTo({
      url: `/pages/bus-progress/bus-progress?id=${tripId}`
    })
  },

  // 点击车辆位置区域：有车辆位置直接打开微信地图导航，否则打开站点位置
  handleViewTrack() {
    const trip = this.data.trip
    if (!trip) return
    if (this.data.vehicleLocation) {
      // 有车辆实时位置，直接打开微信地图
      const loc = this.data.vehicleLocation
      wx.openLocation({
        latitude: loc.latitude,
        longitude: loc.longitude,
        scale: 16,
        name: '车辆当前位置',
        address: this.data.fromStationName + ' → ' + this.data.toStationName
      })
    } else if (this.data.vehicleMarkers && this.data.vehicleMarkers.length > 0) {
      // 没有车辆位置但有站点坐标，打开出发站位置
      const fromMarker = this.data.vehicleMarkers[0]
      wx.openLocation({
        latitude: fromMarker.latitude,
        longitude: fromMarker.longitude,
        scale: 14,
        name: this.data.fromStationName || '出发站',
        address: this.data.fromStationName + ' → ' + this.data.toStationName
      })
    } else if (this.data.hasVehicleLocation) {
      // 地图已显示但无站点坐标，用当前地图中心打开微信地图
      wx.openLocation({
        latitude: this.data.vehicleMapLat,
        longitude: this.data.vehicleMapLng,
        scale: 12,
        name: this.data.fromStationName || '出发站',
        address: this.data.fromStationName + ' → ' + this.data.toStationName
      })
    } else if (trip.status === 2) {
      wx.navigateTo({
        url: `/pages/bus-progress/bus-progress?id=${trip.id}`
      })
    } else {
      wx.showToast({ title: '班次尚未发车，暂无位置信息', icon: 'none' })
    }
  },

  // 用站点坐标构建地图（所有站点标记 + 路线连线）
  buildStationMap() {
    const trip = this.data.trip
    if (!trip || !trip.route) {
      this.setData({ trackPlaceholderText: '暂无路线信息' })
      return
    }

    const markers = []
    const includePoints = []
    const epo = this.data.effectivePassedOrder || 0
    const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []

    // 遍历所有站点，依次添加标记
    routeStations.forEach(function (rs, index) {
      const station = rs.station
      if (!station || !station.latitude || !station.longitude) return
      const stopOrder = rs.stop_order || (index + 1)
      const isFirst = index === 0
      const isLast = index === routeStations.length - 1

      let iconPath = '/images/map-marker-start.png'
      let bgColor = '#999999'
      let prefix = ''
      if (isFirst) {
        iconPath = '/images/map-marker-start.png'
        bgColor = '#34c759'
        prefix = '起 '
      } else if (isLast) {
        iconPath = '/images/map-marker-end.png'
        bgColor = '#ff3b30'
        prefix = '终 '
      } else {
        iconPath = '/images/map-marker-start.png'
        if (stopOrder < epo) bgColor = '#999999'
        else if (stopOrder === epo) bgColor = '#1296db'
        else bgColor = '#666666'
        prefix = '第' + stopOrder + '站 '
      }

      markers.push({
        id: 100 + stopOrder,
        latitude: station.latitude,
        longitude: station.longitude,
        iconPath: iconPath,
        width: isFirst || isLast ? 25 : 20,
        height: isFirst || isLast ? 25 : 20,
        anchor: { x: 0.5, y: 0.5 },
        callout: {
          content: prefix + station.name,
          color: '#ffffff',
          fontSize: 10,
          bgColor: bgColor,
          borderRadius: 5,
          padding: 3,
          display: 'ALWAYS'
        }
      })
      includePoints.push({ latitude: station.latitude, longitude: station.longitude })
    })

    // 没有坐标时用默认中心点（商丘市中心）
    const DEFAULT_LAT = 34.0528
    const DEFAULT_LNG = 115.6562

    const polyline = includePoints.length >= 2 ? [{
      points: includePoints.slice(),
      color: '#1a1a1aCC',
      width: 3,
      dottedLine: true,
      arrowLine: true
    }] : []

    // 中心点取所有站点的中间站，或第一个站
    const center = includePoints.length > 0
      ? includePoints[Math.floor(includePoints.length / 2)]
      : { latitude: DEFAULT_LAT, longitude: DEFAULT_LNG }

    this.setData({
      hasVehicleLocation: true,
      vehicleMarkers: markers,
      vehiclePolyline: polyline,
      vehicleIncludePoints: includePoints,
      vehicleMapLng: center.longitude,
      vehicleMapLat: center.latitude
    })
  },

  // 加载车辆实时位置（班次已发车时调用，仅用于将地图视野聚焦到车辆所在区域，不再绘制车辆标记）
  loadVehicleLocation(tripId) {
    const token = wx.getStorageSync('user_token')
    if (!token) return
    request({
      url: `/api/wx/trips/${tripId}/location`,
      method: 'GET'
    }).then(res => {
      const location = res.data.location
      if (!location) return

      const stationMarkers = this.data.vehicleMarkers || []
      const stationPoints = stationMarkers.map(m => ({ latitude: m.latitude, longitude: m.longitude }))
      // 将车辆实时位置纳入地图视野（聚焦车辆所在区域），但不绘制为标记
      const includePoints = stationPoints.slice()
      includePoints.push({ latitude: location.latitude, longitude: location.longitude })
      const polyline = stationPoints.length >= 2 ? [{
        points: stationPoints,
        color: '#1a1a1aCC',
        width: 3,
        dottedLine: true,
        arrowLine: true
      }] : []

      this.setData({
        vehicleLocation: location,
        vehiclePolyline: polyline,
        vehicleIncludePoints: includePoints,
        vehicleMapLng: location.longitude,
        vehicleMapLat: location.latitude
      })
    }).catch(() => {
      // 站点地图已在显示，获取车辆位置失败不影响地图展示
    })
  },

  // 创建订单
  async createOrder() {
    if (!this.data.isLogin) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }

    const { trip, passengers, contactName, contactPhone, fromStationId, toStationId, intervalAvailableSeats } = this.data

    if (!trip || trip.status !== 1) {
      wx.showToast({ title: '该班次不可预订', icon: 'none' })
      return
    }

    // 优先校验区间余票（比整车余票更精确），未查询到区间余票时回退整车余票
    if (intervalAvailableSeats !== undefined && intervalAvailableSeats !== null) {
      if (intervalAvailableSeats < 0) {
        wx.showToast({ title: '余票查询中，请稍后重试', icon: 'none' })
        return
      }
      if (intervalAvailableSeats < passengers.length) {
        wx.showToast({ title: '该区间已无票', icon: 'none' })
        return
      }
    } else if (trip.available_seats === 0) {
      wx.showToast({ title: '该班次已无票', icon: 'none' })
      return
    }
    if (!fromStationId || !toStationId) {
      wx.showToast({ title: '请选择上车站和下车站', icon: 'none' })
      return
    }
    if (passengers.length === 0) {
      wx.showModal({
        title: '添加乘客',
        content: '请先添加或勾选乘车人身份信息，每位乘客对应一张车票',
        confirmText: '去添加',
        cancelText: '取消',
        success: (res) => {
          if (res.confirm) {
            wx.navigateTo({ url: '/pages/passenger/passenger?selectMode=1' })
          }
        }
      })
      return
    }
    if (!contactName) {
      wx.showToast({ title: '请填写联系人姓名', icon: 'none' })
      return
    }
    // 手机号格式校验
    if (!contactPhone || !validatePhone(contactPhone)) {
      wx.showToast({ title: '请填写正确的联系人手机号（11位）', icon: 'none' })
      return
    }

    // 验证乘客信息
    for (let i = 0; i < passengers.length; i++) {
      const p = passengers[i]
      if (!p.name) {
        wx.showToast({ title: `第${i + 1}位乘客姓名不能为空`, icon: 'none' })
        return
      }
      // 常用乘客（p.id 存在）的身份证号由后端通过 passenger_id 获取，前端无需校验
      if (!p.idCardNo && !p.id) {
        wx.showToast({ title: `第${i + 1}位乘客身份证号不能为空`, icon: 'none' })
        return
      }
    }

    // 二次确认：展示完整行程信息，防止买错票/交错车
    const routeStations = this.data.routeStations
    const fromRS = routeStations.find(s => s.station_id === fromStationId)
    const toRS = routeStations.find(s => s.station_id === toStationId)
    const fromOrder = fromRS ? fromRS.stop_order : 0
    const toOrder = toRS ? toRS.stop_order : 0
    const stopCount = (fromOrder && toOrder) ? (toOrder - fromOrder - 1) : 0
    const directLabel = stopCount === 0 ? '直达' : '经' + stopCount + '站'
    const basePrice = (parseFloat(this.data.intervalPrice) || 0) * passengers.length
    const discount = this.calcDiscount(basePrice)
    const totalPrice = Math.max(0, basePrice - discount).toFixed(2)
    const tripNo = trip.trip_no || ''
    const arrivalText = formatArrivalTime(trip.arrival_time, trip.arrival_day_offset)
    const confirmContent = '车次：' + tripNo + '\n日期：' + trip.trip_date + ' ' + trip.departure_time + '发车\n到达：' + (arrivalText || '详见班次') + '\n上车：' + this.data.fromStationName + '（第' + fromOrder + '站）\n下车：' + this.data.toStationName + '（第' + toOrder + '站）\n' + directLabel + '　' + passengers.length + '人\n合计：¥' + totalPrice
    const confirmed = await new Promise(resolve => {
      wx.showModal({
        title: '请确认行程信息',
        content: confirmContent,
        confirmText: '确认下单',
        cancelText: '再看看',
        success: (res) => resolve(res.confirm),
        fail: () => resolve(false)
      })
    })
    if (!confirmed) return

    this.setData({ orderLoading: true })

    try {
      const res = await request({
        url: '/api/wx/orders',
        method: 'POST',
        data: {
          trip_id: String(trip.id),
          from_station_id: fromStationId,
          to_station_id: toStationId,
          passengers: passengers.map(p => ({
            passenger_id: p.id || 0,
            name: p.name,
            id_card_no: p.id ? '' : p.idCardNo
          })),
          contact_name: contactName,
          contact_phone: contactPhone,
          coupon_id: this.data.selectedCoupon ? this.data.selectedCoupon.id : 0
        }
      })

      const orderId = res.data.order.id
      wx.showToast({ title: '下单成功', icon: 'success' })

      // 弹出支付确认（orderLoading保持true，防止500ms窗口内重复提交）
      const redirect = () => {
        wx.redirectTo({ url: `/pages/order-detail/order-detail?id=${orderId}` })
      }
      this._orderTimer = setTimeout(() => {
        this.setData({ orderLoading: false })
        startPayment(orderId, {
          confirmTitle: '支付提示',
          confirmContent: `订单金额 ¥${(parseFloat(res.data.order.total_price) || 0).toFixed(2)}，是否立即支付？`,
          confirmText: '去支付',
          cancelText: '稍后',
          onPaid: () => setTimeout(redirect, 1000),
          onFail: () => setTimeout(redirect, 1000),
          onError: () => setTimeout(redirect, 1000),
          onCancel: redirect
        })
      }, 500)
    } catch (e) {
      // error handled by request
      this.setData({ orderLoading: false })
    }
  },

  // 分享给好友（分享班次信息，好友点开直达该班次详情）
  onShareAppMessage() {
    var trip = this.data.trip
    if (trip) {
      var title = (this.data.fromStationName || trip.fromName || '班次详情') +
        '→' + (this.data.toStationName || trip.toName || '') +
        ' ' + (trip.departure_time || '')
      var path = '/pages/trip-detail/trip-detail?id=' + this.data.tripId
      if (this.data.fromStationId) {
        path += '&from_sid=' + this.data.fromStationId
      }
      if (this.data.toStationId) {
        path += '&to_sid=' + this.data.toStationId
      }
      return { title: title, path: path }
    }
    return {
      title: '狸猫日志售票 · 班次详情',
      path: '/pages/home/home'
    }
  },

  // 分享到朋友圈
  onShareTimeline() {
    var trip = this.data.trip
    var title = '狸猫日志售票 · 在线订票便捷出行'
    if (trip) {
      title = (this.data.fromStationName || '') +
        '→' + (this.data.toStationName || '') +
        ' ' + (trip.departure_time || '') +
        ' · 狸猫日志售票'
    }
    return { title: title }
  }
})
