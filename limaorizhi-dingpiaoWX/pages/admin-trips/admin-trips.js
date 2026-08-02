// 管理后台 - 班次管理（超管可增删改）
// 接 /api/wx/admin/trips + /api/wx/admin/routes/all（线路筛选下拉）
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
const CLS_MAP = { 0: 'cancel', 1: 'sale', 2: 'travel', 3: 'cancel', 4: 'done' }
// 表单状态选项（与后端编辑弹窗一致，不含"已完成"——已完成是系统流转终态，不可手动设回）
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
    keyword: '',        // 班次号搜索
    tripDate: '',       // 发车日期筛选
    routes: [],         // 线路筛选下拉（含“全部线路”）
    routeIndex: 0,      // 选中的筛选线路下标（0=全部）
    showRoutePicker: false,

    list: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: true,
    loading: false,
    loadingMore: false,

    // 权限
    isSuperAdmin: false,

    // 表单弹层
    formVisible: false,
    formMode: 'add',    // 'add' | 'edit'
    submitting: false,
    // 表单下拉选项（与筛选下拉分开，不含“全部”）
    formRoutes: [],
    formRouteNames: [],
    formVehicles: [],
    formVehicleNames: [],
    formDrivers: [],
    formDriverNames: ['无司机'],
    // 表单数据
    form: {
      id: 0, route_id: 0, vehicle_id: 0, driver_id: 0,
      trip_no: '', trip_date: '', departure_time: '', arrival_time: '',
      arrival_day_offset: 0, total_seats: '', base_price: '', status: 1
    },
    tripRouteIndex: 0,
    tripVehicleIndex: 0,
    tripDriverIndex: 0,
    tripStatusIndex: 0,
    formStatusNames: ['可售', '已发车', '已取消', '下架']
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
        return {
          id: r.id,
          name: r.name || (fromName && toName ? (fromName + ' → ' + toName) : ('线路#' + r.id))
        }
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

  formatTrip(t) {
    const route = t.route || {}
    const fromName = (route.from_station && route.from_station.name) || '-'
    const toName = (route.to_station && route.to_station.name) || '-'
    const vehicle = t.vehicle || {}
    const status = t.status
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
      departure_time: t.departure_time || '',
      arrival_time: t.arrival_time || '',
      arrival_day_offset: t.arrival_day_offset || 0,
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
    const index = e.currentTarget.dataset.index
    this.setData({ routeIndex: index, showRoutePicker: false })
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
      this.setData({
        formVisible: true, formMode: 'add',
        form: { id: 0, route_id: 0, vehicle_id: 0, driver_id: 0, trip_no: '', trip_date: this.todayStr(), departure_time: '', arrival_time: '', arrival_day_offset: 0, total_seats: '', base_price: '', status: 1 },
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
      // 已完成(4)不可手动设回，回退显示为"可售"前的原状态——实际编辑已完成班次无意义，这里兜底取 0
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
          arrival_time: item.arrival_time || '',
          arrival_day_offset: item.arrival_day_offset || 0,
          total_seats: item.total_seats ? String(item.total_seats) : '',
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
  onTripDateChange(e) { this.setData({ 'form.trip_date': e.detail.value }) },
  onDepTimeChange(e) { this.setData({ 'form.departure_time': e.detail.value }) },
  onArrTimeChange(e) { this.setData({ 'form.arrival_time': e.detail.value }) },

  onSave() {
    if (this.data.submitting) return
    const f = this.data.form
    if (!f.route_id) { wx.showToast({ title: '请选择线路', icon: 'none' }); return }
    if (!f.vehicle_id) { wx.showToast({ title: '请选择车辆', icon: 'none' }); return }
    if (!f.trip_date) { wx.showToast({ title: '请选择发车日期', icon: 'none' }); return }
    if (!f.departure_time) { wx.showToast({ title: '请选择发车时间', icon: 'none' }); return }
    if (!f.arrival_time) { wx.showToast({ title: '请选择到达时间', icon: 'none' }); return }
    const totalSeats = parseInt(f.total_seats, 10)
    if (!totalSeats || totalSeats <= 0) { wx.showToast({ title: '座位数必须大于0', icon: 'none' }); return }
    const offset = Number(f.arrival_day_offset) || 0
    if (offset === 0 && f.departure_time >= f.arrival_time) {
      wx.showToast({ title: '当天到达须晚于发车，跨天请填到达天数', icon: 'none' }); return
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
      total_seats: totalSeats,
      base_price: Number(f.base_price) || 0,
      status: Number(f.status) || 0
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
        this.setData({ submitting: true })
        wx.showLoading({ title: '删除中...', mask: true })
        request({ url: '/api/wx/admin/trips/' + id, method: 'DELETE' }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '删除成功', icon: 'success' })
          this.setData({ submitting: false })
          this.loadList(false)
        }).catch(() => {
          wx.hideLoading()
          this.setData({ submitting: false })
        })
      }
    })
  },

  todayStr() {
    const d = new Date()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return d.getFullYear() + '-' + m + '-' + day
  }
})
