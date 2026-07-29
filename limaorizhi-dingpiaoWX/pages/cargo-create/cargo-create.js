// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')
const { request, checkLogin } = require('../../utils/request')
const stationPickerMixin = require('../../utils/station-picker-mixin')
const { formatArrivalTime } = require('../../utils/order-helper')
const { formatDate, validatePhone } = require('../../utils/format')

Page({
  data: {
    today: '',
    tripDate: '',
    tripList: [],
    selectedTrip: null,
    // 表单
    senderName: '',
    senderPhone: '',
    receiverName: '',
    receiverPhone: '',
    cargoType: '日用品',
    weight: '',
    description: '',
    focusedField: '',
    // 运费预估
    feePreview: null,
    // 提交中
    submitting: false,
    cargoTypes: ['日用品', '食品', '文件', '衣物', '电子产品', '其他'],
    maxWeight: 50, // 默认值，从后端配置获取
    // 站点筛选
    stations: [],
    filteredStations: [],
    selectedFromStation: null,
    selectedToStation: null,
    showStationPicker: false,
    stationPickerType: '',
    stationSearchValue: '',
    stationSearchFocused: false,
    // 路线站点（内部使用，自动取首末站）
    routeStations: [],
    fromStationId: 0,
    toStationId: 0
  },

  onLoad() {
    if (!checkLogin()) return
    this._initialized = true
    const today = formatDate(new Date())
    this.setData({ today, tripDate: today })
    this.loadTrips()
    this.loadConfig()
    this.loadStations()
  },

  onUnload() {
    clearTimeout(this._feeTimer)
  },

  onShow() {
    // 用户从登录页返回后恢复页面（onLoad时未登录被拦截）
    if (!this._initialized && checkLogin()) {
      this._initialized = true
      const today = formatDate(new Date())
      this.setData({ today, tripDate: today })
      this.loadTrips()
      this.loadConfig()
      this.loadStations()
    }
  },

  // 加载后端配置获取最大重量
  loadConfig() {
    request({ url: '/api/wx/config', method: 'GET' }).then(res => {
      const config = res.data || {}
      if (config.cargo_max_weight) {
        const maxW = parseFloat(config.cargo_max_weight) || 50
        if (maxW > 0) {
          this.setData({ maxWeight: maxW })
        }
      }
    }).catch((e) => { log.error('加载配置失败', e) })
  },

  // 日期变化
  onDateChange(e) {
    this.setData({ tripDate: e.detail.value, feePreview: null })
    this.loadTrips()
  },

  // 加载站点列表
  loadStations() {
    request({ url: '/api/wx/stations', method: 'GET' }).then(res => {
      const stations = res.data || []
      this.setData({ stations, filteredStations: stations })
    }).catch((e) => { log.error('加载站点失败', e) })
  },

  // 加载班次列表（按站点+日期筛选，用户手动选择）
  loadTrips() {
    const { tripDate, selectedFromStation, selectedToStation } = this.data
    let url = `/api/wx/trips?trip_date=${tripDate}`
    if (selectedFromStation) url += `&from_station_id=${selectedFromStation.id}`
    if (selectedToStation) url += `&to_station_id=${selectedToStation.id}`
    request({
      url,
      method: 'GET'
    }).then(res => {
      const list = (res.data || []).map(item => {
        const fromName = item.route && item.route.from_station ? item.route.from_station.name : ''
        const toName = item.route && item.route.to_station ? item.route.to_station.name : ''
        const arrival_text = formatArrivalTime(item.arrival_time, item.arrival_day_offset)
        return { ...item, fromName, toName, arrival_text }
      })
      this.setData({ tripList: list, selectedTrip: null, routeStations: [], feePreview: null })
    }).catch((e) => { log.error('加载班次列表失败', e) })
  },

  // 用户手动选择班次
  selectTrip(e) {
    const trip = e.currentTarget.dataset.trip
    if (trip.status !== 1 || trip.available_seats <= 0) {
      wx.showToast({ title: '该班次暂无可用余票', icon: 'none' })
      return
    }
    // 预计算跨天到达文案
    const selectedTrip = { ...trip, arrival_text: formatArrivalTime(trip.arrival_time, trip.arrival_day_offset) }
    const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []
    let fromStationId = 0, toStationId = 0
    if (routeStations.length > 0) {
      fromStationId = routeStations[0].station_id
      toStationId = routeStations[routeStations.length - 1].station_id
    }
    this.setData({
      selectedTrip,
      routeStations,
      fromStationId,
      toStationId,
      feePreview: null
    })
  },

  // 返回班次选择
  backToSelectTrip() {
    this.setData({ selectedTrip: null, routeStations: [], feePreview: null })
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
    this.setData({ feePreview: null })
    this.loadTrips()
  },
  swapStations() {
    const { selectedFromStation, selectedToStation } = this.data
    this.setData({ selectedFromStation: selectedToStation, selectedToStation: selectedFromStation, feePreview: null })
    this.loadTrips()
  },
  clearFilter() {
    this.setData({ selectedFromStation: null, selectedToStation: null, feePreview: null })
    this.loadTrips()
  },

  // 输入框聚焦/失焦
  onFieldFocus(e) { this.setData({ focusedField: e.currentTarget.dataset.field }) },
  onFieldBlur() { this.setData({ focusedField: '' }) },

  // 输入处理
  onSenderNameInput(e) { this.setData({ senderName: e.detail.value }) },
  onSenderPhoneInput(e) { this.setData({ senderPhone: e.detail.value }) },
  onReceiverNameInput(e) { this.setData({ receiverName: e.detail.value }) },
  onReceiverPhoneInput(e) { this.setData({ receiverPhone: e.detail.value }) },
  onCargoTypeChange(e) { this.setData({ cargoType: this.data.cargoTypes[e.detail.value] }) },
  onWeightInput(e) {
    this.setData({ weight: e.detail.value })
    // 防抖
    clearTimeout(this._feeTimer)
    this._feeTimer = setTimeout(() => { this.previewFee() }, 500)
  },
  onDescInput(e) { this.setData({ description: e.detail.value }) },

  // 运费预估
  previewFee() {
    if (!this.data.selectedTrip || !this.data.weight || parseFloat(this.data.weight) <= 0) {
      this.setData({ feePreview: null })
      return
    }
    if (!this.data.fromStationId || !this.data.toStationId) {
      this.setData({ feePreview: null })
      return
    }
    request({
      url: '/api/wx/cargo/preview',
      method: 'POST',
      data: {
        trip_id: String(this.data.selectedTrip.id),
        from_station_id: this.data.fromStationId,
        to_station_id: this.data.toStationId,
        weight: parseFloat(this.data.weight)
      }
    }).then(res => {
      this.setData({ feePreview: res.data })
    }).catch((e) => { log.error('运费预估失败', e) })
  },

  // 提交下单
  handleSubmit() {
    if (!this.data.selectedTrip) {
      wx.showToast({ title: '暂无可选班次', icon: 'none' })
      return
    }
    const trip = this.data.selectedTrip
    if (trip.status !== 1 || trip.available_seats <= 0) {
      wx.showToast({ title: '该班次暂无可用余票', icon: 'none' })
      return
    }
    if (!this.data.senderName.trim()) {
      wx.showToast({ title: '请输入寄件人姓名', icon: 'none' })
      return
    }
    if (!validatePhone(this.data.senderPhone)) {
      wx.showToast({ title: '请输入正确的寄件人手机号', icon: 'none' })
      return
    }
    if (!this.data.receiverName.trim()) {
      wx.showToast({ title: '请输入收件人姓名', icon: 'none' })
      return
    }
    if (!validatePhone(this.data.receiverPhone)) {
      wx.showToast({ title: '请输入正确的收件人手机号', icon: 'none' })
      return
    }
    const weight = parseFloat(this.data.weight)
    if (!weight || weight <= 0 || weight > this.data.maxWeight) {
      wx.showToast({ title: `重量需在0.1~${this.data.maxWeight}kg之间`, icon: 'none' })
      return
    }

    this.setData({ submitting: true })
    request({
      url: '/api/wx/cargo',
      method: 'POST',
      data: {
        trip_id: String(this.data.selectedTrip.id),
        from_station_id: this.data.fromStationId,
        to_station_id: this.data.toStationId,
        sender_name: this.data.senderName.trim(),
        sender_phone: this.data.senderPhone.trim(),
        receiver_name: this.data.receiverName.trim(),
        receiver_phone: this.data.receiverPhone.trim(),
        cargo_type: this.data.cargoType,
        weight: weight,
        description: this.data.description.trim()
      }
    }).then(res => {
      this.setData({ submitting: false })
      const orderId = res.data && res.data.id
      wx.showModal({
        title: '下单成功',
        content: '可在订单详情中查看司机信息并拨打电话联系司机',
        confirmText: '查看订单',
        cancelText: '继续托运',
        success: (modalRes) => {
          if (modalRes.confirm && orderId) {
            wx.navigateTo({ url: `/pages/order-detail/order-detail?id=${orderId}` })
          }
        }
      })
    }).catch((e) => {
      log.error('托运下单失败', e)
      this.setData({ submitting: false })
    })
  },

  // 分享给好友（分享货物托运服务，好友点开可直接寄件）
  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 货物托运便捷寄送',
      path: '/pages/cargo-create/cargo-create'
    }
  }
})
