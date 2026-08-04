var log = require('../../utils/log')
const { request } = require('../../utils/request')
const { formatArrivalTime } = require('../../utils/order-helper')
const { safeDate } = require('../../utils/format')

// 改签页：同线路换班次（上下车站/乘客/联系方式不变，仅更换班次/日期/时间）
// 规则（与后端 /orders/:id/change 一致）：
//   - 仅已支付车票订单可改签
//   - 同线路、未驶过上车站
//   - 座位类型保持：无座订单只能改到开放无座的班次；有座订单余座不足时自动降级无座
//   - 价格只降不升，差价自动原路退回
Page({
  data: {
    orderId: 0,
    order: null,
    loading: true,
    tripList: [],
    selectedDate: '',
    todayStr: '',
    maxDateStr: '',
    // 原订单信息
    routeId: 0,
    fromStationId: 0,
    toStationId: 0,
    fromName: '',
    toName: '',
    tripNo: '',
    isStanding: false,   // 原订单是否无座站票
    currentTripId: 0,
    orderDate: '',
    // 确认弹窗
    showConfirm: false,
    selectedTrip: null,
    confirmFareText: '0.00',
    changing: false
  },

  onLoad(options) {
    if (!options.id) {
      wx.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 800)
      return
    }
    // 改签日期范围：今天起 30 天内（过去日期班次不可改签）
    const now = new Date()
    const fmt = d => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    const max = new Date(now.getTime() + 30 * 86400000)
    this.setData({ orderId: options.id, todayStr: fmt(now), maxDateStr: fmt(max) })
    this.loadOrder()
  },

  // 空方法：阻止弹窗内点击冒泡/滚动穿透
  noop() {},

  // 1. 拉订单详情（拿线路、上下车站、原座位类型、原班次）
  loadOrder() {
    request({ url: `/api/wx/orders/${this.data.orderId}`, method: 'GET' }).then(res => {
      const order = res.data.order
      const passengers = res.data.passengers || []
      const isStanding = passengers.some(p => p.seat_type === 1)
      const fromName = order.from_station_name || (order.from_station ? order.from_station.name : '')
      const toName = order.to_station_name || (order.to_station ? order.to_station.name : '')
      this.setData({
        order,
        fromName,
        toName,
        isStanding,
        tripNo: order.trip ? order.trip.trip_no : '',
        routeId: order.route_id,
        fromStationId: order.from_station_id,
        toStationId: order.to_station_id,
        currentTripId: order.trip_id,
        orderDate: safeDate(order.trip_date),
        selectedDate: safeDate(order.trip_date)
      })
      this.loadTrips()
    }).catch((e) => {
      log.error('加载订单详情失败', e)
      this.setData({ loading: false })
    })
  },

  // 2. 查同线路同日期可售班次（从/to区间筛选，客户端再按 route_id 过滤并排除当前班次）
  loadTrips() {
    const { selectedDate, fromStationId, toStationId, routeId, currentTripId } = this.data
    if (!selectedDate || !fromStationId || !toStationId) return
    this.setData({ loading: true })
    request({
      url: `/api/wx/trips?trip_date=${selectedDate}&from_station_id=${fromStationId}&to_station_id=${toStationId}`,
      method: 'GET'
    }).then(res => {
      const rawList = res.data || []
      const isStanding = this.data.isStanding
      const tripList = []
      rawList.forEach(t => {
        // 同线路、排除当前班次
        if (t.route_id !== routeId || t.id === currentTripId) return
        const priceInfo = this._calcTripFare(t)
        if (!priceInfo) return
        const seatsText = t.available_seats === 0 ? '无票' : (t.available_seats <= 5 ? `余${t.available_seats}座` : '有票')
        let selectable = true
        let disabledTip = ''
        // 已发车班次不可改签（避免列表展示但提交被拒；ISO 格式兼容 iOS 解析）
        const depTime = new Date(`${t.trip_date}T${t.departure_time}:00`).getTime()
        if (isNaN(depTime) || depTime <= Date.now()) {
          selectable = false
          disabledTip = '已发车'
        }
        if (selectable && isStanding) {
          // 无座订单：目标班次必须开放无座且有剩余
          if (t.allow_standing !== true || priceInfo.standingAvailable <= 0) {
            selectable = false
            disabledTip = '无座票不可用'
          }
        } else {
          if (t.available_seats === 0 && t.allow_standing !== true) {
            selectable = false
            disabledTip = '已售罄'
          } else if (t.available_seats === 0 && t.allow_standing) {
            disabledTip = '仅剩无座票，改后为无座'
          }
        }
        tripList.push({
          id: t.id,
          trip_no: t.trip_no,
          date: safeDate(t.trip_date),
          departure_time: t.departure_time,
          arrivalText: formatArrivalTime(t.arrival_time, t.arrival_day_offset),
          available_seats: t.available_seats,
          seatsText,
          allow_standing: t.allow_standing === true,
          standing_available: priceInfo.standingAvailable,
          priceText: priceInfo.priceText,
          selectable,
          disabledTip
        })
      })
      tripList.sort((a, b) => (a.date + a.departure_time).localeCompare(b.date + b.departure_time))
      this.setData({ tripList, loading: false })
    }).catch((e) => {
      log.error('查询可改签班次失败', e)
      this.setData({ loading: false })
    })
  },

  // 计算目标班次同一区间的票价（与后端区间票价计算一致：max(起步价, 票价差)）
  // 返回 { fare, standingFare, standingAvailable, priceText }
  _calcTripFare(trip) {
    const stations = (trip.route && trip.route.route_stations) || []
    const fromRS = stations.find(s => s.station_id === this.data.fromStationId)
    const toRS = stations.find(s => s.station_id === this.data.toStationId)
    if (!fromRS || !toRS) return null
    const minFare = trip.route && trip.route.min_fare ? (parseFloat(trip.route.min_fare) || 0) : 0
    let fare = Math.max(0, (parseFloat(toRS.price) || 0) - (parseFloat(fromRS.price) || 0))
    if (minFare > 0 && fare < minFare) fare = minFare
    const discount = trip.standing_discount && trip.standing_discount > 0 && trip.standing_discount < 1 ? trip.standing_discount : 1
    const standingFare = Math.round(fare * discount * 100) / 100
    const standingAvailable = Math.max(0, (trip.standing_quota || 0) - (trip.standing_sold || 0))
    let priceText
    if (this.data.isStanding) {
      priceText = standingFare.toFixed(2) // 无座订单按无座折扣价
    } else if (trip.available_seats > 0) {
      priceText = fare.toFixed(2) // 有座订单余座充足按座位价
    } else if (trip.allow_standing) {
      priceText = standingFare.toFixed(2) // 余座不足自动降级无座，按无座折扣价
    } else {
      priceText = fare.toFixed(2)
    }
    return { fare, standingFare, standingAvailable, priceText }
  },

  // 切换改签日期
  onDateChange(e) {
    const date = e.detail.value
    if (date === this.data.selectedDate) return
    this.setData({ selectedDate: date })
    this.loadTrips()
  },

  // 选中目标班次 → 弹确认框
  onSelectTrip(e) {
    const id = e.currentTarget.dataset.id
    const trip = this.data.tripList.find(t => t.id == id)
    if (!trip) return
    if (!trip.selectable) {
      wx.showToast({ title: trip.disabledTip || '该班次不可改签', icon: 'none' })
      return
    }
    this.setData({
      showConfirm: true,
      selectedTrip: trip,
      confirmFareText: trip.priceText
    })
  },

  cancelConfirm() {
    this.setData({ showConfirm: false })
  },

  // 提交改签
  confirmChange() {
    if (this.data.changing) return
    const trip = this.data.selectedTrip
    if (!trip) return
    this.setData({ changing: true })
    request({
      url: `/api/wx/orders/${this.data.orderId}/change`,
      method: 'POST',
      data: { trip_id: trip.id }
    }).then(res => {
      this.setData({ changing: false, showConfirm: false })
      // 改签成功直接返回订单详情页（onShow 自动刷新新班次），不再多跳一次
      wx.showToast({ title: '改签成功', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 800)
    }).catch(() => {
      // 错误信息由 request 统一 toast 弹出
      this.setData({ changing: false })
    })
  }
})
