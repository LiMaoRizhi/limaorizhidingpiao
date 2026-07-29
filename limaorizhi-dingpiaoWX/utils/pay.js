// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
// 微信支付公共逻辑（order-page-mixin / trip-detail / order-detail 三处共用）
const { request } = require('./request')
var log = require('./log')
var subscribe = require('./subscribe')

/**
 * 发起微信支付流程
 * 统一处理：确认弹窗 → 订阅消息授权 → 请求后端支付参数 → 调用 wx.requestPayment
 *
 * @param {number} orderId - 订单ID
 * @param {object} opts - 可选回调
 * @param {string} opts.confirmTitle - 确认弹窗标题（默认"确认支付"）
 * @param {string} opts.confirmContent - 确认弹窗内容（默认"确认完成支付？"）
 * @param {string} opts.confirmText - 确认按钮文案（默认"确定"）
 * @param {string} opts.cancelText - 取消按钮文案（默认"取消"）
 * @param {string} opts.successText - 支付成功提示文案（默认"支付成功"）
 * @param {function} opts.onPaid - 支付成功回调
 * @param {function} opts.onFail - 支付取消/失败回调
 * @param {function} opts.onError - 支付参数异常或请求失败回调
 * @param {function} opts.onCancel - 用户取消支付确认弹窗回调
 */
function startPayment(orderId, opts) {
  opts = opts || {}
  wx.showModal({
    title: opts.confirmTitle || '确认支付',
    content: opts.confirmContent || '确认完成支付？',
    confirmText: opts.confirmText || '确定',
    cancelText: opts.cancelText || '取消',
    success: function (res) {
      if (!res.confirm) {
        if (opts.onCancel) opts.onCancel()
        return
      }
      // 先请求订阅消息授权（支付成功/发车/到达通知），授权后再发起支付
      subscribe.requestSubscribe(function () {
        wx.showLoading({ title: '发起支付...' })
        request({
          url: '/api/wx/orders/' + orderId + '/pay',
          method: 'POST'
        }).then(function (res) {
          wx.hideLoading()
          if (res.data && res.data.payment_params) {
            callWxPayment(res.data.payment_params, opts)
          } else {
            wx.showToast({ title: '支付参数异常', icon: 'none' })
            if (opts.onError) opts.onError()
          }
        }).catch(function (e) {
          log.error('发起支付失败', e)
          wx.hideLoading()
          if (opts.onError) opts.onError()
        })
      })
    }
  })
}

/**
 * 调用微信支付API
 */
function callWxPayment(params, opts) {
  wx.requestPayment({
    timeStamp: params.timeStamp,
    nonceStr: params.nonceStr,
    package: params.package,
    signType: params.signType,
    paySign: params.paySign,
    success: function () {
      wx.showToast({ title: opts.successText || '支付成功', icon: 'success' })
      if (opts.onPaid) opts.onPaid()
    },
    fail: function (err) {
      // 区分用户主动取消与真实支付错误
      var errMsg = (err && err.errMsg) || ''
      if (errMsg.indexOf('cancel') !== -1) {
        wx.showToast({ title: '支付已取消', icon: 'none' })
      } else {
        wx.showToast({ title: '支付失败，请重试', icon: 'none' })
      }
      if (opts.onFail) opts.onFail()
    }
  })
}

module.exports = { startPayment: startPayment }
