// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
const orderMixin = require('../../utils/order-page-mixin')

Page({
  data: Object.assign({}, orderMixin.data),

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 })
    }
    this.checkLoginAndLoad()
  },

  // 从 mixin 引入公共方法
  checkLoginAndLoad: orderMixin.checkLoginAndLoad,
  loadTabsLayout: orderMixin.loadTabsLayout,
  loadOrders: orderMixin.loadOrders,
  onReachBottom: orderMixin.onReachBottom,
  onTabChange: orderMixin.onTabChange,
  filterOrders: orderMixin.filterOrders,
  goToDetail: orderMixin.goToDetail,
  onOrderAction: orderMixin.onOrderAction,
  onSecondaryAction: orderMixin.onSecondaryAction,
  payOrder: orderMixin.payOrder,
  cancelOrder: orderMixin.cancelOrder,
  refundOrder: orderMixin.refundOrder,
  onLongPressOrder: orderMixin.onLongPressOrder,
  confirmDeleteOrder: orderMixin.confirmDeleteOrder,
  cancelDeleteOrder: orderMixin.cancelDeleteOrder,
  goLogin: orderMixin.goLogin,

  // 分享给好友
  onShareAppMessage() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行',
      path: '/pages/home/home'
    }
  },

  // 分享到朋友圈
  onShareTimeline() {
    return {
      title: '狸猫日志售票 · 在线订票便捷出行'
    }
  }
})
