// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
// 订单操作公共逻辑（取消订单 / 申请退票退款，order-page-mixin 和 order-detail 共用）
const { request } = require('./request')
var log = require('./log')

/**
 * 取消订单（待支付状态）
 * @param {number} orderId - 订单ID
 * @param {function} onSuccess - 取消成功后的回调（如刷新列表/详情）
 */
function cancelOrder(orderId, onSuccess) {
  wx.showModal({
    title: '确认取消',
    content: '确认取消此订单？取消后不可恢复。',
    confirmColor: '#e64340',
    success: async function (res) {
      if (!res.confirm) return
      wx.showLoading({ title: '取消中...' })
      try {
        await request({ url: '/api/wx/orders/' + orderId + '/cancel', method: 'POST' })
        wx.hideLoading()
        wx.showToast({ title: '取消成功', icon: 'success' })
        if (onSuccess) onSuccess()
      } catch (e) {
        wx.hideLoading()
        log.error('取消订单失败', e)
      }
    }
  })
}

/**
 * 申请退票/退款（待出行/待运输状态）
 * 先获取系统配置展示手续费率和实际退款金额，确保用户知情后再确认
 * @param {number} orderId - 订单ID
 * @param {boolean} isCargo - 是否托运订单（true=退款, false=退票）
 * @param {function} onSuccess - 退票成功后的回调
 * @param {object} [orderInfo] - 订单信息（可选，含 total_price 用于计算退款金额）
 */
function refundOrder(orderId, isCargo, onSuccess, orderInfo) {
  var title = isCargo ? '申请退款' : '申请退票'
  var placeholder = isCargo ? '退款原因（选填）' : '退票原因（选填）'
  var defaultReason = isCargo ? '用户主动退款' : '用户主动退票'
  var successToast = isCargo ? '退款成功' : '退票成功'

  wx.showLoading({ title: '加载中...' })
  request({ url: '/api/wx/config', method: 'GET' }).then(function (res) {
    wx.hideLoading()
    var config = (res && res.data) || {}
    var feeRate = parseFloat(config.refund_fee_rate) || 0
    var beforeHours = parseInt(config.refund_before_departure_hours)
    var isNan = isNaN(beforeHours)
    if (isNan) { beforeHours = 2 }

    var content = ''
    var totalPrice = 0
    if (orderInfo && orderInfo.total_price != null) {
      totalPrice = parseFloat(orderInfo.total_price) || 0
    }
    if (feeRate > 0 && totalPrice > 0) {
      var fee = (totalPrice * feeRate / 100)
      var refundAmount = totalPrice - fee
      content = isCargo
        ? '确认申请退款？\n订单金额：¥' + totalPrice.toFixed(2) + '\n手续费率：' + feeRate + '%\n手续费：¥' + fee.toFixed(2) + '\n实际到账：¥' + refundAmount.toFixed(2)
        : '确认申请退票？\n订单金额：¥' + totalPrice.toFixed(2) + '\n手续费率：' + feeRate + '%\n手续费：¥' + fee.toFixed(2) + '\n实际到账：¥' + refundAmount.toFixed(2)
    } else if (feeRate > 0) {
      content = isCargo
        ? '确认申请退款？退款将扣除' + feeRate + '%手续费后退回原支付账户。'
        : '确认申请退票？退票将扣除' + feeRate + '%手续费后退回原支付账户。'
    } else {
      content = isCargo
        ? '确认申请退款？退款将全额退回原支付账户。'
        : '确认申请退票？退票将全额退回原支付账户。'
    }

    wx.showModal({
      title: title,
      content: content,
      editable: true,
      placeholderText: placeholder,
      success: async function (res) {
        if (!res.confirm) return
        wx.showLoading({ title: '处理中...' })
        try {
          await request({
            url: '/api/wx/orders/' + orderId + '/refund',
            method: 'POST',
            data: { reason: res.content || defaultReason }
          })
          wx.hideLoading()
          wx.showToast({ title: successToast, icon: 'success' })
          if (onSuccess) onSuccess()
        } catch (e) {
          wx.hideLoading()
          log.error('退款失败', e)
        }
      }
    })
  }).catch(function (e) {
    wx.hideLoading()
    log.error('获取退票配置失败', e)
    // 配置获取失败时降级为原逻辑（全额退回提示）
    var fallbackContent = isCargo
      ? '确认申请退款？退款将全额退回原支付账户。'
      : '确认申请退票？退票将全额退回原支付账户。'
    wx.showModal({
      title: title,
      content: fallbackContent,
      editable: true,
      placeholderText: placeholder,
      success: async function (res) {
        if (!res.confirm) return
        wx.showLoading({ title: '处理中...' })
        try {
          await request({
            url: '/api/wx/orders/' + orderId + '/refund',
            method: 'POST',
            data: { reason: res.content || defaultReason }
          })
          wx.hideLoading()
          wx.showToast({ title: successToast, icon: 'success' })
          if (onSuccess) onSuccess()
        } catch (e2) {
          wx.hideLoading()
          log.error('退款失败', e2)
        }
      }
    })
  })
}

/**
 * 隐藏/删除订单（软删除，所有状态均可操作）
 * 仅负责API调用，弹窗交互由页面自行管理
 * @param {number} orderId - 订单ID
 * @param {function} onSuccess - 删除成功后的回调
 */
function hideOrder(orderId, onSuccess) {
  wx.showLoading({ title: '删除中...' })
  request({ url: '/api/wx/orders/' + orderId + '/hide', method: 'POST' }).then(() => {
    wx.hideLoading()
    wx.showToast({ title: '删除成功', icon: 'success' })
    if (onSuccess) onSuccess()
  }).catch((e) => {
    wx.hideLoading()
    log.error('删除订单失败', e)
  })
}

module.exports = {
  cancelOrder: cancelOrder,
  refundOrder: refundOrder,
  hideOrder: hideOrder
}
