// 管理后台 - 售票管理（班次 + 票价，照搬后端 trips.vue）
// 接 /api/wx/admin/trips，超管可增删改
const { request } = require('../../utils/request')

// 班次状态：1=可售 2=已发车 3=已取消 0=下架 4=已完成
const STATUS_TABS = [
  { key: '', text: '全部' },
  { key: '1', text: '可售' },
  { key: '2', text: '已发车' },
  { key: '4', text: '已完成' },
  { key: '3', text: '已取消' },
  { key: '0', text: '下架' }
]
const TRIP_STATUS_TEXT = { 0: '下架', 1: '可售', 2: '已发车', 3: '已取消', 4: '已完成' }
const CLS_MAP = { 0: 'ban', 1: 'ok', 2: 'travel', 3: 'ban', 4: 'done' }
// 表单状态选项（与后端编辑弹窗一致，不含“已完成”）
const FORM_STATUS = [
  { value: 1, label: '可售' },
  { value: 2, label: '已发车' },
  { value: 3, label: '已取消' },
  { value: 0, label: '下架' }
]

Page({
  data: {
    statusTabs: STATUS_TABS,
    activeStatus: '',
    keyword: '',         // 班次号搜索
    tripDate: '',        // 发车日期筛选
    routes: [],          // 线路筛选下拉（含“全部线路”）
    routeIndex: 0,
    showRoutePicker: false,

    list: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: true,
    loading: false,
    loadingMore: false,

    isSuperAdmin: false,

    // 表单弹层
    formVisible: false,
    formMode: 'add',
    submitting: false,
    // 表单下拉选项
    formRoutes: [],
    formRouteNames: [],
    formVehicles: [],
    formVehicleNames: [],
    formDrivers: [],
    formDriverNames: ['无司机'],
    formStatusNames: ['可售', '已发车', '已取消', '下架'],
    // 表单数据（对齐后端 editing 字段）
    form: {
      id: 0, route_id: 0, vehicle_id: 0, driver_id: 0, trip_no: '',
      trip_date: '', departure_time: '', arrival_date: '', arrival_time: '',
      total_seats: '', available_seats: '', base_price: '', status: 1
    },
    tripRouteIndex: 0,
    tripVehicleIndex: 0,
    tripDriverIndex: 0,
    tripStatusIndex: 0,

    // 批量生成弹层（简化版：日期多选 + 默认发车/到达时间 + 跨天天数 + 线路/车辆/司机 + 参考票价）
    batchVisible: false,
    batchSubmitting: false,
    batchForm: {
      route_id: 0, vehicle_id: 0, driver_id: 0, base_price: '',
      departure_time: '', arrival_time: '', arrival_day_offset: 0
    },
    batchRouteIndex: 0,
    batchVehicleIndex: 0,
    batchDriverIndex: 0,
    batchDateInput: '',          // 日期选择器临时值，待“添加”进 selectedDates
    selectedDates: [],           // 已选发车日期列表（已去重并排序）

    // 乘客名单弹层（只读）
    passengersVisible: false,
    passengersLoading: false,
    passengersTripNo: '',
    passengersList: [],

    // 每站票价弹层（只读，展示该班次所属线路的每站累计票价）
    fareVisible: false,
    fareLoading: false,
    fareTripNo: '',
    fareRouteName: '',
    fareStations: [],     // [{ idx, name, cumPrice, segPrice, isStart, isEnd }]
    fareTotalPrice: '',   // 全程票价（末站累计价）

    // 清理历史班次弹层（两阶段：预览 → 确认）
    cleanupVisible: false,
    cleanupSubmitting: false,
    cleanupForm: { before_date: '', route_id: 0 },
    cleanupRouteIndex: 0,
    cleanupPreview: null         // 预览返回的统计对象，null=未预览
  },

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    this.setData({ isSuperAdmin: userInfo.admin_role === 1 })
    this.loadRoutes()
    this.loadList()
  },

  // 筛选线路下拉（含“全部线路”）
  loadRoutes() {
    request({ url: '/api/wx/admin/routes/all', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      const routes = [{ id: 0, name: '全部线路' }].concat(raw.map(r => {
        const fromName = r.from_station ? r.from_station.name : ''
        const toName = r.to_station ? r.to_station.name : ''
        return { id: r.id, name: r.name || (fromName && toName ? (fromName + ' → ' + toName) : ('线路#' + r.id)) }
      }))
      this.setData({ routes })
    }).catch(() => {})
  },

  loadList(append) {
    if (append) {
      if (this.data.loadingMore || !this.data.hasMore || this.data.loading) return
    } else {
      if (this.data.loading) return
    }
    const page = append ? this.data.page + 1 : 1
    if (append) this.setData({ loadingMore: true })
    else this.setData({ loading: true })

    const data = { page, page_size: this.data.pageSize }
    if (this.data.activeStatus) data.status = this.data.activeStatus
    if (this.data.tripDate) data.trip_date = this.data.tripDate
    if (this.data.keyword.trim()) data.trip_no = this.data.keyword.trim()
    const route = this.data.routes[this.data.routeIndex]
    if (route && route.id) data.route_id = route.id

    request({ url: '/api/wx/admin/trips', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(t => this.formatTrip(t))
      const list = append ? this.data.list.concat(items) : items
      this.setData({
        list, page, total,
        hasMore: list.length < total,
        loading: false, loadingMore: false
      })
    }).catch(() => {
      this.setData({ loading: false, loadingMore: false, list: append ? this.data.list : [] })
    })
  },

  // 对齐后端表格列：班次号/出发/到达/票价/可售·总座/司机/状态
  formatTrip(t) {
    const route = t.route || {}
    const fromName = (route.from_station && route.from_station.name) || '-'
    const toName = (route.to_station && route.to_station.name) || '-'
    const vehicle = t.vehicle || {}
    const status = t.status
    const offset = t.arrival_day_offset || 0
    return {
      id: t.id,
      trip_no: t.trip_no || ('#' + t.id),
      route_id: t.route_id || route.id || 0,
      vehicle_id: t.vehicle_id || (vehicle.id || 0),
      driver_id: t.driver_id || 0,
      route_name: route.name || (fromName + ' → ' + toName),
      from: fromName,
      to: toName,
      trip_date: t.trip_date || '',
      departure_time: (t.departure_time || '').substring(0, 5),
      arrival_time: (t.arrival_time || '').substring(0, 5),
      arrival_date: this.arrivalDateStr(t.trip_date, offset),
      arrival_day_offset: offset,
      vehicle_plate: vehicle.plate_no || '未分配',
      driver_text: t.driver_id ? ('司机#' + t.driver_id) : '未分配',
      total_seats: t.total_seats || 0,
      available_seats: t.available_seats || 0,
      base_price: (t.base_price || 0).toFixed(2),
      status,
      statusText: TRIP_STATUS_TEXT[status] || '未知',
      statusCls: CLS_MAP[status] || ''
    }
  },

  // trip_date + offset => 到达日期
  arrivalDateStr(tripDate, offset) {
    if (!tripDate) return ''
    const d = new Date(tripDate + 'T00:00:00')
    d.setDate(d.getDate() + (offset || 0))
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return d.getFullYear() + '-' + m + '-' + day
  },
  // 到达日期 - 发车日期 => offset
  offsetFromDates(tripDate, arrivalDate) {
    if (!tripDate || !arrivalDate) return 0
    return Math.round((new Date(arrivalDate + 'T00:00:00').getTime() - new Date(tripDate + 'T00:00:00').getTime()) / 86400000)
  },

  onStatusTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeStatus) return
    this.setData({ activeStatus: key })
    this.loadList(false)
  },
  onKeywordInput(e) { this.setData({ keyword: e.detail.value }) },
  onSearchConfirm() { this.loadList(false) },
  onClearKeyword() { this.setData({ keyword: '' }); this.loadList(false) },
  onDateChange(e) { this.setData({ tripDate: e.detail.value }); this.loadList(false) },
  onClearDate() { this.setData({ tripDate: '' }); this.loadList(false) },
  toggleRoutePicker() { this.setData({ showRoutePicker: !this.data.showRoutePicker }) },
  onRoutePick(e) {
    this.setData({ routeIndex: e.currentTarget.dataset.index, showRoutePicker: false })
    this.loadList(false)
  },
  preventBubble() {},
  onPullDownRefresh() { this.loadList(false); wx.stopPullDownRefresh() },
  onLoadMore() { this.loadList(true) },

  // ===== 表单下拉选项懒加载 =====
  loadFormRoutes() {
    if (this.data.formRoutes.length > 0) return Promise.resolve()
    return request({ url: '/api/wx/admin/routes/all', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      const options = raw.map(r => {
        const fromName = r.from_station ? r.from_station.name : ''
        const toName = r.to_station ? r.to_station.name : ''
        return { id: r.id, name: r.name || (fromName && toName ? (fromName + ' → ' + toName) : ('线路#' + r.id)) }
      })
      this.setData({ formRoutes: options, formRouteNames: options.map(o => o.name) })
    }).catch(() => {})
  },
  loadFormVehicles() {
    if (this.data.formVehicles.length > 0) return Promise.resolve()
    return request({ url: '/api/wx/admin/vehicles/all', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      const options = raw.map(v => ({ id: v.id, plate_no: v.plate_no || ('车辆#' + v.id) }))
      this.setData({ formVehicles: options, formVehicleNames: options.map(o => o.plate_no) })
    }).catch(() => {})
  },
  loadFormDrivers() {
    if (this.data.formDrivers.length > 0) return Promise.resolve()
    return request({ url: '/api/wx/admin/drivers/all', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      const options = raw.map(d => ({ id: d.id, name: d.name || ('司机#' + d.id) }))
      this.setData({ formDrivers: options, formDriverNames: ['无司机'].concat(options.map(o => o.name)) })
    }).catch(() => {})
  },

  // ===== 新增 / 编辑 / 删除 =====
  openAdd() {
    if (!this.data.isSuperAdmin) return
    Promise.all([this.loadFormRoutes(), this.loadFormVehicles(), this.loadFormDrivers()]).then(() => {
      const today = this.todayStr()
      this.setData({
        formVisible: true, formMode: 'add',
        form: { id: 0, route_id: 0, vehicle_id: 0, driver_id: 0, trip_no: '', trip_date: today, departure_time: '', arrival_date: today, arrival_time: '', total_seats: '', available_seats: '', base_price: '', status: 1 },
        tripRouteIndex: 0, tripVehicleIndex: 0, tripDriverIndex: 0, tripStatusIndex: 0
      })
    })
  },

  openEdit(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(t => t.id === id)
    if (!item) return
    Promise.all([this.loadFormRoutes(), this.loadFormVehicles(), this.loadFormDrivers()]).then(() => {
      const rIdx = Math.max(0, this.data.formRoutes.findIndex(r => r.id === item.route_id))
      const vIdx = Math.max(0, this.data.formVehicles.findIndex(v => v.id === item.vehicle_id))
      let dIdx = 0
      if (item.driver_id) {
        const i = this.data.formDrivers.findIndex(d => d.id === item.driver_id)
        dIdx = i >= 0 ? (i + 1) : 0
      }
      const sIdx = Math.max(0, FORM_STATUS.findIndex(s => s.value === item.status))
      this.setData({
        formVisible: true, formMode: 'edit',
        form: {
          id: item.id,
          route_id: item.route_id || 0,
          vehicle_id: item.vehicle_id || 0,
          driver_id: item.driver_id || 0,
          trip_no: item.trip_no && item.trip_no.indexOf('#') === 0 ? '' : (item.trip_no || ''),
          trip_date: item.trip_date || this.todayStr(),
          departure_time: item.departure_time || '',
          arrival_date: item.arrival_date || item.trip_date || this.todayStr(),
          arrival_time: item.arrival_time || '',
          total_seats: item.total_seats ? String(item.total_seats) : '',
          available_seats: item.available_seats ? String(item.available_seats) : '',
          base_price: item.base_price === '0.00' ? '' : item.base_price,
          status: item.status
        },
        tripRouteIndex: rIdx, tripVehicleIndex: vIdx, tripDriverIndex: dIdx, tripStatusIndex: sIdx
      })
    })
  },

  closeForm() {
    if (this.data.submitting) return
    this.setData({ formVisible: false })
  },

  onFormInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ ['form.' + field]: e.detail.value })
  },
  onTripRouteChange(e) {
    const idx = Number(e.detail.value)
    const r = this.data.formRoutes[idx]
    this.setData({ tripRouteIndex: idx, 'form.route_id': r ? r.id : 0 })
  },
  onTripVehicleChange(e) {
    const idx = Number(e.detail.value)
    const v = this.data.formVehicles[idx]
    this.setData({ tripVehicleIndex: idx, 'form.vehicle_id': v ? v.id : 0 })
  },
  onTripDriverChange(e) {
    const idx = Number(e.detail.value)
    const driverId = idx === 0 ? 0 : (this.data.formDrivers[idx - 1] ? this.data.formDrivers[idx - 1].id : 0)
    this.setData({ tripDriverIndex: idx, 'form.driver_id': driverId })
  },
  onTripStatusChange(e) {
    const idx = Number(e.detail.value)
    this.setData({ tripStatusIndex: idx, 'form.status': FORM_STATUS[idx].value })
  },
  onTripDateChange(e) {
    // 发车日期变更时，若到达日期早于发车日期则自动同步为发车日（对齐后端 onTripDateChange）
    const tripDate = e.detail.value
    let arrivalDate = this.data.form.arrival_date
    if (!arrivalDate || arrivalDate < tripDate) arrivalDate = tripDate
    this.setData({ 'form.trip_date': tripDate, 'form.arrival_date': arrivalDate })
  },
  onArrDateChange(e) { this.setData({ 'form.arrival_date': e.detail.value }) },
  onDepTimeChange(e) { this.setData({ 'form.departure_time': e.detail.value }) },
  onArrTimeChange(e) { this.setData({ 'form.arrival_time': e.detail.value }) },

  onSave() {
    if (this.data.submitting) return
    const f = this.data.form
    if (!f.route_id) { wx.showToast({ title: '请选择线路', icon: 'none' }); return }
    if (!f.vehicle_id) { wx.showToast({ title: '请选择车辆', icon: 'none' }); return }
    if (!f.trip_date) { wx.showToast({ title: '请选择发车日期', icon: 'none' }); return }
    if (!f.departure_time) { wx.showToast({ title: '请选择发车时间', icon: 'none' }); return }
    if (!f.arrival_date) { wx.showToast({ title: '请选择到达日期', icon: 'none' }); return }
    if (!f.arrival_time) { wx.showToast({ title: '请选择到达时间', icon: 'none' }); return }
    if (f.arrival_date < f.trip_date) { wx.showToast({ title: '到达日期不能早于发车日期', icon: 'none' }); return }
    const offset = this.offsetFromDates(f.trip_date, f.arrival_date)
    if (offset === 0 && f.departure_time >= f.arrival_time) {
      wx.showToast({ title: '当天到达须晚于发车，跨天请改到达日期', icon: 'none' }); return
    }
    const payload = {
      route_id: Number(f.route_id),
      vehicle_id: Number(f.vehicle_id),
      driver_id: Number(f.driver_id) || 0,
      trip_no: (f.trip_no || '').trim(),
      trip_date: f.trip_date,
      departure_time: f.departure_time,
      arrival_time: f.arrival_time,
      arrival_day_offset: offset,
      total_seats: parseInt(f.total_seats, 10) || 0,
      available_seats: parseInt(f.available_seats, 10) || 0,
      base_price: Number(f.base_price) || 0,
      status: Number(f.status)
    }
    const isEdit = this.data.formMode === 'edit'
    const url = isEdit ? ('/api/wx/admin/trips/' + f.id) : '/api/wx/admin/trips'
    this.setData({ submitting: true })
    wx.showLoading({ title: '保存中...', mask: true })
    request({ url, method: isEdit ? 'PUT' : 'POST', data: payload }).then(() => {
      wx.hideLoading()
      wx.showToast({ title: isEdit ? '已保存' : '已新增', icon: 'success' })
      this.setData({ submitting: false, formVisible: false })
      this.loadList(false)
    }).catch(() => {
      wx.hideLoading()
      this.setData({ submitting: false })
    })
  },

  onDelete(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(t => t.id === id)
    if (!item) return
    wx.showModal({
      title: '删除确认', content: '确定删除班次「' + item.trip_no + '」吗？',
      confirmText: '删除', confirmColor: '#f5222d',
      success: r => {
        if (!r.confirm) return
        wx.showLoading({ title: '删除中...', mask: true })
        request({ url: '/api/wx/admin/trips/' + id + '?force=1', method: 'DELETE' }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '删除成功', icon: 'success' })
          this.loadList(false)
        }).catch(() => { wx.hideLoading() })
      }
    })
  },

  todayStr() {
    const d = new Date()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return d.getFullYear() + '-' + m + '-' + day
  },

  // ===== 批量生成班次 =====
  openBatch() {
    if (!this.data.isSuperAdmin) return
    Promise.all([this.loadFormRoutes(), this.loadFormVehicles(), this.loadFormDrivers()]).then(() => {
      this.setData({
        batchVisible: true,
        batchForm: { route_id: 0, vehicle_id: 0, driver_id: 0, base_price: '', departure_time: '', arrival_time: '', arrival_day_offset: 0 },
        batchRouteIndex: 0, batchVehicleIndex: 0, batchDriverIndex: 0,
        batchDateInput: this.todayStr(), selectedDates: []
      })
    })
  },
  closeBatch() {
    if (this.data.batchSubmitting) return
    this.setData({ batchVisible: false })
  },
  onBatchRouteChange(e) {
    const idx = Number(e.detail.value)
    const r = this.data.formRoutes[idx]
    this.setData({ batchRouteIndex: idx, 'batchForm.route_id': r ? r.id : 0 })
  },
  onBatchVehicleChange(e) {
    const idx = Number(e.detail.value)
    const v = this.data.formVehicles[idx]
    this.setData({ batchVehicleIndex: idx, 'batchForm.vehicle_id': v ? v.id : 0 })
  },
  onBatchDriverChange(e) {
    const idx = Number(e.detail.value)
    const driverId = idx === 0 ? 0 : (this.data.formDrivers[idx - 1] ? this.data.formDrivers[idx - 1].id : 0)
    this.setData({ batchDriverIndex: idx, 'batchForm.driver_id': driverId })
  },
  onBatchDepTimeChange(e) { this.setData({ 'batchForm.departure_time': e.detail.value }) },
  onBatchArrTimeChange(e) { this.setData({ 'batchForm.arrival_time': e.detail.value }) },
  onBatchOffsetChange(e) {
    const v = parseInt(e.detail.value, 10)
    this.setData({ 'batchForm.arrival_day_offset': (isNaN(v) || v < 0) ? 0 : v })
  },
  onBatchBasePriceInput(e) { this.setData({ 'batchForm.base_price': e.detail.value }) },
  onBatchDateInput(e) { this.setData({ batchDateInput: e.detail.value }) },
  addBatchDate() {
    const d = this.data.batchDateInput
    if (!d) { wx.showToast({ title: '请选择日期', icon: 'none' }); return }
    if (this.data.selectedDates.indexOf(d) >= 0) {
      wx.showToast({ title: '该日期已添加', icon: 'none' }); return
    }
    const list = this.data.selectedDates.concat(d).sort()
    this.setData({ selectedDates: list, batchDateInput: '' })
  },
  removeBatchDate(e) {
    const d = e.currentTarget.dataset.date
    const list = this.data.selectedDates.filter(x => x !== d)
    this.setData({ selectedDates: list })
  },
  onBatchSave() {
    if (this.data.batchSubmitting) return
    const f = this.data.batchForm
    if (!f.route_id) { wx.showToast({ title: '请选择线路', icon: 'none' }); return }
    if (!f.vehicle_id) { wx.showToast({ title: '请选择车辆', icon: 'none' }); return }
    if (!f.departure_time) { wx.showToast({ title: '请选择发车时间', icon: 'none' }); return }
    if (!f.arrival_time) { wx.showToast({ title: '请选择到达时间', icon: 'none' }); return }
    if (this.data.selectedDates.length === 0) { wx.showToast({ title: '请至少添加一个发车日期', icon: 'none' }); return }
    if (Number(f.arrival_day_offset) === 0 && f.departure_time >= f.arrival_time) {
      wx.showToast({ title: '当天到达须晚于发车，跨天请改到达天数', icon: 'none' }); return
    }
    const tripDates = this.data.selectedDates.map(d => ({
      date: d,
      departure_time: f.departure_time,
      arrival_time: f.arrival_time,
      arrival_day_offset: Number(f.arrival_day_offset) || 0
    }))
    const payload = {
      route_id: Number(f.route_id),
      vehicle_id: Number(f.vehicle_id),
      driver_id: Number(f.driver_id) || 0,
      base_price: Number(f.base_price) || 0,
      trip_dates: tripDates
    }
    this.setData({ batchSubmitting: true })
    wx.showLoading({ title: '生成中...', mask: true })
    request({ url: '/api/wx/admin/trips/batch', method: 'POST', data: payload }).then(res => {
      wx.hideLoading()
      const created = Array.isArray(res.data) ? res.data.length : ((res.data && res.data.created_count) || tripDates.length)
      wx.showToast({ title: '已生成 ' + created + ' 个班次', icon: 'success' })
      this.setData({ batchSubmitting: false, batchVisible: false })
      this.loadList(false)
    }).catch(() => {
      wx.hideLoading()
      this.setData({ batchSubmitting: false })
    })
  },

  // ===== 乘客名单（只读） =====
  openPassengers(e) {
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(t => t.id === id)
    if (!item) return
    this.setData({
      passengersVisible: true, passengersLoading: true,
      passengersTripNo: item.trip_no, passengersList: []
    })
    request({ url: '/api/wx/admin/trips/' + id + '/passengers', method: 'GET', silent: true }).then(res => {
      const raw = Array.isArray(res.data) ? res.data : ((res.data && res.data.list) || [])
      const list = raw.map(p => {
        const order = p.order || {}
        const fromName = order.from_station_name || (order.from_station && order.from_station.name) || '-'
        const toName = order.to_station_name || (order.to_station && order.to_station.name) || '-'
        return {
          name: p.name || '-',
          from: fromName,
          to: toName,
          id_card_no: p.id_card_no || '-',
          phone: p.phone || '-',
          seat_no: p.seat_no || '-',
          check_status: p.check_status === 1 ? '已核销' : '未核销',
          checked_at: p.checked_at || '-'
        }
      })
      this.setData({ passengersList: list, passengersLoading: false })
    }).catch(() => {
      this.setData({ passengersLoading: false, passengersList: [] })
    })
  },
  closePassengers() { this.setData({ passengersVisible: false }) },

  // ===== 每站票价（只读，展示线路的累计票价表） =====
  openFare(e) {
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(t => t.id === id)
    if (!item) return
    this.setData({
      fareVisible: true, fareLoading: true,
      fareTripNo: item.trip_no, fareRouteName: item.route_name,
      fareStations: [], fareTotalPrice: ''
    })
    request({ url: '/api/wx/admin/routes/' + item.route_id + '/stations', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      // raw 为 RouteStation 数组（按 stop_order ASC，含 station、price）
      // 累计票价制：每站 price = 从起点到该站的总票价；段价 = 当前站 price - 上一站 price
      const stations = raw.map((rs, i) => {
        const name = (rs.station && rs.station.name) || ('站点#' + rs.station_id)
        const cumPrice = Number(rs.price) || 0
        const prevPrice = i > 0 ? (Number(raw[i - 1].price) || 0) : 0
        const seg = i === 0 ? 0 : (cumPrice - prevPrice)
        return {
          idx: i + 1,
          name,
          cumPrice: cumPrice.toFixed(2),
          segPrice: i === 0 ? '—' : seg.toFixed(2),
          isStart: i === 0,
          isEnd: i === raw.length - 1
        }
      })
      const total = raw.length > 0 ? (Number(raw[raw.length - 1].price) || 0).toFixed(2) : '0.00'
      this.setData({ fareStations: stations, fareTotalPrice: total, fareLoading: false })
    }).catch(() => {
      this.setData({ fareStations: [], fareTotalPrice: '', fareLoading: false })
    })
  },
  closeFare() { this.setData({ fareVisible: false }) },

  // ===== 清理历史班次 =====
  openCleanup() {
    if (!this.data.isSuperAdmin) return
    this.setData({
      cleanupVisible: true, cleanupSubmitting: false, cleanupPreview: null,
      cleanupForm: { before_date: this.todayStr(), route_id: 0 },
      cleanupRouteIndex: 0
    })
  },
  closeCleanup() {
    if (this.data.cleanupSubmitting) return
    this.setData({ cleanupVisible: false, cleanupPreview: null })
  },
  onCleanupDateChange(e) { this.setData({ 'cleanupForm.before_date': e.detail.value, cleanupPreview: null }) },
  onCleanupRouteChange(e) {
    const idx = Number(e.detail.value)
    const r = this.data.routes[idx]
    this.setData({ cleanupRouteIndex: idx, 'cleanupForm.route_id': r ? r.id : 0, cleanupPreview: null })
  },
  previewCleanup() {
    const f = this.data.cleanupForm
    if (!f.before_date) { wx.showToast({ title: '请选择截止日期', icon: 'none' }); return }
    this.setData({ cleanupSubmitting: true })
    wx.showLoading({ title: '预览中...', mask: true })
    request({
      url: '/api/wx/admin/trips/cleanup', method: 'POST',
      data: { before_date: f.before_date, route_id: Number(f.route_id) || 0, force: false }
    }).then(res => {
      wx.hideLoading()
      this.setData({ cleanupSubmitting: false, cleanupPreview: res.data || {} })
    }).catch(() => {
      wx.hideLoading()
      this.setData({ cleanupSubmitting: false })
    })
  },
  executeCleanup() {
    const preview = this.data.cleanupPreview
    if (!preview) { wx.showToast({ title: '请先预览', icon: 'none' }); return }
    if (preview.has_active) {
      wx.showModal({
        title: '存在活跃订单',
        content: '将自动取消并退款 ' + (preview.paid_orders || 0) + ' 笔已支付订单，确定继续？',
        confirmText: '继续清理', confirmColor: '#f5222d',
        success: r => { if (r.confirm) this._doCleanupExecute() }
      })
      return
    }
    this._doCleanupExecute()
  },
  _doCleanupExecute() {
    const f = this.data.cleanupForm
    const preview = this.data.cleanupPreview || {}
    this.setData({ cleanupSubmitting: true })
    wx.showLoading({ title: '清理中...', mask: true })
    request({
      url: '/api/wx/admin/trips/cleanup', method: 'POST',
      data: { before_date: f.before_date, route_id: Number(f.route_id) || 0, force: true }
    }).then(res => {
      wx.hideLoading()
      const deleted = (res.data && res.data.trip_count) || preview.trip_count || 0
      wx.showToast({ title: '已清理 ' + deleted + ' 个班次', icon: 'success' })
      this.setData({ cleanupSubmitting: false, cleanupVisible: false, cleanupPreview: null })
      this.loadList(false)
    }).catch(() => {
      wx.hideLoading()
      this.setData({ cleanupSubmitting: false })
    })
  }
})
