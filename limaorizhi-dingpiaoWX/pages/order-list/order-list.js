const orderMixin = require('../../utils/order-page-mixin')

Page({
  data: Object.assign({}, orderMixin.data),

  onLoad(options) {
    // 接收来自「我的」页面的状态筛选参数
    if (options.status) {
      this.setData({ activeTab: options.status })
    }
  },

  // scroll-view 滚动到底部，加载更多
  loadMore() {
    this.loadOrders(true)
  },

  onShow() {
    this.checkLoginAndLoad()
  },

  // 从 mixin 引入公共方法
  checkLoginAndLoad: orderMixin.checkLoginAndLoad,
  loadTabsLayout: orderMixin.loadTabsLayout,
  loadOrders: orderMixin.loadOrders,
  onTabChange: orderMixin.onTabChange,
  filterOrders: orderMixin.filterOrders,
  goToDetail: orderMixin.goToDetail,
  onOrderAction: orderMixin.onOrderAction,
  onSecondaryAction: orderMixin.onSecondaryAction,
  goChangeTicket: orderMixin.goChangeTicket,
  payOrder: orderMixin.payOrder,
  cancelOrder: orderMixin.cancelOrder,
  refundOrder: orderMixin.refundOrder,
  onLongPressOrder: orderMixin.onLongPressOrder,
  confirmDeleteOrder: orderMixin.confirmDeleteOrder,
  cancelDeleteOrder: orderMixin.cancelDeleteOrder,
  goLogin: orderMixin.goLogin,
  onReachBottom: orderMixin.onReachBottom
})
