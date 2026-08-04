const orderMixin = require('../../utils/order-page-mixin')

Page({
  data: Object.assign({}, orderMixin.data),

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 })
    }
    this.checkLoginAndLoad()
  },

  checkLoginAndLoad: orderMixin.checkLoginAndLoad,
  loadTabsLayout: orderMixin.loadTabsLayout,
  loadOrders: orderMixin.loadOrders,
  onReachBottom: orderMixin.onReachBottom,
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

  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行',
      path: '/pages/home/home'
    }
  },

  onShareTimeline() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行'
    }
  }
})
