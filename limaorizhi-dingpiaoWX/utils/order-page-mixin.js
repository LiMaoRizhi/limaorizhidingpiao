// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('./log')
const { request } = require('./request')
const { formatOrderList, filterOrdersByTab } = require('./order-helper')
const { startPayment } = require('./pay')
const { cancelOrder: doCancelOrder, refundOrder: doRefundOrder, hideOrder: doHideOrder } = require('./order-action')

// 订单列表页公共逻辑（search.js 和 order-list.js 共用）
module.exports = {
  data: {
    activeTab: 'all',
    tabs: [
      { key: 'all', text: '全部' },
      { key: '0', text: '待支付' },
      { key: '1', text: '待出行' },
      { key: '2', text: '已完成' },
      { key: '3', text: '已退款' },
      { key: '4', text: '已取消' }
    ],
    orderList: [],
    filteredOrders: [],
    isLogin: false,
    loaded: false,
    currentPage: 1,
    pageSize: 20,
    hasMore: true,
    loadingMore: false,
    showDeleteModal: false,
    deleteTargetId: 0,
    deleteTipExtra: ''
  },

  // 检查登录状态并加载订单
  checkLoginAndLoad() {
    this.loadTabsLayout()
    const token = wx.getStorageSync('user_token')
    if (token) {
      this.setData({ isLogin: true, currentPage: 1, hasMore: true })
      this.loadOrders()
    } else {
      this.setData({ isLogin: false, orderList: [], filteredOrders: [], currentPage: 1, hasMore: true })
    }
  },

  // 加载订单页筛选标签布局（装修配置）
  loadTabsLayout() {
    request({ url: '/api/design/page-layout', method: 'GET', data: { page: 'order_tabs' } }).then(res => {
      const layout = res.data || []
      if (layout.length === 0) return
      const defaultTabs = [
        { key: 'all', text: '全部' },
        { key: '0', text: '待支付' },
        { key: '1', text: '待出行' },
        { key: '2', text: '已完成' },
        { key: '3', text: '已退款' },
        { key: '4', text: '已取消' }
      ]
      const map = {}
      defaultTabs.forEach(t => { map[t.key] = t })
      const newTabs = layout.filter(c => c.visible).map(c => map[c.type]).filter(Boolean)
      if (newTabs.length === 0) return
      const activeTab = newTabs.some(t => t.key === this.data.activeTab) ? this.data.activeTab : newTabs[0].key
      this.setData({ tabs: newTabs, activeTab })
      if (this.data.isLogin) this.filterOrders(activeTab)
    }).catch(() => {})
  },

  // 加载订单列表（支持分页）
  loadOrders(append) {
    if (append && (this.data.loadingMore || !this.data.hasMore)) return
    // 防止非append（刷新）重复触发竞态：旧请求返回覆盖新请求结果
    if (!append && this._ordersLoading) return
    this._ordersLoading = true
    const page = append ? this.data.currentPage + 1 : 1
    if (append) this.setData({ loadingMore: true })
    request({ url: '/api/wx/orders', method: 'GET', data: { page: page, page_size: this.data.pageSize } }).then(res => {
      this._ordersLoading = false
      const newOrders = formatOrderList(res.data.list)
      const allOrders = append ? this.data.orderList.concat(newOrders) : newOrders
      const totalCount = res.data.total || 0
      this.setData({
        orderList: allOrders,
        currentPage: page,
        hasMore: allOrders.length < totalCount,
        loadingMore: false
      })
      this.filterOrders(this.data.activeTab)
    }).catch((e) => {
      this._ordersLoading = false
      log.error('加载订单列表失败', e)
      // append 失败时仅停止加载状态，保留已加载的历史数据
      // 非 append（首次/刷新）失败时才清空列表
      if (append) {
        this.setData({ loadingMore: false })
      } else {
        this.setData({ orderList: [], filteredOrders: [], loadingMore: false })
      }
    })
  },

  // 上拉加载更多
  onReachBottom() {
    this.loadOrders(true)
  },

  onTabChange(e) {
    const key = e.currentTarget.dataset.key
    this.setData({ activeTab: key })
    this.filterOrders(key)
  },

  filterOrders(key) {
    const list = filterOrdersByTab(this.data.orderList, key)
    this.setData({ filteredOrders: list })
  },

  // 点击订单卡片进入详情页（所有状态均可进入，含待支付）
  goToDetail(e) {
    const id = e.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/order-detail/order-detail?id=${id}` })
  },

  onOrderAction(e) {
    const { id, status } = e.currentTarget.dataset
    if (status === '0') {
      this.payOrder(id)
    } else {
      wx.navigateTo({ url: `/pages/order-detail/order-detail?id=${id}` })
    }
  },

  // 支付订单
  payOrder(id) {
    startPayment(id, {
      onPaid: () => {
        // 延迟刷新订单，确保后端已处理支付回调，避免竞态条件导致订单状态不一致
        this._payRefreshTimer = setTimeout(() => { this.loadOrders() }, 1000)
      }
    })
  },

  onUnload() {
    clearTimeout(this._payRefreshTimer)
  },

  // 次要操作：取消订单 / 申请退票退款
  onSecondaryAction(e) {
    const { id, type, cargo } = e.currentTarget.dataset
    if (type === 'cancel') {
      this.cancelOrder(id)
    } else if (type === 'refund') {
      this.refundOrder(id, cargo === 'true' || cargo === true)
    }
  },

  // 取消订单（待支付状态）
  cancelOrder(id) {
    doCancelOrder(id, () => this.loadOrders())
  },

  // 申请退票/退款（待出行/待运输状态）
  // 从 orderList 中查找订单金额，传递给退票弹窗展示手续费明细
  refundOrder(id, isCargo) {
    var orderInfo = null
    var found = this.data.orderList.find(function (o) { return o.id == id })
    if (found) {
      orderInfo = { total_price: found.total_price }
    }
    doRefundOrder(id, isCargo, () => this.loadOrders(), orderInfo)
  },

  // 长按订单卡片 → 弹出删除确认弹窗
  onLongPressOrder(e) {
    const { id, status } = e.currentTarget.dataset
    var extra = ''
    if (String(status) === '1') {
      extra = '\n该订单待出行，请确认您已知晓此订单信息'
    }
    this.setData({ showDeleteModal: true, deleteTargetId: id, deleteTipExtra: extra })
  },

  // 确认删除订单
  confirmDeleteOrder() {
    doHideOrder(this.data.deleteTargetId, () => {
      this.setData({ showDeleteModal: false })
      this.loadOrders()
    })
  },

  // 取消删除
  cancelDeleteOrder() {
    this.setData({ showDeleteModal: false })
  },

  // 跳转登录
  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  }
}
