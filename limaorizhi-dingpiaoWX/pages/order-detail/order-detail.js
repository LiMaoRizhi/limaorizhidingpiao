var log = require('../../utils/log')
const { request, BASE_URL } = require('../../utils/request')
const { formatArrivalTime, getStatusInfo } = require('../../utils/order-helper')
const { startPayment } = require('../../utils/pay')
const { formatDateTime } = require('../../utils/format')
const { cancelOrder: doCancelOrder, refundOrder: doRefundOrder, hideOrder: doHideOrder } = require('../../utils/order-action')

Page({
  data: {
    orderId: 0,
    order: null,
    passengers: [],
    statusText: '',
    statusColor: '',
    qrcodeUrl: '',
    qrcodeError: '',
    qrcodeLoading: false,
    showTrackBtn: false,
    trackBtnText: '查看车辆位置',
    needsReload: false,
    showDeleteModal: false,
    deleteTargetId: 0,
    deleteTipExtra: ''
  },

  onLoad(options) {
    const token = wx.getStorageSync('user_token')
    if (!token) {
      wx.navigateTo({ url: '/pages/login/login' })
      return
    }
    this.setData({ orderId: options.id })
    this.loadOrderDetail(options.id)
  },

  onShow() {
    // 仅在 needsReload 标记为 true 时刷新，避免每次返回都重复加载
    // 支付/取消/退款等操作后会设置 needsReload=true，确保返回时刷新状态
    if (this.data.orderId && this.data.needsReload) {
      this.setData({ needsReload: false })
      this.loadOrderDetail(this.data.orderId)
    }
  },

  loadOrderDetail(id) {
    request({ url: `/api/wx/orders/${id}`, method: 'GET' }).then(res => {
      const order = res.data.order
      const passengers = res.data.passengers || []
      
      const isCargo = order.order_type === 2
      const statusInfo = getStatusInfo(order.status, isCargo)
      const statusText = statusInfo.text
      const statusColor = statusInfo.color

      // 优先使用冗余站名（删除线路/站点后仍可显示），回退到关联站名
      const fromName = order.from_station_name || (order.from_station ? order.from_station.name : '')
      const toName = order.to_station_name || (order.to_station ? order.to_station.name : '')

      // 显示“查看车辆位置”按钮：班次未发车(status=1)或已发车(status=2)均可查看
      // bus-progress页面会根据有无GPS自动展示地图或占位提示
      const showTrackBtn = order.trip && (order.trip.status === 1 || order.trip.status === 2)
      const trackBtnText = order.trip && order.trip.status === 2 ? '查看车辆位置' : '查看到站进度'

      // 跨天到达时间（优先用订单冗余字段，回退到 trip 关联）
      const arrivalTime = order.arrival_time || (order.trip ? order.trip.arrival_time : '')
      const arrivalDayOffset = order.arrival_day_offset != null ? order.arrival_day_offset : (order.trip ? order.trip.arrival_day_offset : 0)
      const arrivalText = formatArrivalTime(arrivalTime, arrivalDayOffset)
      // 支付流水信息（后端订单详情新增 payment 字段，展示交易单号等）
      const paymentInfo = res.data.payment || { paid: false }
      if (paymentInfo.pay_time) {
        paymentInfo.pay_time_text = formatDateTime(paymentInfo.pay_time)
      }
      // 乘车座位文本（在二维码旁大字展示，方便老年人一眼看清）
      // 有座：3号座 / 3、4号座；全无座：无座；混合：3号座（含无座）
      const seatTypes = passengers.map(p => p.seat_type)
      const hasSeat = seatTypes.includes(0) && passengers.some(p => p.seat_type === 0 && p.seat_no)
      const allStanding = passengers.length > 0 && seatTypes.every(t => t === 1)
      let seatText = ''
      if (allStanding) {
        seatText = '无座'
      } else if (hasSeat) {
        const seatNos = passengers.filter(p => p.seat_type === 0 && p.seat_no).map(p => p.seat_no)
        seatText = seatNos.join('、') + '号座'
        if (seatTypes.includes(1)) seatText += '（含无座）'
      }
      this.setData({
        order: {
          ...order,
          fromName,
          toName,
          arrivalText,
          totalPriceText: (parseFloat(order.total_price) || 0).toFixed(2),
          insuranceFeeText: (parseFloat(order.insurance_fee) || 0).toFixed(2),
          pay_time: formatDateTime(order.pay_time),
          created_at: formatDateTime(order.created_at)
        },
        paymentInfo,
        passengers,
        seatText,
        statusText,
        statusColor,
        isCargo,
        showTrackBtn,
        trackBtnText
      })

      // 仅车票订单+待出行状态才请求二维码（已取消/已退款/已完成不生成核销码）
      if (!isCargo && order.status === 1) {
        this.loadQrcode(order.order_no)
      }
    }).catch((e) => { log.error('加载订单详情失败', e) })
  },

  // 去支付
  handlePay() {
    startPayment(this.data.orderId, {
      confirmContent: `支付金额 ¥${this.data.order.totalPriceText}，确认支付？`,
      onPaid: () => {
        // 支付回调要过几秒才到后端，立刻刷新还是"待支付"，轮询确认后再刷
        this._payPollTimes = 0
        this._pollOrderPaid()
      }
    })
  },

  // 轮询订单状态：每1秒查一次，确认已支付（status=1）就停，最多8次
  _pollOrderPaid() {
    var self = this
    if (self._payPollTimes >= 8) {
      self.loadOrderDetail(self.data.orderId)
      return
    }
    self._payPollTimes++
    request({ url: `/api/wx/orders/${self.data.orderId}`, method: 'GET' }).then(function (res) {
      if (res.data && res.data.order && res.data.order.status === 1) {
        self.setData({ needsReload: false })
        self.loadOrderDetail(self.data.orderId)
      } else {
        self._payPollTimer = setTimeout(function () { self._pollOrderPaid() }, 1000)
      }
    }).catch(function () {
      self.loadOrderDetail(self.data.orderId)
    })
  },

  onUnload() {
    clearTimeout(this._payPollTimer)
  },

  // 退单
  handleCancel() {
    doCancelOrder(this.data.orderId, () => this.loadOrderDetail(this.data.orderId))
  },

  // 申请退票/退款（传递订单金额用于展示手续费明细）
  handleRefund() {
    doRefundOrder(this.data.orderId, this.data.isCargo, () => this.loadOrderDetail(this.data.orderId), { total_price: this.data.order.total_price })
  },

  // 改签：跳转改签选班次页（返回后刷新状态）
  handleChange() {
    this.setData({ needsReload: true })
    wx.navigateTo({ url: `/pages/change-ticket/change-ticket?id=${this.data.orderId}` })
  },

  // 下载二维码图片（携带鉴权头）
  // 后端错误响应返回HTTP 200 + JSON，需验证下载的文件是否为有效PNG图片
  loadQrcode(orderNo) {
    this.setData({ qrcodeLoading: true, qrcodeError: '', qrcodeUrl: '' })
    const token = wx.getStorageSync('user_token')
    wx.downloadFile({
      url: `${BASE_URL}/api/wx/qrcode/${orderNo}`,
      header: { 'Authorization': token ? 'Bearer ' + token : '' },
      success: (res) => {
        if (res.statusCode !== 200) {
          this.setData({ qrcodeLoading: false, qrcodeError: '二维码加载失败' })
          return
        }
        // 验证下载的文件是否为有效PNG图片（防止JSON错误响应被当图片显示）
        const fs = wx.getFileSystemManager()
        fs.readFile({
          filePath: res.tempFilePath,
          position: 0,
          length: 4,
          success: (readRes) => {
            const bytes = new Uint8Array(readRes.data)
            // PNG文件头: 0x89 0x50 0x4E 0x47
            if (bytes.length >= 4 && bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4E && bytes[3] === 0x47) {
              this.setData({ qrcodeUrl: res.tempFilePath, qrcodeLoading: false, qrcodeError: '' })
            } else {
              // 不是PNG，读取完整内容解析错误信息
              fs.readFile({
                filePath: res.tempFilePath,
                encoding: 'utf8',
                success: (errRes) => {
                  let msg = '二维码加载失败'
                  try {
                    const errData = JSON.parse(errRes.data)
                    msg = errData.message || msg
                  } catch (e) { log.warn('解析二维码错误响应失败', e) }
                  this.setData({ qrcodeLoading: false, qrcodeError: msg })
                },
                fail: () => {
                  this.setData({ qrcodeLoading: false, qrcodeError: '二维码加载失败' })
                }
              })
            }
          },
          fail: () => {
            this.setData({ qrcodeLoading: false, qrcodeError: '二维码加载失败' })
          }
        })
      },
      fail: () => {
        this.setData({ qrcodeLoading: false, qrcodeError: '网络错误，二维码加载失败' })
      }
    })
  },

  // 看二维码
  previewQrcode() {
    if (this.data.qrcodeUrl) {
      wx.previewImage({
        urls: [this.data.qrcodeUrl]
      })
    }
  },

  // 复制单号
  copyOrderNo() {
    if (this.data.order) {
      wx.setClipboardData({
        data: this.data.order.order_no,
        success: () => {
          wx.showToast({ title: '已复制', icon: 'success' })
        }
      })
    }
  },

  // 联系司机
  callDriver(e) {
    const phone = e.currentTarget.dataset.phone
    if (!phone) {
      wx.showToast({ title: '暂无司机电话', icon: 'none' })
      return
    }
    wx.makePhoneCall({ phoneNumber: phone })
  },

  // 长按路线卡片 → 弹出删除确认弹窗
  onLongPressDelete(e) {
    const { id, status } = e.currentTarget.dataset
    var extra = ''
    if (String(status) === '1') {
      extra = '\n该订单待出行，请确认您已知晓此订单信息'
    }
    this.setData({ showDeleteModal: true, deleteTargetId: id, deleteTipExtra: extra })
  },

  // 确认删单
  confirmDelete() {
    doHideOrder(this.data.deleteTargetId, () => {
      this.setData({ showDeleteModal: false })
      wx.navigateBack()
    })
  },

  // 不删了
  cancelDelete() {
    this.setData({ showDeleteModal: false })
  },

  // 查看车辆位置
  handleViewTrack() {
    const tripId = this.data.order.trip_id
    if (tripId) {
      // 导航离开时标记需要重新加载，返回时刷新状态
      this.setData({ needsReload: true })
      wx.navigateTo({
        url: `/pages/bus-progress/bus-progress?id=${tripId}`
      })
    }
  },

  // 分享给好友（分享行程信息，好友点开可查看行程详情）
  onShareAppMessage() {
    var order = this.data.order
    if (order && order.fromName) {
      var title = '我的行程：' + order.fromName +
        '→' + (order.toName || '') +
        ' ' + (order.departure_time || '') +
        ' · 狸猫日志售票'
      return {
        title: title,
        path: '/pages/order-detail/order-detail?id=' + this.data.orderId
      }
    }
    return {
      title: '狸猫日志售票 · 在线订票便捷出行',
      path: '/pages/home/home'
    }
  }
})
