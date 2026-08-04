// 管理后台 - 订单管理
// 接 /api/wx/admin/orders（列表/详情/状态流转/超管退款）
const { request } = require('../../utils/request')

// 状态标签：与后端 model/consts.go 一致
// 车票订单：0待支付 1待出行 2已完成 3已退款 4已取消
// 托运订单：0待支付 1待运输 2运输中 3已到达 4已取消 5已取件
const STATUS_TABS = [
  { key: '', text: '全部' },
  { key: '0', text: '待支付' },
  { key: '1', text: '待出行' },
  { key: '2', text: '已完成' },
  { key: '3', text: '已退款' },
  { key: '4', text: '已取消' }
]

const TYPE_TABS = [
  { key: '', text: '全部' },
  { key: '1', text: '车票' },
  { key: '2', text: '托运' }
]

Page({
  data: {
    adminRole: 0,           // 1=超管 2=普通管理员（控制退款按钮可见性）
    statusTabs: STATUS_TABS,
    typeTabs: TYPE_TABS,
    activeStatus: '',       // 当前选中的状态筛选（空字符串=全部）
    activeType: '',         // 当前选中的订单类型筛选
    keyword: '',            // 搜索关键字（订单号/手机号）
    inputFocused: '',       // 当前聚焦的输入框（黑色输入框聚焦变蓝）
    startDate: '',
    endDate: '',

    list: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: true,
    loading: false,
    loadingMore: false,
    showFilter: false,      // 高级筛选面板（日期范围）展开状态

    // 详情弹窗
    detailVisible: false,
    detailLoading: false,
    detail: null,           // { order, passengers }
    actionLoading: false,   // 防止状态更新/退款重复点击

    showRefundModal: false,
    refundReason: ''
  },

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    this.setData({ adminRole: userInfo.admin_role || 0 })
    this.loadList()
  },

  // 输入框聚焦/失焦（黑框点击变蓝）
  onFieldFocus(e) {
    this.setData({ inputFocused: e.currentTarget.dataset.field || '' })
  },
  onFieldBlur() {
    this.setData({ inputFocused: '' })
  },

  // 拉取订单列表
  loadList(append) {
    if (append) {
      if (this.data.loadingMore || !this.data.hasMore || this.data.loading) return
    } else {
      if (this.data.loading) return
    }
    const page = append ? this.data.page + 1 : 1
    if (append) this.setData({ loadingMore: true })
    else this.setData({ loading: true })

    const data = {
      page,
      page_size: this.data.pageSize
    }
    if (this.data.activeStatus) data.status = this.data.activeStatus
    if (this.data.activeType) data.order_type = this.data.activeType
    if (this.data.keyword) {
      // 后端按 order_no 模糊匹配；手机号字段单独传
      const kw = this.data.keyword.trim()
      if (/^\d+$/.test(kw) && kw.length >= 4) {
        data.contact_phone = kw
      } else {
        data.order_no = kw
      }
    }
    if (this.data.startDate) data.start_date = this.data.startDate
    if (this.data.endDate) data.end_date = this.data.endDate

    request({ url: '/api/wx/admin/orders', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(o => this.formatOrder(o))
      const list = append ? this.data.list.concat(items) : items
      this.setData({
        list,
        page,
        total,
        hasMore: list.length < total,
        loading: false,
        loadingMore: false
      })
    }).catch(() => {
      this.setData({
        loading: false,
        loadingMore: false,
        list: append ? this.data.list : []
      })
    })
  },

  // 列表项格式化（同时支持车票/托运两套状态文案）
  formatOrder(o) {
    const isCargo = o.order_type === 2
    const statusText = this.statusText(o.status, isCargo)
    const statusCls = this.statusCls(o.status)
    const fromName = o.from_station_name || (o.from_station && o.from_station.name) || '-'
    const toName = o.to_station_name || (o.to_station && o.to_station.name) || '-'
    const contact = isCargo ? (o.sender_name || '-') : (o.contact_name || '-')
    const phone = isCargo ? (o.sender_phone || '-') : (o.contact_phone || '-')
    const countText = isCargo
      ? (o.weight ? o.weight + 'kg' : '-')
      : (o.passenger_count + '人')
    return {
      id: o.id,
      order_no: o.order_no,
      order_type: o.order_type,
      typeText: isCargo ? '托运' : '车票',
      from: fromName,
      to: toName,
      route: fromName + ' → ' + toName,
      trip_date: o.trip_date || '',
      departure_time: o.departure_time || '',
      contact,
      phone,
      countText,
      total_price: (parseFloat(o.total_price) || 0).toFixed(2),
      status: o.status,
      statusText,
      statusCls,
      created_at: o.created_at || ''
    }
  },

  statusText(s, isCargo) {
    if (isCargo) {
      return { 0: '待支付', 1: '待运输', 2: '运输中', 3: '已到达', 4: '已取消', 5: '已取件' }[s] || '未知'
    }
    return { 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消', 5: '已核销' }[s] || '未知'
  },
  statusCls(s) {
    return { 0: 'pay', 1: 'travel', 2: 'done', 3: 'refund', 4: 'cancel', 5: 'done' }[s] || ''
  },

  // 状态/类型 tab 切换
  onStatusTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeStatus) return
    this.setData({ activeStatus: key })
    this.loadList(false)
  },
  onTypeTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeType) return
    this.setData({ activeType: key })
    this.loadList(false)
  },

  // 搜索
  onKeywordInput(e) {
    this.setData({ keyword: e.detail.value })
  },
  onSearchConfirm() {
    this.loadList(false)
  },
  onClearKeyword() {
    this.setData({ keyword: '' })
    this.loadList(false)
  },

  // 高级筛选（日期）
  toggleFilter() {
    this.setData({ showFilter: !this.data.showFilter })
  },
  onStartDateChange(e) {
    this.setData({ startDate: e.detail.value })
  },
  onEndDateChange(e) {
    this.setData({ endDate: e.detail.value })
  },
  onClearDate() {
    this.setData({ startDate: '', endDate: '' })
    this.loadList(false)
  },
  onApplyDate() {
    this.setData({ showFilter: false })
    this.loadList(false)
  },

  // 下拉刷新 / 上拉加载
  onPullDownRefresh() {
    this.loadList(false)
    wx.stopPullDownRefresh()
  },
  onLoadMore() {
    this.loadList(true)
  },

  // 点击订单 → 拉详情
  onOrderTap(e) {
    const id = e.currentTarget.dataset.id
    this.setData({ detailVisible: true, detailLoading: true, detail: null })
    request({ url: `/api/wx/admin/orders/${id}`, method: 'GET', silent: true }).then(res => {
      const d = res.data || {}
      const order = d.order || {}
      const passengers = (d.passengers || []).map(p => ({
        id: p.id,
        name: p.name || '-',
        id_card_no: p.id_card_no || '-',
        phone: p.phone || '-',
        seat_no: p.seat_no || '-',
        check_status: p.check_status,
        check_text: p.check_status === 1 ? '已核销' : '未核销'
      }))
      const isCargo = order.order_type === 2
      const detail = {
        id: order.id,
        order_no: order.order_no,
        order_type: order.order_type,
        typeText: isCargo ? '托运' : '车票',
        isCargo,
        from: order.from_station_name || (order.from_station && order.from_station.name) || '-',
        to: order.to_station_name || (order.to_station && order.to_station.name) || '-',
        trip_date: order.trip_date || '-',
        departure_time: order.departure_time || '-',
        route_name: order.route && order.route.name ? order.route.name : '',
        passenger_count: order.passenger_count,
        weight: order.weight,
        countText: isCargo ? (order.weight ? order.weight + 'kg' : '-') : (order.passenger_count + '人'),
        total_price: (parseFloat(order.total_price) || 0).toFixed(2),
        status: order.status,
        statusText: this.statusText(order.status, isCargo),
        statusCls: this.statusCls(order.status),
        // 联系人信息
        contact_name: order.contact_name || '-',
        contact_phone: order.contact_phone || '-',
        sender_name: order.sender_name || '-',
        sender_phone: order.sender_phone || '-',
        receiver_name: order.receiver_name || '-',
        receiver_phone: order.receiver_phone || '-',
        cargo_type: order.cargo_type || '',
        description: order.description || '',
        // 用户
        user_nickname: order.user && order.user.nickname ? order.user.nickname : '-',
        user_phone: order.user && order.user.phone ? order.user.phone : '-',
        // 时间
        pay_time: order.pay_time || '',
        pay_method: order.pay_method || '',
        created_at: order.created_at || '',
        // 可执行操作
        actions: this.availableActions(order.status, isCargo),
        canRefund: this.canRefund(order.status, isCargo)
      }
      this.setData({ detail, detailLoading: false, passengers })
    }).catch(() => {
      this.setData({ detailLoading: false })
      wx.showToast({ title: '加载详情失败', icon: 'none' })
    })
  },

  // 算这单还能做啥操作
  // 规则严格对齐后端 allowedTransitions（退款不在此处，走专用 Refund 接口）
  availableActions(status, isCargo) {
    const actions = []
    if (isCargo) {
      // 托运：0待支付→4已取消；1待运输→2运输中；2运输中→3已到达；3已到达→5已取件
      if (status === 0) actions.push({ to: 4, label: '取消订单', danger: true })
      if (status === 1) actions.push({ to: 2, label: '开始运输', danger: false })
      if (status === 2) actions.push({ to: 3, label: '确认到达', danger: false })
      if (status === 3) actions.push({ to: 5, label: '确认取件', danger: false })
    } else {
      // 车票：0待支付→4已取消；1待出行→2已完成
      if (status === 0) actions.push({ to: 4, label: '取消订单', danger: true })
      if (status === 1) actions.push({ to: 2, label: '标记完成', danger: false })
    }
    return actions
  },

  // 退款条件：仅超管 + (待出行/已完成 车票 或 待运输/运输中 托运)
  canRefund(status, isCargo) {
    if (this.data.adminRole !== 1) return false
    if (isCargo) return status === 1 || status === 2
    return status === 1 || status === 2
  },

  closeDetail() {
    this.setData({ detailVisible: false, detail: null })
  },
  preventBubble() {
    // 阻止冒泡（点击弹窗内容不关闭）
  },

  // 状态流转
  onStatusAction(e) {
    const to = Number(e.currentTarget.dataset.to)
    const label = e.currentTarget.dataset.label
    const id = this.data.detail.id
    const cur = this.data.detail
    wx.showModal({
      title: '确认操作',
      content: `确定将订单 ${cur.order_no} 标记为「${label}」吗？`,
      confirmText: '确定',
      success: r => {
        if (!r.confirm) return
        this.setData({ actionLoading: true })
        request({
          url: `/api/wx/admin/orders/${id}/status`,
          method: 'PUT',
          data: { status: to }
        }).then(() => {
          wx.showToast({ title: '操作成功', icon: 'success' })
          this.setData({ actionLoading: false, detailVisible: false })
          this.loadList(false)
        }).catch(() => {
          this.setData({ actionLoading: false })
        })
      }
    })
  },

  // 退款（超管）
  openRefundModal() {
    this.setData({ showRefundModal: true, refundReason: '' })
  },
  onReasonInput(e) {
    this.setData({ refundReason: e.detail.value })
  },
  cancelRefund() {
    this.setData({ showRefundModal: false })
  },
  confirmRefund() {
    const id = this.data.detail.id
    const reason = this.data.refundReason.trim()
    wx.showModal({
      title: '退款确认',
      content: '退款将原路退回用户账户，1-3个工作日到账。确定继续？',
      confirmText: '确定退款',
      confirmColor: '#f5222d',
      success: r => {
        if (!r.confirm) return
        this.setData({ actionLoading: true, showRefundModal: false })
        request({
          url: `/api/wx/admin/orders/${id}/refund`,
          method: 'POST',
          data: { reason }
        }).then(res => {
          wx.showToast({ title: res.message || '退款已提交', icon: 'none' })
          this.setData({ actionLoading: false, detailVisible: false })
          this.loadList(false)
        }).catch(() => {
          this.setData({ actionLoading: false })
        })
      }
    })
  },

  // 复制订单号
  onCopyOrderNo() {
    if (!this.data.detail) return
    wx.setClipboardData({
      data: this.data.detail.order_no,
      success: () => wx.showToast({ title: '已复制', icon: 'none' })
    })
  }
})
