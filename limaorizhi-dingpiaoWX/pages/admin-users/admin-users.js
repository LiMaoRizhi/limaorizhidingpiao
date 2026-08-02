// 管理后台 - 用户管理
// 接 /api/wx/admin/users(列表/详情) + /api/wx/admin/users/:id/status(封禁/解封)
const { request } = require('../../utils/request')

// 用户状态：1=正常 0=封禁 2=已注销
const STATUS_TABS = [
  { key: '', text: '全部' },
  { key: '1', text: '正常' },
  { key: '0', text: '封禁' },
  { key: '2', text: '已注销' }
]

Page({
  data: {
    statusTabs: STATUS_TABS,
    activeStatus: '',
    keyword: '',

    list: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: true,
    loading: false,
    loadingMore: false,

    detailVisible: false,
    detailLoading: false,
    detail: null,
    recentOrders: [],
    passengers: [],
    actionLoading: false
  },

  onLoad() {
    this.loadList()
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
    const kw = this.data.keyword.trim()
    if (kw) {
      // 纯数字按手机号搜，否则按昵称搜
      if (/^\d+$/.test(kw) && kw.length >= 4) {
        data.phone = kw
      } else {
        data.nickname = kw
      }
    }

    request({ url: '/api/wx/admin/users', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(u => this.formatUser(u))
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

  formatUser(u) {
    const statusText = { 0: '封禁', 1: '正常', 2: '已注销' }[u.status] || '未知'
    const statusCls = { 0: 'ban', 1: 'ok', 2: 'cancel' }[u.status] || ''
    return {
      id: u.id,
      nickname: u.nickname || '未设置',
      avatar_url: u.avatar_url || '',
      phone: u.phone || '-',
      status: u.status,
      statusText,
      statusCls,
      order_count: u.order_count || 0,
      total_amount: (u.total_amount || 0).toFixed(0),
      created_at: (u.created_at || '').replace('T', ' ').slice(0, 16)
    }
  },

  onStatusTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeStatus) return
    this.setData({ activeStatus: key })
    this.loadList(false)
  },

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

  onPullDownRefresh() {
    this.loadList(false)
    wx.stopPullDownRefresh()
  },
  onLoadMore() {
    this.loadList(true)
  },

  onUserTap(e) {
    const id = e.currentTarget.dataset.id
    this.setData({ detailVisible: true, detailLoading: true, detail: null })
    request({ url: `/api/wx/admin/users/${id}`, method: 'GET', silent: true }).then(res => {
      const d = res.data || {}
      const u = d.user || {}
      const detail = {
        id: u.id,
        nickname: u.nickname || '未设置',
        avatar_url: u.avatar_url || '',
        phone: u.phone || '-',
        status: u.status,
        statusText: { 0: '封禁', 1: '正常', 2: '已注销' }[u.status] || '未知',
        statusCls: { 0: 'ban', 1: 'ok', 2: 'cancel' }[u.status] || '',
        created_at: (u.created_at || '').replace('T', ' ').slice(0, 16),
        order_count: d.order_count || 0,
        total_amount: (d.total_amount || 0).toFixed(2),
        refund_count: d.refund_count || 0,
        cargo_count: d.cargo_count || 0,
        canBan: u.status === 1,
        canUnban: u.status === 0
      }
      const recentOrders = (d.orders || []).map(o => ({
        order_no: o.order_no,
        typeText: o.order_type === 2 ? '托运' : '车票',
        from: o.from_station_name || (o.from_station && o.from_station.name) || '-',
        to: o.to_station_name || (o.to_station && o.to_station.name) || '-',
        price: (o.total_price || 0).toFixed(2),
        statusText: { 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消' }[o.status] || '未知',
        statusCls: { 0: 'pay', 1: 'travel', 2: 'done', 3: 'refund', 4: 'cancel' }[o.status] || '',
        created_at: (o.created_at || '').replace('T', ' ').slice(0, 16)
      }))
      const passengers = (d.passengers || []).map(p => ({
        id: p.id,
        name: p.name || '-',
        id_card_no: p.id_card_no || '-',
        phone: p.phone || '-'
      }))
      this.setData({ detail, detailLoading: false, recentOrders, passengers })
    }).catch(() => {
      this.setData({ detailLoading: false })
      wx.showToast({ title: '加载详情失败', icon: 'none' })
    })
  },

  closeDetail() {
    this.setData({ detailVisible: false, detail: null })
  },
  preventBubble() {},

  // 封禁/解封
  onToggleStatus(e) {
    const action = e.currentTarget.dataset.action // ban / unban
    const cur = this.data.detail
    if (!cur) return
    const toStatus = action === 'ban' ? 0 : 1
    const label = action === 'ban' ? '封禁' : '解封'
    wx.showModal({
      title: '确认操作',
      content: `确定${label}用户「${cur.nickname}」吗？`,
      confirmText: '确定',
      confirmColor: action === 'ban' ? '#f5222d' : '#1a1a1a',
      success: r => {
        if (!r.confirm) return
        this.setData({ actionLoading: true })
        request({
          url: `/api/wx/admin/users/${cur.id}/status`,
          method: 'PUT',
          data: { status: toStatus }
        }).then(() => {
          wx.showToast({ title: `${label}成功`, icon: 'success' })
          this.setData({ actionLoading: false, detailVisible: false })
          this.loadList(false)
        }).catch(() => {
          this.setData({ actionLoading: false })
        })
      }
    })
  }
})
