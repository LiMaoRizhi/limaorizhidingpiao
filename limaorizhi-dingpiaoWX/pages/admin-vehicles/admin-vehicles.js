// 管理后台 - 车辆管理（接 /api/wx/admin/vehicles）
const { request } = require('../../utils/request')

// 车辆状态：1=可用 0=维修
const STATUS_TABS = [
  { key: '', text: '全部' },
  { key: '1', text: '可用' },
  { key: '0', text: '维修' }
]

// 车型选项
const VEHICLE_TYPES = ['大巴', '中巴', '商务车', '小轿车']

Page({
  data: {
    statusTabs: STATUS_TABS,
    activeStatus: '',
    keyword: '',
    inputFocused: '',
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
    formMode: 'add', // 'add' | 'edit'
    editingId: null,
    submitting: false,
    vehicleTypes: VEHICLE_TYPES,
    typeIndex: 0,
    form: {
      plate_no: '',
      vehicle_type: '',
      seat_count: '',
      status: 1
    }
  },

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    this.setData({ isSuperAdmin: userInfo.admin_role === 1 })
    this.loadList()
  },

  // 输入框聚焦/失焦（黑线框点击变蓝）
  onFieldFocus(e) {
    this.setData({ inputFocused: e.currentTarget.dataset.field || '' })
  },
  onFieldBlur() {
    this.setData({ inputFocused: '' })
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
    if (this.data.keyword.trim()) data.plate_no = this.data.keyword.trim()

    request({ url: '/api/wx/admin/vehicles', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(v => ({
        id: v.id,
        plate_no: v.plate_no || '-',
        vehicle_type: v.vehicle_type || '未分类',
        seat_count: v.seat_count || 0,
        status: v.status,
        statusText: v.status === 1 ? '可用' : '维修',
        statusCls: v.status === 1 ? 'ok' : 'repair'
      }))
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

  onStatusTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeStatus) return
    this.setData({ activeStatus: key })
    this.loadList(false)
  },
  onKeywordInput(e) { this.setData({ keyword: e.detail.value }) },
  onSearchConfirm() { this.loadList(false) },
  onClearKeyword() { this.setData({ keyword: '' }); this.loadList(false) },
  onPullDownRefresh() { this.loadList(false); wx.stopPullDownRefresh() },
  onLoadMore() { this.loadList(true) },

  // 新增 / 编辑 / 删除
  preventBubble() {
    // 阻止冒泡（点击弹窗内容不关闭）
  },

  openAdd() {
    if (!this.data.isSuperAdmin) return
    const defaultType = VEHICLE_TYPES[0]
    this.setData({
      formVisible: true,
      formMode: 'add',
      editingId: null,
      typeIndex: 0,
      form: {
        plate_no: '',
        vehicle_type: defaultType,
        seat_count: '',
        status: 1
      }
    })
  },

  openEdit(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(v => v.id === id)
    if (!item) return
    let typeIndex = VEHICLE_TYPES.indexOf(item.vehicle_type)
    if (typeIndex < 0) typeIndex = 0
    const vehicleType = typeIndex >= 0 && VEHICLE_TYPES.indexOf(item.vehicle_type) >= 0 ? item.vehicle_type : VEHICLE_TYPES[typeIndex]
    this.setData({
      formVisible: true,
      formMode: 'edit',
      editingId: id,
      typeIndex,
      form: {
        plate_no: item.plate_no === '-' ? '' : item.plate_no,
        vehicle_type: vehicleType,
        seat_count: String(item.seat_count || ''),
        status: item.status === 1 ? 1 : 0
      }
    })
  },

  closeForm() {
    if (this.data.submitting) return
    this.setData({ formVisible: false })
  },

  onPlateInput(e) {
    this.setData({ 'form.plate_no': e.detail.value })
  },

  onTypeChange(e) {
    const idx = Number(e.detail.value)
    this.setData({ typeIndex: idx, 'form.vehicle_type': VEHICLE_TYPES[idx] })
  },

  onSeatInput(e) {
    this.setData({ 'form.seat_count': e.detail.value })
  },

  onStatusChange(e) {
    this.setData({ 'form.status': e.detail.value ? 1 : 0 })
  },

  onSave() {
    if (this.data.submitting) return
    const { plate_no, vehicle_type, seat_count, status } = this.data.form
    const plate = (plate_no || '').trim()
    if (!plate) {
      wx.showToast({ title: '请输入车牌号', icon: 'none' })
      return
    }
    const seat = parseInt(seat_count, 10)
    if (!seat || seat <= 0) {
      wx.showToast({ title: '座位数必须大于0', icon: 'none' })
      return
    }
    const payload = {
      plate_no: plate,
      vehicle_type: (vehicle_type || '').trim(),
      seat_count: seat,
      status: status === 1 ? 1 : 0
    }
    const isEdit = this.data.formMode === 'edit' && this.data.editingId
    const url = isEdit ? `/api/wx/admin/vehicles/${this.data.editingId}` : '/api/wx/admin/vehicles'
    const method = isEdit ? 'PUT' : 'POST'
    this.setData({ submitting: true })
    wx.showLoading({ title: '保存中...', mask: true })
    request({ url, method, data: payload }).then(() => {
      wx.hideLoading()
      wx.showToast({ title: '保存成功', icon: 'success' })
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
    const item = this.data.list.find(v => v.id === id)
    if (!item) return
    wx.showModal({
      title: '删除确认',
      content: `确定删除车辆「${item.plate_no}」吗？删除后不可恢复。`,
      confirmText: '删除',
      confirmColor: '#f5222d',
      success: r => {
        if (!r.confirm) return
        this.setData({ submitting: true })
        wx.showLoading({ title: '删除中...', mask: true })
        request({ url: `/api/wx/admin/vehicles/${id}`, method: 'DELETE' }).then(() => {
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
  }
})
