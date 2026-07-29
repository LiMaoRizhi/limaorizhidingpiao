// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')

const { driverRequest: request } = require('../../utils/request')
const { formatArrivalTime } = require('../../utils/order-helper')

Page({
  data: {
    tripId: 0,
    trip: null,
    isTracking: false,
    lastLocation: null,
    reportCount: 0,
    statusText: '待开始',
    startBtnLoading: false,
    trackError: false, // 定位是否启动失败（控制状态点颜色）
    routeStations: [], // 站点序列
    currentPassedOrder: 0 // 当前已驶过到第几站
  },

  onLoad(options) {
    const tripId = options.id
    const app = getApp()
    if (tripId) {
      this.setData({ tripId: parseInt(tripId) })
      this.loadTripDetail(tripId)
    }
    // 从全局恢复定位状态（司机返回页面时能看到正在追踪）
    this.syncTrackingStatus()
  },

  onShow() {
    // 每次显示页面时同步全局追踪状态
    this.syncTrackingStatus()
    // 定时刷新状态显示（定位在后台持续上报，页面需同步显示最新计数）
    this._statusTimer = setInterval(() => {
      this.syncTrackingStatus()
    }, 3000)
  },

  onHide() {
    // 离开页面时清除定时器，但定位继续在后台运行
    if (this._statusTimer) {
      clearInterval(this._statusTimer)
      this._statusTimer = null
    }
  },

  // 从全局 app 实例同步定位追踪状态到页面
  syncTrackingStatus() {
    const app = getApp()
    const t = app.getDriverTrackingStatus()
    if (t.isTracking && t.tripId === this.data.tripId) {
      let statusText = `定位中 · 已上报${t.reportCount}次`
      let trackError = false
      // 定位启动失败时提示司机
      if (t.trackError && t.reportCount === 0) {
        statusText = t.trackError
        trackError = true
      }
      this.setData({
        isTracking: true,
        reportCount: t.reportCount,
        lastLocation: t.lastLocation,
        statusText,
        trackError
      })
    } else if (!t.isTracking && this.data.isTracking) {
      // 全局已停止追踪，同步页面状态
      this.setData({
        isTracking: false,
        statusText: '行程已结束'
      })
    }
  },

  // 加载班次详情（从司机今日班次列表中查找）
  loadTripDetail(id) {
    request({
      url: '/api/wx/driver/trips',
      method: 'GET'
    }).then(res => {
      const trips = res.data || []
      const trip = trips.find(t => t.id == id)
      if (trip) {
        const fromName = trip.route && trip.route.from_station ? trip.route.from_station.name : ''
        const toName = trip.route && trip.route.to_station ? trip.route.to_station.name : ''
        // 解析站点序列（每站预计算跨天到达文案）
        const routeStations = (trip.route && trip.route.route_stations)
          ? trip.route.route_stations.slice().sort((a, b) => a.stop_order - b.stop_order).map(s => ({
              ...s,
              arrival_text: formatArrivalTime(s.arrival_time, s.arrival_day_offset)
            }))
          : []
        this.setData({
          trip: { ...trip, fromName, toName },
          routeStations,
          currentPassedOrder: trip.current_passed_order || 0
        })
      }
    }).catch((e) => { log.error('加载司机班次详情失败', e) })
  },

  // 标记已驶过某站（点击站点列表项）
  handleMarkStation(e) {
    const order = e.currentTarget.dataset.order
    const { tripId, currentPassedOrder } = this.data
    // 已标记过的站点不重复标记
    if (order <= currentPassedOrder) return
    wx.showModal({
      title: '确认标记过站',
      content: `确认车辆已驶过第${order}站？标记后该站及之前站点的乘客将无法下单/支付。`,
      confirmColor: '#007aff',
      success: async (res) => {
        if (res.confirm) {
          try {
            await request({
              url: `/api/wx/driver/trips/${tripId}/mark-station`,
              method: 'PUT',
              data: { passed_order: order }
            })
            this.setData({ currentPassedOrder: order })
            wx.showToast({ title: '标记成功', icon: 'success' })
          } catch (e) {
            log.error('标记过站失败', e)
          }
        }
      }
    })
  },

  // 重置过站标记
  handleResetStation() {
    const { tripId } = this.data
    wx.showModal({
      title: '重置过站标记',
      content: '确认重置所有过站标记？重置后所有站点均可下单。',
      confirmColor: '#ff9500',
      success: async (res) => {
        if (res.confirm) {
          try {
            await request({
              url: `/api/wx/driver/trips/${tripId}/mark-station`,
              method: 'PUT',
              data: { passed_order: 0 }
            })
            this.setData({ currentPassedOrder: 0 })
            wx.showToast({ title: '已重置', icon: 'success' })
          } catch (e) {
            log.error('重置标记失败', e)
          }
        }
      }
    })
  },

  // 开始行程（司机点击按钮 → 班次状态改为已发车 → 启动全局定位上报）
  async handleStart() {
    const { tripId } = this.data
    this.setData({ startBtnLoading: true })
    try {
      // 1. 通知后端班次已发车
      await request({
        url: `/api/wx/driver/trips/${tripId}/start`,
        method: 'PUT'
      })
      this.setData({ startBtnLoading: false })
      // 2. 启动全局定位追踪（不随页面销毁而停止）
      const app = getApp()
      app.startDriverTracking(tripId)
      this.syncTrackingStatus()
    } catch (e) {
      this.setData({ startBtnLoading: false })
    }
  },

  // 结束行程（到站后司机手动点击 → 停止全局定位上报 → 通知后端清除位置记录）
  async handleStop() {
    wx.showModal({
      title: '确认结束行程',
      content: '结束后将停止车辆位置上报，乘客将无法再查看车辆位置',
      confirmColor: '#ff3b30',
      success: async (res) => {
        if (res.confirm) {
          const app = getApp()
          app.stopDriverTracking()
          // 通知后端结束行程，清除该班次位置记录
          try {
            await request({
              url: `/api/wx/driver/trips/${this.data.tripId}/end`,
              method: 'PUT'
            })
          } catch (e) {
            log.error('结束行程请求失败', e)
          }
          this.setData({ isTracking: false, statusText: '行程已结束' })
          wx.showToast({ title: '行程已结束', icon: 'success' })
        }
      }
    })
  },

  onUnload() {
    // 页面销毁时只清除定时器，不停止定位
    // 定位逻辑在 app.js 全局管理，页面销毁不影响定位持续运行
    if (this._statusTimer) {
      clearInterval(this._statusTimer)
      this._statusTimer = null
    }
  }
})
